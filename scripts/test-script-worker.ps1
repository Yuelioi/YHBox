$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$outputDirectory = Join-Path $repositoryRoot ".task"
$worker = Join-Path $outputDirectory "Yotta.ScriptWorker.exe"

New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
& go build -mod=readonly -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui" -o $worker ./cmd/yotta-script-worker
if ($LASTEXITCODE -ne 0) {
    throw "building the isolated script worker failed with exit code $LASTEXITCODE"
}

$env:YOTTA_SCRIPT_WORKER_TEST_EXE = $worker
& go test -count=1 -run '^TestWindowsRuntime' ./internal/scriptengine
if ($LASTEXITCODE -ne 0) {
    throw "the isolated script worker smoke test failed with exit code $LASTEXITCODE"
}
