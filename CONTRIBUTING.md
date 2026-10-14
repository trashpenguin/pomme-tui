# Contributing

Supported release target: Windows x64. The frontend and backend communicate through a versioned JSON-line protocol over local pipes. Authentication stays in the backend. Never add token values to protocol messages, logs, tests, issue reports, screenshots, or committed fixtures.

Run `scripts/Test.ps1` before submitting changes. Test playback changes manually with your own active subscription. Live tests are opt-in and are never run in CI; they may play audio or use a signed-in account. Keep ordinary tests independent of Apple credentials.

The frontend keeps its MIT license; backend and Sonora changes use GPL-3.0-or-later. Do not add proprietary CDM binaries or credentials. Preserve upstream attribution. Include the matching complete source with every binary release.

Build in Developer PowerShell with Go 1.26, Rust 1.98.1, MSVC, and Windows SDK installed. The Windows workflow runs offline unit checks and builds a ZIP; it does not sign in, test audio devices, or publish a GitHub Release.
