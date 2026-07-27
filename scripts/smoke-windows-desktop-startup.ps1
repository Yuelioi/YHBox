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
$smokeRoot = if ($InPlace) { $bin } else { Join-Path $scratchRoot "app" }
$profileRoot = Join-Path $scratchRoot "profile"
if (-not $InPlace) {
    New-Item -ItemType Directory -Path $smokeRoot | Out-Null
}

$process = $null
$secondProcess = $null
$previousStorageRoot = [System.Environment]::GetEnvironmentVariable("YOTTA_ROOT", "Process")
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
        [System.Environment]::SetEnvironmentVariable("YOTTA_ROOT", $profileRoot, "Process")
        $process = Start-Process -FilePath (Join-Path $smokeRoot "Yotta.exe") -WorkingDirectory $smokeRoot `
            -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru
    } finally {
        [System.Environment]::SetEnvironmentVariable("__COMPAT_LAYER", $previousCompatLayer, "Process")
        [System.Environment]::SetEnvironmentVariable("YOTTA_ROOT", $previousStorageRoot, "Process")
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
    $manifest = Join-Path $profileRoot "root.json"
    if (-not (Test-Path -LiteralPath $manifest -PathType Leaf)) {
        throw "Yotta.exe did not claim the isolated storage profile: $manifest"
    }
    foreach ($unexpected in @((Join-Path $smokeRoot "data"), (Join-Path $smokeRoot "settings.json"))) {
        if (Test-Path -LiteralPath $unexpected) {
            throw "Yotta.exe wrote storage beside the executable: $unexpected"
        }
    }

    # Windows PowerShell 5 may leave ExitCode unset on a Start-Process
    # -PassThru object. Use Process directly for the short-lived handoff probe.
    $secondStart = New-Object System.Diagnostics.ProcessStartInfo
    $secondStart.FileName = Join-Path $smokeRoot "Yotta.exe"
    $secondStart.WorkingDirectory = $smokeRoot
    $secondStart.UseShellExecute = $false
    $secondStart.CreateNoWindow = $true
    $secondStart.EnvironmentVariables["__COMPAT_LAYER"] = "RunAsInvoker"
    $secondStart.EnvironmentVariables["YOTTA_ROOT"] = $profileRoot
    $secondProcess = New-Object System.Diagnostics.Process
    $secondProcess.StartInfo = $secondStart
    if (-not $secondProcess.Start()) {
        throw "failed to start the second Yotta.exe instance"
    }
    if (-not $secondProcess.WaitForExit(10000)) {
        throw "second Yotta.exe did not hand off to the existing profile owner within 10 seconds"
    }
    $secondExitCode = $secondProcess.ExitCode
    if ($secondExitCode -ne 0) {
        throw "second Yotta.exe exited with code $secondExitCode instead of handing off"
    }
    $process.Refresh()
    if ($process.HasExited) {
        throw "first Yotta.exe exited while handling the second-instance launch"
    }

    $mode = if ($InPlace) { "in-place" } else { "isolated" }
    Write-Host "$mode desktop startup smoke OK: one profile owner remained alive and a second launch handed off cleanly"
} finally {
    [System.Environment]::SetEnvironmentVariable("YOTTA_ROOT", $previousStorageRoot, "Process")
    foreach ($runningProcess in @($secondProcess, $process)) {
        if ($null -ne $runningProcess -and -not $runningProcess.HasExited) {
            Stop-Process -Id $runningProcess.Id -Force -ErrorAction SilentlyContinue
            if (-not $runningProcess.WaitForExit(10000)) {
                throw "Yotta.exe did not exit within the smoke cleanup deadline"
            }
        }
    }
    $resolvedSmoke = [System.IO.Path]::GetFullPath($scratchRoot)
    $resolvedTask = [System.IO.Path]::GetFullPath($taskRoot)
    if (-not $resolvedSmoke.StartsWith($resolvedTask + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "refusing to clean desktop smoke directory outside .task"
    }
    $removed = $false
    for ($attempt = 1; $attempt -le 5; $attempt++) {
        try {
            Remove-Item -LiteralPath $resolvedSmoke -Recurse -Force
            $removed = $true
            break
        } catch {
            if ($attempt -eq 5) {
                throw
            }
            Start-Sleep -Milliseconds 250
        }
    }
    if (-not $removed) {
        throw "desktop smoke scratch directory was not removed: $resolvedSmoke"
    }
}
