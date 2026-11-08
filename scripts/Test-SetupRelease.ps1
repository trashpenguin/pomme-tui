param([Parameter(Mandatory=$true)][string]$SetupPath, [Parameter(Mandatory=$true)][string]$Version)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'Pomme.Helpers.ps1')
. (Join-Path $PSScriptRoot 'Setup.Helpers.ps1')
$registration = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\{6A02AC7C-757E-4540-A60C-267CF9702C2D}_is1'
if (Test-Path -LiteralPath $registration) { Write-Host 'Setup integration skipped: preserve existing Pomme uninstall registration.'; return }
if (@(Get-PommeMissingPrerequisites).Count) { Write-Host 'Setup integration skipped: build-host prerequisites are missing; CI never installs host runtimes.'; return }
$fixture = Join-Path ([IO.Path]::GetTempPath()) ('pomme-setup-test-' + [guid]::NewGuid())
$installed = Join-Path $fixture 'Installed With Spaces'
New-Item -ItemType Directory -Path $fixture | Out-Null
$completed = $false
try {
    $process = Start-Process -FilePath ([IO.Path]::GetFullPath($SetupPath)) -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART','/NOICONS','/TASKS=""',('/DIR="{0}"' -f $installed),('/LOG="{0}"' -f (Join-Path $fixture 'setup.log'))) -Wait -PassThru -WindowStyle Hidden
    if ($process.ExitCode -ne 0) { throw "Setup failed with exit $($process.ExitCode); see $fixture\setup.log" }
    $actual = & (Join-Path $installed 'pomme.cmd') version
    if ($LASTEXITCODE -ne 0 -or ($actual -join ' ') -notmatch [regex]::Escape($Version)) { throw 'Setup installed the wrong player version.' }
    & (Join-Path $installed 'pomme.cmd') doctor
    if ($LASTEXITCODE -ne 0) { throw 'Installed player prerequisite check failed.' }
    if (!(Test-Path -LiteralPath $registration)) { throw 'Setup did not register its uninstaller.' }
    $completed = $true
} finally {
    $uninstaller = Join-Path $installed 'unins000.exe'
    if (Test-Path -LiteralPath $uninstaller) {
        $process = Start-Process -FilePath $uninstaller -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru -WindowStyle Hidden
        if ($process.ExitCode -ne 0) { throw "Test uninstall failed; retained $fixture for inspection." }
    }
    if (Test-Path -LiteralPath $registration) { throw 'Test uninstall left its registration; retained test folder for inspection.' }
    $resolved = [IO.Path]::GetFullPath($fixture)
    $temp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    if (!$resolved.StartsWith($temp, [StringComparison]::OrdinalIgnoreCase) -or (Split-Path $resolved -Leaf) -notlike 'pomme-setup-test-*') { throw 'Unexpected setup test path.' }
    if ($completed) { Remove-Item -LiteralPath $resolved -Recurse -Force }
    else { Write-Host "Retained failed setup test and logs at $fixture" }
}
Write-Host 'Actual setup install, version, readiness, uninstall registration and cleanup checks passed.'
