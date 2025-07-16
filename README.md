# Vault Unsealer

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An automated tool to unseal HashiCorp Vault instances using unseal keys stored in Bitwarden.

## Overview
This tool automates the Vault unsealing process by:
- Retrieving unseal keys from Bitwarden
- Applying them sequentially to specified Vault instances
- Verifying successful unsealing operations

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
| `VERIFY_CERT` | Enables cert verification, set to `false` when using self-signed certificates | `true` |

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
  -e VERIFY_CERT="true" \
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

## Security Considerations
- Access tokens and unseal keys are only stored in memory during execution
- All URLs must be HTTPS (validated during startup)
- The unsealer exits immediately after completion
- No sensitive data is written to logs (only lengths are logged)
- Environment variables are validated before use

## Limitations
- Currently supports exactly 4 unseal keys (not configurable)
- No automatic retry on failed unseal attempts
- Requires full access to Bitwarden secrets
- All vault instances must use the same unseal keys

## Contributing
Contributions are welcome! Please note:
- When reporting issues, do not include sensitive data like tokens or URLs
- Security-related issues should be reported privately
- Pull requests should not modify the core unsealing logic without discussion

## Version History
- 1.0.1: Certificate verification
  - Enables self-signed certificates
  - Adds a new variable, `VERIFY_CERT`

- 1.0.0: Initial release
  - Basic unsealing functionality
  - Environment variable configuration
  - Docker support

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
