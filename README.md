# Vault Unsealer

An automated tool to unseal HashiCorp Vault instances using unseal keys stored in Bitwarden.

## Overview

This tool automates the Vault unsealing process by:
- Retrieving unseal keys from Bitwarden
- Applying them sequentially to specified Vault instances
- Verifying successful unsealing operations

## Prerequisites

- Bitwarden organization account
- Access to Bitwarden API
- Vault unseal keys stored in Bitwarden
- Docker runtime environment

## Configuration

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `API_URL` | Bitwarden API endpoint | `https://api.bitwarden.com` |
| `IDENTITY_URL` | Bitwarden identity URL | `https://identity.bitwarden.com` |
| `VAULT_URLS` | Comma-separated Vault URLs | `https://vault1.example.com,https://vault2.example.com` |
| `ORGANIZATION_ID` | Bitwarden organization ID | `123e4567-e89b-12d3-a456-426614174000` |
| `ACCESS_TOKEN` | Bitwarden access token | `your_access_token` |
| `UNSEAL_KEY_1` | Bitwarden secret ID for first unseal key | `unseal-key-1` |
| `UNSEAL_KEY_2` | Bitwarden secret ID for second unseal key | `unseal-key-2` |
| `UNSEAL_KEY_3` | Bitwarden secret ID for third unseal key | `unseal-key-3` |
| `UNSEAL_KEY_4` | Bitwarden secret ID for fourth unseal key | `unseal-key-4` |

## Usage

### Building the Container

```bash
docker build -t vault-unsealer .
```

### Running the Unsealer

```bash
docker run --rm \
  -e API_URL="https://api.bitwarden.com" \
  -e IDENTITY_URL="https://identity.bitwarden.com" \
  -e VAULT_URLS="https://vault1.example.com,https://vault2.example.com" \
  -e ORGANIZATION_ID="your_org_id" \
  -e ACCESS_TOKEN="your_access_token" \
  -e UNSEAL_KEY_1="secret_id_1" \
  -e UNSEAL_KEY_2="secret_id_2" \
  -e UNSEAL_KEY_3="secret_id_3" \
  -e UNSEAL_KEY_4="secret_id_4" \
  vault-unsealer
```

## Technical Specifications

### System Constraints

- Maximum URL length: 2048 characters
- Maximum token length: 1024 characters
- Client timeout: 30 seconds
- Required unseal keys: 4

### Security Features

- No unseal keys stored in container
- Secure retrieval from Bitwarden
- Environment variable validation
- Automatic cleanup after execution

### Error Handling

The system implements comprehensive error handling for:
- URL validation
- Environment variable verification
- Unseal key validation
- Unsealing operation monitoring

### Logging

The unsealer provides detailed logging with timestamps for:
- Initialization status
- Environment variable validation
- Key retrieval confirmation
- Unsealing progress
- Success/failure status
