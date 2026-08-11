param(
    [string]$BinDirectory = "bin",
    [int]$Seconds = 4
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if ($Seconds -lt 1 -or $Seconds -gt 60) {
    throw "Seconds must be between 1 and 60"
}

$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$binRoot = (Resolve-Path -LiteralPath (Join-Path $repoRoot $BinDirectory)).Path
$taskRoot = Join-Path $repoRoot ".task"
New-Item -ItemType Directory -Path $taskRoot -Force | Out-Null
$scratchRoot = Join-Path $taskRoot ("storage-migration-" + [guid]::NewGuid().ToString("N"))
$profileRoot = Join-Path $scratchRoot "profile"
$legacyRoot = Join-Path $profileRoot "data\workspace\runs"
$migrationRoot = Join-Path $profileRoot "backups\migrations\layout-1-to-2"
$journalPath = Join-Path $migrationRoot "journal.json"
$blocker = "0190c7d4-1e40-7cc5-a783-57b16d5c8e3a.json"
$cli = Join-Path $binRoot "Yotta.CLI.exe"
$gui = Join-Path $binRoot "Yotta.exe"

foreach ($path in @($cli, $gui)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "storage migration smoke input is missing: $path"
    }
}

$process = $null
$previousStorageRoot = [Environment]::GetEnvironmentVariable("YOTTA_ROOT", "Process")
$previousCompatLayer = [Environment]::GetEnvironmentVariable("__COMPAT_LAYER", "Process")

