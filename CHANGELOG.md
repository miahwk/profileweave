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
- Browser discovery for locally installed Chromium-family browsers.
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
- Per-user Windows amd64/arm64 installers with Start Menu integration, safe
  application shutdown, upgrade reuse, and uninstall-time data retention.
- Cross-platform `--open`, `--shutdown`, and bounded local log-file application
  lifecycle commands, including Windows GUI installer shortcuts.

### Changed

- Renamed the product to ProfileWeave to avoid the existing Atlas browser
  product name.
- Profile deletion now moves browser data to an application trash directory
  and rolls it back if metadata deletion fails.
- Build metadata is exposed consistently through `--version` and `/health`.
- The capability endpoint now derives feature status from the active runtime
  provider instead of a second hard-coded capability table.
- Direct browser executable paths are disabled in the initial release; legacy
  selections migrate to a non-runnable state until an installed browser is
  selected, closing a path-to-process execution boundary.

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
