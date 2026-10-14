param([string]$Destination = (Join-Path $env:LOCALAPPDATA 'Programs\PommeTUI'), [switch]$AddToPath)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'Pomme.Helpers.ps1')
$Destination = [IO.Path]::GetFullPath($Destination)
if ($Destination -eq [IO.Path]::GetFullPath($PSScriptRoot)) { throw 'Choose an installation folder separate from the extracted release.' }
$files = @('pomme-ui.exe','apple-native-engine.exe','pomme.cmd','Run-Pomme.ps1','Pomme.Helpers.ps1','Start Pomme.cmd','Sign In.cmd','README.md','LICENSE','LICENSE-AMTUI.txt','NOTICE.md','THIRD-PARTY.md','SHA256SUMS.txt')
foreach ($name in $files) { if (!(Test-Path -LiteralPath (Join-Path $PSScriptRoot $name) -PathType Leaf)) { throw "Release file missing: $name" } }
$checksums = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'SHA256SUMS.txt')
foreach ($name in @('pomme-ui.exe','apple-native-engine.exe')) {
    $expected = @($checksums | Where-Object { $_ -match ('  ' + [regex]::Escape($name) + '$') })
    $actual = Get-PommeSha256 -Path (Join-Path $PSScriptRoot $name)
    if ($expected.Count -ne 1 -or $actual -ne ($expected[0] -split '  ')[0]) { throw "Checksum failed: $name" }
}
New-Item -ItemType Directory -Force -Path $Destination | Out-Null
foreach ($name in $files) { Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination (Join-Path $Destination $name) -Force }
if (Test-Path -LiteralPath (Join-Path $PSScriptRoot 'licenses')) { Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'licenses') -Destination $Destination -Recurse -Force }
if ($AddToPath) {
    $current = [Environment]::GetEnvironmentVariable('Path','User')
    $entries = @($current -split ';' | Where-Object { $_ })
    if ($entries -notcontains $Destination) { [Environment]::SetEnvironmentVariable('Path', (($entries + $Destination) -join ';'), 'User') }
    Write-Host 'Open a new terminal to use the pomme command.'
}
Write-Host "Installed Pomme TUI to $Destination"
Write-Host 'No sign-in data was copied. Run pomme doctor, then pomme login.'
