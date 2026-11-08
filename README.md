# Pomme TUI

<img src="assets/pomme.svg" width="96" alt="Pomme terminal apple logo">

**Apple Music. Straight from your terminal.**

A Windows terminal player with native audio playback, search, a queue, synced lyrics, artwork, themes, and visualizers. An active Apple Music subscription is required. Pomme is an unofficial community project, independent of Apple.

**Development branch:** the artwork browsing, Search, Stats, controls and mini-player changes described below are [unreleased](CHANGELOG.md#unreleased). The current rc.5 downloads do not include them.

## Install and play

1. [Download the Windows setup](https://github.com/trashpenguin/pomme-tui/releases/download/v0.2.6-rc.5/pomme-tui-0.2.6-rc.5-setup-x64.exe) and run it. See the [prerelease notes](https://github.com/trashpenguin/pomme-tui/releases/tag/v0.2.6-rc.5) for validation and limits.
2. Follow setup. It checks required components, downloads and installs missing ones from Google/Microsoft, and adds a Start menu shortcut. Allow Windows permission prompts when requested, then open Pomme and complete Apple Music sign-in.
3. Press **/**, enter a song or artist, press **Enter** to search, then **Enter** to play a song or open an artist, album or playlist.

You need Windows 10/11 x64, internet access and your own Apple Music subscription. **Setup handles missing Chrome/Widevine, Visual C++ x64 Runtime and WebView2 prerequisites.** Downloads are checked for a trusted Google/Microsoft signature before execution. If a component needs a restart, setup asks you to restart Windows yourself before opening Pomme.

For portable use, extract the Windows ZIP and open **Pomme.exe**. Portable users provide the prerequisites themselves. Widevine comes from the official Chrome installation; it is not bundled separately in Pomme. Chrome does not run during playback. See [setup details](docs/windows-setup.md), including validation limits.

When Windows Terminal is available, `Pomme.exe` and the `pomme` command open the dedicated **Pomme** profile with the apple icon in the terminal tab. The launcher registers its own profile fragment and embedded icon under `%LOCALAPPDATA%\Microsoft\Windows Terminal\Fragments\PommeTUI`; it preserves the contents of your existing Terminal settings. See [Microsoft’s profile-fragment documentation](https://learn.microsoft.com/en-us/windows/terminal/json-fragment-extensions).

If setup fails, run `pomme.cmd doctor` from the extracted folder. Install missing dependencies from their official sources: [Chrome](https://www.google.com/chrome/), [Microsoft Visual C++ x64 Runtime](https://aka.ms/vc14/vc_redist.x64.exe), [WebView2](https://developer.microsoft.com/microsoft-edge/webview2/).

## Easier Windows setup in 0.2.6-rc.5

A single setup download installs Pomme, handles missing prerequisites, adds shortcuts and registers an uninstaller. Existing components are reused. Your account/profile is retained on uninstall. See [release notes](releases/v0.2.6-rc.5.md).

## Dependency security fixes from 0.2.6-rc.4

Updates the native backend's certificate-trust packages for CVE-2026-111943, Tokio for a Unix I/O-safety advisory, and a yanked locked package. The production Rust scan reports zero known vulnerabilities; one build-time macro maintenance warning remains. See the [dependency audit](docs/rust-dependency-audit.md). Update earlier candidates using the rebuilt rc.4 package.

## Repeat key correction from 0.2.6-rc.3

Both **r** and **R** cycle repeat in the player. **F5** reloads the audio engine or retries a failed connection. Shift/Caps Lock can no longer turn the repeat key into a reload. The help screen and failure/initialization hints use the same shortcuts. See [release notes](releases/v0.2.6-rc.4.md).

## Playback recovery in 0.2.6-rc.2

Interrupted downloads resume the same song at its last audible position, with up to three automatic attempts. The interface shows **Reconnecting audio** and keeps pause/skip available during recovery. Failed loads can be retried with Play. Late events cannot rewind the saved position, and slow downloads can continue progressing while locating it. Audio-output changes are detected even while the audio queue is full or paused.

The packaged backend passed an actual Apple Music test with a deliberately interrupted download, automatic recovery, paused seek/reload, and the next queued song after a near-end seek. All 57 Rust tests and frontend Go tests/vet passed locally. Realtek wired-output switching/reconnection and isolated fresh sign-in passed on rc.3. Rust advisory review is recorded for rc.4. USB/Bluetooth reconnection, clean Windows/first-time prerequisite installation and extended playback soak checks remain outstanding. See [release notes](releases/v0.2.6-rc.2.md), [changelog](CHANGELOG.md), and [verification](VERIFICATION.md).

## What's new in 0.2.5

- **Search:** find songs, albums, playlists and artists with category counts and a selected artwork preview. Enter plays a song or opens an artist/album/playlist; e plays an entire selected album/playlist. Tab switches category, arrows browse and / edits the query. Back restores the search selection.
- **Discovery:** browse Apple Music recommendations, open supported albums, playlists, artists and genres, then play or queue items.
- **Listening stats:** view play counts and listening time by date range. In Top Tracks, use Up/Down to select and Enter to play, or click to select and double-click to play. Left/Right changes the range and r refreshes. Stats are collected locally by Pomme starting with this version; older Apple Music listening history is not imported.
- **Queue manager:** play, reorder and remove queued tracks, or clear the queue.
- **Library:** browse songs, albums, playlists, artists and favorites in dedicated pages. Artists have portrait cards in the same terminal art style as song covers; open a card to browse top songs, albums and album tracks. Stats Top Artists opens the same browser. In Stats, Tab also focuses Top Albums; Click or Enter opens the selected album’s songs; e/E plays the whole album. Esc from its track page returns to Stats. Recently played is shown in its own panel.
- **Security:** the bundled Rustls dependency is updated to a patched version.

The current setup executable, portable Windows ZIP, matching source ZIP and SHA-256 checksums are attached to the [v0.2.6-rc.5 release](https://github.com/trashpenguin/pomme-tui/releases/tag/v0.2.6-rc.5).

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
| Tab / Shift+Tab | Next / previous focus or search category |
| Enter | Play selected result or queue entry |
| a / A in search | Append / play next |
| L | Open the selected Library page |
| Enter in Discovery | Open a recommendation |
| e in Discovery | Play the selected recommendation |
| a / A in Discovery or Library | Queue selected / play next |
| w, then y / Enter in Discovery | Confirm saving selected item to library |
| q in Discovery; Esc/q in Queue Manager | Open queue manager / return |
| Delete / Ctrl+Up / Ctrl+Down in queue | Remove / move selected track |
| Ctrl+X with queue focused | Clear the queue |
| Enter in Library | Open the selected category or item |
| Esc in Library | Return to the Library menu to choose another category |
| ← / → in Stats; r | Change date range / refresh stats |
| Tab in Stats; Enter | Focus Tracks/Artists/Albums; play track or open artist/album |
| Tab in artist details | Switch Top songs / Albums |
| Enter / click in artist browser | Open artist or album / play song |
| e / E on an album or playlist | Play the whole selected album/playlist, including from its track page |
| a / A in artist browser; Esc | Queue selected / play next; back |
| Ctrl+P / Ctrl+N in search | Previous / next search in history |
| PgUp / PgDn | Page through lists and help |
| Left / right in player | Seek five seconds |
| Up / down in player | Volume |
| s | Shuffle |
| r / R on the main screen | Cycle repeat |
| v / t | Visualizer / theme |
| m / T | Mini player / sleep timer |
| F1 / ? | Controls for the current screen; ? remains text in search input |
| a on the main screen | Account menu; Enter in account signs out |
| F5 on the main screen | Recover the audio engine / retry a failed connection |
| q / Ctrl+C | Quit |

Lyrics use LRCLIB and can be missing or inaccurate. Missing lyrics/artwork do not prevent playback. Optional Last.fm support is inherited from the frontend and is off unless configured. Run `pomme lastfm-auth` to configure its authentication in the Pomme profile.

## Your data

Pomme stores its settings and native session under `%LOCALAPPDATA%\PommeTUI`, outside the program folder. Signing out clears Pomme's Apple Music session without clearing another Sonora application's session. Credentials are sensitive local files protected by your Windows user-folder permissions; they are not encrypted by Pomme. Do not upload this directory, session files, or login logs to GitHub.

Advanced environment variables:

- `POMME_DATA_DIR`: separate absolute profile directory, useful for testing.
- `SONORA_WIDEVINE_CDM`: path to an existing installed x64 `widevinecdm.dll`.
- `POMME_OUTPUT`: preferred output endpoint; `-Output` overrides it.
- `GOMEMLIMIT`: Go runtime soft memory limit; defaults to `64MiB`.

For a setup installation, uninstall Pomme TUI through Windows Settings → Apps. Portable users remove the program directory and any manually added user PATH entry. To erase the session as well, sign out first or separately remove the PommeTUI user-data folder. Uninstall keeps your profile and shared prerequisite installations.

## Scope and limitations

This is the **0.2.6-rc.5 Windows prerelease**. Automated UI, backend, installer and release-package checks and muted live Apple Music recovery pass; a clean Windows VM and clean-machine playback have not been verified. Search returns at most 50 items per category. The initial search overview remains limited. Queues are capped at 1,000 catalog songs. Locally uploaded songs without catalog encodes, offline downloads, lossless, and seamless gapless playback are not supported.

Native playback still uses a separate Widevine helper. A prior full-interface snapshot measured roughly 236 MiB combined working set; this is a snapshot, not a memory guarantee. The Go soft limit does not cap total app memory. Subscription/authentication and Apple's service behavior may affect availability.

## Build from source

Install Go 1.26, Rust 1.98.1, and Visual Studio 2022 Build Tools with Desktop development with C++ and a Windows SDK. Keep the source directory path free of spaces (for example `C:\src\pomme-tui`); the native compiler dependency uses whitespace-delimited build flags. The installed app itself supports paths with spaces. In a **Developer PowerShell** with `go` and `cargo` on PATH:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\Build.ps1
```

The build reads `VERSION`, runs Go tests/vet, Rust library/controller/playback tests and launcher checks, and creates the setup executable, Windows ZIP, matching source ZIP, and `SHA256SUMS-0.2.6-rc.5.txt` under `dist/`. Production dependencies use `backend/Cargo.lock`; the isolated playback-test workspace adds test-only dependencies without changing it. The pinned Inno Setup 7.1.1 compiler is downloaded, hash/publisher verified and extracted in portable mode under `target/` for setup builds. Do not commit `dist/`, `target/`, credentials, or local profiles. See [CONTRIBUTING.md](CONTRIBUTING.md) and [RELEASE-CHECKLIST.md](RELEASE-CHECKLIST.md).

## Credits and licenses

The interface is a modified [applemusic-tui](https://github.com/k1y0miiii/applemusic-tui) (amtui), MIT licensed. The native playback uses modified [Sonora](https://github.com/sonorahq/sonora), GPL-3.0-or-later. The backend and combined distribution use GPL-3.0-or-later; the frontend retains its MIT license. See [NOTICE.md](NOTICE.md), [LICENSE](LICENSE), and [THIRD-PARTY.md](THIRD-PARTY.md). Release binaries must be accompanied by their matching source archive. Widevine is not redistributed.

## Earlier changes

### 0.2.0

- **Session restore:** queue, selected track, position, volume, shuffle and repeat are saved every 15 seconds and on normal exit. Reopening restores paused. Signing out clears the saved queue.
- **Library:** choose a category in the Library panel and press `Enter`; `L` opens the selected item's dedicated page. Each page stays in its selected category; press `Esc` to return to the menu and choose another. Library albums and playlists use cover-art cards; open one to browse and play its tracks. `PgUp`/`PgDn` fetch 50-item pages. Select an artist to browse their tracks and albums. Apple library permissions and availability still apply.
- **Queue:** `Delete` removes the selected track, `Ctrl+Up/Down` moves it, and `Ctrl+X` clears the focused queue. Press `S` twice within five seconds to save a dated playlist to Apple Music. A failed or interrupted save may leave a partial playlist; check your library before retrying.
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

- **Mini player:** `m` toggles a compact card with a Pomme wordmark, separate title/artist lines, aligned progress/times, playback state and timer. The card scales down for small terminals and stays bounded in wide windows. Arrows seek/change volume while mini mode is active; `m` restores the previous main-panel focus. `F1` opens its controls.
- **Sleep timer:** `T` cycles off / 15 / 30 / 60 / 90 minutes. Expiry pauses playback; timers are not restored after exit.
- **Search history:** `Ctrl+P` and `Ctrl+N` recall up to 30 searches in the search overlay. History stays in your local settings folder.
- **Help:** `?`, then `PgUp/PgDn` to browse controls.
- **Logo:** original Pomme apple/terminal artwork in `assets/`, embedded in the Windows launcher.

Session and search metadata are local JSON files under the Pomme profile. They are not encrypted. No account tokens are included in session history.

Page navigation: PgUp/PgDn (or Ctrl+B/Ctrl+F) moves the queue and search/library selection by a screenful. At the end of a full library batch, paging loads the next batch. Help uses the same keys and displays the current page number. These keys apply only while Pomme is focused.

### 0.1.2

- Steady progress bar by default, clamped progress values, and plain-text playback symbols for predictable terminal spacing.
- Six visualizers: bars, torus, sphere, wave, mirror, and waterfall. Press `v` to cycle.
- `Pomme.exe` launcher; opening `pomme-ui.exe` directly also runs dependency setup when needed.

Local UI tests, package checks, direct-launch setup, and authenticated backend catalog checks passed. Clean-machine playback remains unverified; this is an unsigned prerelease.
