param([switch]$Install, [Parameter(Mandatory=$true)][string]$ResultPath)
$ErrorActionPreference = 'Stop'
$helpers = Join-Path $PSScriptRoot 'Setup.Pomme.Helpers.ps1'
if (!(Test-Path -LiteralPath $helpers)) { $helpers = Join-Path $PSScriptRoot 'Pomme.Helpers.ps1' }
. $helpers
. (Join-Path $PSScriptRoot 'Setup.Helpers.ps1')
try {
    if (![Environment]::Is64BitProcess -or ![Environment]::Is64BitOperatingSystem) { throw 'Setup requires 64-bit Windows 10 or 11 and native 64-bit PowerShell.' }
    if ($Install) {
        $code = Install-PommePrerequisites -Directory $PSScriptRoot
        $message = if ($code -eq 3010) { 'Restart Windows before opening Pomme to finish installing required components.' } else { 'All required components are ready.' }
    } else {
        $missing = @(Get-PommeMissingPrerequisites)
        $code = if ($missing.Count) { 10 } else { 0 }
        $message = if ($missing.Count) { 'Setup will download and install: ' + ($missing.Name -join ', ') + '.' } else { 'All required components are already installed. No prerequisite downloads are needed.' }
    }
    [IO.File]::WriteAllText($ResultPath, $message, [Text.Encoding]::ASCII)
    exit $code
} catch {
    [IO.File]::WriteAllText($ResultPath, $_.Exception.Message, [Text.Encoding]::ASCII)
    exit 1
}
