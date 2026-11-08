function Get-PommeSha256 {
    param([Parameter(Mandatory=$true)][string]$Path)
    $stream = [IO.File]::OpenRead($Path)
    $hash = [Security.Cryptography.SHA256]::Create()
    try { return ([BitConverter]::ToString($hash.ComputeHash($stream))).Replace('-','').ToLowerInvariant() }
    finally { $hash.Dispose(); $stream.Dispose() }
}

function Find-PommeCdm {
    param([string]$Override, [string[]]$Roots)
    if ($Override) {
        if (!(Test-Path -LiteralPath $Override -PathType Leaf)) { throw 'SONORA_WIDEVINE_CDM points to a missing file.' }
        return $Override
    }
    foreach ($root in $Roots) {
        $found = Get-ChildItem -LiteralPath $root -Directory -ErrorAction SilentlyContinue |
            Where-Object { $_.Name -match '^\d+(\.\d+){3}$' } |
            Sort-Object { [version]$_.Name } -Descending |
            ForEach-Object {
                Join-Path $_.FullName 'WidevineCdm\_platform_specific\win_x64\widevinecdm.dll'
                Join-Path $_.FullName '_platform_specific\win_x64\widevinecdm.dll'
            } |
            Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
        if ($found) { return $found }
    }
    return $null
}

function Test-PommeWebView {
    foreach ($base in @('HKLM:\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients', 'HKLM:\SOFTWARE\Microsoft\EdgeUpdate\Clients', 'HKCU:\Software\Microsoft\EdgeUpdate\Clients')) {
        foreach ($key in @(Get-ChildItem -LiteralPath $base -ErrorAction SilentlyContinue)) {
            $item = Get-ItemProperty -LiteralPath $key.PSPath -ErrorAction SilentlyContinue
            if ($item.name -match 'WebView2' -and $item.pv -and $item.pv -ne '0.0.0.0') { return $true }
        }
    }
    return $false
}

function Get-PommeCdmRoots {
    $roots = @(
        (Join-Path $env:LOCALAPPDATA 'Google\Chrome\Application'),
        (Join-Path $env:LOCALAPPDATA 'Google\Chrome\User Data\WidevineCdm'),
        (Join-Path $env:ProgramFiles 'Google\Chrome\Application')
    )
    if (${env:ProgramFiles(x86)}) { $roots += Join-Path ${env:ProgramFiles(x86)} 'Google\Chrome\Application' }
    return $roots
}

function Test-PommeVCRuntime {
    $missing = @('msvcp140.dll','vcruntime140.dll','vcruntime140_1.dll') | Where-Object {
        !(Test-Path -LiteralPath (Join-Path $env:windir "System32\$_") -PathType Leaf)
    }
    return @($missing).Count -eq 0
}
