# Hyperliquid Python SDK Go Port

This repository contains a traceable Go port of the official Hyperliquid Python SDK.

The Python SDK in `hyperliquid-python-sdk/` is the source of truth. The Go `sdk/` directory intentionally mirrors the Python package structure. Do not add a separate ergonomic facade. Do not rename concepts unless Go requires it.

## Rules

1. Keep `sdk/` structurally close to `hyperliquid-python-sdk/hyperliquid/`.
2. Every exported Go type or function should reference its upstream Python source in a comment.
3. Preserve exchange action payloads exactly.
4. Preserve msgpack and EIP-712 signing behavior exactly.
5. Signing, msgpack, order wire, and float formatting changes require golden tests.
6. Update `portmap.yaml` whenever upstream behavior is ported or intentionally left unported.

## Update Workflow

1. Pull or update `hyperliquid-python-sdk/`.
2. Diff upstream code:

   ```bash
   git -C hyperliquid-python-sdk diff OLD..NEW -- hyperliquid
   ```

3. Find affected entries in `portmap.yaml`.
4. Port behavior into the corresponding `sdk/` package.
5. Add or update golden tests for signing or wire-format behavior.
6. Run Go tests.

## Package Mapping

- `sdk/api` mirrors `hyperliquid/api.py`.
- `sdk/info` mirrors `hyperliquid/info.py`.
- `sdk/exchange` mirrors `hyperliquid/exchange.py`.
- `sdk/signing` mirrors `hyperliquid/utils/signing.py`.
- `sdk/types` mirrors `hyperliquid/utils/types.py`.
- `sdk/constants` mirrors `hyperliquid/utils/constants.py`.
- `sdk/websocket` mirrors `hyperliquid/websocket_manager.py`.

## Compatibility Priority

The highest priority is wire compatibility with the Python SDK. Idiomatic Go improvements are secondary and should not make upstream diffs harder to port.
