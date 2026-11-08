# Pomme TUI 0.2.6-rc.3 verification

## Isolated installation and fresh sign-in — 8 November 2026

The user confirmed that no clean VM or separate Windows PC was available, so this pass used a newly installed copy and empty Pomme profile on the existing Windows 11 Home PC. This does **not** establish clean-Windows compatibility.

- Downloaded the public rc.3 Windows ZIP and checksum file, verified the archive hash, and installed to a separate path containing spaces without changing PATH or replacing the normal installation.
- Installed version reports 0.2.6-rc.3. Dependency preflight found the existing Chrome Widevine, Visual C++ x64 Runtime, and WebView2 installations. Default Windows output was selected.
- Release checks passed for binary privacy, installation, corruption rejection, version, output options, dependency errors, fresh-profile unauthenticated startup, and shutdown.
- The empty profile contained no Apple credentials and inherited token overrides were removed. The user completed fresh sign-in directly in Apple's packaged sign-in window; existing account credentials were not copied.
- A newly started installed backend restored the new authentication, returned 25 catalog songs, and advanced a selected song during muted native playback.
- Pause/seek to approximately 12 seconds and clean shutdown passed. A second process restored authentication, the same queued song, the paused position, and mute setting. Play resumed beyond that saved position; the test then paused and exited cleanly.

The isolated install/fresh sign-in/session persistence check passed. Full interactive TUI startup was not re-exercised in this pass. Clean Windows user/VM testing, first-time prerequisite installation, physical output-device checks, and Rust advisory review remain outstanding. The normal Pomme profile was not replaced or signed out.

## Release-candidate checks — 7 November 2026

Checked on 7 November 2026. This frontend correction reserves `r`/`R` for repeat and uses F5 for engine reload and failed-connection retry. It updates help, initialization/failure hints, and README controls.

The new regressions failed before the correction and passed afterward. The full Go suite and vet pass, including help layout and the existing reload tests updated to F5. The native backend is unchanged from the 0.2.6-rc.2 build whose verification is recorded below; its live playback-recovery results remain applicable.

The release remains a prerelease with the physical-output, clean-Windows, and Rust advisory checks below still outstanding.

---

# Pomme TUI 0.2.6-rc.2 verification

Checked on 7 November 2026 on a Windows development PC. This is a prerelease verification record; physical output-device and clean-machine checks below remain open. Earlier records are preserved as history.

## Automated checks

- `scripts/Test.ps1` passed: frontend Go tests and vet, 12 backend library tests, 18 controller tests, 19 stream/Apple stream tests, 6 engine tests, and 2 Apple playback error-classification tests. Total: **57 passing Rust tests**, plus the Go suite.
- The playback tests cover interrupted HTTP bodies, retained decoder errors, stalled and progressing position lookups, retry backoff/limits, late clock events, pause/selection/clear/sign-out cancellation, failed-load Play, and normal queue endings.
- New failure regressions were observed failing before their fixes and passing afterward. Independent code review approved the two final timing/lookup fixes with no remaining important findings.
- Apple-only dependency and launcher checks passed. Ordinary tests require no Apple account; the broader upstream GUI/Spotify workspace is excluded from the playback-test workspace.
- Optimized package checks passed for binary build-path privacy, installation to a path with spaces, corrupted-executable rejection, version, audio options, missing dependencies, fresh-profile startup, and shutdown.

## Actual Apple Music playback

The opt-in `TestNativeLiveRecoversInterruptedDownload` passed against the packaged 0.2.6-rc.2 backend in **24.53 seconds**. It used an isolated account copy and muted audio. A local TLS tunnel throttled and deliberately cut one real CDN connection without decrypting traffic.

- The same song resumed automatically around its seven-second audible position, without a Play command triggering recovery.
- The selected queue item and mute setting were preserved.
- Pausing, seeking to 12 seconds, and reopening the paused player preserved position and pause state.
- Seeking near the end started the next queued song, including a duplicate catalog ID, while remaining muted.

Reproduce explicitly with a signed-in subscription and `AMTUI_NATIVE_BACKEND` pointing at the candidate backend:

```powershell
$env:NATIVE_RECOVERY_LIVE_TEST = '1'
go -C frontend test ./engine -run '^TestNativeLiveRecoversInterruptedDownload$' -count=1 -v -timeout=3m
```

Set `SONORA_WIDEVINE_CDM` to the installed Chrome CDM if necessary. The test does not create a new sign-in or modify the normal saved queue.

## Dependency review and release status

Go `govulncheck` reported no vulnerabilities for this change. Rust advisory review was not rerun because `cargo-audit` was unavailable locally. The existing unused `Paced::drain` warning remains. Production lockfiles were not changed by the playback repair.

Physical USB/Bluetooth disconnection/reconnection, default-device switching during playback and while paused, a clean Windows user/VM, and Rust dependency advisory review remain outstanding. Fresh Apple sign-in and physical media-key/game compatibility were not retested. Binaries are unsigned. No new memory benchmark, offline playback, lossless support, or seamless gapless playback is claimed.

The release ships the Windows ZIP, matching complete source ZIP, and versioned archive checksum file together. `VERSION`, the release tag, notes, frontend version, and archive names identify **0.2.6-rc.2**.

---

# Historical: Pomme TUI 0.2.0 candidate verification

Checked on 16 October 2026 on Windows. The public release remains 0.1.2 while this candidate is being verified.

