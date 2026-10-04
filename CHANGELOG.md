# Changelog

All notable changes to Archseal are documented here.

## 1.0.0 — 2026-10-04

First stable architecture-contract release.

### Core

- deterministic layer boundary enforcement
- relative import resolution
- configurable wildcard path aliases
- dependency-cycle detection using strongly connected components
- reproducible SHA-256 architecture seal
- strict policy validation with unknown-field rejection
- overlapping-layer rejection
- repository-local alias validation
- `archseal doctor`
- text, JSON, and SARIF 2.1.0 output
- stable exit-code contract
- JavaScript and TypeScript source support

### Repository engineering

- reusable `Divonto/archseal@v1` composite GitHub Action
- CI on Go 1.23 and current stable Go
- seed fuzz tests plus a dedicated fuzzing gate
- Linux, macOS, and Windows cross-build verification on amd64 and arm64
- deterministic build reproducibility check
- tag-driven cross-platform release workflow
- release SHA-256 checksum manifest
- CODEOWNERS and Dependabot configuration
- architecture and threat-model documentation
- structured pull-request and bug-report templates

### Security properties

- no network access during verification
- no execution of analyzed project code
- no package installation by the verifier
- no third-party Go runtime dependencies
- deterministic sorted findings
- repository-contained alias resolution
- fail-closed policy parsing
