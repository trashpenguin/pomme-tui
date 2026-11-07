# Publishing Pomme TUI

The project repository is [trashpenguin/pomme-tui](https://github.com/trashpenguin/pomme-tui). Keep source, tests, documentation, notices, and release metadata together. Never upload `target/`, the working `dist/` directory, credentials, session files, or a personal AppData profile as repository source.

## Prepare a release

1. Set `VERSION`, add the matching entry to `CHANGELOG.md`, and add `releases/v<VERSION>.md`.
2. Update the README download link and `VERIFICATION.md` with actual results and outstanding checks.
3. Commit the reviewed source, run the Windows build, and wait for hosted CI. `scripts/Build.ps1` reads `VERSION` and produces the Windows ZIP, matching source ZIP, and versioned archive checksum file. It includes release notes and verification in the application package.
4. Follow [RELEASE-CHECKLIST.md](RELEASE-CHECKLIST.md). A build from a source ZIP uses its explicit source manifest; it does not include arbitrary local files.

The Windows workflow runs ordinary tests and packages unsigned artifacts. It does not sign in, run live account tests, or publish releases automatically.

## Publish with GitHub CLI

For the current candidate, publish **v0.2.6-rc.2 as a prerelease** from its verified commit. Create a draft, upload all three assets, compare their checksums, and publish after validation. Use `--notes-file releases/v0.2.6-rc.2.md` to preserve the exact Markdown notes and `--target <commit>` to bind the tag to the reviewed source.

Required release assets:

- `pomme-tui-0.2.6-rc.2-windows-x64.zip`
- `pomme-tui-0.2.6-rc.2-source.zip`
- `SHA256SUMS-0.2.6-rc.2.txt`

Verify the public release flags, tag commit, asset names and bytes after publication. The source ZIP records its commit in `SOURCE-COMMIT.txt` and lists packaged source files in `SOURCE-MANIFEST.txt`. Do not reuse archives from another version or update an older release's binaries in place.

This candidate retains open physical-output, clean-Windows, and Rust advisory checks documented in `VERIFICATION.md`. Preserve the prerelease status and those limits in public notes.
