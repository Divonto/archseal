# Security

Archseal is designed as a static verifier with a deliberately small trust boundary.

## Supported version

Security fixes target the current v1 line.

## Verifier guarantees

During `archseal check`, the verifier:

- reads configured source roots
- parses static import syntax
- resolves relative paths and explicit aliases
- constructs a dependency graph
- evaluates architecture policy
- computes a deterministic SHA-256 seal

It does not execute analyzed JavaScript or TypeScript, invoke Node.js, install packages, run project scripts, or make network requests.

## Untrusted input

Source text, import specifiers, policy JSON, paths, and alias declarations are treated as untrusted input.

Unknown policy fields are rejected. Alias targets that escape the repository are rejected. Overlapping layer ownership is rejected.

See [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) for the security model.

## Reporting a vulnerability

Do not open a public issue for a security vulnerability.

Use GitHub's private security-advisory flow for this repository and include:

1. affected version or commit
2. minimal reproduction
3. security impact
4. expected invariant
5. any suggested mitigation

Please avoid including unrelated private source code or credentials.
