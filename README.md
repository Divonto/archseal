# Archseal

**Architecture is a contract. Seal it in CI.**

Archseal v1 is a deterministic architecture verifier for JavaScript and TypeScript codebases.

One Go binary. Zero third-party runtime dependencies. No daemon. No cloud. No AI. No execution of analyzed project code.

```text
ARCHSEAL v1

Files scanned: 42
Import edges:  61
Violations:    0
Cycles:        0
Seal:          sha256:6e4f...

SEALED  Architecture contract holds.
```

## Why Archseal exists

Architecture diagrams do not stop code from crossing boundaries. Archseal converts those boundaries into executable policy and fails CI when the dependency graph violates the contract.

The v1 contract is deliberately narrow:

- enforce forbidden layer-to-layer dependencies
- resolve relative imports and configured path aliases
- detect dependency cycles with deterministic SCC analysis
- reject ambiguous or overlapping layer definitions
- emit stable text, JSON, and SARIF 2.1.0 output
- generate a reproducible SHA-256 seal over policy plus analyzed source
- validate policy and filesystem assumptions with `archseal doctor`

## Install

```bash
go install github.com/Divonto/archseal/cmd/archseal@latest
```

Or run directly from the repository:

```bash
go run ./cmd/archseal check
```

## Quick start

```bash
archseal init
archseal doctor
archseal check
```

A policy looks like this:

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

and a rule that denies `domain -> infrastructure`, Archseal returns exit code `1`:

```text
DENY  src/domain/order.ts:1
      domain -> infrastructure via "@infra/db"

OPEN  Architecture contract failed.
```

This works for:

```ts
import x from "../infrastructure/x";
export { x } from "../infrastructure/x";
const x = require("../infrastructure/x");
const x = await import("../infrastructure/x");
import { x } from "@infra/x";
```

Supported source extensions:

```text
.ts  .tsx  .js  .jsx  .mjs  .cjs
```

Package imports that do not match a configured alias remain outside the v1 dependency graph.

## Cycle detection

With `"forbid_cycles": true`, Archseal detects strongly connected components in the project dependency graph.

```text
CYCLE src/core/a.ts -> src/core/b.ts

OPEN  Architecture contract failed.
```

Cycle output is stable and sorted so CI logs do not change randomly between runs.

## Architecture seal

Every successful or failed check includes a content-addressed seal:

```text
sha256:<64 hex characters>
```

The seal is computed from:

1. the canonical Archseal policy
2. sorted analyzed file paths
3. exact analyzed source bytes

No timestamp, hostname, random value, or network state is included. The same architecture state produces the same seal.

This makes the result suitable for build evidence, audit logs, and reproducibility checks.

## Doctor

```bash
archseal doctor
```

Doctor fails closed when the policy cannot be trusted. It verifies:

- schema validity
- non-overlapping layers
- valid rule references
- repository-local aliases
- readable roots
- readable layer directories

Example:

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

## Machine output

JSON:

```bash
archseal check --format json
```

SARIF 2.1.0:

```bash
archseal check --format sarif > archseal.sarif
```

The SARIF contract exposes:

- `ARCH001` — forbidden architecture dependency
- `ARCH002` — dependency cycle

This makes Archseal compatible with tooling that consumes standard static-analysis results.

## CI

A minimal GitHub Actions gate:

```yaml
name: architecture

on: [push, pull_request]

jobs:
  archseal:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23.x"
      - run: go run github.com/Divonto/archseal/cmd/archseal@latest doctor
      - run: go run github.com/Divonto/archseal/cmd/archseal@latest check
```

## CLI

```text
archseal init
archseal doctor [--config path]
archseal check [--config path] [--format text|json|sarif]
archseal version
```

`archseal check --json` remains available as a compatibility alias.

Exit codes:

```text
0  architecture is sealed
1  boundary violations or dependency cycles found
2  usage, policy, or configuration error
```

## Security model

During `check`, Archseal:

- reads source files
- resolves dependency paths
- computes a deterministic graph and seal
- performs no network requests
- does not execute project code
- does not install packages
- does not invoke a JavaScript runtime

See [SECURITY.md](SECURITY.md).

## Design principles

**Deterministic.** Same policy and source bytes, same result and same seal.

**Fail closed.** Unknown config fields, invalid aliases, overlapping layers, unreadable roots, and invalid rules are errors.

**Explainable.** Every boundary failure identifies the source file, source layer, target layer, import path, and line.

**Small attack surface.** The verifier uses the Go standard library only.

**CI-first.** Stable ordering, explicit exit codes, JSON, SARIF, and no interactive behavior.

## Non-goals

Archseal is not a TypeScript compiler, package vulnerability scanner, formatter, general linter, or AI code reviewer.

Its job is smaller: **make architecture boundaries executable.**

## Versioning

v1 defines the stable policy and CLI contract. Breaking changes require a new major version.

See [CHANGELOG.md](CHANGELOG.md).

## License

MIT
