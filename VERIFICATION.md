# Unreleased browsing and controls verification — 9 November 2026

This section describes development changes after the published rc.5 release. No
new release/tag or replacement of rc.5 assets is implied.

- Go tests/vet cover Library and Search artwork navigation, artist/album/playlist
  detail flows, ranked Stats playback, keyboard scopes, stale response rejection,
  initialization guards, bounded artwork loading, Unicode and terminal sizes.
- Native library and full backend binary tests passed: 14 protocol/transport tests
  and 23 catalog/controller tests. The shipped playback regressions remain part of
  the standard test script; no audio recovery implementation or dependency version
  changed in this feature update. The final local pass also verified all 27 shipped
  stream/engine/Apple playback regressions, Apple-only dependencies and the
  launcher/setup prerequisite test scripts.
- Explicit read-only Apple Music checks verified catalog artist search, real song
  artist/album references, portraits, artist top songs/albums and album tracks.
  Artist search failed before the backend fix and passed afterward. No live audio
  played during these catalog checks.
- Independent reviews found and resolved queue initialization/notice-routing,
  same-name album ranking, Search scrolling, mini overlay visibility and launcher
  entrypoint issues. Complete control scope is recorded in [the audit](docs/controls-audit.md).
- Rendered narrow/wide fixtures verified artwork, controls help and mini layout.
  Help preserves complete sections at 70×22, 100×30, 140×45 and 100×48; mini tests
  cover 20×5 through 200×35.
- An explicit Windows Terminal 1.24 probe accepted the Pomme profile, preserved
  paths/arguments with spaces, and ran the actual child launcher's version command
  without recursion. Profile registration tests preserve existing JSONC settings.
- Local optimized previews passed privacy, installation with spaces, corruption
  rejection, version, audio/output options, dependency-error, fresh-profile and
  shutdown checks. Only application source, tests and documentation are committed;
  profiles, credentials, screenshots, binaries and local test output are excluded.

The clean-machine prerequisite installation, signing, extended playback soak and
USB/Bluetooth release gates documented below remain open. This update does not
convert the existing prerelease into a public-ready stable release.

---

# Pomme TUI 0.2.6-rc.5 verification

## Windows setup — 8 November 2026

Adds a native x64 per-user Inno Setup wizard alongside the portable ZIP. Setup detects missing prerequisites, downloads official vendor installers, verifies Authenticode status/publisher, installs and rechecks them, then installs Pomme with shortcuts and a registered uninstaller. Restart requirements persist across retries and clear after a new boot; setup does not reboot Windows or offer immediate launch while a restart is pending.

- New prerequisite regressions failed before implementation/fixes and now pass: no downloads on a ready machine, unsigned-file rejection, installer failure propagation, readiness recheck, independent missing components after another requested restart, persistent separate component IDs and boot-based clearing.
- An actual prototype setup executable with the rc.4 payload passed isolated installation to a path with spaces, version/doctor, uninstall registration and uninstall cleanup on this Windows PC with existing prerequisites. No normal shortcuts or uninstall entry were replaced. A temporary helper extraction mismatch was fixed and the executable test repeated successfully.
- Downloaded the actual Microsoft Visual C++ x64 installer, WebView2 bootstrapper and Google Chrome x64 MSI. Each passed the expected Microsoft Corporation/Google LLC signature check. These prerequisites were not reinstalled on the host. Direct MSI File-table inspection did not verify CDM contents because Chrome's MSI wraps an embedded installer.
- Independent review found restart/failure-handling issues; fixes were reviewed with no remaining important findings.

## Published rc.5 checks — 8 November 2026

