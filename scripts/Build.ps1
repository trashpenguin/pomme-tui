param([string]$Version = '0.2.0', [switch]$SkipTests)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'Pomme.Helpers.ps1')
$root = Split-Path $PSScriptRoot -Parent
$dist = Join-Path $root 'dist'
$release = Join-Path $dist "pomme-tui-$Version-windows-x64"
function Invoke-Checked {
    param([string]$Tool, [string[]]$Arguments)
    & $Tool @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Tool failed with exit code $LASTEXITCODE" }
}
if ($Version -notmatch '^\d+\.\d+\.\d+([-.][A-Za-z0-9.]+)?$') { throw 'Use a semantic version such as 0.1.0.' }
if (Test-Path -LiteralPath $release) { throw 'Release folder already exists. Choose a new version or move the previous build.' }
if (!$SkipTests) { & (Join-Path $PSScriptRoot 'Test.ps1'); if ($LASTEXITCODE -ne 0) { throw 'Tests failed.' } }
New-Item -ItemType Directory -Force -Path $release | Out-Null
# Avoid embedding the builder's Windows username and local paths in release binaries.
$savedFlags = $env:CARGO_ENCODED_RUSTFLAGS
$savedNativeFlags = $env:AWS_LC_SYS_CFLAGS
$flags = @(if ($savedFlags) { $savedFlags -split [char]31 } elseif ($env:RUSTFLAGS) { $env:RUSTFLAGS -split '\s+' | Where-Object { $_ } })
$flags += "--remap-path-prefix=$([Environment]::GetFolderPath('UserProfile'))=.build-home"
$flags += "--remap-path-prefix=$root=."
$env:CARGO_ENCODED_RUSTFLAGS = $flags -join [char]31
# C error records have independent filename metadata. Override only the
# OpenSSL diagnostic filename; error codes and line numbers remain intact.
$privacyHeader = (Join-Path $PSScriptRoot 'native-privacy.h').Replace('\','/')
if ($privacyHeader -match '\s') { throw 'Build from a directory path without spaces, such as C:\src\pomme-tui.' }
$env:AWS_LC_SYS_CFLAGS = (($savedNativeFlags + ' /FI' + $privacyHeader).Trim())
try { Invoke-Checked -Tool cargo -Arguments @('+1.98.1','build','--locked','--release','--manifest-path',(Join-Path $root 'backend\Cargo.toml'),'--features','backend','--target-dir',(Join-Path $root 'target')) }
finally {
    $env:CARGO_ENCODED_RUSTFLAGS = $savedFlags
    $env:AWS_LC_SYS_CFLAGS = $savedNativeFlags
}
Invoke-Checked -Tool go -Arguments @('-C',(Join-Path $root 'frontend'),'build','-trimpath','-ldflags',"-s -w -X main.version=$Version",'-o',(Join-Path $release 'pomme-ui.exe'),'.')
Invoke-Checked -Tool go -Arguments @('run','github.com/akavel/rsrc@v0.10.2','-ico',(Join-Path $root 'assets\pomme.ico'),'-arch','amd64','-o',(Join-Path $root 'frontend\cmd\pomme\icon_windows_amd64.syso'))
Invoke-Checked -Tool go -Arguments @('-C',(Join-Path $root 'frontend'),'build','-trimpath','-ldflags','-s -w','-o',(Join-Path $release 'Pomme.exe'),'./cmd/pomme')
Copy-Item -LiteralPath (Join-Path $root 'target\release\apple-native-engine.exe') -Destination $release
& (Join-Path $PSScriptRoot 'Remove-NativeSourcePaths.ps1') -Path (Join-Path $release 'apple-native-engine.exe')
foreach ($name in @('pomme.cmd','Run-Pomme.ps1','Pomme.Helpers.ps1','Start Pomme.cmd','Sign In.cmd','Install.ps1')) { Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination $release }
foreach ($name in @('README.md','LICENSE','NOTICE.md','THIRD-PARTY.md')) { Copy-Item -LiteralPath (Join-Path $root $name) -Destination $release }
Copy-Item -LiteralPath (Join-Path $root 'frontend\LICENSE') -Destination (Join-Path $release 'LICENSE-AMTUI.txt')
New-Item -ItemType Directory -Force (Join-Path $release 'assets') | Out-Null
foreach ($asset in @('pomme.svg','pomme.png','pomme.ico')) { Copy-Item (Join-Path $root ('assets\' + $asset)) (Join-Path $release 'assets') }
Copy-Item -LiteralPath (Join-Path $root 'licenses') -Destination $release -Recurse
$sums = foreach ($name in @('pomme-ui.exe','apple-native-engine.exe','Pomme.exe')) { '{0}  {1}' -f (Get-PommeSha256 -Path (Join-Path $release $name)), $name }
$sums | Set-Content -LiteralPath (Join-Path $release 'SHA256SUMS.txt') -Encoding ascii
& (Join-Path $PSScriptRoot 'Test-Release.ps1') -ReleasePath $release
Compress-Archive -LiteralPath $release -DestinationPath (Join-Path $dist "pomme-tui-$Version-windows-x64.zip")
Write-Host "Windows release: $release"
exit 0
