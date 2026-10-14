param(
    [ValidateSet('play','login','doctor','version','help','lastfm-auth')][string]$Command = 'play',
    [string]$Output,
    [switch]$DefaultOutput
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'Pomme.Helpers.ps1')
try {
    if ($Command -eq 'help') {
        Write-Host 'pomme [play|login|doctor|version|help|lastfm-auth] [-Output "Spotify"] [-DefaultOutput]'
        Write-Host 'Default: follow Windows sound output. Output selects a device name, including Wave Link channels.'
        exit 0
    }
    $frontend = Join-Path $PSScriptRoot 'pomme-ui.exe'
    $backend = Join-Path $PSScriptRoot 'apple-native-engine.exe'
    if ($Command -eq 'version') { & $frontend --version; exit $LASTEXITCODE }
    if (![Environment]::Is64BitOperatingSystem) { throw 'Pomme requires 64-bit Windows 10 or Windows 11.' }
    if (!(Test-Path -LiteralPath $frontend) -or !(Test-Path -LiteralPath $backend)) { throw 'Executables are missing. Extract the complete Windows release ZIP first.' }
    if (!$env:LOCALAPPDATA) { throw 'Windows LOCALAPPDATA is unavailable.' }
    if (!$env:POMME_DATA_DIR) { $env:POMME_DATA_DIR = Join-Path $env:LOCALAPPDATA 'PommeTUI' }
    $env:POMME_DATA_DIR = [IO.Path]::GetFullPath($env:POMME_DATA_DIR)
    $env:AMTUI_CONFIG_DIR = Join-Path $env:POMME_DATA_DIR 'settings'
    $env:AMTUI_NATIVE_BACKEND = $backend
    if (!$env:GOMEMLIMIT) { $env:GOMEMLIMIT = '64MiB' }
    if ($Command -eq 'lastfm-auth') {
        New-Item -ItemType Directory -Force -Path $env:POMME_DATA_DIR | Out-Null
        & $frontend lastfm-auth
        exit $LASTEXITCODE
    }
    # Deliberately replace the prototype's inherited Wave Link setting.
    $env:APPLE_TERMINAL_OUTPUT = if ($DefaultOutput) { '' } elseif ($Output) { $Output } else { $env:POMME_OUTPUT }
    $roots = @(
        (Join-Path $env:LOCALAPPDATA 'Google\Chrome\Application'),
        (Join-Path $env:ProgramFiles 'Google\Chrome\Application')
    )
    if (${env:ProgramFiles(x86)}) { $roots += Join-Path ${env:ProgramFiles(x86)} 'Google\Chrome\Application' }
    $cdm = Find-PommeCdm -Override $env:SONORA_WIDEVINE_CDM -Roots $roots
    $runtimeRoot = Join-Path $env:windir 'System32'
    $missingRuntime = @('msvcp140.dll','vcruntime140.dll','vcruntime140_1.dll') | Where-Object { !(Test-Path -LiteralPath (Join-Path $runtimeRoot $_)) }
    $webview = Test-PommeWebView
    if ($Command -eq 'doctor') {
        Write-Host "Pomme data: $env:POMME_DATA_DIR"
        Write-Host "Audio: $(if ($env:APPLE_TERMINAL_OUTPUT) { $env:APPLE_TERMINAL_OUTPUT } else { 'Windows default' })"
        Write-Host "Widevine found: $([bool]$cdm)"
        Write-Host "Visual C++ runtime found: $($missingRuntime.Count -eq 0)"
        Write-Host "WebView2 runtime found: $webview"
    }
    if (!$cdm) { throw 'Widevine was not found. Install Google Chrome for Windows, or set SONORA_WIDEVINE_CDM to your installed x64 CDM. See README.md.' }
    if ($missingRuntime.Count) { throw 'Install Microsoft Visual C++ Redistributable (x64): https://aka.ms/vc14/vc_redist.x64.exe' }
    if (!$webview) { throw 'Install Microsoft Edge WebView2 Evergreen Runtime: https://developer.microsoft.com/microsoft-edge/webview2/' }
    $env:SONORA_WIDEVINE_CDM = $cdm
    if ($Command -eq 'doctor') { exit 0 }
    New-Item -ItemType Directory -Force -Path $env:POMME_DATA_DIR | Out-Null
    if ($Command -eq 'login') { & $backend login } else { & $frontend }
    exit $LASTEXITCODE
} catch {
    Write-Host "Pomme: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
