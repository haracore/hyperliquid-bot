# SDK Port Status

This directory is a traceable Go port of `hyperliquid-python-sdk/hyperliquid`.

Current status:

- `sdk/constants`: initial port complete.
- `sdk/types`: initial `Cloid`, metadata, spot metadata, and builder types ported.
- `sdk/api`: initial HTTP POST and error behavior ported.
- `sdk/signing`: float helpers, ordered msgpack payloads, action hash, phantom agent, L1 signing, user-signed transaction signing wrappers, order wire, and order action helpers ported.
- `sdk/info`: most `/info` request methods ported as direct one-liners with dynamic `out any` responses. Metadata initialization is available through `NewInitialized`/`Initialize`, and websocket remapping is wired through `Subscribe`/`Unsubscribe`.
- `sdk/exchange`: broad action-method port is present, including order/cancel/modify/leverage/margin/referrer/subaccounts/transfers/agent/builder/multisig conversion/deploy/validator/abstraction/noop/gossip methods.
- `sdk/websocket`: working manager scaffold is present, including connect, ping, read dispatch, subscribe, unsubscribe, and Python-compatible identifier helpers.

Important compatibility note:

Python dict insertion order matters for `msgpack.packb(action)` and therefore for signing hashes. Go maps must not be used for signed action payloads. Use `signing.OrderedMap` or dedicated ordered structs with explicit msgpack behavior.

Verified so far:

- `signing.ActionHash` matches Python for `test_phantom_agent_creation_matches_production`.
- `signing.SignL1Action` matches Python for dummy, order, cloid order, vault, TPSL, create sub account, and schedule cancel golden cases.
- `signing.SignUSDTransferAction` and `signing.SignWithdrawFromBridgeAction` match Python golden cases.
- Additional user-signed wrappers are present for spot transfer, USD class transfer, send asset, agent approval, builder fee approval, token delegate, user abstraction, and convert-to-multisig actions.
- `types.CloidFromInt(1)` matches Python's `Cloid.from_int(1)`.
- `signing.FloatToIntForHashing` handles Python's arbitrary-size integer test case with `big.Int`.
- `exchange.SlippagePrice` has a Python-rounding unit test.
- `info.RemapCoinSubscription` and websocket identifier helpers have unit tests.

Remaining verification work:

- Many exchange action methods compile and mirror payload construction, but only signing-layer golden tests currently prove byte-level compatibility.
- Future upstream Python SDK updates should be ported by diffing `hyperliquid-python-sdk/hyperliquid` and updating `sdk/portmap.yaml`.

Run:

```bash
make tidy
make fmt
make test
```
