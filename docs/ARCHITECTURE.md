# Archseal Architecture

Archseal is built around one invariant:

> The same policy and the same analyzed source bytes must produce the same architecture decision and the same seal.

## Pipeline

```text
.archseal.json
      |
      v
strict policy parser
      |
      +----> doctor: schema / paths / aliases
      |
      v
source discovery
      |
      v
static import extraction
      |
      v
specifier resolution
(relative + configured aliases)
      |
      v
dependency graph
      |
      +----> boundary rules ----> ARCH001
      |
      +----> SCC cycle scan ----> ARCH002
      |
      v
sorted report
      |
      +----> text
      +----> JSON
      +----> SARIF 2.1.0
      |
      v
SHA-256 architecture seal
```

## Determinism boundary

The seal includes:

- canonical JSON policy
- analyzed paths in sorted order
- exact analyzed source bytes

The seal excludes:

- wall-clock time
- hostname
- environment variables
- network state
- filesystem modification timestamps
- random values

## Failure model

Archseal fails closed for malformed policy, unknown policy fields, overlapping layer ownership, invalid rule references, unsafe aliases, unreadable roots, and unreadable layer paths.

Architecture violations and dependency cycles are contract failures and return exit code 1. Configuration or usage failures return exit code 2.

## Resolver scope

v1 resolves project-relative imports and explicit aliases declared in `.archseal.json`.

Package resolution through Node.js, package managers, TypeScript compiler plugins, or arbitrary JavaScript execution is intentionally outside the trusted core.

## Trust boundary

The verifier parses text and filesystem metadata. It never imports or executes the analyzed JavaScript/TypeScript code.

That constraint is intentional: architecture verification should not require trusting the application it is verifying.
