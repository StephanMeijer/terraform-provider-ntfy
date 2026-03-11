# Contributing

Thank you for your interest in contributing to this project.

## Development Setup

1. **Install Go** (version 1.21 or later)
2. **Clone the repository**
3. **Install dependencies**:
   ```bash
   go mod download
   ```
4. **Run tests**:
   ```bash
   go test ./...
   ```
5. **Build the provider**:
   ```bash
   go build
   ```

## Running Acceptance Tests

Acceptance tests require Docker and a running ntfy server:

```bash
# Start ntfy container
docker compose up -d
bash scripts/setup-test-ntfy.sh

# Run acceptance tests
TF_ACC=1 NTFY_URL=http://localhost:8080 NTFY_USERNAME=admin NTFY_PASSWORD=admin \
  go test -v -timeout 300s -run TestAcc ./internal/provider/

# Tear down
docker compose down -v

# Or use the justfile shortcut:
just test-acc
```

Required environment variables:
- `TF_ACC=1` — enables acceptance tests
- `NTFY_URL` — ntfy server URL (default: http://localhost:8080)
- `NTFY_USERNAME` — admin username (default: admin)
- `NTFY_PASSWORD` — admin password

## Generating Documentation

Documentation is auto-generated from schema annotations using terraform-plugin-docs:

```bash
go generate ./...
# or
just docs
```

After making schema changes, always regenerate docs and commit the updated `docs/` directory.

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature-name`)
3. Make your changes and add tests
4. Ensure all tests pass (`go test ./...`)
5. Commit your changes using [Conventional Commits](https://www.conventionalcommits.org/)
6. Push to your fork and submit a Pull Request
7. Ensure PR description clearly describes the problem and solution

## Code Style

- Follow the standard Go coding conventions
- Use `gofmt` for code formatting
- Add comments to exported functions and types
- Write unit tests for new functionality
- Ensure code passes `go vet` and `golangci-lint`

## Commit Messages

Use conventional commit format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `chore:` for maintenance tasks
- `test:` for test changes

## GPG Signing

Releases are signed with GPG for Terraform Registry publishing. To set up GPG signing:

### Generating a GPG Key

```bash
gpg --batch --gen-key <<EOF
Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: Stephan Meijer
Name-Email: your-email@example.com
Expire-Date: 0
%no-protection
%commit
EOF
```

### Exporting the Key

```bash
# Get the key fingerprint
gpg --list-secret-keys --keyid-format=long

# Export the private key (base64 encoded for GitHub secret)
gpg --armor --export-secret-keys YOUR_KEY_FINGERPRINT | base64 -w 0
```

### GitHub Secrets

Add the following secrets to the GitHub repository settings:

- `GPG_PRIVATE_KEY`: The base64-encoded private key from the export step
- `GPG_PASSPHRASE`: The passphrase for the GPG key (empty string if no passphrase)

The release workflow will automatically import the key and use it to sign release artifacts.
