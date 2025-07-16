// MIT License
//
// Copyright (c) 2024 Thomas Mack (https://github.com/mackcoding)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	sdk "github.com/bitwarden/sdk-go"
)

const (
	logDateFormat      = "1/2/2006 3:04 PM"
	maxURLLength       = 2048
	maxTokenLength     = 1024
	clientTimeout      = 30 * time.Second
	requiredUnsealKeys = 4
)

func main() {
	log("init", "Initializing unsealer...")

	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()

	apiUrl, identityUrl, vaultUrls, orgId, token, verifyCert := getVariables()

	bwClient, err := getClient(ctx, apiUrl, identityUrl, token, orgId)
	if err != nil {
		exitUnsealer("failed to initialize client: %v", err)
	}

	unsealKeys, err := getUnsealKeys(ctx, bwClient)
	if err != nil {
		exitUnsealer("failed to get unseal keys: %v", err)
	}

	log("init", "Successfully retrieved %d unseal keys", len(unsealKeys))
	unsealVault(unsealKeys, vaultUrls, verifyCert)

	os.Exit(0)
}

func validateURL(urlStr string) error {
	if len(urlStr) > maxURLLength {
		return fmt.Errorf("URL exceeds maximum length of %d characters", maxURLLength)
	}
	_, err := url.Parse(urlStr)
	return err
}

func validateVaultURLs(urls []string) error {
	for _, u := range urls {
		if err := validateURL(u); err != nil {
			return fmt.Errorf("invalid vault URL %s: %v", u, err)
		}
	}
	return nil
}

func getClient(ctx context.Context, apiUrl string, identityUrl string, token string, orgId string) (sdk.BitwardenClientInterface, error) {
	log("getClient", "creating new bitwarden client")
	bwClient, err := sdk.NewBitwardenClient(&apiUrl, &identityUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bitwarden client: %v", err)
	}

	err = bwClient.AccessTokenLogin(token, &orgId)
	if err != nil {
		return nil, fmt.Errorf("failed to login with access token: %v", err)
	}

	return bwClient, nil
}

func getVariables() (apiUrl string, identityUrl string, vaultUrls []string, orgId string, token string, verifyCert string) {
	apiUrl = getEnv("API_URL")
	if err := validateURL(apiUrl); err != nil {
		exitUnsealer("invalid API URL: %v", err)
	}

	identityUrl = getEnv("IDENTITY_URL")
	if err := validateURL(identityUrl); err != nil {
		exitUnsealer("invalid Identity URL: %v", err)
	}

	vaultUrlsStr := getEnv("VAULT_URLS")
	vaultUrls = strings.Split(vaultUrlsStr, ",")
	if err := validateVaultURLs(vaultUrls); err != nil {
		exitUnsealer("invalid Vault URLs: %v", err)
	}

	orgId = getEnv("ORGANIZATION_ID")
	if orgId == "" {
		exitUnsealer("organization ID cannot be empty")
	}

	token = getEnv("ACCESS_TOKEN")
	if len(token) > maxTokenLength {
		exitUnsealer("access token exceeds maximum length of %d characters", maxTokenLength)
	}

	verifyCert = getEnv("VERIFY_CERT")

	log("getVariables", "API_URL: %s", apiUrl)
	log("getVariables", "IDENTITY_URL: %s", identityUrl)
	log("getVariables", "VAULT_URLS: %v", vaultUrls)
	log("getVariables", "ORGANIZATION_ID: %s", orgId)
	log("getVariables", "ACCESS_TOKEN: [length=%d]", len(token))
	log("getVariables", "VERIFY_CERT: %s", verifyCert)
	return
}

func getUnsealKeys(ctx context.Context, bwClient sdk.BitwardenClientInterface) ([]string, error) {
	unsealKeys := make([]string, 0, requiredUnsealKeys)

	for i := 1; i <= requiredUnsealKeys; i++ {
		unsealKey := getEnv(fmt.Sprintf("UNSEAL_KEY_%d", i))

		key, err := bwClient.Secrets().Get(unsealKey)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve unseal key %d: %v", i, err)
		}

		if key.Value == "" {
			return nil, fmt.Errorf("empty unseal key received for key %d", i)
		}

		log("getUnsealKeys", "UNSEAL_KEY_%d: [length=%d]", i, len(key.Value))
		unsealKeys = append(unsealKeys, key.Value)
	}

	if len(unsealKeys) != requiredUnsealKeys {
		return nil, fmt.Errorf("incorrect number of unseal keys: got %d, want %d", len(unsealKeys), requiredUnsealKeys)
	}

	return unsealKeys, nil
}

func log(action string, format string, args ...interface{}) {
	date := time.Now().Format(logDateFormat)
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("(%s) [%s]: %s\n", date, action, msg)
}

func getEnv(name string) string {
	env := os.Getenv(name)
	if env == "" {
		exitUnsealer("required environment variable not set: %s", name)
	}
	return env
}

func exitUnsealer(format string, args ...interface{}) {
	log("FATAL", format, args...)
	os.Exit(1)
}

func unsealVault(keys []string, urls []string, verifyCert string) {
	log("unsealVault", "Unsealing vault...")

	skipVerify := strings.ToLower(verifyCert) == "false"

	client := &http.Client{
		Timeout: clientTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipVerify,
			},
		},
	}

	for _, addr := range urls {
		for i, key := range keys {
			payload := map[string]string{
				"key": key,
			}

			jsonData, err := json.Marshal(payload)
			if err != nil {
				exitUnsealer("failed to marshal unseal payload: %v", err)
			}

			req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/sys/unseal", addr), bytes.NewBuffer(jsonData))
			if err != nil {
				exitUnsealer("failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				exitUnsealer("failed to send unseal request to %s with key %d: %v", addr, i+1, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				exitUnsealer("unseal request %d failed with status: %s for node %s", i+1, resp.Status, addr)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				exitUnsealer("failed to decode response: %v", err)
			}

			if sealed, ok := result["sealed"].(bool); ok && !sealed {
				log("unsealVault", "node %s unsealed successfully", addr)
				break
			}
		}
	}

	log("unsealVault", "vault unsealing completed successfully")
}
