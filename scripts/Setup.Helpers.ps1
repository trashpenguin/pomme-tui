# Called only by setup; ordinary player startup never installs software.
function Get-PommeMissingPrerequisites {
    if (!(Test-PommeVCRuntime)) {
        [pscustomobject]@{ Id='vc'; Name='Microsoft Visual C++ x64 Runtime'; Publisher='Microsoft Corporation'; Uri='https://aka.ms/vc14/vc_redist.x64.exe'; File='vc_redist.x64.exe' }
    }
    if (!(Test-PommeWebView)) {
        [pscustomobject]@{ Id='webview'; Name='Microsoft Edge WebView2 Runtime'; Publisher='Microsoft Corporation'; Uri='https://go.microsoft.com/fwlink/p/?LinkId=2124703'; File='MicrosoftEdgeWebview2Setup.exe' }
    }
    if (!(Find-PommeCdm -Override $env:SONORA_WIDEVINE_CDM -Roots (Get-PommeCdmRoots))) {
        [pscustomobject]@{ Id='chrome'; Name='Google Chrome (includes Widevine)'; Publisher='Google LLC'; Uri='https://dl.google.com/dl/chrome/install/googlechromestandaloneenterprise64.msi'; File='GoogleChromeStandaloneEnterprise64.msi' }
    }
}

function Save-PommePrerequisite($Prerequisite, [string]$Directory) {
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    $path = Join-Path $Directory $Prerequisite.File
    $savedProgress = $ProgressPreference
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $Prerequisite.Uri -OutFile $path -UseBasicParsing -TimeoutSec 300 -MaximumRedirection 5
    } finally { $ProgressPreference = $savedProgress }
    return $path
}

function Test-PommeInstallerSignature([string]$Path, [string]$Publisher) {
    $signature = Get-AuthenticodeSignature -LiteralPath $Path
    return $signature.Status -eq 'Valid' -and $null -ne $signature.SignerCertificate -and
        $signature.SignerCertificate.GetNameInfo([Security.Cryptography.X509Certificates.X509NameType]::SimpleName, $false) -eq $Publisher
}

function Start-PommePrerequisiteInstaller($Prerequisite, [string]$Path) {
    $parameters = @{ FilePath=$Path; ArgumentList=@('/install','/quiet','/norestart'); PassThru=$true; WindowStyle='Hidden' }
    if ($Prerequisite.Id -eq 'chrome') {
        $parameters.FilePath = Join-Path $env:windir 'System32\msiexec.exe'
        $parameters.ArgumentList = @('/i', ('"{0}"' -f $Path), '/qn', '/norestart')
    } elseif ($Prerequisite.Id -eq 'webview') {
        $parameters.ArgumentList = @('/silent','/install')
    }
    if ($Prerequisite.Id -ne 'webview') { $parameters.Verb = 'RunAs' }
    $process = Start-Process @parameters
    try {
        if (!$process.WaitForExit(900000)) { throw "$($Prerequisite.Name) is still running. Wait for it to finish, then retry setup." }
        return $process.ExitCode
    } finally { $process.Dispose() }
}

function Get-PommeBootTime { return (Get-CimInstance Win32_OperatingSystem).LastBootUpTime.ToUniversalTime() }

function Get-PommePendingRestart([string]$Path = (Join-Path $env:LOCALAPPDATA 'PommeTUI\setup\restart-required.txt')) {
    if (!(Test-Path -LiteralPath $Path)) { return @() }
    if ((Get-Item -LiteralPath $Path).LastWriteTimeUtc -lt (Get-PommeBootTime)) {
        Remove-Item -LiteralPath $Path -Force
        return @()
    }
    return @(Get-Content -LiteralPath $Path | Where-Object { $_ })
}

function Add-PommePendingRestart([string]$Id, [string]$Path = (Join-Path $env:LOCALAPPDATA 'PommeTUI\setup\restart-required.txt')) {
    $pending = @(@(Get-PommePendingRestart -Path $Path) + @($Id) | Select-Object -Unique)
    New-Item -ItemType Directory -Force -Path (Split-Path $path -Parent) | Out-Null
    $pending | Set-Content -LiteralPath $path -Encoding ascii
}

function Install-PommePrerequisites([string]$Directory) {
    foreach ($item in @(Get-PommeMissingPrerequisites)) {
        if (@(Get-PommePendingRestart) -contains $item.Id) { continue }
        Write-Host "Installing $($item.Name)..."
        $path = Save-PommePrerequisite -Prerequisite $item -Directory $Directory
        if (!(Test-PommeInstallerSignature -Path $path -Publisher $item.Publisher)) {
            throw "The downloaded $($item.Name) installer failed its trusted publisher signature check. Nothing from this download was run. Retry setup with a working internet connection."
        }
        $code = Start-PommePrerequisiteInstaller -Prerequisite $item -Path $path
        if ($code -eq 3010) { Add-PommePendingRestart -Id $item.Id }
        elseif ($code -ne 0) { throw "$($item.Name) installation failed (exit $code). Allow the Windows permission prompt, check your internet connection, and retry setup." }
    }
    $pending = @(Get-PommePendingRestart)
    $missing = @(Get-PommeMissingPrerequisites | Where-Object { $pending -notcontains $_.Id })
    if ($missing.Count) { throw "Required components are still missing: $(($missing.Name) -join ', '). Restart Windows if an installer requested it, then retry setup." }
    if ($pending.Count) { return 3010 }
    return 0
}
