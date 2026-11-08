# Controls audit — local changes, 2026-11-09

Scope: main player and focus panels, mini player, Search query/results, Library lists,
artist/album/playlist browsing, Stats, Discovery, save confirmation, Queue Manager,
Account and Help. This audit and implementation remain uncommitted and unpublished.

## Findings fixed

- Help sliced rendered lines across pages, separating headings from their entries.
  It now packs whole sections, uses one column when narrow and two when wide, and
  opens on the current screen's controls. F1 works on every ready screen; ? opens
  help outside query input. Esc/F1/?/Ctrl+C closes help; other action keys are inert.
- Ctrl+Delete was not decoded by the current Bubble Tea version. Ctrl+X clears the
  focused queue or Queue Manager. Clearing no longer acts from other main panels.
- Queue Manager Enter bypassed audio initialization protection. It now follows the
  same playback guard as the main queue; Ctrl+C returns instead of being ignored.
- Main Enter unnecessarily blocked Library navigation while audio was initializing.
- Queue/save/scrobble notices were treated as playback failures. Non-playback actions
  now use separate notices and cannot fail a pending playback operation.
- Search L opened the previous Library category instead of the active search type.
- Search query input swallowed literal /; AC/DC and similar names remain intact.
- Mini rendering hid active browsing pages while keys were routed into those pages.
- Help reversed Up/Down volume descriptions and mislabeled Discovery P. P queues
  the current collection from its details; q opens Queue Manager.
- Main and artist details now support Shift+Tab in the reverse direction. Collection
  paging accepts Ctrl+B/F as alternatives to Page Up/Down.

## Intentional context reuse

| Keys | Main player | Browsing/input scope |
| --- | --- | --- |
| r/R | Repeat | Stats refresh; collection retry; literal text in Search input |
| a/A | a opens Account | Queue/play next in results; literal text in query input |
| q | Quit | Discovery opens Queue Manager; Queue Manager returns |
| Tab | Main focus | Search category; Stats ranking; artist songs/albums; no Library category cycling |
| Enter | Play/open focused item | Search submits query or opens result; collection song plays; Account signs out |
| e/E | No main action | Play selected song/whole collection; literal query text while typing |
| / | Open Search | Edit Search query outside input; literal slash while typing |
| Ctrl+C | Quit | Back/cancel in pages and dialogs; close Help |
| s/S and t/T | Lower/uppercase distinct | Letter case is stated in Help |

F5 remains engine reload/retry; r/R never reloads. Search history Ctrl+P/N is
scoped to Search and cannot skip the playing song. Queue removal/reordering keys
are scoped to the queue. No account action is dispatched by closing Help.

## Verification

- Regression tests reproduced each behavioral defect before fixes and pass afterward.
- Full Go tests and vet; help fit/whole-section tests at 70×22, 100×30, 140×45, 100×48.
- Protocol fixtures verify queue/save success does not fail audio initialization.
- Keyboard scopes, F1 in every overlay, typed ? and /, reverse focus, scrolled Search
  selection, and mini overlay visibility are covered. Visual fixture renders inspect
  narrow and wide controls pages. No live account/audio or GitHub mutation is used.

Optional global hotkeys retain their existing default-off behavior, duplicate-binding
validation and conflict rollback. Existing Windows tests cover registration, routing
and release using synthetic test shortcuts. Hardware media keys are unchanged.

Mini player follow-up: entering mini mode focuses the player so arrows seek/adjust
volume; Tab cannot select hidden panels, and expanding restores the prior main
panel when available. Its F1 page describes mini controls.
