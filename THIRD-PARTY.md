# Third-party notices

Pomme's frontend preserves amtui's MIT license (LICENSE-AMTUI.txt in a Windows release, frontend/LICENSE in source). Go dependency notices are in licenses/go/; Go's license is licenses/LICENSE-GO.txt.

The backend and modified Sonora components are GPL-3.0-or-later (LICENSE). Matching modified source must accompany each release. Use this project's source archive for the same release version, including backend/, sonora/, lockfiles, and build scripts. The pinned upstream Sonora commit alone does not contain Pomme's modifications.

The compiled C++ Widevine shim includes the MIT notice in licenses/LICENSE-WIDEVINE-SHIM.txt. Chromium CDM header notices are included in licenses/LICENSE-CHROMIUM-CDM-HEADERS.txt (BSD). The proprietary Widevine runtime itself is not distributed.

licenses/SONORA-THIRD-PARTY.md preserves Sonora's broader upstream dependency license inventory. Paths and GUI font/icon statements in that inventory refer to the included sonora/ source tree; Pomme's terminal backend does not embed Sonora's GUI fonts or icon packs. Exact dependency versions are recorded in backend/Cargo.lock and frontend/go.sum.
