param(
    [ValidateRange(1024, 65535)]
    [int]$DebugPort = 9227,
    [ValidateRange(1024, 65535)]
    [int]$VitePort = 9245,
    [switch]$SkipLauncher
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$relativeRunRoot = ".task/workflow-editor-smoke/$stamp"
$runRoot = Join-Path $root $relativeRunRoot
$binDir = Join-Path $runRoot 'bin'
$profileRoot = Join-Path $runRoot 'profile'
$screenshot = Join-Path $runRoot 'workflow-editor.png'
$assetsScreenshot = Join-Path $runRoot 'assets.png'
$workflowsScreenshot = Join-Path $runRoot 'workflows.png'
$schedulesScreenshot = Join-Path $runRoot 'schedules.png'
$subgraphScreenshot = Join-Path $runRoot 'subgraph.png'
$launcherScreenshot = Join-Path $runRoot 'launcher.png'
$appProcess = $null
$viteProcess = $null
$viteListenerPID = $null
$debugEndpoint = $null
$previousStorageRoot = $env:YOTTA_ROOT
$previousDebugPort = $env:YOTTA_WEBVIEW_DEBUG_PORT
$previousDebugProfile = $env:YOTTA_WEBVIEW_DEBUG_PROFILE
$previousFrontendDevServer = $env:FRONTEND_DEVSERVER_URL

New-Item -ItemType Directory -Force -Path $binDir | Out-Null

try {
    Push-Location $root
    task windows:build DEV=true BIN_DIR="$relativeRunRoot/bin"
    if ($LASTEXITCODE -ne 0) {
        throw "Wails DEV build failed with exit code $LASTEXITCODE"
    }
    # The production host intentionally embeds requireAdministrator. CDP UI
    # smoke runs hidden and cannot answer UAC, so compile an unmanifested
    # development-only host against the same freshly built frontend/assets.
    go build -mod=readonly -buildvcs=false -gcflags='all=-l' -o (Join-Path $binDir 'Yotta.SmokeHost.exe') .
    if ($LASTEXITCODE -ne 0) {
        throw "Wails smoke host build failed with exit code $LASTEXITCODE"
    }

    # A single malformed user Source must be isolated without preventing the
    # real desktop host from starting or hiding the rest of the workflow list.
    # Seed an explicitly owned root before adding the Source fixture; a
    # non-empty profile without root.json must fail closed.
    $utf8NoBom = [System.Text.UTF8Encoding]::new($false)
    New-Item -ItemType Directory -Force -Path $profileRoot | Out-Null
    [System.IO.File]::WriteAllText(
        (Join-Path $profileRoot 'root.json'),
        '{"format":"yotta.storage-root","version":"1"}',
        $utf8NoBom
    )
    $workflowStore = Join-Path $profileRoot 'data/workspace/workflows'
    New-Item -ItemType Directory -Force -Path $workflowStore | Out-Null
    [System.IO.File]::WriteAllText((Join-Path $workflowStore '.yotta-workflow-source-store'), "yotta/workflow-source-store/1`n", $utf8NoBom)
    [System.IO.File]::WriteAllText((Join-Path $workflowStore 'damaged-workflow.json'), '{"format":"yotta.workflow","version":"1",', $utf8NoBom)

    try {
        Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$VitePort" -TimeoutSec 1 | Out-Null
    } catch {
        $viteOut = Join-Path $runRoot 'vite.out.log'
        $viteErr = Join-Path $runRoot 'vite.err.log'
        $pnpm = Get-Command pnpm -CommandType Application -ErrorAction Stop | Select-Object -First 1
        $viteProcess = Start-Process -FilePath $pnpm.Source -ArgumentList @('-C', 'frontend', 'dev', '--host', '127.0.0.1', '--port', $VitePort) -WorkingDirectory $root -WindowStyle Hidden -RedirectStandardOutput $viteOut -RedirectStandardError $viteErr -PassThru
        for ($attempt = 0; $attempt -lt 100; $attempt++) {
            Start-Sleep -Milliseconds 100
            $listener = Get-NetTCPConnection -State Listen -LocalPort $VitePort -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($listener) {
                $viteListenerPID = $listener.OwningProcess
                break
            }
        }
        if (-not $viteListenerPID) {
            throw "Vite did not listen on port $VitePort"
        }
    }

    $env:YOTTA_WEBVIEW_DEBUG_PORT = [string]$DebugPort
    $env:YOTTA_WEBVIEW_DEBUG_PROFILE = Join-Path $runRoot 'webview2'
    $env:FRONTEND_DEVSERVER_URL = "http://127.0.0.1:$VitePort"
    $env:YOTTA_ROOT = $profileRoot
    $appOut = Join-Path $runRoot 'yotta.out.log'
    $appErr = Join-Path $runRoot 'yotta.err.log'
    $appProcess = Start-Process -FilePath (Join-Path $binDir 'Yotta.SmokeHost.exe') -WorkingDirectory $binDir -WindowStyle Hidden -RedirectStandardOutput $appOut -RedirectStandardError $appErr -PassThru

    for ($attempt = 0; $attempt -lt 300; $attempt++) {
        Start-Sleep -Milliseconds 100
        if ($appProcess.HasExited) {
            throw "Yotta exited before exposing WebView CDP; see $appErr"
        }
        $debugListeners = Get-NetTCPConnection -State Listen -LocalPort $DebugPort -ErrorAction SilentlyContinue
        if (-not $debugListeners) {
            continue
        }
        if ($debugListeners | Where-Object { $_.LocalAddress -in @('127.0.0.1', '0.0.0.0') }) {
            $candidateEndpoint = "http://127.0.0.1:$DebugPort"
        } elseif ($debugListeners | Where-Object { $_.LocalAddress -in @('::1', '::') }) {
            $candidateEndpoint = "http://[::1]:$DebugPort"
        } else {
            continue
        }
        try {
            Invoke-WebRequest -UseBasicParsing -Uri "$candidateEndpoint/json" -TimeoutSec 1 | Out-Null
            $debugEndpoint = $candidateEndpoint
            break
        } catch {
            if ($attempt -eq 299) {
                throw "WebView CDP listened on port $DebugPort but did not expose /json"
            }
        }
    }
    if (-not $debugEndpoint) {
        throw "WebView CDP did not listen on IPv4 or IPv6 loopback port $DebugPort"
    }

    $smokeArgs = @(
        'run', './cmd/workflow-editor-smoke',
        '-endpoint', $debugEndpoint,
        '-screenshot', $screenshot,
        '-assets-screenshot', $assetsScreenshot,
        '-workflows-screenshot', $workflowsScreenshot,
        '-schedules-screenshot', $schedulesScreenshot,
        '-subgraph-screenshot', $subgraphScreenshot
    )
    if (-not $SkipLauncher) {
        $smokeArgs += @('-launcher-screenshot', $launcherScreenshot)
    } else {
        $smokeArgs += '-launcher-screenshot='
    }
    & go @smokeArgs
    if ($LASTEXITCODE -ne 0) {
        throw "Workflow editor smoke failed with exit code $LASTEXITCODE"
    }
} finally {
    $env:YOTTA_ROOT = $previousStorageRoot
    $env:YOTTA_WEBVIEW_DEBUG_PORT = $previousDebugPort
    $env:YOTTA_WEBVIEW_DEBUG_PROFILE = $previousDebugProfile
    $env:FRONTEND_DEVSERVER_URL = $previousFrontendDevServer
    if ($appProcess -and -not $appProcess.HasExited) {
        Stop-Process -Id $appProcess.Id -Force -ErrorAction SilentlyContinue
    }
    if ($viteListenerPID) {
        Stop-Process -Id $viteListenerPID -Force -ErrorAction SilentlyContinue
    } elseif ($viteProcess -and -not $viteProcess.HasExited) {
        Stop-Process -Id $viteProcess.Id -Force -ErrorAction SilentlyContinue
    }
    Pop-Location -ErrorAction SilentlyContinue
}
