# Contributing

Keep Archseal narrow, deterministic, and explainable.

Before opening a pull request:

```bash
go test ./...
go vet ./...
go run ./cmd/archseal check
```

New behavior should include a focused test. Avoid network-dependent checks in the core verifier.
