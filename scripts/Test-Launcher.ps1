$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'Pomme.Helpers.ps1')
$fixture = Join-Path ([IO.Path]::GetTempPath()) ('pomme-test-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path $fixture | Out-Null
try {
    foreach ($version in @('9.0.0.0','100.0.0.0')) {
        $dir = Join-Path $fixture "$version\WidevineCdm\_platform_specific\win_x64"
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
        Set-Content -LiteralPath (Join-Path $dir 'widevinecdm.dll') -Value 'fixture'
    }
    $found = Find-PommeCdm -Roots @($fixture)
    if ($found -notlike '*100.0.0.0*') { throw 'CDM discovery did not select newest version.' }
    if (Find-PommeCdm -Roots @((Join-Path $fixture 'missing'))) { throw 'Missing CDM falsely detected.' }
    if ((Find-PommeCdm -Override $found) -ne $found) { throw 'Explicit CDM was not preserved.' }
    $rejected = $false
    try { Find-PommeCdm -Override (Join-Path $fixture 'missing.dll') | Out-Null } catch { $rejected = $true }
    if (!$rejected) { throw 'Missing override was accepted.' }
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'Run-Pomme.ps1') help
    if ($LASTEXITCODE -ne 0) { throw 'Help failed without installed dependencies.' }
    Write-Host 'Launcher discovery and help tests passed.'
} finally {
    # Only the unique test folder created above may be removed.
    $resolved = [IO.Path]::GetFullPath($fixture)
    $temp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    if ($resolved.StartsWith($temp, [StringComparison]::OrdinalIgnoreCase) -and (Split-Path $resolved -Leaf) -like 'pomme-test-*') {
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}
