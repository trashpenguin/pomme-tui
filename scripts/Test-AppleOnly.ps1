$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$tree = & cargo +1.98.1 tree --locked --manifest-path (Join-Path $root 'backend\Cargo.toml') --features backend --target x86_64-pc-windows-msvc --prefix none
if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect backend dependency graph.' }
if ($tree -match '^librespot-') { throw 'Unused Spotify dependencies are enabled in the Apple-only build.' }
Write-Host 'Apple-only dependency check passed.'
