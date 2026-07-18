param(
    [ValidateRange(1024, 65535)]
    [int]$DebugPort = 9227,
    [ValidateRange(1024, 65535)]
    [int]$VitePort = 9245
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$relativeRunRoot = ".task/workflow-editor-smoke/$stamp"
$runRoot = Join-Path $root $relativeRunRoot
$binDir = Join-Path $runRoot 'bin'
$screenshot = Join-Path $runRoot 'workflow-editor.png'
$assetsScreenshot = Join-Path $runRoot 'assets.png'
$workflowsScreenshot = Join-Path $runRoot 'workflows.png'
$launcherScreenshot = Join-Path $runRoot 'launcher.png'
$appProcess = $null
$viteProcess = $null
$viteListenerPID = $null

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
    $workflowStore = Join-Path $binDir 'data/workspace/workflows'
    New-Item -ItemType Directory -Force -Path $workflowStore | Out-Null
    $utf8NoBom = [System.Text.UTF8Encoding]::new($false)
    [System.IO.File]::WriteAllText((Join-Path $workflowStore '.yotta-workflow-source-store'), "yotta/workflow-source-store/3.1`n", $utf8NoBom)
    [System.IO.File]::WriteAllText((Join-Path $workflowStore 'damaged-workflow.json'), '{"format":"yotta.workflow","version":"3.1",', $utf8NoBom)

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
    $appOut = Join-Path $runRoot 'yotta.out.log'
    $appErr = Join-Path $runRoot 'yotta.err.log'
    $appProcess = Start-Process -FilePath (Join-Path $binDir 'Yotta.SmokeHost.exe') -WorkingDirectory $binDir -WindowStyle Hidden -RedirectStandardOutput $appOut -RedirectStandardError $appErr -PassThru

    for ($attempt = 0; $attempt -lt 100; $attempt++) {
        Start-Sleep -Milliseconds 100
        try {
            Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$DebugPort/json" -TimeoutSec 1 | Out-Null
            break
        } catch {
            if ($appProcess.HasExited) {
                throw "Yotta exited before exposing WebView CDP; see $appErr"
            }
            if ($attempt -eq 99) {
                throw "WebView CDP did not listen on port $DebugPort"
            }
        }
    }

    go run ./cmd/workflow-editor-smoke -endpoint "http://127.0.0.1:$DebugPort" -screenshot $screenshot -assets-screenshot $assetsScreenshot -workflows-screenshot $workflowsScreenshot -launcher-screenshot $launcherScreenshot
    if ($LASTEXITCODE -ne 0) {
        throw "Workflow editor smoke failed with exit code $LASTEXITCODE"
    }
} finally {
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
