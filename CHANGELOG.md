# Changelog

## [Unreleased]

## [0.2.6-rc.2] - 2026-11-07

### Fixed

- Resume interrupted Apple Music downloads automatically on the same song at the last audible position, with up to three attempts and delays of 1, 2, and 4 seconds.
- Keep pause, skip, and track selection available while reconnecting. Pausing, selecting another track, clearing the queue, or signing out cancels scheduled recovery.
- Reload a failed track when Play is pressed, instead of resuming an empty decoder.
- Preserve stream and decoder errors so interrupted audio does not silently advance the queue as if the song had finished.
- Ignore late clock events from a failed decoder so recovery does not rewind to the original load position.
- Wait for slow downloads that are still progressing when locating the saved position, and treat a stalled lookup as retryable.
- Detect output failures while the audio queue is full, paused, or idle, and preserve the output-recovery event when audio channels close.

### Changed

- Show **Reconnecting audio** during automatic recovery.
- Bound Apple playback connection, read, and loading waits, and distinguish transport/server failures from account, catalog, and rate-limit responses.
- Add stream fault tests and an opt-in, muted live Apple Music test that cuts one encrypted CDN connection, checks automatic recovery, and verifies a near-end queue transition.
- Keep the release version in `VERSION`; build and package the matching source, release notes, verification record, and archive checksums.

### Validation

- 57 Rust tests, frontend Go tests/vet, Apple-only dependency checks, launcher checks, and Windows package checks passed locally.
- The exact packaged backend passed real Apple Music automatic recovery after a deliberately interrupted download, paused seek/reload, and the next queued song after a near-end seek.
- Go vulnerability scanning reported no vulnerabilities. See [VERIFICATION.md](VERIFICATION.md) for scope and remaining release checks.

This is a Windows x64 prerelease. Physical USB/Bluetooth reconnection, clean Windows installation testing, and Rust dependency advisory review remain outstanding. Builds are unsigned.

## [0.2.5] - 2026-10-18

### Added

- Apple Music discovery, local listening statistics, and dedicated queue controls.
- Expanded library browsing and navigation.

### Fixed

- Library keyboard navigation and the clickable top search field.
- Updated the bundled Rustls dependency to a patched version.

[Unreleased]: https://github.com/trashpenguin/pomme-tui/compare/v0.2.6-rc.2...HEAD
[0.2.6-rc.2]: https://github.com/trashpenguin/pomme-tui/compare/v0.2.5...v0.2.6-rc.2
[0.2.5]: https://github.com/trashpenguin/pomme-tui/releases/tag/v0.2.5
