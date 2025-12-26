# Vault Unsealer

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An automated daemon to monitor and unseal HashiCorp Vault instances using unseal keys stored in Bitwarden.

## Overview
This tool runs as a background service that automates the Vault unsealing process by:
- Securely retrieving unseal keys from Bitwarden into memory
- Continuously monitoring the health status of specified Vault instances
- Automatically applying unseal keys when a sealed node is detected
- Handling network interruptions and service restarts gracefully

## Contributing
This project was created as my first significant Go script, and I welcome contributions to help improve it. Here's how you can contribute:

### Pull Requests
- Fork the repository and create your feature branch from `main`
- Include detailed descriptions of your changes in the PR
- Update the README.md to reflect any new features, changes to environment variables, or modified behaviors
- Ensure your code follows the existing style and structure
- Add yourself to the contributors list if you'd like

### Issue Reporting
- Use the GitHub issue tracker
- For bugs, include:
  - Clear description of the issue
  - Steps to reproduce
  - Expected vs actual behavior
  - Environment details (Go version, OS, etc.)
- Do not include sensitive data like tokens, keys, or private URLs
- Security vulnerabilities should be reported privately

### Guidelines
- Keep changes focused and minimal
- Maintain the environment variable configuration approach
- Major architectural changes should be discussed in an issue first
- Test your changes thoroughly before submitting
- Document any new features or behavior changes

### Security
- Never commit sensitive data or credentials
- Do not add features that could compromise security
- Maintain the current security practices (in-memory only, no persistent storage)

I appreciate all contributions, whether they're bug fixes, documentation improvements, or feature additions. This is a learning experience for me as well, so constructive feedback is always welcome!

## Prerequisites
- Bitwarden organization account
- Access to Bitwarden API
- Vault unseal keys stored in Bitwarden
- Docker runtime environment

## Configuration

### Required Environment Variables
| Variable | Description | Example | Default |
|----------|-------------|---------|---------|
| `API_URL` | Bitwarden API endpoint | `https://api.bitwarden.com` | - |
| `IDENTITY_URL` | Bitwarden identity URL | `https://identity.bitwarden.com` | - |
| `VAULT_URLS` | Comma-separated Vault URLs | `https://vault1.example.com,https://vault2.example.com` | - |
| `ORGANIZATION_ID` | Bitwarden organization ID | `123e4567-e89b-12d3-a456-426614174000` | - |
| `ACCESS_TOKEN` | Bitwarden access token | `your_access_token` | - |
| `UNSEAL_KEY_1` | Bitwarden secret ID for first unseal key | `unseal-key-1` | - |
| `UNSEAL_KEY_2` | Bitwarden secret ID for second unseal key | `unseal-key-2` | - |
| `UNSEAL_KEY_3` | Bitwarden secret ID for third unseal key | `unseal-key-3` | - |
| `UNSEAL_KEY_4` | Bitwarden secret ID for fourth unseal key | `unseal-key-4` | - |
| `VERIFY_CERT` | Enables cert verification, set to `false` when using self-signed certificates | `true` | `true` |
| `POLL_INTERVAL` | Frequency to check Vault health status | `60s` | `60s` |

## Usage

### Building the Container
```bash
docker build -t vault-unsealer .
```

### Running the Unsealer (Docker)
```bash
docker run -d --restart always --name vault-unsealer \
  -p 8080:8080 \
  -e API_URL="https://api.bitwarden.com" \
  -e IDENTITY_URL="https://identity.bitwarden.com" \
  -e VAULT_URLS="https://vault1.example.com,https://vault2.example.com" \
  -e ORGANIZATION_ID="your_org_id" \
  -e ACCESS_TOKEN="your_access_token" \
  -e UNSEAL_KEY_1="secret_id_1" \
  -e UNSEAL_KEY_2="secret_id_2" \
  -e UNSEAL_KEY_3="secret_id_3" \
  -e UNSEAL_KEY_4="secret_id_4" \
  -e VERIFY_CERT="true" \
  -e POLL_INTERVAL="30s" \
  vault-unsealer
```

### Kubernetes Deployment
This daemon is designed to run in Kubernetes. Below is an example deployment configuration.

**1. Create a Secret for credentials:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vault-unsealer-config
type: Opaque
stringData:
  ORGANIZATION_ID: "your_org_id"
  ACCESS_TOKEN: "your_access_token"
  UNSEAL_KEY_1: "secret_id_1"
  UNSEAL_KEY_2: "secret_id_2"
  UNSEAL_KEY_3: "secret_id_3"
  UNSEAL_KEY_4: "secret_id_4"
```

**2. Deploy the Unsealer:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vault-unsealer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vault-unsealer
  template:
    metadata:
      labels:
        app: vault-unsealer
    spec:
      containers:
      - name: unsealer
        image: your-registry/vault-unsealer:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: API_URL
          value: "https://api.bitwarden.com"
        - name: IDENTITY_URL
          value: "https://identity.bitwarden.com"
        - name: VAULT_URLS
          value: "https://vault1.example.com,https://vault2.example.com"
        - name: POLL_INTERVAL
          value: "30s"
        envFrom:
        - secretRef:
            name: vault-unsealer-config
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 10
```

## Health & Monitoring

The daemon exposes an HTTP server on port `8080` to provide health status and operational metrics.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | `GET` | Returns `200 OK` ("status": "ok") if the service is running. |
| `/ready` | `GET` | Returns `200 OK` if unseal keys are successfully loaded in memory. Returns `503` if keys are missing. |
| `/metrics` | `GET` | Returns JSON statistics about unseal operations. |

**Example Metrics Response:**
```json
{
  "unseal_attempts": 42,
  "unseal_successes": 2,
  "unseal_failures": 0
}
```

## Technical Specifications

### System Constraints
- Maximum URL length: 2048 characters
- Maximum token length: 1024 characters
- Client timeout: 30 seconds
- Required unseal keys: 4

### Security Features
- No unseal keys stored in container filesystem
- Secure retrieval from Bitwarden into memory
- Environment variable validation
- Graceful shutdown handling

### Error Handling
The system implements comprehensive error handling for:
- URL validation
- Environment variable verification
- Unseal key validation
- Network connectivity issues
- Vault health check failures
- Infinite recursion protection during auth failures

### Logging
The unsealer provides structured logging for:
- Service initialization and configuration
- Key retrieval status
- Polling activities
- Unsealing attempts and results
- Error conditions

## Version History
- 1.1.0: Daemon mode
  - Continuous monitoring and unsealing
  - Configurable polling interval
  - Structured logging
  - Graceful shutdown
  - Health and Metrics endpoints

- 1.0.1: Certificate verification
  - Enables self-signed certificates
  - Adds a new variable, `VERIFY_CERT`

- 1.0.0: Initial release
  - Basic unsealing functionality
  - Environment variable configuration
  - Docker support

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.