# UI Layer

This directory contains the project UI layer.

It is intentionally separate from `sdk/` and `execution/`.

- `sdk/` is the low-level traceable Go port of the official Hyperliquid Python SDK.
- `execution/` is the bot/client execution package built on top of `sdk/`.
- `ui/` is the browser interface built on top of `execution/client`.

The UI layer must not import `hyperliquid-bot/sdk/...` directly.

## Structure

```text
ui/
├── config.example.toml # Example UI config
├── cmd/ui/             # HTTP server entrypoint
├── internal/app/       # Routes, handlers, templates, and form parsing
└── internal/config/    # TOML config loader
```

## Run

```bash
go run ./ui/cmd/ui -config ui/config.toml
```

Without `-config`, the UI uses environment-compatible defaults.

Copy the example config first:

```bash
cp ui/config.example.toml ui/config.toml
```

## Secret Providers

The UI uses `execution/credentials` and supports the same providers:

```toml
[secrets]
provider = "env" # env, vault, or vault-userpass
account = "main"
prefix = "accounts"
```

For Vault token auth:

```toml
[secrets]
provider = "vault"

[secrets.vault]
addr = "http://127.0.0.1:8200"
token_env = "VAULT_TOKEN"
mount = "secret"
prefix = "hyperliquid"
```

For Vault userpass auth:

```toml
[secrets]
provider = "vault-userpass"

[secrets.vault_userpass]
addr = "http://127.0.0.1:8200"
username_env = "VAULT_USERNAME"
password_env = "VAULT_PASSWORD"
otp_env = "VAULT_OTP"
auth_mount = "userpass"
mount = "secret"
prefix = "hyperliquid"
```

For env provider, execution currently reads these fixed environment variables:

```bash
export HYPERLIQUID_ADDRESS=0x...
export HYPERLIQUID_PRIVATE_KEY=0x...
export HYPERLIQUID_VAULT_ADDRESS=0x...
```

State-changing actions require a confirmation checkbox in the UI. The private
key is resolved on the server through `execution/credentials` and is not
rendered back into pages.
