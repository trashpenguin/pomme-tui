# Pomme TUI

<img src="assets/pomme.svg" width="96" alt="Pomme terminal apple logo">

**Apple Music. Straight from your terminal.**

A Windows terminal player with native audio playback, search, a queue, synced lyrics, artwork, themes, and visualizers. An active Apple Music subscription is required. Pomme is an unofficial community project, independent of Apple.

## Install and play

1. Download and extract the Windows x64 release ZIP from this repository's Releases page. For a local build, find it under `dist/`.
2. Double-click **Pomme.exe** (or **Start Pomme.cmd**). If the current console is too small, Pomme opens a 140×45 Windows Terminal window so the lyrics cover is visible. On first launch, complete Apple Music sign-in in the temporary window.
3. Press **/**, enter a song or artist, press **Enter** to search, then **Enter** to play.

Requirements: Windows 10/11 x64, internet access, an Apple Music subscription, Google Chrome with its installed x64 Widevine module, Microsoft Visual C++ x64 Runtime, and WebView2 Evergreen Runtime for sign-in. Chrome does not run during playback. Widevine is loaded from your existing Chrome installation; Pomme does not include or download a CDM.

If setup fails, run `pomme.cmd doctor` from the extracted folder. Install missing dependencies from their official sources: [Chrome](https://www.google.com/chrome/), [Microsoft Visual C++ x64 Runtime](https://aka.ms/vc14/vc_redist.x64.exe), [WebView2](https://developer.microsoft.com/microsoft-edge/webview2/).

### Use the `pomme` command anywhere

From PowerShell in the extracted Windows release folder:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\Install.ps1 -AddToPath
```

This installs for your Windows user without administrator access. Open a **new terminal**, then:

```powershell
pomme
pomme doctor
pomme login
pomme version
pomme help
```

The ZIP also works without installation. The installer verifies executable checksums before copying. To update, quit Pomme and run the new release's installer; your AppData profile is retained. Builds are unsigned; distribution signing has not been set up. Download only from the project owner's trusted release page.

## Audio and Wave Link

Pomme follows Windows' default output and recovers playback when it changes. To use an Elgato Wave Link channel instead:

```powershell
pomme -Output "Spotify"
```

Use the name of your actual Windows audio endpoint. Matching is case-insensitive; an exact match is preferred, then a name containing the text. With an explicit device, change your monitor destination inside Wave Link. To return to following Windows:

```powershell
pomme -DefaultOutput
```

For a persistent preference, set the user environment variable `POMME_OUTPUT` to the endpoint name. Remove it to restore the default. Do not set an output if you do not have that device.

## Controls

| Key | Action |
| --- | --- |
| Space | Play/pause |
| n / p | Next / previous |
| / | Search |
| Tab | Move focus or search category |
| Enter | Play selected result or queue entry |
| a / A in search | Append / play next |
| Left / right in player | Seek five seconds |
| Up / down in player | Volume |
| s / r | Shuffle / cycle repeat |
| v / t | Visualizer / theme |
| a outside search | Account; Enter in account signs out |
| R | Recover the audio engine |
| ? | Full keyboard help |
| q / Ctrl+C | Quit |

Lyrics use LRCLIB and can be missing or inaccurate. Missing lyrics/artwork do not prevent playback. Optional Last.fm support is inherited from the frontend and is off unless configured. Run `pomme lastfm-auth` to configure its authentication in the Pomme profile.

## Your data

Pomme stores its settings and native session under `%LOCALAPPDATA%\PommeTUI`, outside the program folder. Signing out clears Pomme's Apple Music session without clearing another Sonora application's session. Credentials are sensitive local files protected by your Windows user-folder permissions; they are not encrypted by Pomme. Do not upload this directory, session files, or login logs to GitHub.

Advanced environment variables:

- `POMME_DATA_DIR`: separate absolute profile directory, useful for testing.
- `SONORA_WIDEVINE_CDM`: path to an existing installed x64 `widevinecdm.dll`.
- `POMME_OUTPUT`: preferred output endpoint; `-Output` overrides it.
- `GOMEMLIMIT`: Go runtime soft memory limit; defaults to `64MiB`.

To uninstall, remove the program directory and its entry from your user PATH. To erase the session as well, sign out first or separately remove the PommeTUI user-data folder. The program does not erase user data when its files are removed.

## Scope and limitations

This is the **0.2.5 Windows prerelease**. Automated UI, backend, installer and release-package checks pass; a clean Windows VM and clean-machine playback have not been verified. Search returns at most 50 items per category. The initial search overview remains limited. Queues are capped at 1,000 catalog songs. Locally uploaded songs without catalog encodes, offline downloads, lossless, and seamless gapless playback are not supported.

Native playback still uses a separate Widevine helper. A prior full-interface snapshot measured roughly 236 MiB combined working set; this is a snapshot, not a memory guarantee. The Go soft limit does not cap total app memory. Subscription/authentication and Apple's service behavior may affect availability.

## Build from source

Install Go 1.26, Rust 1.98.1, and Visual Studio 2022 Build Tools with Desktop development with C++ and a Windows SDK. Keep the source directory path free of spaces (for example `C:\src\pomme-tui`); the native compiler dependency uses whitespace-delimited build flags. The installed app itself supports paths with spaces. In a **Developer PowerShell** with `go` and `cargo` on PATH:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\Build.ps1
```

The build runs Go tests/vet, Rust library tests, and launcher checks, uses the locked dependencies, and creates `dist/pomme-tui-0.2.5-windows-x64.zip`. Do not commit `dist/`, `target/`, credentials, or local profiles. See [CONTRIBUTING.md](CONTRIBUTING.md) and [RELEASE-CHECKLIST.md](RELEASE-CHECKLIST.md).

## Credits and licenses

The interface is a modified [applemusic-tui](https://github.com/k1y0miiii/applemusic-tui) (amtui), MIT licensed. The native playback uses modified [Sonora](https://github.com/sonorahq/sonora), GPL-3.0-or-later. The backend and combined distribution use GPL-3.0-or-later; the frontend retains its MIT license. See [NOTICE.md](NOTICE.md), [LICENSE](LICENSE), and [THIRD-PARTY.md](THIRD-PARTY.md). Release binaries must be accompanied by their matching source archive. Widevine is not redistributed.

## Changes in 0.1.2

- Steady progress bar by default, clamped progress values, and plain-text playback symbols for predictable terminal spacing.
- Six visualizers: bars, torus, sphere, wave, mirror, and waterfall. Press `v` to cycle.
- `Pomme.exe` launcher; opening `pomme-ui.exe` directly also runs dependency setup when needed.

Local UI tests, package checks, direct-launch setup, and authenticated backend catalog checks passed. Clean-machine playback remains unverified; this is an unsigned prerelease.

## New in 0.2.5

- **Discovery:** browse Apple Music recommendations in a dedicated page, open albums, playlists, artists and genres, and queue supported recommendations.
- **Stats:** select Last 7 Days, Last 30 Days, This Month, This Year or All Time. Pomme records plays and listening time locally from this version onward; the history is stored with Pomme's settings.
- **Queue controls:** review, play, reorder and remove queued tracks from the queue manager; discovery offers a confirmation before saving supported items to the library.
- **Library navigation:** open category pages directly, page through large lists, and use Tab to change categories. Ctrl+L no longer changes library categories.

## New in 0.2.0

- **Session restore:** queue, selected track, position, volume, shuffle and repeat are saved every 15 seconds and on normal exit. Reopening restores paused. Signing out clears the saved queue.
- **Library:** choose a category in the Library panel and press `Enter`; `L` opens the selected item's dedicated page. Press `Tab` on a library page to cycle songs, albums, playlists, artists and favorites. `PgUp`/`PgDn` fetch 50-item pages. Select an artist to browse their tracks and albums. Apple library permissions and availability still apply.
- **Queue:** `Delete` removes the selected track, `Ctrl+Up/Down` moves it, and `Ctrl+Delete` clears the queue. Press `S` twice within five seconds to save a dated playlist to Apple Music. A failed or interrupted save may leave a partial playlist; check your library before retrying.
- **Media keys:** Uses Windows System Media Transport Controls for play/pause, next and previous while Pomme is in the background, with current track metadata. Windows selects the active media session; start playback in Pomme before testing keyboard controls.
- **Optional global shortcuts:** Disabled by default. After the player starts, edit `%LOCALAPPDATA%\PommeTUI\settings\hotkeys.json` (or `settings\hotkeys.json` under your custom `POMME_DATA_DIR`) and restart Pomme. Set `enabled` to `true` to enable the combinations below, or `false` to release them on the next restart. While disabled, Pomme registers no global keyboard combinations.

```json
{
  "enabled": false,
  "play_pause": "Ctrl+Alt+Space",
  "next": "Ctrl+Alt+Right",
  "previous": "Ctrl+Alt+Left"
}
```

Customize combinations with Ctrl, Alt, and Shift plus a letter, digit, F1–F24, Space, Left, Right, Up, or Down. Include Ctrl or Alt. Each combination must be different. Enabled shortcuts apply while another app or game is focused and may consume that combination; leave them disabled or choose combinations your games do not use. If registration fails, all Pomme global shortcuts are released and the player shows a notice. Hardware media-button support remains separate.

- **Mini player:** `m` toggles the compact view; it also fits smaller terminals.
- **Sleep timer:** `T` cycles off / 15 / 30 / 60 / 90 minutes. Expiry pauses playback; timers are not restored after exit.
- **Search history:** `Ctrl+P` and `Ctrl+N` recall up to 30 searches in the search overlay. History stays in your local settings folder.
- **Help:** `?`, then `PgUp/PgDn` to browse controls.
- **Logo:** original Pomme apple/terminal artwork in `assets/`, embedded in the Windows launcher.

Session and search metadata are local JSON files under the Pomme profile. They are not encrypted. No account tokens are included in session history.

Page navigation: PgUp/PgDn (or Ctrl+B/Ctrl+F) moves the queue and search/library selection by a screenful. At the end of a full library batch, paging loads the next batch. Help uses the same keys and displays the current page number. These keys apply only while Pomme is focused.
