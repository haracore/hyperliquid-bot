# Secrets Layer

This directory contains the reusable secrets layer.

It is intentionally separate from `sdk/`, `execution/`, and `ui/`.

Other modules should depend on this package through the `Provider` interface
instead of reading environment variables or Vault directly.

## Design

```text
secrets.Provider
├── EnvProvider      # Reads secrets from environment variables
└── VaultProvider    # Reads HashiCorp Vault KV v2 secrets over HTTP
```

The generic primitive is a secret bundle:

```go
bundle, err := provider.Get(ctx, secrets.Ref{Path: "accounts/main"})
privateKey := bundle.Get("private_key")
```

For Hyperliquid account-like secrets, use `AccountResolver`:

```go
resolver := secrets.NewAccountResolver(provider, "accounts")
account, err := resolver.Account(ctx, "main")
```

Expected account fields:

```text
address
private_key
vault_address
```

Only `private_key` is required by `AccountResolver`.

## Env Provider

```go
provider := secrets.NewEnvProvider(secrets.EnvConfig{
    PathFields: map[string]secrets.EnvFields{
        "accounts/main": {
            "address":       "HYPERLIQUID_ADDRESS",
            "private_key":   "HYPERLIQUID_PRIVATE_KEY",
            "vault_address": "HYPERLIQUID_VAULT_ADDRESS",
        },
    },
})
```

## Vault Provider

This implementation targets HashiCorp Vault KV v2.

```go
provider := secrets.NewVaultProvider(secrets.VaultConfig{
    Address: "http://127.0.0.1:8200",
    Token:   os.Getenv("VAULT_TOKEN"),
    Mount:   "secret",
    Prefix:  "hyperliquid",
})
```

The call below reads:

```text
GET /v1/secret/data/hyperliquid/accounts/main
```

```go
bundle, err := provider.Get(ctx, secrets.Ref{Path: "accounts/main"})
```

## Local Vault

For local development, start a Vault dev server:

```bash
docker compose -f secrets/docker-compose.vault.yml up -d
```

Use the dev root token:

```bash
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=dev-root-token
```

Write a Hyperliquid account secret:

```bash
docker compose -f secrets/docker-compose.vault.yml exec vault \
  vault kv put secret/hyperliquid/accounts/main \
  address=0x... \
  private_key=0x... \
  vault_address=0x...
```

Read it through the CLI:

```bash
go run ./secrets/cmd/secrets account \
  -provider vault \
  -vault-mount secret \
  -vault-prefix hyperliquid \
  -account main
```

Stop the local Vault:

```bash
docker compose -f secrets/docker-compose.vault.yml down
```

The compose file uses Vault dev mode. It is convenient for local checks, but it
is not a production Vault setup.

## CLI

Run the secrets CLI with:

```bash
go run ./secrets/cmd/secrets --help
```

Read a Hyperliquid account from environment variables:

```bash
go run ./secrets/cmd/secrets account -provider env -account main
```

By default the CLI redacts sensitive fields such as `private_key`. Use
`-reveal` only when you intentionally need to print the full value:

```bash
go run ./secrets/cmd/secrets account -provider env -account main -reveal
```

Read a generic bundle from environment variables:

```bash
go run ./secrets/cmd/secrets get \
  -provider env \
  -path accounts/main \
  -env-field address=HYPERLIQUID_ADDRESS \
  -env-field private_key=HYPERLIQUID_PRIVATE_KEY
```

Read an account from Vault:

```bash
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=...

go run ./secrets/cmd/secrets account \
  -provider vault \
  -vault-mount secret \
  -vault-prefix hyperliquid \
  -account main
```

This is the old token-based mode. It keeps working as long as `VAULT_TOKEN`
contains a token with read access.

## Vault Userpass Login

The CLI also supports login-based access. In this mode it logs in through
Vault's `userpass` auth method, receives a short-lived client token, then reads
the secret with that token:

```bash
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_USERNAME=trader
export VAULT_PASSWORD=...

go run ./secrets/cmd/secrets account \
  -provider vault-userpass \
  -vault-mount secret \
  -vault-prefix hyperliquid \
  -account main
```

If Vault Login MFA is enforced on the userpass auth mount, provide the MFA value
expected by Vault:

```bash
go run ./secrets/cmd/secrets account \
  -provider vault-userpass \
  -vault-username trader \
  -vault-password '...' \
  -vault-mfa 'totp-method-id-or-name:123456' \
  -vault-prefix hyperliquid \
  -account main
```

Or split method and one-time passcode:

```bash
go run ./secrets/cmd/secrets account \
  -provider vault-userpass \
  -vault-username trader \
  -vault-password '...' \
  -vault-mfa-method totp-method-id-or-name \
  -vault-otp 123456 \
  -vault-prefix hyperliquid \
  -account main
```

You can also login only and export the returned token:

```bash
go run ./secrets/cmd/secrets login-vault \
  -vault-username trader \
  -vault-password '...' \
  -vault-mfa 'totp-method-id-or-name:123456' \
  -reveal
```

Vault's built-in Login MFA works on auth-method login. It does not wrap the
token auth method itself, so this mode exists alongside the old token-based
mode rather than replacing it.

This reads:

```text
GET /v1/secret/data/hyperliquid/accounts/main
```

Read a specific Vault KV v2 version:

```bash
go run ./secrets/cmd/secrets get \
  -provider vault \
  -path accounts/main \
  -version 3
```

## Ownership

Agents working on `ui/` or `execution/` can import this package later. This
package must not import `sdk/`, `execution/`, or `ui/`.
