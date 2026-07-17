# Fingerprint Browser Agent Guide

## Mission

Build a local-first browser profile manager for authorized QA, privacy research, and session isolation. The product must be honest about its guarantees: profile isolation and configuration coherence are deliverables; bypassing a site's risk controls is not.

## Required stack and architecture

- Frontend: Vue 3, TypeScript, Vite.
- Backend: Go, standard library first.
- Architecture: DDD with explicit `domain`, `application`, `infrastructure`, and `interfaces` layers.
- Domain and application packages must not import infrastructure or HTTP packages.
- UI code must call the HTTP API through the frontend API adapter; components must not know storage or process details.
- Browser command construction and process management belong in the browser infrastructure layer.
- Fingerprint coherence rules belong in the fingerprint domain layer.

## File-size guardrails

- Vue and TypeScript source files should remain at or below 400 lines.
- Go source files should remain at or below 300 lines.
- A small, justified overage is acceptable when splitting would reduce clarity.
- Generated files, lockfiles, fixtures, and third-party code are exempt.
- Before delivery, run `scripts/check-lines.ps1` and split files that exceed the limits without a clear reason.

## Security and product boundaries

- Bind the application server to loopback by default.
- Never build browser commands through a shell or concatenate untrusted input into a command string.
- Validate browser paths, profile IDs, URLs, proxy endpoints, locale, and timezone at trust boundaries.
- Do not persist proxy passwords in plaintext. The initial release supports unauthenticated proxies only.
- Never expose an unauthenticated CDP endpoint on a non-loopback interface.
- Do not claim a fingerprint is undetectable. Surface unsupported or partially applied settings as warnings.
- Do not add CAPTCHA solving, credential stuffing, account farming, or target-specific evasion logic.

## Data and compatibility

- Profile metadata is stored locally under the configured data directory.
- Each profile receives a separate browser user-data directory.
- Storage updates must be atomic and concurrency-safe.
- Preserve backward compatibility in persisted JSON or add an explicit schema migration.

## Verification

Run the relevant checks after every material change:

```powershell
go test ./...
go vet ./...
pnpm --dir frontend test
pnpm --dir frontend build
powershell -ExecutionPolicy Bypass -File scripts/check-lines.ps1
```

Tests should cover domain invariants, fingerprint coherence rules, repository persistence, browser argument construction, HTTP behavior, and critical frontend state transitions.
