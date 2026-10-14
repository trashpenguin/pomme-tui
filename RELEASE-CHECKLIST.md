# Release checklist

1. Run `scripts/Build.ps1 -Version <version>` in Developer PowerShell. Use a fresh release directory.
2. Review dependency advisories: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` inside `frontend/`, and `cargo audit --file backend/Cargo.lock`. Document relevant findings before publishing.
3. Extract the generated ZIP elsewhere. Run `pomme.cmd version`, `pomme.cmd doctor`, and test installation to a temporary folder. Check output selection, sign-in, search, playback, lyrics, recovery, and quitting.
4. Test a clean Windows user/VM without developer tools. Confirm missing Chrome/CDM, WebView2, and Visual C++ runtime messages are useful.
5. Review the exact Git-tracked files for local paths, credentials, logs, profiles, binaries, or proprietary modules. Review archive entries, not just `.gitignore`.
6. Publish the binary ZIP and its matching complete source ZIP together with executable/archive checksums. Preserve licenses and attribution. Consider Windows code signing before broader distribution.
7. Report platform and playback limitations accurately. Do not claim clean-machine validation or lower memory without measuring it.
