# Release checklist

1. Set `VERSION`, update `CHANGELOG.md`, `releases/v<version>.md`, README, and `VERIFICATION.md`, then commit the reviewed source. Run `scripts/Build.ps1` in Developer PowerShell. Use a fresh release directory. The build emits the Windows ZIP, matching source ZIP, and versioned archive checksums together.
2. Review dependency advisories: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` inside `frontend/`, and `cargo audit --file backend/Cargo.lock`. Document relevant findings before publishing.
3. Extract the generated ZIP elsewhere. Run `pomme.cmd version`, `pomme.cmd doctor`, and test installation to a temporary folder. Check output selection, sign-in, search, playback, lyrics, recovery, and quitting.
4. Test a clean Windows user/VM without developer tools. Confirm missing Chrome/CDM, WebView2, and Visual C++ runtime messages are useful.
5. Review the exact Git-tracked files for local paths, credentials, logs, profiles, binaries, or proprietary modules. Review archive entries, not just `.gitignore`.
6. Wait for the exact commit's Windows CI result. Create a draft release targeting that commit, attach the setup executable, Windows/source ZIPs and versioned archive checksums, verify the uploaded bytes, then publish with the appropriate prerelease status. Check the actual setup install/uninstall flow and separately verify first-time prerequisite installation on a clean machine; an integration-test skip is not a pass. Preserve licenses and attribution. Consider Windows code signing before broader distribution.
7. Report platform and playback limitations accurately. Do not claim clean-machine validation or lower memory without measuring it.

Playback reliability gate:

- Run the new stream/controller regressions through `scripts/Test.ps1`.
- With an existing signed-in subscription and `AMTUI_NATIVE_BACKEND` pointing at the candidate backend, explicitly set `NATIVE_RECOVERY_LIVE_TEST=1` and run `go -C frontend test ./engine -run '^TestNativeLiveRecoversInterruptedDownload$' -count=1 -v -timeout=3m`. Set `SONORA_WIDEVINE_CDM` to the installed Chrome CDM if needed. This test uses an isolated account copy, muted audio, and a local TLS tunnel that drops one media connection without decrypting it.
- Check mid-song and near-end playback, ordinary end-of-queue behavior, repeat, pause during recovery, and switching tracks while reconnecting.
- Unplug/reconnect a USB or Bluetooth output during active playback, switch the Windows default output, and repeat while paused. Playback must recover at the last audible position; a paused player must stay paused.
- Confirm an extended network outage stops after the bounded retry budget and a later Play reloads the song. Authentication and unavailable-track failures must not loop or advance the queue silently.
