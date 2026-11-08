# Publishing Pomme TUI

The project repository is [trashpenguin/pomme-tui](https://github.com/trashpenguin/pomme-tui). Keep source, tests, documentation, notices, and release metadata together. Never upload `target/`, the working `dist/` directory, credentials, session files, or a personal AppData profile as repository source.

## Prepare a release

1. Set `VERSION`, add the matching entry to `CHANGELOG.md`, and add `releases/v<VERSION>.md`.
2. Update the README download link and `VERIFICATION.md` with actual results and outstanding checks.
3. Commit the reviewed source, run the Windows build, and wait for hosted CI. `scripts/Build.ps1` reads `VERSION` and produces the setup executable, Windows ZIP, matching source ZIP, and versioned archive checksum file. It includes release notes and verification in the application package.
4. Follow [RELEASE-CHECKLIST.md](RELEASE-CHECKLIST.md). A build from a source ZIP uses its explicit source manifest; it does not include arbitrary local files.

The Windows workflow runs ordinary tests and packages unsigned artifacts. It does not sign in, run live account tests, or publish releases automatically.

## Publish with GitHub CLI

Use short, version-first release titles: `v<VERSION> — <short description>`, for example `v0.2.6-rc.5 — Windows setup`. Omit the repeated product name so GitHub's release navigation stays readable. Use `Windows preview` for historical Windows prereleases without a specific feature title. A title cleanup does not change version numbers, tags or release status.

For the current candidate, publish **v0.2.6-rc.5 as a prerelease** from its verified commit. Create a draft, upload all four assets, compare their checksums, and publish after validation. Use `--notes-file releases/v0.2.6-rc.5.md` to preserve the exact Markdown notes and `--target <commit>` to bind the tag to the reviewed source.

Required release assets:

- `pomme-tui-0.2.6-rc.5-setup-x64.exe`

- `pomme-tui-0.2.6-rc.5-windows-x64.zip`
- `pomme-tui-0.2.6-rc.5-source.zip`
- `SHA256SUMS-0.2.6-rc.5.txt`

Verify the public release flags, tag commit, asset names and bytes after publication. The source ZIP records its commit in `SOURCE-COMMIT.txt` and lists packaged source files in `SOURCE-MANIFEST.txt`. Do not reuse archives from another version or update an older release's binaries in place.

This candidate retains open USB/Bluetooth, clean-Windows, first-time prerequisite and extended playback soak checks documented in `VERIFICATION.md`. Rust advisory review is complete with one explicit upstream maintenance warning. Preserve the prerelease status and those limits in public notes.
