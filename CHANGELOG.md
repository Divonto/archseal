# Changelog

All notable changes to Archseal are documented here.

## 1.0.0 — 2026-10-04

First stable architecture-contract release.

### Added

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
- CI verification for tests, vet, build, policy, JSON, and SARIF

### Security properties

- no network access during verification
- no execution of analyzed project code
- no third-party Go dependencies
- deterministic sorted findings
