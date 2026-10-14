# Attribution

Pomme TUI combines separately licensed components:

| Component | Origin | License |
| --- | --- | --- |
| `frontend/` | k1y0miiii/applemusic-tui, upstream `613dbcd2bab8c56282ea84e3ed49f1fe3a4353b0`, with native engine and release changes | MIT; see `frontend/LICENSE` |
| `backend/` | Native JSON-line controller and Apple Music sign-in integration developed for this project | GPL-3.0-or-later |
| `sonora/` | sonorahq/sonora, upstream `63fcb51e06d5ad6ac2e8ac4e4db53d07d3ded93b` | GPL-3.0-or-later; see `sonora/COPYING` |

Sonora modifications: optional Windows output selection, controller build dependencies, Pomme-specific credential/cache and WebView2 sign-in folders, and an optional Spotify feature disabled in Pomme's Apple-only backend. The bundled source includes those modifications. Original author notices remain intact. The frontend's original module path is retained to preserve internal imports; this project is a fork, not an upstream release.

Google's Widevine CDM is an external runtime dependency and is not included. Apple Music, Apple, Chrome, Elgato, and Last.fm names identify their respective services/products and imply no endorsement.
