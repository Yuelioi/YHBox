param(
    [string]$BrowserPath = 'C:\Program Files\Google\Chrome\Application\chrome.exe',
    [int]$Port = 9337
)

$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $BrowserPath -PathType Leaf)) {
    throw "Browser executable not found: $BrowserPath"
}

$workspace = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$taskRoot = Join-Path $workspace '.task'
New-Item -ItemType Directory -Force -Path $taskRoot | Out-Null
$profile = Join-Path $taskRoot ('browser-cdp-smoke-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $profile | Out-Null
$previousSmoke = $env:YOTTA_BROWSER_CDP_SMOKE
$previousEndpoint = $env:YOTTA_BROWSER_CDP_ENDPOINT

try {
    Start-Process -FilePath $BrowserPath -ArgumentList @(
        "--remote-debugging-port=$Port",
        '--remote-debugging-address=127.0.0.1',
        "--user-data-dir=$profile",
        '--no-first-run',
        '--no-default-browser-check',
        'about:blank'
    ) -WindowStyle Hidden | Out-Null

    $ready = $false
    for ($attempt = 0; $attempt -lt 30; $attempt++) {
        try {
            Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$Port/json" -TimeoutSec 1 | Out-Null
            $ready = $true
            break
        }
        catch {
            Start-Sleep -Milliseconds 250
        }
    }
    if (-not $ready) {
        throw 'Browser CDP endpoint did not become ready'
    }

    $env:YOTTA_BROWSER_CDP_SMOKE = '1'
    $env:YOTTA_BROWSER_CDP_ENDPOINT = "http://127.0.0.1:$Port"
    & go test ./internal/appbootstrap -run '^TestBrowserCDPWorkflowSmoke$' -count=1 -v
    if ($LASTEXITCODE -ne 0) {
        throw "Browser CDP smoke exited $LASTEXITCODE"
    }
}
finally {
    if ($null -eq $previousSmoke) { Remove-Item Env:YOTTA_BROWSER_CDP_SMOKE -ErrorAction SilentlyContinue } else { $env:YOTTA_BROWSER_CDP_SMOKE = $previousSmoke }
    if ($null -eq $previousEndpoint) { Remove-Item Env:YOTTA_BROWSER_CDP_ENDPOINT -ErrorAction SilentlyContinue } else { $env:YOTTA_BROWSER_CDP_ENDPOINT = $previousEndpoint }
    Get-CimInstance Win32_Process |
        Where-Object { $_.CommandLine -and $_.CommandLine.Contains($profile) } |
        ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

    $resolvedRoot = [System.IO.Path]::GetFullPath($taskRoot)
    $resolvedProfile = [System.IO.Path]::GetFullPath($profile)
    if (-not $resolvedProfile.StartsWith(
        $resolvedRoot + [System.IO.Path]::DirectorySeparatorChar,
        [System.StringComparison]::OrdinalIgnoreCase
    )) {
        throw 'Refusing to clean a browser profile outside .task'
    }
    if (Test-Path -LiteralPath $resolvedProfile) {
        Remove-Item -LiteralPath $resolvedProfile -Recurse -Force
    }
}
