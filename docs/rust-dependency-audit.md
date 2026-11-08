# Rust production dependency audit — 8 November 2026

Scope: the complete `backend/Cargo.lock` used by the Windows release, including locked packages for other targets. This is a known-advisory scan, not a proof that all dependencies are defect-free. The unrelated upstream workspace and test-only workspace are outside this production audit.

Tool: `cargo-audit` 0.22.2. RustSec database commit: `309392b18f6b17022f82c701f300433126901068` (latest database change 6 November 2026; 1,422 advisories).

## Findings and fixes

| Package | rc.3 | rc.4 | Finding |
| --- | --- | --- | --- |
| rustls-webpki | 0.103.15 | 0.103.16 | [RUSTSEC-2026-0382](https://rustsec.org/advisories/RUSTSEC-2026-0382.html): emSign distrust enforcement |
| webpki-roots | 1.0.9 | 1.0.10 | [RUSTSEC-2026-0381](https://rustsec.org/advisories/RUSTSEC-2026-0381.html): bundled emSign root distrust |
| webpki-root-certs | 1.0.9 | 1.0.10 | [RUSTSEC-2026-0383](https://rustsec.org/advisories/RUSTSEC-2026-0383.html): bundled emSign root distrust |
| tokio | 1.53.1 | 1.53.3 | [RUSTSEC-2026-0386](https://rustsec.org/advisories/RUSTSEC-2026-0386.html): Unix AsyncFd I/O safety; Unix API does not apply to the Windows executable |
| chacha20 | 0.10.1 | 0.10.2 | Yanked package in the full lockfile; absent from the Windows backend feature tree |

The three certificate advisories share CVE-2026-111943. The rc.3 scan exited 1 with three vulnerabilities, one unsoundness warning and one yanked package. After the lockfile updates, the same scan exits 0 with **zero known vulnerabilities**, no unsoundness warnings and no yanked packages. Cargo also consolidated Windows API package references onto compatible versions already in the lockfile.

## Remaining maintenance warning

[RUSTSEC-2024-0436](https://rustsec.org/advisories/RUSTSEC-2024-0436.html) reports `paste` 1.0.15 as unmaintained. It is a compile-time procedural macro used through `lofty` by the music backend. No patched `paste` version is listed. This is retained as an explicit upstream maintenance risk; no advisory is ignored or suppressed. Replacing the upstream macro dependency requires a separate compatibility review.

## Reproduction

Install the pinned audit CLI, fetch the official RustSec advisory database, and run:

```powershell
cargo audit --file backend/Cargo.lock --db <advisory-db-checkout> --no-fetch
```

For identical advisory inputs, check out the database commit above. For a new release, fetch the current database instead. Re-run the release tests after dependency changes. Record test and packaged-backend results in [VERIFICATION.md](../VERIFICATION.md).

Earlier rc.3 artifacts retain their original dependencies. Updating repository source does not patch an already installed executable; use the rebuilt rc.4 Windows package.
