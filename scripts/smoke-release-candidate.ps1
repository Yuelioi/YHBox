param(
    [string]$StageDirectory = "artifacts/staging/Yotta"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Get-Sha256 {
    param([string]$LiteralPath)
    $stream = [System.IO.File]::OpenRead($LiteralPath)
    $sha256 = [System.Security.Cryptography.SHA256]::Create()
    try {
        return ([System.BitConverter]::ToString($sha256.ComputeHash($stream))).Replace("-", "").ToLowerInvariant()
    } finally {
        $sha256.Dispose()
        $stream.Dispose()
    }
}

$root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$stage = (Resolve-Path -LiteralPath (Join-Path $root $StageDirectory)).Path
$manifestPath = Join-Path $stage "artifact-manifest.json"
$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json

$expected = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::Ordinal)
[void]$expected.Add("artifact-manifest.json")
foreach ($record in $manifest.files) {
    $relative = ([string]$record.path).Replace("/", [System.IO.Path]::DirectorySeparatorChar)
    if (-not $expected.Add(([string]$record.path))) {
        throw "duplicate release manifest path: $($record.path)"
    }
    $path = Join-Path $stage $relative
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "release manifest file is missing: $($record.path)"
    }
    if ((Get-Item -LiteralPath $path).Length -ne [long]$record.size) {
        throw "release manifest size mismatch: $($record.path)"
    }
    if ((Get-Sha256 -LiteralPath $path) -ne [string]$record.sha256) {
        throw "release manifest digest mismatch: $($record.path)"
    }
}

$actual = Get-ChildItem -LiteralPath $stage -File -Recurse | ForEach-Object {
    $_.FullName.Substring($stage.Length + 1).Replace("\", "/")
}
foreach ($path in $actual) {
    if (-not $expected.Contains($path)) {
        throw "release staging contains an unmanifested file: $path"
    }
}
if ($actual.Count -ne $expected.Count) {
    throw "release staging file set does not match the artifact manifest"
}

$smokeRoot = Join-Path $root ".task/release-candidate-smoke"
if (Test-Path -LiteralPath $smokeRoot) {
    Remove-Item -LiteralPath $smokeRoot -Recurse -Force
}
Copy-Item -LiteralPath $stage -Destination $smokeRoot -Recurse

$env:YOTTA_SCRIPT_WORKER_TEST_EXE = Join-Path $smokeRoot "Yotta.ScriptWorker.exe"
& go test -count=1 -run '^TestWindowsRuntime' ./internal/scriptengine
if ($LASTEXITCODE -ne 0) {
    throw "staged script worker smoke failed with exit code $LASTEXITCODE"
}

$env:YOTTA_WASM_PLUGIN_RUNNER = Join-Path $smokeRoot "Yotta.WasmPluginRunner.exe"
& go run ./cmd/plugin-smoke
if ($LASTEXITCODE -ne 0) {
    throw "staged plugin runner smoke failed with exit code $LASTEXITCODE"
}

$legacySource = Join-Path $smokeRoot "legacy-source.json"
[System.IO.File]::WriteAllText($legacySource, '{"format":"yotta.workflow","version":"3"}', [System.Text.UTF8Encoding]::new($false))
$cliOutput = & (Join-Path $smokeRoot "Yotta.CLI.exe") --data-root (Join-Path $smokeRoot "cli-data") validate $legacySource 2>$null
if ($LASTEXITCODE -eq 0 -or ($cliOutput -join "`n") -notmatch 'diagnostics') {
    throw "staged headless CLI did not strictly reject a legacy Workflow source"
}

$appOut = Join-Path $smokeRoot "yotta.out.log"
$appErr = Join-Path $smokeRoot "yotta.err.log"
$process = Start-Process -FilePath (Join-Path $smokeRoot "Yotta.exe") -WorkingDirectory $smokeRoot -WindowStyle Hidden -RedirectStandardOutput $appOut -RedirectStandardError $appErr -PassThru
try {
    Start-Sleep -Seconds 3
    if ($process.HasExited) {
        throw "staged Yotta exited during startup smoke; see $appErr"
    }
} finally {
    if (-not $process.HasExited) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "release candidate manifest and frozen payload smoke passed"
