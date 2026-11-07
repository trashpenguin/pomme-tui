param(
    [Parameter(Mandatory=$true)][string]$Version,
    [Parameter(Mandatory=$true)][string]$Destination
)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
Add-Type -AssemblyName System.IO.Compression.FileSystem
Add-Type -AssemblyName System.IO.Compression
if ($Version -notmatch '^\d+\.\d+\.\d+([-.][A-Za-z0-9.]+)?$') { throw 'Invalid source version.' }
if (Test-Path -LiteralPath $Destination) { throw 'Source archive already exists.' }
if (Test-Path -LiteralPath (Join-Path $root '.git')) {
    $files = @(& git -C $root ls-files)
    if ($LASTEXITCODE -ne 0) { throw 'Cannot enumerate Git source files.' }
    $sourceCommit = (& git -C $root rev-parse HEAD).Trim()
} elseif (Test-Path -LiteralPath (Join-Path $root 'SOURCE-MANIFEST.txt')) {
    $files = @(Get-Content -LiteralPath (Join-Path $root 'SOURCE-MANIFEST.txt') | Where-Object { $_ })
    $sourceCommit = (Get-Content -LiteralPath (Join-Path $root 'SOURCE-COMMIT.txt') -Raw).Trim()
} else {
    throw 'Source packaging requires a Git checkout or the manifest from a matching source archive.'
}
$files = @($files | Sort-Object -Unique)
$prefix = "pomme-tui-$Version-source/"
$archive = [IO.Compression.ZipFile]::Open([IO.Path]::GetFullPath($Destination), [IO.Compression.ZipArchiveMode]::Create)
try {
    foreach ($file in $files) {
        if ($file -match '(^|/)(\.git|target|dist|cache|credentials\.json|session\.json)(/|$)' -or [IO.Path]::IsPathRooted($file)) { throw 'Unexpected generated or private source path.' }
        $full = [IO.Path]::GetFullPath((Join-Path $root $file))
        if (!$full.StartsWith([IO.Path]::GetFullPath($root) + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Source path escaped the repository.' }
        [IO.Compression.ZipFileExtensions]::CreateEntryFromFile($archive, $full, $prefix + $file.Replace('\','/'), [IO.Compression.CompressionLevel]::Optimal) | Out-Null
    }
    foreach ($metadata in @(
        @{ Name = 'SOURCE-MANIFEST.txt'; Text = (($files -join "`n") + "`n") },
        @{ Name = 'SOURCE-COMMIT.txt'; Text = ($sourceCommit + "`n") }
    )) {
        $entry = $archive.CreateEntry($prefix + $metadata.Name)
        $writer = [IO.StreamWriter]::new($entry.Open(), [Text.UTF8Encoding]::new($false))
        try { $writer.Write($metadata.Text) } finally { $writer.Dispose() }
    }
} finally { $archive.Dispose() }
Write-Host "Matching source archive: $Destination"
