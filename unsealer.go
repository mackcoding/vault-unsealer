package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	sdk "github.com/bitwarden/sdk-go"
	"github.com/hashicorp/go-hclog"
)

type Unsealer struct {
	logger       hclog.Logger
	client       *http.Client
	bw           sdk.BitwardenClientInterface
	keys         []string
	keysMu       sync.RWMutex
	vaults       []string
	attempts     int64
	successes    int64
	failures     int64
	working      sync.Map
	wg           sync.WaitGroup
	orgID        string
	token        string
	apiURL       string
	identityURL  string
	healthServer *http.Server
}

func main() {
	log := hclog.New(&hclog.LoggerOptions{Name: "vault-unsealer", Level: hclog.Info})

	vaultsRaw := strings.Split(getEnvRequired("VAULT_URLS"), ",")
	vaults := make([]string, 0, len(vaultsRaw))
	for _, v := range vaultsRaw {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			vaults = append(vaults, trimmed)
		}
	}
	if len(vaults) == 0 {
		log.Error("no valid vault URLs provided")
		os.Exit(1)
	}
	orgID := getEnvRequired("ORGANIZATION_ID")
	token := getEnvRequired("ACCESS_TOKEN")

	pollIntStr := getEnv("POLL_INTERVAL", "60s")
	pollInt, err := time.ParseDuration(pollIntStr)
	if err != nil {
		log.Warn("invalid POLL_INTERVAL, defaulting to 60s", "error", err)
		pollInt = 60 * time.Second
	}
	if pollInt < time.Second {
		log.Warn("POLL_INTERVAL too short, enforcing 1s minimum")
		pollInt = time.Second
	}

	verifyCert := getEnv("VERIFY_CERT", "true") == "true"

	u := &Unsealer{
		logger:      log,
		vaults:      vaults,
		orgID:       orgID,
		token:       token,
		apiURL:      apiURL,
		identityURL: identityURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				IdleConnTimeout:       90 * time.Second,
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: !verifyCert},
			},
		},
	}

	if err := u.initBitwardenClient(); err != nil {
		log.Error("bitwarden init failed", "error", err)
		os.Exit(1)
	}

	if err := u.fetchKeys(); err != nil {
		log.Error("failed to fetch keys", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	u.initHealthServer()
	go u.startHealthServer()
	go u.keyRefreshLoop(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(pollInt)
	defer ticker.Stop()

	u.unsealAll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info("waiting for in-flight unseals to complete")
			u.wg.Wait()
			log.Info("shutting down health server")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := u.healthServer.Shutdown(shutdownCtx); err != nil {
				log.Error("health server shutdown failed", "error", err)
			}
			return
		case <-sig:
			log.Info("shutting down")
			cancel()
			u.wg.Wait()
			log.Info("shutting down health server")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := u.healthServer.Shutdown(shutdownCtx); err != nil {
				log.Error("health server shutdown failed", "error", err)
			}
			return
		case <-ticker.C:
			u.unsealAll(ctx)
		}
	}
}

func (u *Unsealer) initBitwardenClient() error {
	var err error
	if u.apiURL != "" && u.identityURL != "" {
		u.bw, err = sdk.NewBitwardenClient(&u.apiURL, &u.identityURL)
	} else {
		u.bw, err = sdk.NewBitwardenClient(nil, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := u.bw.AccessTokenLogin(u.token, &u.orgID); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	return nil
}

func (u *Unsealer) fetchKeys() error {
	return u.doFetchKeys(true)
}

func (u *Unsealer) doFetchKeys(allowRelogin bool) error {
	// Note: Bitwarden SDK doesn't support context timeouts
	// If this hangs, the entire refresh loop blocks
	keys := make([]string, 0, 4)

	for i := 1; i <= 4; i++ {
		keyName := fmt.Sprintf("UNSEAL_KEY_%d", i)
		keyID := os.Getenv(keyName)
		if keyID == "" {
			return fmt.Errorf("environment variable %s not set", keyName)
		}

		secret, err := u.bw.Secrets().Get(keyID)
		if err != nil {
			if allowRelogin && (strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "auth")) {
				u.logger.Warn("authentication error detected, attempting re-login")
				if reloginErr := u.initBitwardenClient(); reloginErr != nil {
					return fmt.Errorf("re-login failed: %w", reloginErr)
				}
				return u.doFetchKeys(false)
			}
			return fmt.Errorf("failed to get key %d: %w", i, err)
		}
		if secret.Value == "" {
			return fmt.Errorf("empty value for key %d", i)
		}
		keys = append(keys, secret.Value)
	}

	u.keysMu.Lock()
	u.keys = keys
	u.keysMu.Unlock()

	u.logger.Info("loaded keys", "count", len(keys))
	return nil
}

func (u *Unsealer) keyRefreshLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			u.logger.Error("panic in key refresh loop", "panic", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
			if err := u.fetchKeys(); err != nil {
				u.logger.Error("key refresh failed", "error", err)
			} else {
				u.logger.Info("keys refreshed")
			}
		}
	}
}

