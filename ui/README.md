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
├── cmd/ui/       # HTTP server entrypoint
└── internal/app/ # Routes, handlers, templates, and form parsing
```

## Run

```bash
go run ./ui/cmd/ui
```

Optional environment variables:

```bash
export HYPERLIQUID_ADDRESS=0x...
export HYPERLIQUID_PRIVATE_KEY=0x...
export HYPERLIQUID_VAULT_ADDRESS=0x...
export HYPERLIQUID_BASE_URL=https://api.hyperliquid.xyz
```

State-changing actions require a confirmation checkbox in the UI. The private
key is read from `HYPERLIQUID_PRIVATE_KEY` or the `-private-key` flag and is not
rendered back into pages.
