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
