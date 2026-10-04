# Archseal

[![CI](https://github.com/Divonto/archseal/actions/workflows/ci.yml/badge.svg)](https://github.com/Divonto/archseal/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-black.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23%2B-black.svg)](go.mod)

**Architecture is a contract. Seal it in CI.**

Archseal is a deterministic architecture verifier for JavaScript and TypeScript codebases.

It converts architectural intent into an executable CI invariant: forbidden dependency edges fail, cycles fail, malformed policy fails closed, and every analyzed architecture state receives a reproducible SHA-256 seal.

```text
ARCHSEAL v1

Files scanned: 42
Import edges:  61
Violations:    0
Cycles:        0
Seal:          sha256:6e4f...

SEALED  Architecture contract holds.
```

## The contract

Archseal v1 guarantees a deliberately small surface:

- deterministic layer-boundary enforcement
- project-relative and configured alias resolution
- dependency-cycle detection with strongly connected components
- strict JSON policy parsing and unambiguous layer ownership
- stable text, JSON, and SARIF 2.1.0 output
- content-addressed SHA-256 architecture seals
- no analyzed-code execution and no verifier network access
- zero third-party Go runtime dependencies

The design is intentionally narrower than a general linter. Archseal owns one question:

> **Does this repository still obey its declared architecture?**

## Use it as a GitHub Action

```yaml
name: architecture

on: [push, pull_request]

jobs:
  archseal:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: Divonto/archseal@v1
```

Custom policy path or machine output:

```yaml
- uses: Divonto/archseal@v1
  with:
    config: config/architecture.json
    format: sarif
```

The action validates the policy with `doctor` before enforcing it.

## Install the CLI

```bash
go install github.com/Divonto/archseal/cmd/archseal@latest
```

Or run from source:

```bash
go run ./cmd/archseal check
```

## Quick start

```bash
archseal init
archseal doctor
archseal check
```

Example policy:

```json
{
  "version": 1,
  "roots": ["src"],
  "layers": {
    "domain": ["src/domain"],
    "application": ["src/application"],
    "infrastructure": ["src/infrastructure"],
    "interfaces": ["src/interfaces"]
  },
  "rules": [
    {
      "from": "domain",
      "deny": ["application", "infrastructure", "interfaces"]
    },
    {
      "from": "application",
      "deny": ["infrastructure", "interfaces"]
    }
  ],
  "aliases": {
    "@domain/*": "src/domain/*",
    "@infra/*": "src/infrastructure/*"
  },
  "forbid_cycles": true
}
```

## Boundary enforcement

Given:

```ts
// src/domain/order.ts
import { db } from "@infra/db";
```

and a policy denying `domain -> infrastructure`:

```text
DENY  src/domain/order.ts:1
      domain -> infrastructure via "@infra/db"

OPEN  Architecture contract failed.
```

Archseal recognizes static imports, exports, `require()`, dynamic `import()`, and configured aliases across:

```text
.ts  .tsx  .js  .jsx  .mjs  .cjs
```

Bare package imports that do not match an explicit Archseal alias remain outside the v1 dependency graph.

## Cycle detection

With `"forbid_cycles": true`, dependency cycles are contract failures:

```text
CYCLE src/core/a.ts -> src/core/b.ts

OPEN  Architecture contract failed.
```

Cycle detection uses deterministic strongly connected component analysis. Findings are sorted before output.

## Architecture seal

Every check produces:

```text
sha256:<64 hex characters>
```

The seal covers canonical policy JSON, sorted analyzed paths, and exact source bytes.

It deliberately excludes timestamps, hostnames, environment variables, random values, filesystem mtimes, and network state.

That makes the seal useful as build evidence: identical architecture state produces an identical seal.

## Doctor

```bash
archseal doctor
```

Doctor validates the verifier's trust assumptions before a check:

```text
ARCHSEAL DOCTOR

OK  policy schema is valid
OK  layer paths are non-overlapping
OK  rules reference declared layers
OK  aliases are repository-local
OK  1 root(s) are readable
OK  4 layer(s) resolve to readable directories

READY  Policy is production-valid.
```

Invalid policy is an error, not a warning.

## Machine contracts

JSON:

```bash
archseal check --format json
```

SARIF 2.1.0:

```bash
archseal check --format sarif > archseal.sarif
```

Stable SARIF rule IDs:

```text
ARCH001  forbidden architecture dependency
ARCH002  dependency cycle
```

## CLI contract

```text
archseal init
archseal doctor [--config path]
archseal check [--config path] [--format text|json|sarif]
archseal version
```

`archseal check --json` is kept as a compatibility alias.

Exit codes:

```text
0  architecture is sealed
1  boundary violations or dependency cycles found
2  usage, policy, or configuration error
```

## Engineering guarantees

The repository itself is verified as a product artifact, not only compiled once:

- tests run on Go 1.23 and current stable Go
- import parsing receives a fuzzing gate in CI
- binaries cross-compile for Linux, macOS, and Windows on amd64 and arm64
- release tags produce archives plus SHA-256 checksums
- GitHub Actions dependencies are maintained by Dependabot
- ownership is explicit through CODEOWNERS
- architecture and trust boundaries are documented separately

See [Architecture](docs/ARCHITECTURE.md), [Threat model](docs/THREAT_MODEL.md), [Security](SECURITY.md), and [Changelog](CHANGELOG.md).

## Security model

During `check`, Archseal reads source and filesystem metadata, resolves paths, constructs a graph, evaluates policy, and computes a seal.

It does **not** execute the analyzed application, invoke Node.js, install packages, run project scripts, or contact a network service.

## Release model

A `v*` tag triggers a verification gate, then builds:

```text
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
windows/amd64
windows/arm64
```

Release archives are published with a `SHA256SUMS` manifest.

## Non-goals

Archseal is not a TypeScript compiler, dependency vulnerability scanner, formatter, package resolver, general linter, or AI code reviewer.

Its job is smaller and stricter:

**make architecture boundaries executable.**

## License

MIT
