# contracts-reconciler

Contract reconciliation engine for MuxCore. Enables third-party modules to use their own contract repos while preserving Go nominal type safety.

Library only — import `github.com/Muxcore-Media/contracts-reconciler/reconciler`. There is no CLI binary in this repo.

## What it does

When a module declares it implements `MediaAdminService` from `github.com/some-dev/contracts-media-admin`, the reconciler:

1. Parses the Go interface from both the third-party repo and the canonical `github.com/Muxcore-Media/contracts-media-admin`
2. Compares method signatures structurally (names, params, return types)
3. If they match: generates a `go.mod replace` directive normalizing the import to the canonical path
4. If they differ: reports exactly which methods mismatch

## Published canonical contracts

| Interface | Canonical repo | Version |
|-----------|----------------|---------|
| `MediaAdminService` | `github.com/Muxcore-Media/contracts-media-admin` | v0.1.0 |
| `Downloader` / `DownloaderService` | `github.com/Muxcore-Media/contracts-downloader` | v0.1.0 |
| `Indexer` | `github.com/Muxcore-Media/contracts-indexer` | v0.1.0 |
| `NotificationProvider` | `github.com/Muxcore-Media/contracts-notification` | v0.1.0 |

Other entries in `reconciler/canonical.go` are reserved for future contract repos. See [Spool and Marketplace](https://github.com/Muxcore-Media/core/wiki/Spool-and-Marketplace#canonical-contract-registry).

## Philosophy

MuxCore contracts are **patterns**, not org-bound dependencies. Two modules implementing the same interface pattern should be interchangeable regardless of which GitHub org published the `.go` file. The reconciler is the marketplace/spool tooling that makes this work without sacrificing compile-time type safety.

## Usage

```go
import "github.com/Muxcore-Media/contracts-reconciler/reconciler"

r := &reconciler.Resolver{}

// Check a single declaration
directive, err := r.Resolve(reconciler.Declaration{
    Repo:      "github.com/some-dev/contracts-media-admin",
    Version:   "v1.2.0",
    Interface: "MediaAdminService",
})
// directive = {OldPath: "github.com/some-dev/contracts-media-admin",
//              NewPath: "github.com/Muxcore-Media/contracts-media-admin",
//              Version: "v0.1.0"}

// Apply to go.mod
reconciler.ApplyReplaceDirectives(".", []reconciler.ReplaceDirective{*directive})
```

Set `Resolver.CacheDir` to reuse cloned contract repos across invocations; otherwise clones go under the system temp directory.

## API

| Function | Description |
|----------|-------------|
| `Resolver.Resolve(Declaration)` | Check a declaration, return replace directive if compatible |
| `Resolver.ResolveAll([]Declaration)` | Batch process, return directives + errors |
| `ApplyReplaceDirectives(workdir, []ReplaceDirective)` | Run `go mod edit -replace` for each |
| `GenerateReplaceBlock([]ReplaceDirective)` | Format replace directives as a go.mod block |
| `DryRun([]Declaration)` | Preview without modifying files |
| `ParseDir(dir)` | Extract all exported interface specs from a Go package |
| `ParseFile(path)` | Extract exported interface specs from one `.go` file |
| `FindInterface(specs, name)` | Look up an interface by name in parsed specs |
| `ParseGoModFile(path)` | Parse `require` directives from a go.mod |
| `ParseReplaceDirective(line)` | Parse a replace directive line |
| `Canonical(name)` | Look up canonical repo for an interface name |
| `RegisteredCanonicals()` | All known canonical contract entries |
