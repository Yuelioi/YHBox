param(
    [string]$BinDirectory = "bin",
    [int]$Seconds = 5,
    [switch]$InPlace
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ($Seconds -lt 1 -or $Seconds -gt 60) {
    throw "Seconds must be between 1 and 60"
}

$root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$bin = (Resolve-Path -LiteralPath (Join-Path $root $BinDirectory)).Path
$taskRoot = Join-Path $root ".task"
New-Item -ItemType Directory -Path $taskRoot -Force | Out-Null
$scratchRoot = Join-Path $taskRoot ("desktop-startup-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $scratchRoot | Out-Null
$smokeRoot = if ($InPlace) { $bin } else { $scratchRoot }

$process = $null
try {
    if (-not $InPlace) {
        foreach ($file in @("Yotta.exe", "Yotta.ScriptWorker.exe", "Yotta.WasmPluginRunner.exe", "capture_wgc.dll")) {
            $source = Join-Path $bin $file
            if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
                throw "desktop startup smoke input is missing: $source"
            }
            Copy-Item -LiteralPath $source -Destination (Join-Path $smokeRoot $file)
        }
    }

    $stdout = Join-Path $scratchRoot "stdout.log"
    $stderr = Join-Path $scratchRoot "stderr.log"
    # The production manifest intentionally requires elevation. The smoke
    # separately verifies that manifest and the WINDOWS_GUI subsystem, then
    # uses the Windows test shim only for this child so unattended checks can
    # exercise application startup without an interactive UAC prompt.
    $previousCompatLayer = [System.Environment]::GetEnvironmentVariable("__COMPAT_LAYER", "Process")
    try {
        [System.Environment]::SetEnvironmentVariable("__COMPAT_LAYER", "RunAsInvoker", "Process")
        $process = Start-Process -FilePath (Join-Path $smokeRoot "Yotta.exe") -WorkingDirectory $smokeRoot `
            -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru
    } finally {
        [System.Environment]::SetEnvironmentVariable("__COMPAT_LAYER", $previousCompatLayer, "Process")
    }
    Start-Sleep -Seconds $Seconds
    if ($process.HasExited) {
        $errorText = if (Test-Path -LiteralPath $stderr) {
            [System.IO.File]::ReadAllText($stderr)
        } else {
            ""
        }
        throw "Yotta.exe exited during isolated startup smoke with code $($process.ExitCode): $errorText"
    }
    $mode = if ($InPlace) { "in-place" } else { "isolated" }
    Write-Host "$mode desktop startup smoke OK: Yotta.exe remained alive for $Seconds seconds"
} finally {
    if ($null -ne $process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    }
    $resolvedSmoke = [System.IO.Path]::GetFullPath($scratchRoot)
    $resolvedTask = [System.IO.Path]::GetFullPath($taskRoot)
    if (-not $resolvedSmoke.StartsWith($resolvedTask + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "refusing to clean desktop smoke directory outside .task"
    }
    Remove-Item -LiteralPath $resolvedSmoke -Recurse -Force
}
