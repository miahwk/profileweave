# Contributing to ProfileWeave

Thanks for improving ProfileWeave. Contributions must support authorized QA,
privacy research, or session isolation and must preserve the product's honest
security boundaries.

## Before starting

- Search existing issues and proposals.
- Discuss broad architecture changes before implementing them.
- Never submit credentials, cookies, browsing history, real customer data, or
  a browser profile directory.
- Report vulnerabilities privately according to `SECURITY.md`.

## Development setup

Install Go 1.25+, Node.js 22, and pnpm 11, then run:

```powershell
pnpm --dir frontend install --frozen-lockfile
powershell -ExecutionPolicy Bypass -File scripts/verify.ps1
```

The required individual checks are:

```powershell
go test ./...
go vet ./...
pnpm --dir frontend test
pnpm --dir frontend build
powershell -ExecutionPolicy Bypass -File scripts/check-lines.ps1
```

## Architecture and code style

- Keep Go domain and application packages independent from infrastructure and
  HTTP packages.
- Keep browser process construction in browser infrastructure and fingerprint
  coherence rules in the fingerprint domain.
- Route frontend network access through its API adapter.
- Prefer the Go standard library when it is clear and safe; a maintained
  dependency is welcome when it measurably reduces risk or complexity.
- Keep Vue/TypeScript files at or below 400 lines and Go files at or below 300
  lines unless a small overage clearly improves readability.
- Add or update tests for behavior changes and persistence compatibility.

See `AGENTS.md` and `docs/architecture.md` for the complete constraints.

## Pull requests

Keep each pull request focused. Explain user value, security and persistence
impact, verification performed, and any new third-party dependency. Update the
changelog for user-visible changes. UI changes should include screenshots when
helpful. Avoid generated build output unless the repository explicitly tracks
it.

By submitting a contribution, you agree that it is provided under the Apache
License 2.0 in `LICENSE` and that you have the right to submit it.
