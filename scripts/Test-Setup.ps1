$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'Pomme.Helpers.ps1')
. (Join-Path $PSScriptRoot 'Setup.Helpers.ps1')
function Assert-Setup($Condition, $Message) { if (!$Condition) { throw $Message } }
function Assert-SetupFailure([scriptblock]$Action, [string]$Pattern) {
    $message = ''
    try { & $Action | Out-Null } catch { $message = $_.Exception.Message }
    Assert-Setup ($message -match $Pattern) "Expected failure matching $Pattern; got: $message"
}
Assert-Setup (!(Test-PommeInstallerSignature -Path $PSCommandPath -Publisher 'Microsoft Corporation')) 'An unsigned file must fail the real signature check.'
$restartFile = Join-Path ([IO.Path]::GetTempPath()) ('pomme-restart-test-' + [guid]::NewGuid() + '.txt')
$script:boot = [datetime]::UtcNow.AddHours(-1)
function Get-PommeBootTime { return $script:boot }
try {
    Add-PommePendingRestart -Id 'vc' -Path $restartFile
    Add-PommePendingRestart -Id 'webview' -Path $restartFile
    $pending = @(Get-PommePendingRestart -Path $restartFile)
    Assert-Setup ($pending.Count -eq 2 -and $pending -contains 'vc' -and $pending -contains 'webview') 'Restart state must retain independent component IDs.'
    $script:boot = [datetime]::UtcNow.AddMinutes(1)
    Assert-Setup (@(Get-PommePendingRestart -Path $restartFile).Count -eq 0 -and !(Test-Path -LiteralPath $restartFile)) 'Restart state must clear after a new Windows boot.'
} finally { if (Test-Path -LiteralPath $restartFile) { Remove-Item -LiteralPath $restartFile -Force } }
$script:missing = @()
$script:downloaded = 0
$script:started = 0
$script:validSignature = $true
$script:installerExit = 0
$script:becomesReady = $true
$script:pendingRestart = @()
function Get-PommePendingRestart { return $script:pendingRestart }
function Add-PommePendingRestart($Id) { $script:pendingRestart += $Id }
function Get-PommeMissingPrerequisites { return $script:missing }
function Save-PommePrerequisite($Prerequisite, $Directory) { $script:downloaded++; return (Join-Path $Directory 'fixture.exe') }
function Test-PommeInstallerSignature($Path, $Publisher) { return $script:validSignature }
function Start-PommePrerequisiteInstaller($Prerequisite, $Path) {
    $script:started++
    if ($script:becomesReady) { $script:missing = @() }
    return $script:installerExit
}
$fixture = [pscustomobject]@{ Name='Fixture runtime'; Publisher='Fixture publisher'; Id='fixture' }
$directory = [IO.Path]::GetTempPath()
$result = Install-PommePrerequisites -Directory $directory
Assert-Setup ($result -eq 0 -and $script:downloaded -eq 0 -and $script:started -eq 0) 'Ready machines must not download or reinstall prerequisites.'

$script:missing = @($fixture); $script:validSignature = $false
Assert-SetupFailure { Install-PommePrerequisites -Directory $directory } 'signature'
Assert-Setup ($script:started -eq 0) 'Untrusted downloads must never execute.'

$script:validSignature = $true; $script:installerExit = 1603; $script:becomesReady = $false
Assert-SetupFailure { Install-PommePrerequisites -Directory $directory } '1603'

$script:installerExit = 0
Assert-SetupFailure { Install-PommePrerequisites -Directory $directory } 'still missing'

$script:becomesReady = $true
Assert-Setup ((Install-PommePrerequisites -Directory $directory) -eq 0) 'Successful installation must recheck readiness.'

$script:missing = @($fixture); $script:installerExit = 3010; $script:becomesReady = $false
Assert-Setup ((Install-PommePrerequisites -Directory $directory) -eq 3010) 'Required restarts must reach setup rather than being reported as ready.'
Assert-Setup ($script:pendingRestart -contains 'fixture') 'Restart requirements must survive a failed later component and retries.'
$script:missing = @()
Assert-Setup ((Install-PommePrerequisites -Directory $directory) -eq 3010) 'A retry with present DLLs must preserve an outstanding restart.'
$script:missing = @([pscustomobject]@{Name='Other runtime';Id='other';Publisher='Fixture publisher'})
$script:installerExit = 0
Assert-SetupFailure { Install-PommePrerequisites -Directory $directory } 'still missing'
Write-Host 'Setup prerequisite safety, skip, failure, recheck and restart tests passed.'
