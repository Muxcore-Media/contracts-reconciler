# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Do not open a public issue.** Report via GitHub Security Advisories for this repository:

https://github.com/Muxcore-Media/contracts-reconciler/security/advisories

Acknowledgment within **72 hours**. Target patch: **7 days** critical, **30 days** moderate.

## Scope

This repository is a **Go library** (`github.com/Muxcore-Media/contracts-reconciler/reconciler`). It is not MuxCore core: there is no HTTP API, gRPC mesh, RBAC, Docker image, or runtime daemon here.

Security-relevant behavior is limited to what consumers invoke when reconciling contract modules:

- **`go.mod replace`**: `ApplyReplaceDirectives` runs `go mod edit -replace`. A forged or mismatched replace can redirect a build to an attacker-controlled module path. Callers must treat replace output as trusted only after a successful structural interface match (or after explicit human review of dry-run output).
- **Cache directory**: `Resolver.CacheDir` (or the process temp dir) holds cloned contract repos. A writable shared cache can be poisoned with unexpected source; isolate `CacheDir` per job/user and do not reuse untrusted cache trees.
- **Git clone of declared repos**: `Resolve` / `CloneAndParse` clone third-party and canonical contract repositories by import path and version. Supplying untrusted `Declaration.Repo` / URLs can cause network fetches of malicious trees or unexpected local filesystem layout under the cache. Only reconcile declarations from trusted catalogs (e.g. spool/marketplace metadata you already accept).

Out of scope for this repo: MuxCore HTTP/gRPC transport security, cluster join tokens, module capability enforcement, and container hardening — report those against [Muxcore-Media/core](https://github.com/Muxcore-Media/core/security/advisories).

## Disclosure Policy

1. Reporter submits private report
2. Maintainers triage within 72 hours, assign severity
3. Fix developed privately; reporter credited (with permission)
4. GitHub Security Advisory published with fix release
5. CVE requested for critical vulnerabilities

We follow **coordinated disclosure**. Default window: 30 days before public disclosure.

## Safe Harbor

We will not pursue legal action against researchers who:
- Test against their own systems and clones
- Avoid accessing or modifying data that does not belong to them
- Make a good-faith effort to avoid degradation of service during testing
- Follow this policy's reporting and disclosure process