function Start-YottaSmokeProcess {
    param(
        [Parameter(Mandatory = $true)]
        [string]$StandardOutput,
        [Parameter(Mandatory = $true)]
        [string]$StandardError
    )
    try {
        [Environment]::SetEnvironmentVariable("YOTTA_ROOT", $profileRoot, "Process")
        [Environment]::SetEnvironmentVariable("__COMPAT_LAYER", "RunAsInvoker", "Process")
        return Start-Process -FilePath $gui -WorkingDirectory $binRoot -WindowStyle Hidden `
            -RedirectStandardOutput $StandardOutput -RedirectStandardError $StandardError -PassThru
    } finally {
        [Environment]::SetEnvironmentVariable("YOTTA_ROOT", $previousStorageRoot, "Process")
        [Environment]::SetEnvironmentVariable("__COMPAT_LAYER", $previousCompatLayer, "Process")
    }
}

function Stop-YottaSmokeProcess {
    param(
        [Parameter(Mandatory = $true)]
        [System.Diagnostics.Process]$Target
    )
    if (-not $Target.HasExited) {
        Stop-Process -Id $Target.Id -Force -ErrorAction SilentlyContinue
        if (-not $Target.WaitForExit(10000)) {
            throw "Yotta.exe did not exit within the smoke cleanup deadline"
        }
    }
}

try {
    New-Item -ItemType Directory -Path (Join-Path $profileRoot "config") -Force | Out-Null
    New-Item -ItemType Directory -Path $legacyRoot -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $repoRoot "internal\storage\migrate\testdata\layout-1\root.json") `
        -Destination (Join-Path $profileRoot "root.json")
    Copy-Item -LiteralPath (Join-Path $repoRoot "internal\storage\migrate\testdata\layout-1\config\settings.json") `
        -Destination (Join-Path $profileRoot "config\settings.json")
    Copy-Item -LiteralPath (Join-Path $repoRoot "internal\storage\migrate\testdata\invalid-legacy-run\.yotta-run-store") `
        -Destination (Join-Path $legacyRoot ".yotta-run-store")
    Copy-Item -LiteralPath (Join-Path $repoRoot "internal\storage\migrate\testdata\invalid-legacy-run\$blocker") `
        -Destination (Join-Path $legacyRoot $blocker)

    $planText = & $cli --data-root $profileRoot migrate plan
    if ($LASTEXITCODE -ne 0) {
        throw "production migrate plan failed with code $LASTEXITCODE"
    }
    $plan = $planText | ConvertFrom-Json
    if ($plan.from -ne "1" -or $plan.to -ne "2" -or $plan.legacyRunRecords -ne 1) {
        throw "production migrate plan did not describe the frozen layout 1 fixture"
    }
    if (Test-Path -LiteralPath $migrationRoot) {
        throw "production migrate plan wrote migration state"
    }

    $savedErrorActionPreference = $ErrorActionPreference
    try {
        # Windows PowerShell promotes native stderr to ErrorRecord when the
        # global preference is Stop. This failure is the recovery fixture's
        # expected transition, so capture its real process exit code directly.
        $ErrorActionPreference = "Continue"
        $applyOutput = & $cli --data-root $profileRoot migrate apply 2>&1
        $applyExitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $savedErrorActionPreference
    }
    if ($applyExitCode -eq 0) {
        throw "invalid legacy Run unexpectedly migrated: $applyOutput"
    }
    $journal = Get-Content -LiteralPath $journalPath -Raw | ConvertFrom-Json
    if ($journal.state -ne "recovery-required" -or $journal.blockedEntry -ne $blocker) {
        throw "production apply did not persist the exact recovery blocker"
    }

    $process = Start-YottaSmokeProcess `
        -StandardOutput (Join-Path $scratchRoot "recovery.out.log") `
        -StandardError (Join-Path $scratchRoot "recovery.err.log")
    Start-Sleep -Seconds $Seconds
    if ($process.HasExited) {
        throw "recovery GUI exited early with code $($process.ExitCode)"
    }
    Stop-YottaSmokeProcess -Target $process
    $process = $null
    $journalAfterKill = Get-Content -LiteralPath $journalPath -Raw | ConvertFrom-Json
    if ($journalAfterKill.state -ne "recovery-required") {
        throw "recovery GUI kill changed the durable recovery state"
    }

    & $cli --data-root $profileRoot migrate quarantine $blocker | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "production quarantine failed with code $LASTEXITCODE"
    }
    & $cli --data-root $profileRoot migrate resume | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "production resume failed with code $LASTEXITCODE"
    }
    $healthText = & $cli --data-root $profileRoot health
    if ($LASTEXITCODE -ne 0) {
        throw "production health failed with code $LASTEXITCODE"
    }
    $health = $healthText | ConvertFrom-Json
    if (-not $health.supported -or $health.layoutVersion -ne "3" -or
        -not $health.databases.healthy -or
        -not $health.databases.content.healthy -or
        -not $health.databases.runs.healthy) {
        throw "migrated production health did not report layout 3 and two healthy databases"
    }

    $process = Start-YottaSmokeProcess `
        -StandardOutput (Join-Path $scratchRoot "migrated.out.log") `
        -StandardError (Join-Path $scratchRoot "migrated.err.log")
    Start-Sleep -Seconds $Seconds
    if ($process.HasExited) {
        throw "migrated GUI exited early with code $($process.ExitCode)"
    }
    Stop-YottaSmokeProcess -Target $process
    $process = $null

    Write-Host "production storage migration smoke OK: dry-run stayed read-only, recovery GUI survived kill, quarantine/resume completed layout 1 to 2 to 3, both databases are healthy, and migrated GUI remained alive"
} finally {
    [Environment]::SetEnvironmentVariable("YOTTA_ROOT", $previousStorageRoot, "Process")
    [Environment]::SetEnvironmentVariable("__COMPAT_LAYER", $previousCompatLayer, "Process")
    if ($null -ne $process) {
        Stop-YottaSmokeProcess -Target $process
    }
    $resolvedScratch = [IO.Path]::GetFullPath($scratchRoot)
    $resolvedTask = [IO.Path]::GetFullPath($taskRoot)
    if (-not $resolvedScratch.StartsWith(
        $resolvedTask + [IO.Path]::DirectorySeparatorChar,
        [StringComparison]::OrdinalIgnoreCase
    )) {
        throw "refusing to clean storage migration smoke directory outside .task"
    }
    if (Test-Path -LiteralPath $resolvedScratch) {
        Remove-Item -LiteralPath $resolvedScratch -Recurse -Force
    }
}
