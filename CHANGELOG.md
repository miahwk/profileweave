# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

The first public release is planned as `v0.1.0`; no stable version has been
published yet.

### Added

- Open-source governance, security, release, and upgrade documentation.
- Multi-platform CI, dependency auditing, release checksums, SBOM generation,
  and build provenance attestation.
- Per-process control tokens, cross-process data-directory locking, and
  browser process-tree shutdown on Windows, Linux, and macOS.
- Local profile creation, update, duplication, validation, launch, stop, and
  session listing.
- Browser discovery and custom executable selection for Chromium-family
  browsers.
- Separate browser user-data directories and atomic JSON metadata storage.
- Fingerprint configuration coherence diagnostics and applied-support status.
- Loopback Go HTTP API and Vue 3 management console.
- Recoverable profile deletion with recycle-bin listing, restoration, explicit
  permanent purge, and browser-data rollback on metadata failures.
- Deterministic Chinese, English, and European desktop configuration templates.
- Replaceable runtime-provider metadata with provenance, license, version
  management, and observable capability boundaries.
- Local runtime doctor API and management-console diagnostics panel.
- A `/self-check` page for inspecting the browser environment actually exposed
  through standard Web APIs, plus a one-click editor start-page preset.
- Frontend profile-manager state tests covering partial loads, lifecycle
  actions, recoverable deletion, runtime diagnostics, and busy-state cleanup.

### Changed

- Renamed the product to ProfileWeave to avoid the existing Atlas browser
  product name.
- Profile deletion now moves browser data to an application trash directory
  and rolls it back if metadata deletion fails.
- Build metadata is exposed consistently through `--version` and `/health`.
- The capability endpoint now derives feature status from the active runtime
  provider instead of a second hard-coded capability table.

### Security

- All API requests now require a loopback Host; unsafe requests also require a
  random token and a non-cross-site Origin/Fetch Metadata posture.
- The management console now sends CSP, clickjacking, referrer, permissions,
  content-type, and cross-origin isolation headers.
- Session errors no longer expose browser paths, proxy values, URLs, or argv.
- OS target and language preferences are now reported honestly as diagnostic-only
  capabilities; browser launch failures are visible in the profile card.
- Doctor responses normalize all required collections, hide browser executable
  paths, and the diagnostics dialog traps/restores keyboard focus.
