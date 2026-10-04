# Threat Model

## Assets

Archseal protects the integrity of an architecture policy decision in local development and CI.

The important assets are:

- policy integrity
- deterministic findings
- deterministic architecture seal
- CI exit status
- source-path containment

## Untrusted inputs

Archseal treats these as untrusted:

- source files
- import specifiers
- policy JSON
- repository paths
- aliases

## Security properties

### No analyzed-code execution

Archseal does not invoke Node.js, npm, pnpm, yarn, TypeScript, project scripts, or imported modules.

### Repository-local alias resolution

Alias targets must remain inside the repository boundary. Absolute and parent-escaping targets are rejected.

### Fail-closed policy parsing

Unknown JSON fields are errors. A misspelled security-relevant field cannot silently downgrade enforcement.

### Unambiguous layer ownership

Overlapping layer path declarations are rejected so one file cannot be classified into two policy domains.

### Stable findings

Files, violations, cycles, and alias matching are sorted before decisions are emitted.

## Out of scope

Archseal does not attempt to detect malicious source code, package vulnerabilities, runtime sandbox escapes, compiler bugs, or compromised CI runners.

## Reporting

Security issues should be reported through a private GitHub security advisory rather than a public issue.