The 0.2.0-rc.3 package adds opt-in configurable global shortcuts, disabled by default. All frontend tests and vet passed. Windows shortcut tests passed ten repetitions covering disabled defaults, input validation, action dispatch, conflicting registrations, and release on shutdown. The package reuses the unchanged rc.2 native engine and launcher; release installation, privacy, corruption, and fresh-profile checks passed locally under Windows PowerShell 5.1. No physical keyboard/game compatibility test has been performed. Hosted run 38887814312 still fails fresh-profile hello with an invalid JSON request; this was subsequently reproduced locally with a BOM-emitting Console.InputEncoding and fixed by selecting BOM-free UTF-8 before Process.Start. Both the failing baseline and passing fix were verified on 17 October 2026; a hosted rerun is pending.

- Frontend tests and Go vet passed, including search history, mini-player layout, sleep-timer expiry, and paginated help.
- Rust library tests passed (12), and all six controller tests passed. Controller checks cover paused session restoration, queue reorder/remove/clear, and existing playback behavior.
- A read-only test with an existing Apple session passed catalog search, all five library categories, artist details, paused restore, and queue editing. It requested no audio and created no playlist.
- The initial media-key implementation failed the user's external-keyboard test. The replacement uses Windows System Media Transport Controls; the 0.2.0-rc.2 package compiled, initialized the controls without error, and appeared in Windows media-session enumeration. Physical-key validation is pending.
- GitHub's Windows build exposed a Windows PowerShell compatibility issue in the release test. Sending UTF-8 bytes directly fixes the unavailable StandardInputEncoding property but did not prevent the startup writer from emitting a BOM; the subsequent Console.InputEncoding fix handles that case. The revised release test passed under Windows PowerShell 5.1 locally; hosted CI must be rerun.
- Release tests cover installation to a path with spaces, corruption rejection, version, audio options, dependency errors, isolated fresh-profile startup, shutdown, and binary privacy checks.
- The 0.2.0 candidate adds paged library browsing, saved paused sessions, queue editing, playlist creation, mini mode, sleep timer, search history, and original logo assets.

Still requiring manual validation: external keyboard play/pause and skip in the replacement build, playlist creation, a clean Windows installation, and a fresh Apple sign-in. No new RAM benchmark is claimed. Dependency-advisory results below are historical and have not been rerun for this candidate.

---
# Pomme TUI 0.1.0 verification

Checked on 14 October 2026 on a Windows development PC. This is a prerelease preparation record, not a claim of exhaustive testing.

## Automated checks

- Go `test ./...` passed for all six frontend packages; `go vet ./...` passed.
- Rust library tests: 12 passed; native controller tests: four passed.
- Launcher discovery tests passed for newest installed CDM selection, an explicit path, a missing path, and help without dependencies.
- The Windows release compiled with Go 1.26 and Rust 1.98.1/MSVC.
- Extracted-release installation passed in a temporary directory with spaces; the test left the user's PATH unchanged.
- A corrupted executable was rejected before creating an installation directory.
- The installed `pomme version` command worked. Doctor's audio-option tests verified Windows default, an explicit Wave Link endpoint, and rejection of a missing CDM override.
- A fresh isolated profile returned protocol 1 with authentication false and shut down cleanly. It created no Apple credentials.
- Actual runtime preflight found installed Widevine, Visual C++ x64 Runtime, and WebView2 on the test PC.
- The packaged backend restored an existing native session and returned 25 songs for Kingslayer. This read-only catalog check used a temporary cache junction to the existing session; it did not duplicate credentials or request audio/sign-out.
- Source-file checks found no credential files, JWT-shaped tokens, private-key blocks, executables, proprietary CDM DLLs, or local settings among publishable source files.
- Release privacy checks found no builder-profile paths in UTF-8 or UTF-16 executable text after diagnostic-filename cleanup. Compiler source-filename records retain their byte lengths and offsets; matching executable hashes are generated afterward.

## Dependency and code review

Go's govulncheck reported no known vulnerabilities after updating x/text to 0.41.0. Cargo-audit reported zero known vulnerabilities in the actual backend lockfile after updating networking dependencies and excluding unused Spotify support. Cargo-audit still reports the `paste` macro crate as unmaintained; this is a maintenance warning, not a clean bill of health.

The broader vendored Sonora source tree includes upstream components outside Pomme's Windows backend build. The advisory result above applies to `backend/Cargo.lock`, not to every possible Sonora GUI build.

Focused independent review found no Critical issue. Its license-notice and shared WebView2-profile findings were fixed. Native shim MIT and Chromium BSD notices are included in the binary ZIP. The source archive includes the modified frontend, backend, vendored Sonora, build configuration, and notices.

Artwork now rejects HTTP failures, bodies above 4 MiB, and dimensions above 2048 pixels. Lyrics reject bodies above 2 MiB. Regression tests failed against the old behavior and passed after the fixes. A startup-sensitive cancellation test now measures termination from cancellation after helper startup, rather than timing Windows executable startup.

## Remaining manual validation

A clean Windows VM, GitHub-hosted CI execution, fresh Apple sign-in through the new Pomme profile, physical output-device switching, actual token revocation, and optional Last.fm authentication were not exercised in this release pass. Earlier native playback development confirmed audible playback and output switching; that does not replace a fresh release test on another machine.

Builds are unsigned. Credentials use the Windows user folder's access permissions and are not encrypted by Pomme. No lossless, offline downloads, or seamless gapless support is claimed. The prior approximately 236 MiB full-interface working-set snapshot is not a new benchmark or a memory cap.
