# AGENTS.md — contracts-reconciler

MuxCore **contracts library** (`muxcore.json` type `contracts`). Not a gRPC sidecar — no ports, TLS, or capability mesh. Workspace deploy context: [`../AGENTS.md`](../AGENTS.md).

## Module identity

| Field | Value |
|-------|-------|
| Directory | `contracts-reconciler` |
| Type | `contracts` (library) |
| Package | `github.com/Muxcore-Media/contracts-reconciler/reconciler` |

## What it does

Structural interface matching and `go.mod replace` generation for third-party contract modules. `Resolver.Resolve` clones repos by Go import path (default allow-list: `github.com`), walks nested packages (`gen/...`, `muxcore/*/v1`), and compares exported Server interfaces against the canonical registry in `reconciler/canonical.go`.

## Build & test

Use Nix when Go is not on PATH:

```bash
cd contracts-reconciler
nix-shell -p go --run 'go test ./...'
```

Set `Resolver.CacheDir` to reuse cloned contract trees across invocations (or seed fixture dirs under `contracts-reconciler-cache/<import-path>-<version>` for offline tests).

## Agent rules

- Published registry keys use proto-generated `*ServiceServer` interface names.
- Reserved registry rows (`Reserved: true`) return `(nil, nil)` from `Resolve` — do not expect git clones.
- `ApplyReplaceDirectives` uses `go mod edit -replace=old=new@version` (not `=>`).
- Match existing Go patterns; run `gofmt` and `go test ./...` before finishing.
- Do not edit polluted workspace dumps (see `MASTER-ROADMAP.md` Appendix H).
