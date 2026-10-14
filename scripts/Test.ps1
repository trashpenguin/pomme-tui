$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
foreach ($step in @('go-test','go-vet','rust-test','controller-test','apple-only','launcher-test')) {
    switch ($step) {
        'go-test' { & go -C (Join-Path $root 'frontend') test ./... }
        'go-vet' { & go -C (Join-Path $root 'frontend') vet ./... }
        'rust-test' { & cargo +1.98.1 test --locked --manifest-path (Join-Path $root 'backend\Cargo.toml') --lib --target-dir (Join-Path $root 'target') }
        'controller-test' { & cargo +1.98.1 test --locked --release --manifest-path (Join-Path $root 'backend\Cargo.toml') --features backend --bin apple-native-engine controller::tests --target-dir (Join-Path $root 'target') }
        'apple-only' { & (Join-Path $PSScriptRoot 'Test-AppleOnly.ps1') }
        'launcher-test' { & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'Test-Launcher.ps1') }
    }
    if ($LASTEXITCODE -ne 0) { throw "$step failed" }
}
exit 0
