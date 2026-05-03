# Execution Layer

This directory contains the project execution layer.

It is intentionally separate from `sdk/`.

- `sdk/` is the low-level traceable Go port of the official Hyperliquid Python SDK.
- `execution/` contains CLI commands, safety flags, formatting, orchestration, and user-facing workflows built on top of `sdk/`.

Future work for the execution agent should happen here unless the user explicitly asks to change `sdk/`.

## Structure

```text
execution/
├── cmd/                 # User-facing CLI commands
└── internal/clientutil/ # Shared helpers for execution commands
```

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

Private key can be passed with:

```bash
export HYPERLIQUID_PRIVATE_KEY=0x...
```

Address can be passed with:

```bash
export HYPERLIQUID_ADDRESS=0x...
```
