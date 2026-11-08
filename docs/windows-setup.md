# Windows setup

Download the versioned `setup-x64.exe`, run it, follow the setup wizard and open Pomme. Setup installs into the current user's LocalAppData Programs folder by default. It adds a Start menu shortcut, offers an optional desktop shortcut and registers an uninstaller in Windows Settings. Installation does not add a PATH entry or copy credentials from another installation.

Setup detects required components using the same checks as the launcher. It skips working components and obtains missing ones from their official publishers:

| Component | Official source | Installation |
| --- | --- | --- |
| Visual C++ x64 Runtime | Microsoft `aka.ms/vc14/vc_redist.x64.exe` | Microsoft installer, with Windows permission prompt |
| WebView2 Evergreen | Microsoft `go.microsoft.com/fwlink/p/?LinkId=2124703` | Microsoft bootstrapper; user/machine scope follows Microsoft's updater |
| Chrome x64 / Widevine | Google `dl.google.com/dl/chrome/install/googlechromestandaloneenterprise64.msi` | Official Chrome MSI, with Windows permission prompt |

Before execution, downloads must have a valid Windows Authenticode signature and the expected Google LLC or Microsoft Corporation publisher. Download errors, denied elevation, installer errors and missing components after installation stop setup with an explanation and allow retry. The wizard explains prerequisite installation before the Install step. Chrome is a full browser installation and provides the CDM; Pomme does not include a separate Widevine binary.

If a component requests a restart, setup retains its component ID in `%LOCALAPPDATA%\PommeTUI\setup\restart-required.txt` until Windows boots again. Other prerequisites are still checked independently. Setup finishes installing Pomme, disables immediate launch, instructs the user to restart manually and returns exit code 3010. It never initiates a Windows reboot. Successful setup with no restart needed returns 0. Ordinary player startup does not download/install software.

Silent setup supports Inno Setup options such as `/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /DIR="C:\path"`. Uninstall preserves the Pomme profile and shared Google/Microsoft prerequisites. Portable users can continue using the ZIP with manually installed prerequisites.

## Validation limits

The final rc.5 setup passed exact-version install/readiness/uninstall checks locally and in [Windows CI](https://github.com/trashpenguin/pomme-tui/actions/runs/42472544346). Visual inspection confirmed that the complete prerequisite explanation is visible after fixing its text-area height. Later static inspection of the official signed Chrome MSI's nested archives confirmed the x64 Widevine DLL in Chrome 157.0.8103.39. This establishes payload contents, not first-time provisioning on a clean Windows machine.

Prerequisite tests cover skip, untrusted signature rejection, installer failure, readiness recheck, separate restart IDs, retry persistence and clearing state after a new boot. The prototype setup passed an actual isolated install/version/readiness/uninstall check with existing prerequisites. The official Google/Microsoft downloads were obtained and their expected publisher signatures verified. The Chrome MSI wraps an embedded installer; direct MSI File-table inspection did not verify CDM contents. No host prerequisites were removed/reinstalled for testing.

A clean Windows VM/PC was unavailable. First-time prerequisite installation and post-reboot provisioning therefore remain unverified. This is an unsigned prerelease, not a claim of completed clean-machine validation. An active Apple Music subscription, internet and personal Apple sign-in are still required. Setup does not add offline downloads, lossless audio or seamless gapless playback.

Implementation references: [Microsoft WebView2 distribution](https://learn.microsoft.com/microsoft-edge/webview2/concepts/distribution), [Visual C++ installer options](https://learn.microsoft.com/cpp/windows/redistributing-visual-cpp-files), [Google MSI deployment](https://support.google.com/chrome/a/answer/7650247), [Inno Setup](https://jrsoftware.org/).
