$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$target = Join-Path $root 'target'
# Test only the crates shipped by Pomme. The full upstream workspace also resolves
# unrelated GPUI/Spotify targets and cannot use Pomme's Apple-only dependency lock.
$workspace = Join-Path $target 'playback-tests'
if (Test-Path -LiteralPath $workspace) {
    $resolved = (Resolve-Path -LiteralPath $workspace).Path
    if ($resolved -ne [IO.Path]::GetFullPath($workspace)) { throw 'Unexpected playback test directory.' }
    if (!$resolved.StartsWith([IO.Path]::GetFullPath($target) + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Playback tests must stay under target.' }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}
New-Item -ItemType Directory -Force (Join-Path $workspace 'crates') | Out-Null
$manifest = Get-Content -LiteralPath (Join-Path $root 'sonora/Cargo.toml') -Raw
$manifest = $manifest.Replace('members = ["crates/*"]', 'members = ["crates/music", "crates/storage", "crates/widevine", "crates/webview"]')
Set-Content -LiteralPath (Join-Path $workspace 'Cargo.toml') -Value $manifest
foreach ($crate in @('music', 'storage', 'widevine', 'webview')) {
    Copy-Item -LiteralPath (Join-Path $root "sonora/crates/$crate") -Destination (Join-Path $workspace 'crates') -Recurse
}
# Seed production versions from the shipped backend lock; Cargo adds unit-test-only
# dependencies in this disposable workspace without changing either tracked lockfile.
Copy-Item -LiteralPath (Join-Path $root 'backend/Cargo.lock') -Destination (Join-Path $workspace 'Cargo.lock')
foreach ($filter in @('stream::tests', 'engine::tests', 'apple::playback::tests')) {
    & cargo +1.98.1 test --release --manifest-path (Join-Path $workspace 'Cargo.toml') -p music --no-default-features --features widevine,bundled-sqlite --lib $filter --target-dir $target
    if ($LASTEXITCODE -ne 0) { throw "Playback regression tests failed: $filter" }
}
