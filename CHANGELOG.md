# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Open-source governance, security, release, and upgrade documentation.
- Multi-platform CI, dependency auditing, release checksums, SBOM generation,
  and build provenance attestation.
- Per-process control tokens, cross-process data-directory locking, and
  browser process-tree shutdown on Windows, Linux, and macOS.

### Changed

- Renamed the product to ProfileWeave to avoid the existing Atlas browser
  product name.
- Profile deletion now moves browser data to an application trash directory
  and rolls it back if metadata deletion fails.
- Build metadata is exposed consistently through `--version` and `/health`.

### Security

- All API requests now require a loopback Host; unsafe requests also require a
  random token and a non-cross-site Origin/Fetch Metadata posture.
- The management console now sends CSP, clickjacking, referrer, permissions,
  content-type, and cross-origin isolation headers.
- Session errors no longer expose browser paths, proxy values, URLs, or argv.

## [0.1.0] - 2026-07-17

### Added

- Local profile creation, update, duplication, validation, launch, stop, and
  session listing.
- Browser discovery and custom executable selection for Chromium-family
  browsers.
- Separate browser user-data directories and atomic JSON metadata storage.
- Fingerprint configuration coherence diagnostics and applied-support status.
- Loopback Go HTTP API and Vue 3 management console.
