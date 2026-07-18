$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$previousNativeSmoke = $env:YOTTA_WINDOWS_NATIVE_SMOKE

Push-Location $root
try {
    $env:YOTTA_WINDOWS_NATIVE_SMOKE = "1"
    & go test `
        ./internal/automation/installed `
        ./internal/services/recording `
        ./internal/services/tools `
        ./pkg/winutil `
        -run '^(TestNativeWindowsDriverEndToEnd|TestNativeRecorderProducesCanonicalEncodableInput|TestWindowCaptureReturnsExactForegroundMetadata|TestCaptureKeyboardProcQueuesExactForegroundWindow|TestInspectForegroundWindowState)$' `
        -count=1 `
        -v
    if ($LASTEXITCODE -ne 0) {
        throw "Windows automation native smoke failed with exit code $LASTEXITCODE"
    }
} finally {
    if ($null -eq $previousNativeSmoke) {
        Remove-Item Env:YOTTA_WINDOWS_NATIVE_SMOKE -ErrorAction SilentlyContinue
    } else {
        $env:YOTTA_WINDOWS_NATIVE_SMOKE = $previousNativeSmoke
    }
    Pop-Location
}