func (u *Unsealer) unsealAll(ctx context.Context) {
	for _, vault := range u.vaults {
		u.wg.Add(1)
		go func(addr string) {
			defer u.wg.Done()
			u.unsealWithRetry(ctx, addr)
		}(vault)
	}
}

func (u *Unsealer) unsealWithRetry(ctx context.Context, addr string) {
	defer func() {
		if r := recover(); r != nil {
			u.logger.Error("panic in unseal retry", "vault", addr, "panic", r)
			atomic.AddInt64(&u.failures, 1)
		}
	}()

	if _, exists := u.working.LoadOrStore(addr, true); exists {
		u.logger.Debug("unseal already in progress for vault", "vault", addr)
		return
	}
	defer u.working.Delete(addr)

	backoff := time.Second
	for i := 0; i < 3; i++ {
		if err := u.unseal(ctx, addr); err == nil {
			return
		} else if i < 2 {
			u.logger.Warn("unseal attempt failed, retrying", "vault", addr, "attempt", i+1, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}
	atomic.AddInt64(&u.failures, 1)
}

func (u *Unsealer) unseal(ctx context.Context, addr string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", addr+"/v1/sys/health", nil)
	if err != nil {
		return fmt.Errorf("invalid vault URL: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	resp.Body.Close()

	switch resp.StatusCode {
	case 200, 429, 472, 473:
		return nil
	case 503:
		// Sealed, continue to unseal
	default:
		return fmt.Errorf("vault unhealthy, status code: %d", resp.StatusCode)
	}

	atomic.AddInt64(&u.attempts, 1)
	u.logger.Info("unsealing", "vault", addr)

	u.keysMu.RLock()
	keys := u.keys
	u.keysMu.RUnlock()

	for i, key := range keys {
		if i > 0 {
			req, err := http.NewRequestWithContext(ctx, "GET", addr+"/v1/sys/health", nil)
			if err == nil {
				resp, err := u.client.Do(req)
				if err == nil {
					resp.Body.Close()
					switch resp.StatusCode {
					case 200, 429, 472, 473:
						u.logger.Info("unsealed (quorum)", "vault", addr)
						atomic.AddInt64(&u.successes, 1)
						return nil
					}
				}
			}
		}

		data, err := json.Marshal(map[string]string{"key": key})
		if err != nil {
			u.logger.Warn("failed to marshal unseal request", "vault", addr, "error", err)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, "PUT", addr+"/v1/sys/unseal", bytes.NewBuffer(data))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := u.client.Do(req)
		if err != nil {
			continue
		}

		var result map[string]interface{}
		decodeErr := json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if decodeErr != nil {
			u.logger.Warn("bad response from vault", "vault", addr, "error", decodeErr)
			continue
		}

		if sealed, ok := result["sealed"].(bool); ok && !sealed {
			u.logger.Info("unsealed", "vault", addr)
			atomic.AddInt64(&u.successes, 1)
			return nil
		}
	}

	return fmt.Errorf("failed to unseal")
}

func (u *Unsealer) initHealthServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		u.keysMu.RLock()
		ready := len(u.keys) > 0
		u.keysMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		if !ready {
			w.WriteHeader(503)
		}
		json.NewEncoder(w).Encode(map[string]bool{"ready": ready})
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{
			"unseal_attempts":  atomic.LoadInt64(&u.attempts),
			"unseal_successes": atomic.LoadInt64(&u.successes),
			"unseal_failures":  atomic.LoadInt64(&u.failures),
		})
	})

	u.healthServer = &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func (u *Unsealer) startHealthServer() {
	defer func() {
		if r := recover(); r != nil {
			u.logger.Error("panic in health server", "panic", r)
		}
	}()

	u.logger.Info("health server starting", "addr", ":8080")
	if err := u.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		u.logger.Error("health server failed", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s not set", key))
	}
	return v
}
