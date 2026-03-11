# Terraform Provider for ntfy

[![Tests](https://github.com/StephanMeijer/terraform-provider-ntfy/actions/workflows/test.yml/badge.svg)](https://github.com/StephanMeijer/terraform-provider-ntfy/actions/workflows/test.yml)

A custom [OpenTofu](https://opentofu.org/) / [Terraform](https://www.terraform.io/) provider for managing [ntfy](https://ntfy.sh/) server resources.

## Installation

### Terraform Registry

```hcl
terraform {
  required_providers {
    ntfy = {
      source  = "stephanmeijer/ntfy"
      version = "~> 0.1.0"
    }
  }
}
```

### Manual Installation

Download the appropriate binary from the [releases page](https://github.com/StephanMeijer/terraform-provider-ntfy/releases) and place it in your Terraform plugins directory.

## Resources

- **`ntfy_user`** - Creates and manages ntfy users.
  - `username` (Required): The username for the ntfy user.
  - `password` (Computed, Sensitive): The auto-generated password for the ntfy user.
  - `role` (Computed): The role of the user (user or admin).
  - `tier` (Optional, Computed): The tier to assign to the user.
  - `id` (Computed): The unique identifier for this user (same as username).
- **`ntfy_access`** - Manages topic access permissions (ACLs) for users.
  - `username` (Required): The username to grant access to.
  - `topic` (Required): The topic to grant access to.
  - `permission` (Required): The permission level: `read-write`, `read-only`, `write-only`, or `deny`.
  - `id` (Computed): The unique identifier for this access grant (format: username/topic).
- **`ntfy_token`** - Issues API tokens using user credentials.
  - `username` (Required): The username to create the token for.
  - `password` (Required, Sensitive): The password of the user to create the token for.
  - `label` (Optional): An optional label for the token.
  - `expires` (Optional): Token expiry as Unix timestamp. 0 means no expiry.
  - `token` (Computed, Sensitive): The generated API token.
  - `id` (Computed): The unique identifier for this token (same as the token value).

## Data Sources

- **`ntfy_user`** - Reads an existing ntfy user.
  - `username` (Required): The username to look up.
  - `role` (Computed): The role of the user (user or admin).
  - `id` (Computed): The unique identifier for this user (same as username).
- **`ntfy_access`** - Reads an existing access grant.
  - `username` (Required): The username to look up.
  - `topic` (Required): The topic to look up.
  - `permission` (Computed): The permission level: `read-write`, `read-only`, `write-only`, or `deny`.
  - `id` (Computed): The unique identifier (format: username/topic).

## Import

Resources can be imported using the following commands:

```bash
# Import a user (note: password cannot be recovered from API)
terraform import ntfy_user.example myuser

# Import an access grant
terraform import ntfy_access.example myuser/alerts

# ntfy_token does not support import
```

## Usage Example

```hcl
resource "ntfy_user" "example" {
  username = "myuser"
  tier     = "default"  # optional
}

resource "ntfy_access" "example" {
  username   = ntfy_user.example.username
  topic      = "alerts"
  permission = "read-write"
}

resource "ntfy_token" "example" {
  username = ntfy_user.example.username
  password = ntfy_user.example.password
  label    = "automation"
  expires  = 0
}

data "ntfy_user" "existing" {
  username = "existinguser"
}

data "ntfy_access" "existing" {
  username = "existinguser"
  topic    = "alerts"
}
```

## Provider Configuration

| Attribute  | Type   | Required | Default              | Env Var        | Description                  |
|------------|--------|----------|----------------------|----------------|------------------------------|
| `url`      | String | No       | `http://localhost:80` | `NTFY_URL`     | ntfy server URL              |
| `username` | String | No       | `admin`              | `NTFY_USERNAME` | Admin username               |
| `password` | String | No       | -                    | `NTFY_PASSWORD` | Admin password (sensitive)   |

Environment variables are read first; explicit config values override them.

## Testing

### Unit Tests
Run unit tests (no Docker required):
```bash
go test -v ./...
# or
just test
```

### Acceptance Tests
Acceptance tests require a running ntfy server (Docker):
```bash
# Start ntfy and run acceptance tests
just test-acc

# Or manually:
docker compose up -d
bash scripts/setup-test-ntfy.sh
TF_ACC=1 NTFY_URL=http://localhost:8080 NTFY_USERNAME=admin NTFY_PASSWORD=admin \
  go test -v -timeout 300s -run TestAcc ./...
docker compose down -v
```

## Building

```bash
make build
```

## Installing for Development

```bash
make install
```

This builds the provider and moves it to `~/go/bin/`.

## Development Setup

Create a `.tofurc` (or `.terraformrc`) file to use the local build:

```hcl
provider_installation {
  dev_overrides {
    "stephanmeijer/ntfy" = "/home/steve/go/bin"
  }
  direct {}
}
```

> **Note:** The `direct {}` block is required.

Set the environment variable to point to this file:

```bash
export TF_CLI_CONFIG_FILE="$HOME/.tofurc"
```