- Full local tests passed: 57 Rust tests, Go tests/vet, Apple-only dependency checks, launcher checks and setup prerequisite regressions. [Windows CI run 42472544346](https://github.com/trashpenguin/pomme-tui/actions/runs/42472544346) passed for release commit `f8a17d8ada17308ca689fbcc0aabe0499e33cf9f`, including the actual setup install/version/readiness/uninstall test and artifact upload.
- The final setup executable passed isolated installation and uninstall locally. The portable package passed the existing privacy, installation, corruption-rejection, version, output/dependency-error, fresh-profile and shutdown checks. The final setup passed the builder-path scan.
- Visual inspection caught a clipped prerequisite explanation. After correction, the complete consent/download/subscription/restart text was visible, and the rebuilt setup passed integration again. The preview was closed without installing to the normal location.
- Further static inspection of the signed Google MSI's nested archives confirmed Chrome 157.0.8103.39 contains `WidevineCdm/_platform_specific/win_x64/widevinecdm.dll`. MSI SHA-256: `55edacd348b4c1c9b291dab19e4665929c7a127c013d9625970be924f893273f`. Vendor installers were not executed/reinstalled on the host.
- The native backend is byte-identical to the live-tested rc.4 executable (SHA-256 `2f2906c46d56e442cc756f855ec0da926a44befe86a11a40e48402fb9fba0b4f`); its actual interrupted-download recovery validation still applies.
- The matching source ZIP has 913 entries, records the release commit and matches its version, lockfile and setup scripts. No credentials/session/target/dist entries were found. All four assets were downloaded and compared byte-for-byte after corrected draft upload and after [public prerelease publication](https://github.com/trashpenguin/pomme-tui/releases/tag/v0.2.6-rc.5). The public tag resolves to the verified release commit.

Clean-machine prerequisite installation/post-reboot readiness remains unverified because no VM/other PC was available. Existing rc.4 playback/security results below apply to unchanged native/Go player code; the launcher only shares its existing prerequisite checks with setup. USB/Bluetooth, extended playback soak and signing remain open. This remains a prerelease.

---

# Pomme TUI 0.2.6-rc.4 verification

## Production Rust dependency review — 8 November 2026

Scanned the complete production `backend/Cargo.lock` with `cargo-audit` 0.22.2 against RustSec database commit `309392b18f6b17022f82c701f300433126901068`. The rc.3 lockfile reported three certificate-trust vulnerabilities (one shared CVE), one Tokio Unix I/O-safety warning and one yanked package.

Updated `rustls-webpki` to 0.103.16, `webpki-roots` and `webpki-root-certs` to 1.0.10, Tokio to 1.53.3 and `chacha20` to 0.10.2. The patched 383-package lockfile scan exits 0 with **zero known vulnerabilities**, no unsoundness warnings and no yanked packages. The unmaintained compile-time `paste` macro remains explicitly documented; no advisory is suppressed. See [the audit](docs/rust-dependency-audit.md) for scope, advisory links and reproduction.

The full `scripts/Test.ps1` pass succeeded after updating the lockfile: frontend Go tests/vet, all 57 Rust library/controller/playback tests, Apple-only dependency checks and launcher checks. Existing dead-code/profile warnings remain; no test failed.

The earlier rc.3 executable is not patched by changing source; use the rebuilt rc.4 package.

## Published rc.4 artifact checks — 8 November 2026

- [Windows CI run 42461236785](https://github.com/trashpenguin/pomme-tui/actions/runs/42461236785) passed for release commit `dd5be58ff9561704a5f9af67959a7778047a4ee3`, including all 57 Rust tests, Go checks, optimized build, package checks and artifact upload.
- The exact locally packaged backend passed the muted live Apple Music interrupted-download recovery test in 20.56 seconds: same-song recovery around seven seconds, pause/seek/reopen preservation and the next queued song after a near-end seek.
- Optimized binary privacy, installation with spaces, corrupted-executable rejection, version, audio options, dependency-error handling, fresh-profile startup and shutdown checks passed locally and on CI.
- The matching source ZIP contains 903 entries, records the release commit, and has the same production lockfile and version as that commit. No target/dist/credentials/session entries were present.
- The [published rc.4 prerelease](https://github.com/trashpenguin/pomme-tui/releases/tag/v0.2.6-rc.4) has its Windows ZIP, matching source ZIP and versioned SHA-256 file. All three were downloaded and compared byte-for-byte after draft upload and again after publication. The public tag resolves to the verified release commit. rc.3's public notes now point users to the security update; older binaries were not replaced.

Clean Windows/first-time prerequisite installation, USB/Bluetooth reconnection, subjective audio/gap checks, extended playback soak testing and signing remain outstanding. The rc.3 isolated sign-in and Realtek results below are historical; the physical tests were not repeated for this dependency-only update. This remains a prerelease.

---

# Pomme TUI 0.2.6-rc.3 verification

## Realtek wired-output checks — 8 November 2026

Tested the public rc.3 packaged backend with the isolated fresh-sign-in profile and volume zero. The user physically disconnected/reconnected Realtek wired headphones. This pass verifies playback state and audible-clock advancement rather than a subjective listening assessment.

- Changing Windows default output from headphones to Realtek speakers during playback left the same song advancing without a manual Play command or reported playback error.
- Changing back to headphones while paused preserved the paused position at 93,732 ms and volume zero.
- Unplugging the headphones during playback caused Windows to select the Realtek speakers; the same muted song continued advancing automatically. Windows visually confirmed the unplugged endpoint and default-speaker state.
- The first reinsertion was not immediately detected. After reseating, Windows detected the headphones as default again, but playback had already been paused; immediate reconnection while actively playing remains unverified.
- Repeated unplug/reconnect while paused at 10,000 ms preserved pause, exact position, and volume zero. Windows visually confirmed both endpoint states. Play afterward resumed the same song beyond 21 seconds.
- Repeated the complete unplug/reconnect cycle while actively playing, with Windows endpoint state recorded beside playback samples. Captured 85 unplugged-state samples (DeviceState 8) and 394 active-state samples (DeviceState 1). Every sample in that cycle reported playing, volume zero, and no playback error; position advanced from 10,000 through 66,390 ms, including after endpoint return, without another Play command. This resolves the initially inconclusive active-return transition.
- Headphones were restored as Windows' default output and the test backend was paused and shut down.

A scratch monitoring-script file-sharing error interrupted the first observation run. Captured samples were retained, the writer was corrected, and the paused unplug/reconnect/resume cycle was repeated successfully. Pomme reported no playback error in the sampled phases.

Realtek wired-output default switching and physical unplug/reconnect checks passed on this setup, both during playback and while paused. USB/Bluetooth hardware and subjective audio/gap-duration checks remain outstanding. No production code or release binary changed for this validation.

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
