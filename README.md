# contracts-reconciler

Contract reconciliation engine for MuxCore. Enables third-party modules to use their own contract repos while preserving Go nominal type safety.

## What it does

When a module declares it implements `MediaLibrary` from `github.com/some-dev/contracts-media`, the reconciler:

1. Parses the Go interface from both the third-party repo and the canonical `github.com/Muxcore-Media/contracts-media`
2. Compares method signatures structurally (names, params, return types)
3. If they match: generates a `go.mod replace` directive normalizing the import to the canonical path
4. If they differ: reports exactly which methods mismatch

## Philosophy

MuxCore contracts are **patterns**, not org-bound dependencies. Two modules implementing the same interface pattern should be interchangeable regardless of which GitHub org published the `.go` file. The reconciler is the marketplace/spool tooling that makes this work without sacrificing compile-time type safety.

## Usage

```go
import "github.com/Muxcore-Media/contracts-reconciler/reconciler"

r := &reconciler.Resolver{}

// Check a single declaration
directive, err := r.Resolve(reconciler.Declaration{
    Repo:      "github.com/some-dev/contracts-media",
    Version:   "v1.2.0",
    Interface: "MediaLibrary",
})
// directive = {OldPath: "github.com/some-dev/contracts-media",
//              NewPath: "github.com/Muxcore-Media/contracts-media",
//              Version: "v1.0.0"}

// Apply to go.mod
reconciler.ApplyReplaceDirectives(".", []reconciler.ReplaceDirective{*directive})
```

## CLI

```bash
go run ./cmd/reconciler check \
  --repo github.com/some-dev/contracts-media \
  --version v1.2.0 \
  --interface MediaLibrary
```

## API

| Function | Description |
|----------|-------------|
| `Resolver.Resolve(Declaration)` | Check a declaration, return replace directive if compatible |
| `Resolver.ResolveAll([]Declaration)` | Batch process, return directives + errors |
| `ApplyReplaceDirectives(dir, []ReplaceDirective)` | Run `go mod edit -replace` for each |
| `DryRun([]Declaration)` | Preview without modifying files |
| `ParseDir(dir)` | Extract all exported interface specs from a Go package |
| `Canonical(name)` | Look up canonical repo for an interface name |
