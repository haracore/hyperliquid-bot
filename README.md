# Hyperliquid Bot

This repository is split into explicit layers so different agents can work without stepping on each other.

## Layers

```text
.
├── hyperliquid-python-sdk/   # Official upstream Python SDK submodule
├── sdk/                      # Traceable Go port of the Python SDK
├── execution/                # Bot/client execution layer built on top of sdk
├── secrets/                  # Secret provider interface and implementations
└── ui/                       # Browser UI built on top of execution
```

## Ownership

- `sdk/` is the SDK layer. It mirrors the official Python SDK and should stay easy to diff against upstream.
- `execution/` is the execution layer. It contains runnable commands and bot-facing workflows.
- `secrets/` is the provider interface layer for account secrets. It must not import `sdk/`, `execution/`, or `ui/`.
- `ui/` is the browser UI layer. It depends on `execution/client` and must not import `sdk/` directly.
- `hyperliquid-python-sdk/` is a git submodule and should be treated as read-only upstream reference.

## Documentation

- [sdk/README.md](sdk/README.md): SDK purpose, structure, commands, compatibility notes.
- [sdk/PORTING.md](sdk/PORTING.md): rules for porting Python SDK changes into Go.
- [sdk/STATUS.md](sdk/STATUS.md): current SDK implementation status.
- [sdk/OPEN_QUESTIONS.md](sdk/OPEN_QUESTIONS.md): production-readiness risks and remaining verification work.
- [sdk/portmap.yaml](sdk/portmap.yaml): machine-readable mapping from Python SDK symbols to Go SDK symbols.
- [execution/README.md](execution/README.md): execution-layer commands and usage.
- [secrets/README.md](secrets/README.md): secret provider interface, env provider, and Vault provider.
- [ui/README.md](ui/README.md): UI-layer server, structure, and usage.

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

## Submodules

Clone with submodules:

```bash
git clone --recurse-submodules <repo-url>
```

Initialize submodules after a regular clone:

```bash
git submodule update --init --recursive
```
