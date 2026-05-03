# Hyperliquid Go SDK Port

This repository contains a traceable Go port of the official Hyperliquid Python SDK.

The goal is not to create a polished high-level Go wrapper. The goal is to keep a Go `sdk/` that maps clearly to the original Python code so that an AI agent or developer can port upstream Python SDK updates with minimal guesswork.

## Source of Truth

The official Python SDK is stored in:

```text
hyperliquid-python-sdk/
```

The Go port is stored in:

```text
sdk/
```

The Python SDK remains the upstream reference. When behavior is unclear, compare against `hyperliquid-python-sdk/hyperliquid`.

## Project Structure

```text
.
├── hyperliquid-python-sdk/      # Official upstream Python SDK clone
├── sdk/                         # Traceable Go SDK port
│   ├── api/                     # Mirrors hyperliquid/api.py
│   ├── constants/               # Mirrors hyperliquid/utils/constants.py
│   ├── exchange/                # Mirrors hyperliquid/exchange.py
│   ├── info/                    # Mirrors hyperliquid/info.py
│   ├── signing/                 # Mirrors hyperliquid/utils/signing.py
│   ├── types/                   # Mirrors hyperliquid/utils/types.py
│   └── websocket/               # Mirrors hyperliquid/websocket_manager.py
├── tests/golden/                # Golden compatibility fixtures
├── go.mod
├── go.sum
├── Makefile
├── PORTING.md
├── portmap.yaml
└── README.md
```

## Markdown Files

### `README.md`

This file. It explains the repository goal, structure, commands, and maintenance workflow.

### `PORTING.md`

Porting rules for humans and AI agents.

Use it when changing the Go SDK after the Python SDK changes. It explains the core rule: keep `sdk/` structurally close to the Python SDK and preserve wire/signing compatibility.

### `sdk/STATUS.md`

Current implementation status.

Use it to see which SDK parts are ported, which compatibility tests exist, and what the current known verification status is.

### `sdk/OPEN_QUESTIONS.md`

Production-readiness risks and remaining verification work.

Use it before real-money trading. It lists missing golden tests, integration test needs, websocket hardening questions, and other open risks.

### `portmap.yaml`

Machine-readable mapping from Python SDK symbols to Go SDK symbols.

This is the most important file for future AI agents. It answers: “which Go code corresponds to this Python file/function?”

Example:

```yaml
- python: hyperliquid/utils/signing.py::sign_l1_action
  go: sdk/signing/signing.go::SignL1Action
  status: ported
```

## Commands

Format Go code:

```bash
make fmt
```

Download/update Go dependencies:

```bash
make tidy
```

Run tests:

```bash
make test
```

Equivalent direct command:

```bash
go test ./...
```

## Compatibility Notes

Signed Hyperliquid actions are sensitive to exact payload encoding.

Python `dict` insertion order affects `msgpack.packb(action)`, and those bytes affect action hashes and signatures. For that reason, signed Go payloads must use:

```go
signing.OrderedMap
```

Do not replace signed action payloads with plain `map[string]any` unless the payload is not signed or order does not matter.

## Current State

The Go SDK compiles and core signing compatibility is tested against golden values from the official Python SDK.

Implemented packages include:

- `sdk/api`
- `sdk/constants`
- `sdk/types`
- `sdk/signing`
- `sdk/info`
- `sdk/exchange`
- `sdk/websocket`

Run:

```bash
make test
```

to verify the current state.

## Production Readiness

This repository is a strong SDK-porting base, but before using it for real-money trading, read:

```text
sdk/OPEN_QUESTIONS.md
```

The main remaining task is adding more golden tests for every exchange action builder used by the bot, plus testnet integration tests.

## Updating From Upstream Python SDK

1. Update `hyperliquid-python-sdk/`.
2. Diff upstream changes:

   ```bash
   git -C hyperliquid-python-sdk diff OLD..NEW -- hyperliquid
   ```

3. Find affected mappings in `portmap.yaml`.
4. Port behavior into `sdk/`.
5. Add or update golden tests.
6. Run:

   ```bash
   make fmt
   make test
   ```

7. Update `sdk/STATUS.md` and `sdk/OPEN_QUESTIONS.md` if the risk/status changed.
