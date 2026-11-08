param([Parameter(Mandatory=$true)][string]$ReleasePath, [Parameter(Mandatory=$true)][string]$Version)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'Pomme.Helpers.ps1')
. (Join-Path $PSScriptRoot 'Setup.Helpers.ps1')
$root = Split-Path $PSScriptRoot -Parent
$tools = Join-Path $root 'target\setup-tools-7.1.1'
$compiler = Join-Path $tools 'ISCC.exe'
if (!(Test-Path -LiteralPath $compiler)) {
    New-Item -ItemType Directory -Force -Path $tools | Out-Null
    $installer = Join-Path $tools 'innosetup-7.1.1-x64.exe'
    Invoke-WebRequest -UseBasicParsing -Uri 'https://github.com/jrsoftware/issrc/releases/download/is-7_1_1/innosetup-7.1.1-x64.exe' -OutFile $installer
    if ((Get-PommeSha256 $installer) -ne '8a5a6711c3766a6e5fb74f280ba0e3551e719ce94538d901fdbd1f8f9920b3f0' -or
        !(Test-PommeInstallerSignature $installer 'Pyrsys B.V.')) { throw 'Inno Setup compiler installer verification failed.' }
    # Official portable mode avoids installing a development tool into Windows.
    $process = Start-Process -FilePath $installer -ArgumentList @('/PORTABLE=1','/CURRENTUSER','/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="{0}"' -f $tools)) -Wait -PassThru -WindowStyle Hidden
    if ($process.ExitCode -ne 0 -or !(Test-Path -LiteralPath $compiler)) { throw 'Portable Inno Setup compiler extraction failed.' }
}
& $compiler ('/DAppVersion=' + $Version) ('/DReleaseDir=' + [IO.Path]::GetFullPath($ReleasePath)) ('/DOutputDir=' + (Join-Path $root 'dist')) (Join-Path $PSScriptRoot 'Pomme.Setup.iss')
if ($LASTEXITCODE -ne 0) { throw 'Windows setup compilation failed.' }
& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'Test-SetupRelease.ps1') -SetupPath (Join-Path $root "dist\pomme-tui-$Version-setup-x64.exe") -Version $Version
if ($LASTEXITCODE -ne 0) { throw 'Windows setup integration failed.' }
