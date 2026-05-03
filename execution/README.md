# Execution Layer

This directory contains the project execution layer.

It is intentionally separate from `sdk/`.

- `sdk/` is the low-level traceable Go port of the official Hyperliquid Python SDK.
- `execution/` contains CLI commands, safety flags, formatting, orchestration, and user-facing workflows built on top of `sdk/`.

Future work for the execution agent should happen here unless the user explicitly asks to change `sdk/`.

## Structure

```text
execution/
├── client/              # Reusable execution package for bots and agents
├── credentials/         # Execution adapter over the root secrets.Provider interface
├── cmd/                 # Thin user-facing CLI wrappers around client/
└── internal/clientutil/ # Shared helpers for execution commands
```

`client/` is the Go package to use from bot code. It owns the calls into
`sdk/` for balances, positions, open orders, order placement, cancel, and
modify operations.

`cmd/` is only for manual runs and smoke checks. Commands parse flags, build
`client` requests, and print responses.

## Secrets

Execution commands use `execution/credentials`, which adapts the root
`secrets.Provider` interface for Hyperliquid accounts.

Default behavior reads account `main` from environment variables:

```bash
export HYPERLIQUID_ADDRESS=0x...
export HYPERLIQUID_PRIVATE_KEY=0x...
export HYPERLIQUID_VAULT_ADDRESS=0x...
```

The default env mapping is:

```text
accounts/main.address       -> HYPERLIQUID_ADDRESS
accounts/main.private_key   -> HYPERLIQUID_PRIVATE_KEY
accounts/main.vault_address -> HYPERLIQUID_VAULT_ADDRESS
```

Common secret flags:

```bash
-secret-provider env|vault
-account main
-secret-prefix accounts
-vault-addr http://127.0.0.1:8200
-vault-token ...
-vault-mount secret
-vault-prefix hyperliquid
```

Explicit command flags such as `-address`, `-private-key`, and
`-vault-address` override values resolved from secrets.

## Commands

Balances and positions:

```bash
go run ./execution/cmd/balances -address 0x...
go run ./execution/cmd/perp-positions -address 0x...
```

Perp/futures orders:

```bash
go run ./execution/cmd/perp-open-orders -address 0x...
go run ./execution/cmd/perp-order -testnet -coin BTC -side buy -size 0.001 -price 25000 -confirm
go run ./execution/cmd/perp-cancel-order -testnet -coin BTC -oid 123 -confirm
go run ./execution/cmd/perp-cancel-cloid -testnet -coin BTC -cloid 0x00000000000000000000000000000001 -confirm
go run ./execution/cmd/perp-modify-order -testnet -coin BTC -oid 123 -side buy -size 0.001 -price 26000 -confirm
```

Spot orders:

```bash
go run ./execution/cmd/spot-open-orders -address 0x...
go run ./execution/cmd/spot-order -testnet -coin PURR/USDC -side buy -size 24 -price 0.5 -confirm
go run ./execution/cmd/spot-cancel-order -testnet -coin PURR/USDC -oid 123 -confirm
go run ./execution/cmd/spot-cancel-cloid -testnet -coin PURR/USDC -cloid 0x00000000000000000000000000000001 -confirm
go run ./execution/cmd/spot-modify-order -testnet -coin PURR/USDC -oid 123 -side buy -size 24 -price 0.5 -confirm
```

## Safety

Commands that change exchange state require `-confirm`.

Read-only commands only require an address. State-changing commands require a
private key.
