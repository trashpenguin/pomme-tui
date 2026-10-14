param([Parameter(Mandatory=$true)][string]$ReleasePath)
$ErrorActionPreference = 'Stop'
$ReleasePath = [IO.Path]::GetFullPath($ReleasePath)
foreach ($name in @('pomme-ui.exe','apple-native-engine.exe')) {
    $bytes = [IO.File]::ReadAllBytes((Join-Path $ReleasePath $name))
    $profileRoot = [Environment]::GetFolderPath('UserProfile')
    foreach ($encoding in @([Text.Encoding]::UTF8,[Text.Encoding]::Unicode)) {
        $binary = $encoding.GetString($bytes)
        if ($binary.IndexOf($profileRoot,[StringComparison]::OrdinalIgnoreCase) -ge 0 -or $binary.IndexOf($profileRoot.Replace('\','/'),[StringComparison]::OrdinalIgnoreCase) -ge 0) {
            throw 'Release binary contains local build paths. Check the compiler path-remapping and source-filename cleanup.'
        }
    }
}
$fixture = Join-Path ([IO.Path]::GetTempPath()) ('pomme-release-test-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path $fixture | Out-Null
$previous = @{}
foreach ($key in @('POMME_DATA_DIR','POMME_OUTPUT','APPLE_TERMINAL_OUTPUT','SONORA_WIDEVINE_CDM')) { $previous[$key] = [Environment]::GetEnvironmentVariable($key,'Process') }
try {
    $installed = Join-Path $fixture 'Installed With Spaces'
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $ReleasePath 'Install.ps1') -Destination $installed
    if ($LASTEXITCODE -ne 0) { throw 'Install failed.' }
    $tampered = Join-Path $fixture 'Tampered Release'
    Copy-Item -LiteralPath $ReleasePath -Destination $tampered -Recurse
    [IO.File]::AppendAllText((Join-Path $tampered 'pomme-ui.exe'),'corrupt fixture')
    $rejectedDestination = Join-Path $fixture 'Rejected Install'
    $savedErrorPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $rejected = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $tampered 'Install.ps1') -Destination $rejectedDestination 2>&1
        $corruptExit = $LASTEXITCODE
    } finally { $ErrorActionPreference = $savedErrorPreference }
    if ($corruptExit -eq 0 -or (Test-Path -LiteralPath $rejectedDestination)) { throw 'Installer accepted a corrupted executable.' }
    $version = & (Join-Path $installed 'pomme.cmd') version
    if ($LASTEXITCODE -ne 0 -or $version -notmatch 'Pomme TUI') { throw 'Installed command did not report Pomme version.' }
    $env:POMME_DATA_DIR = Join-Path $fixture 'Fresh Profile'
    $env:POMME_OUTPUT = ''
    $env:APPLE_TERMINAL_OUTPUT = 'Spotify'
    $cdmFixture = Join-Path $fixture 'widevinecdm.dll'
    Set-Content -LiteralPath $cdmFixture -Value 'discovery fixture, never loaded'
    $env:SONORA_WIDEVINE_CDM = $cdmFixture
    $doctor = & (Join-Path $installed 'pomme.cmd') doctor
    if ($doctor -notcontains 'Audio: Windows default') { throw 'Prototype output setting leaked into Pomme default.' }
    $doctor = & (Join-Path $installed 'pomme.cmd') doctor -Output 'Wave Link Spotify'
    if ($doctor -notcontains 'Audio: Wave Link Spotify') { throw 'Explicit audio option was lost.' }
    $env:SONORA_WIDEVINE_CDM = Join-Path $fixture 'missing.dll'
    $missing = & (Join-Path $installed 'pomme.cmd') doctor
    if ($LASTEXITCODE -eq 0 -or ($missing -join ' ') -notmatch 'missing file') { throw 'Missing CDM did not fail clearly.' }

    $info = New-Object System.Diagnostics.ProcessStartInfo
    $info.FileName = Join-Path $installed 'apple-native-engine.exe'
    $info.UseShellExecute = $false
    $info.CreateNoWindow = $true
    $info.RedirectStandardInput = $true
    $info.RedirectStandardOutput = $true
    $info.RedirectStandardError = $true
    $info.EnvironmentVariables['POMME_DATA_DIR'] = $env:POMME_DATA_DIR
    $info.EnvironmentVariables.Remove('SONORA_APPLE_MEDIA_USER_TOKEN')
    $info.EnvironmentVariables.Remove('SONORA_APPLE_BEARER_TOKEN')
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $info
    if (!$process.Start()) { throw 'Backend did not start.' }
    try {
        $process.StandardInput.WriteLine('{"id":1,"method":"hello","params":{}}')
        $replyTask = $process.StandardOutput.ReadLineAsync()
        if (!$replyTask.Wait(30000)) { throw 'Fresh profile hello timed out.' }
        $reply = $replyTask.Result | ConvertFrom-Json
        if ($reply.id -ne 1 -or $reply.result.protocol -ne 1 -or $reply.result.authed -ne $false) { throw 'Fresh profile inherited another application session.' }
        $process.StandardInput.WriteLine('{"id":2,"method":"shutdown","params":{}}')
        $process.StandardInput.Close()
        if (!$process.WaitForExit(10000) -or $process.ExitCode -ne 0) { throw 'Backend did not shut down cleanly.' }
    } finally {
        if (!$process.HasExited) { $process.Kill(); $process.WaitForExit() }
        $process.Dispose()
    }
    if (Test-Path -LiteralPath (Join-Path $env:POMME_DATA_DIR 'cache\apple\credentials.json')) { throw 'Fresh profile test created credentials.' }
    Write-Host 'Release privacy, install, corruption rejection, version, audio options, dependency error, fresh profile, and shutdown checks passed.'
} finally {
    foreach ($key in $previous.Keys) { [Environment]::SetEnvironmentVariable($key,$previous[$key],'Process') }
    $resolved = [IO.Path]::GetFullPath($fixture)
    $temp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    if ($resolved.StartsWith($temp,[StringComparison]::OrdinalIgnoreCase) -and (Split-Path $resolved -Leaf) -like 'pomme-release-test-*') { Remove-Item -LiteralPath $resolved -Recurse -Force }
}
