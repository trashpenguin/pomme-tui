# Changelog

## [Unreleased]

## [0.2.6-rc.4] - 2026-11-08

### Security

- Update `rustls-webpki` to 0.103.16, `webpki-roots` and `webpki-root-certs` to 1.0.10 for the emSign certificate-distrust advisories RUSTSEC-2026-0382, RUSTSEC-2026-0381 and RUSTSEC-2026-0383 (CVE-2026-111943).
- Update Tokio to 1.53.3 for RUSTSEC-2026-0386, affecting its Unix `AsyncFd` API. Update the locked, yanked `chacha20` 0.10.1 package to 0.10.2.
- The full production lockfile scan now reports zero known vulnerabilities, no unsoundness warnings and no yanked packages. The unmaintained build-time `paste` dependency remains documented in [the audit](docs/rust-dependency-audit.md).

Includes the playback recovery and repeat/F5 corrections from rc.2/rc.3. This remains an unsigned Windows x64 prerelease; see [verification](VERIFICATION.md) for completed checks and outstanding release gates.

## [0.2.6-rc.3] - 2026-11-07

### Fixed

- Reserve both `r` and `R` for cycling repeat in the player. Caps Lock or Shift can no longer turn the repeat key into an engine reload.
- Move manual engine reload and failed-connection retry to **F5**. Update help, initialization warnings, the connection-failure screen, and README controls to use the same binding.
- Add regressions proving repeat preserves engine/initialization state and F5 reloads; update the existing reload tests to use F5.

The native playback backend is unchanged from 0.2.6-rc.2. Its automatic recovery fixes remain included. See [VERIFICATION.md](VERIFICATION.md) for validation and outstanding hardware/clean-Windows checks.

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

[Unreleased]: https://github.com/trashpenguin/pomme-tui/compare/v0.2.6-rc.4...HEAD
[0.2.6-rc.4]: https://github.com/trashpenguin/pomme-tui/compare/v0.2.6-rc.3...v0.2.6-rc.4
[0.2.6-rc.3]: https://github.com/trashpenguin/pomme-tui/compare/v0.2.6-rc.2...v0.2.6-rc.3
[0.2.6-rc.2]: https://github.com/trashpenguin/pomme-tui/compare/v0.2.5...v0.2.6-rc.2
[0.2.5]: https://github.com/trashpenguin/pomme-tui/releases/tag/v0.2.5
