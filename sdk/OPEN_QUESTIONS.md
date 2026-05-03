# Open Questions and Production Readiness

This file tracks the remaining questions and risks for the traceable Go port of the official Hyperliquid Python SDK.

The SDK compiles and core signing compatibility is covered by golden tests, but this should not be treated as production-ready for real-money trading until the items below are closed.

## Highest Risk

Signed exchange actions must match the Python SDK byte-for-byte.

The signing layer has golden tests, but many `sdk/exchange` methods still need method-level golden tests that compare the exact action payload and signature against the official Python SDK.

## Open Questions

1. Do all `sdk/exchange` action builders produce the exact same ordered payloads as Python?
2. Do optional fields match Python behavior exactly: omitted field vs `null`, empty string vs missing value?
3. Do deploy, validator, abstraction, multisig, and transfer actions match Python on real examples?
4. Does `MarketOpen`/`MarketClose` round prices exactly like Python for spot and perp assets with different `szDecimals`?
5. Should public trading methods continue using `float64` to mirror Python, or should a later production wrapper use decimal strings?
6. Does websocket behavior need reconnect, resubscribe, backoff, and error channels for bot usage?
7. Should `/info` response types remain `any`, or should frequently used responses get typed structs?
8. How should upstream Python SDK updates be tracked: git submodule, pinned commit, or vendored snapshot?

## Required Golden Tests

Add Python-vs-Go golden tests for these exchange methods before production use:

- `Order`
- `BulkOrders`
- `ModifyOrder`
- `BulkModifyOrdersNew`
- `Cancel`
- `CancelByCloid`
- `BulkCancel`
- `BulkCancelByCloid`
- `ScheduleCancel`
- `UpdateLeverage`
- `UpdateIsolatedMargin`
- `SetReferrer`
- `CreateSubAccount`
- `USDClassTransfer`
- `SendAsset`
- `SubAccountTransfer`
- `SubAccountSpotTransfer`
- `VaultUSDTransfer`
- `USDTransfer`
- `SpotTransfer`
- `TokenDelegate`
- `WithdrawFromBridge`
- `ApproveAgent`
- `ApproveBuilderFee`
- `ConvertToMultiSigUser`
- `MultiSig`
- `UseBigBlocks`
- `AgentEnableDexAbstraction`
- `AgentSetAbstraction`
- `UserDexAbstraction`
- `UserSetAbstraction`
- `Noop`
- `GossipPriorityBid`

Lower-frequency deploy and validator actions also need golden tests:

- `SpotDeployRegisterToken`
- `SpotDeployUserGenesis`
- `SpotDeployFreezeUser`
- `SpotDeployTokenActionInner`
- `SpotDeployGenesis`
- `SpotDeployRegisterSpot`
- `SpotDeployRegisterHyperliquidity`
- `SpotDeploySetDeployerTradingFeeShare`
- `PerpDeployRegisterAsset`
- `PerpDeploySetOracle`
- `CSignerInner`
- `CValidatorRegister`
- `CValidatorChangeProfile`
- `CValidatorUnregister`

## Required Integration Tests

Run against Hyperliquid testnet with a test wallet:

- `/info` smoke tests:
  - `Meta`
  - `SpotMeta`
  - `AllMids`
  - `L2Snapshot`
  - `UserState`
  - `OpenOrders`
- `/exchange` smoke tests:
  - place tiny limit order
  - cancel by oid
  - place tiny order with cloid
  - cancel by cloid
  - schedule cancel set/unset
- websocket smoke tests:
  - subscribe `allMids`
  - subscribe `l2Book`
  - unsubscribe
  - verify ping/pong path

## Known Design Tradeoffs

- The SDK intentionally uses `any` and dynamic payloads in many places to stay close to Python.
- `signing.OrderedMap` is mandatory for signed payloads because Python dict insertion order affects msgpack bytes.
- The SDK mirrors Python's `float` behavior where possible. This is useful for parity but not ideal for production trading logic.
- Websocket manager mirrors the Python SDK's simple behavior. Production bots may require stronger reconnect and resubscribe handling.

## Before Production Checklist

- [ ] Add golden tests for every trading action used by the bot.
- [ ] Add testnet integration tests for order lifecycle.
- [ ] Add websocket reconnect/resubscribe behavior if the bot depends on streams.
- [ ] Review all optional fields for Python parity.
- [ ] Pin the upstream Python SDK commit in `sdk/portmap.yaml`.
- [ ] Document the process for updating Go after upstream Python changes.
- [ ] Run tests with race detector: `go test -race ./...`.
