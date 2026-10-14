# Put Pomme TUI on your GitHub

This folder is the source repository. Upload the source ZIP's contents or push this Git repository. Keep the folder structure intact: frontend/, backend/, sonora/, scripts/, licenses/, and .github/.

Do not upload the local target/ cache, dist/ working folder, personal AppData profile, or your earlier prototype directory. The supplied source ZIP contains only reviewed Git-tracked source and notices.

## Git push

Create an empty repository named **pomme-tui** on GitHub, choosing the visibility you want. Do not initialize another README/license there. In PowerShell inside this source folder, replace YOUR_USERNAME with your account:

```powershell
git remote add origin https://github.com/YOUR_USERNAME/pomme-tui.git
git push -u origin main
```

If you extracted the source ZIP instead of using the prepared repository, initialize it first:

```powershell
git init -b main
git add .
git commit -m "Prepare Pomme TUI 0.1.0"
```

The Windows workflow runs tests and builds an unsigned ZIP. It does not publish a release or sign into Apple Music.

## GitHub Release

After reviewing the repository, create a **v0.1.0 prerelease**. Attach the supplied Windows x64 ZIP, matching source ZIP, and archive checksum file together. Explain that it is an unofficial Apple Music client, requires a subscription and installed Chrome CDM, and has been checked on a development PC but not yet a clean Windows VM.

Use RELEASE-CHECKLIST.md for future versions. Do not reuse a source ZIP from a different binary version.
