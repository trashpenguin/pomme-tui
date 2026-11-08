# Artist browser implementation record

Implemented as local previews on 8–9 November 2026, then approved for repository publication by the user on 9 November 2026. The original local-only development constraints below applied before that approval.

**Goal:** Artist portraits in Library → Artists and Stats → Top Artists, opening a shared artist page with playable top songs and browsable albums.

**Architecture:** Keep the existing terminal half-block artwork renderer. Library supplies catalog artist IDs and portraits; Stats resolves artists through the actual artist references of a song in local history. A shared browser handles portraits, songs, albums and album tracks. Scope async results by request ID and use bounded thumbnail caching. No arbitrary name-to-ID guessing.

**Tech stack:** Existing Go/Bubble Tea/Lip Gloss frontend and Rust native catalog RPC; no new third-party dependency.

**Spec:** User confirmed both artist sections; pictures must use the existing cover-art style; clicking an artist must show songs/albums. Existing local Stats playback work is preserved.

## Global constraints

- Development was local-only until publication was authorized on 9 November. No new public release was requested.
- Preserve the existing Stats track playback, equal card widths, range keys and stable scrolled mouse selection.
- Use real catalog artist IDs, not artist names or song IDs as artist IDs. Missing art falls back to initials.
- Native audio recovery and account storage are outside scope. Tests use fixtures; any live check uses an isolated profile without audio playback.

## Review focus

- Late network/artwork responses after changing artist, closing or paging cannot replace the current page.
- Small terminals, long Unicode names and missing pictures fit their boxes and match mouse hit testing.
- Artist-name collisions/collaborations use real references and allow a choice rather than opening an arbitrary profile.
- Back navigation restores the correct library grid or Stats range/selection; albums open their track list.
- Artwork requests are bounded/cached and failures never prevent artist navigation or song playback.

### Task 1: Native artist data

Files: backend/src/{catalog.rs,protocol.rs,main.rs}; frontend/engine/native.go.

- [x] Add failing protocol/catalog tests for song-artist references and profile artwork.
- [x] Add `song_artists(id)` RPC using `track.artist_refs` and `artist_images`; extend artist-page results with the profile in Artists.
- [x] Run backend library and catalog tests; keep the existing RPC backward compatible.

### Task 2: Shared artist browsing

Files: frontend/artist_browser.go, frontend/artist_artwork.go, frontend/artist_browser_test.go; main.go, overlay.go, mouse.go.

- [x] Add failing tests for library artist grid, detail songs/albums, album tracks, playback, back navigation, stale responses and geometry.
- [x] Implement the shared browser, existing artwork style, lazy bounded portraits, mouse/keyboard selection and request scoping.
- [x] Add Stats artist cards with Tab focus and name-to-real-reference resolution through a history song; preserve track controls.
- [x] Verify the new regressions and existing Stats suite.

### Task 3: Local validation and preview

- [x] Run Go tests/vet and relevant Rust tests; obtain a fresh-context read-only review and fix important findings.
- [x] Build a separate local preview with the updated backend; verify its version, package checksums and UI fixture rendering.
- [x] Update local notes/help/changelog, report controls and limitations. Leave source uncommitted and do not contact GitHub for mutations.

## Ledger

- Initial development ruling: implement inline and leave changes uncommitted while the user requested local-only previews. Repository publication was authorized on 9 November.
- Ruling: resolve Stats artists through song artist references — history has artist names, which are not reliable catalog IDs.

- Verification: full Go tests and vet passed; native library 13 tests and backend binary 21 tests passed. Fresh-context review findings fixed and re-reviewed with no remaining important findings.
- Real read-only catalog probe passed: song artist IDs, artist profile, 96x96 portrait, 10 top songs, 11 albums, 13 album tracks and 50 library portraits. No audio or library mutations.
- Visual QA: actual View fixture renders reviewed at gallery, artist songs, albums and Stats; sample Stats counts are fixture data. Narrow-terminal geometry/help tests passed.
- Preview: outputs/pomme-artists-preview contains matching locally built frontend/backend, local version and recomputed binary checksums. These previews were initially uncommitted and involved no GitHub writes.