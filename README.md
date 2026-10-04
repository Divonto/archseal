# Archseal

**Architecture is a contract. Seal it in CI.**

Archseal is a tiny, deterministic CLI that prevents forbidden dependency edges in JavaScript and TypeScript codebases.

No daemon. No cloud. No AI. No runtime dependency. One binary, one policy file, one exit code.

```text
ARCHSEAL

Files scanned: 42
Import edges:  61
Violations:    1

DENY  src/domain/order.ts:3
      domain -> infrastructure via "../infrastructure/db"

OPEN  Architecture boundary violation detected.
```

## Why

Layered architectures often exist only in diagrams. The compiler can still accept a dependency that quietly reverses the intended direction.

Archseal turns that direction into an executable invariant.

## Quick start

```bash
go install github.com/Divonto/archseal/cmd/archseal@latest
archseal init
archseal check
```

The generated `.archseal.json` starts with a clean-architecture style policy:

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
    }
  ]
}
```

If `domain` imports `infrastructure`, `archseal check` exits with code `1`.

## What it checks

Archseal scans static relative imports in `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, and `.cjs` files.

It understands:

```ts
import x from "../infrastructure/x";
export { x } from "../infrastructure/x";
const x = require("../infrastructure/x");
const x = await import("../infrastructure/x");
```

Bare package imports are ignored in v0.1. Only project-relative dependency edges are evaluated.

## CI

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
      - run: go run github.com/Divonto/archseal/cmd/archseal@latest check
```

## Design constraints

Archseal is intentionally narrow:

- deterministic output: same repository state, same result
- fail closed on invalid configuration
- sorted findings for stable CI logs
- no network access during checks
- no code execution from the repository being analyzed
- zero third-party Go dependencies

## CLI

```text
archseal init
archseal check
archseal check --config path/to/.archseal.json
archseal check --json
archseal version
```

Exit codes:

```text
0  architecture is sealed
1  boundary violations found
2  usage or configuration error
```

## Non-goals

Archseal is not a linter, formatter, code reviewer, dependency vulnerability scanner, or full compiler. It enforces one thing: declared architecture boundaries.

## Roadmap

`v0.2` can add path aliases, cycle detection, and GitHub annotations without changing the core contract.

## License

MIT
