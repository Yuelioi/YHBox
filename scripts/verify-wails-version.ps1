param(
    [string]$ExpectedVersion,
    [switch]$CheckInstalled
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$toolchains = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath "toolchains.json")) | ConvertFrom-Json
$expectedRuntimeVersion = [string]$toolchains.wails.runtime

function Get-WailsVersion {
    param(
        [string]$Path,
        [string]$Pattern
    )

    $text = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath $Path))
    $matches = [regex]::Matches($text, $Pattern)
    if ($matches.Count -eq 0) {
        throw "Wails version pin not found in ${Path}"
    }
    $versions = @($matches | ForEach-Object { $_.Groups[1].Value } | Sort-Object -Unique)
    if ($versions.Count -ne 1) {
        throw "Conflicting Wails version pins in ${Path}: $($versions -join ', ')"
    }
    return $versions[0]
}

$moduleVersion = Get-WailsVersion "go.mod" 'github\.com/wailsapp/wails/v3\s+(v[^\s]+)'
if ([string]::IsNullOrWhiteSpace($ExpectedVersion)) {
    $ExpectedVersion = [string]$toolchains.wails.go
}

$pins = [ordered]@{
    "go.mod"                        = $moduleVersion
    ".github/workflows/ci.yml"      = Get-WailsVersion ".github/workflows/ci.yml" 'github\.com/wailsapp/wails/v3/cmd/wails3@(v[^\s]+)'
    ".github/workflows/release.yml" = Get-WailsVersion ".github/workflows/release.yml" 'github\.com/wailsapp/wails/v3/cmd/wails3@(v[^\s]+)'
    "README.md"                     = Get-WailsVersion "README.md" 'github\.com/wailsapp/wails/v3/cmd/wails3@(v[^\s]+)'
}

$mismatches = @()
foreach ($entry in $pins.GetEnumerator()) {
    if ($entry.Value -ne $ExpectedVersion) {
        $mismatches += "$($entry.Key)=$($entry.Value)"
    }
}
if ($mismatches.Count -gt 0) {
    throw "Wails version mismatch; expected ${ExpectedVersion}: $($mismatches -join ', ')"
}

$frontend = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath "frontend/package.json")) | ConvertFrom-Json
if ([string]$frontend.dependencies.'@wailsio/runtime' -ne $expectedRuntimeVersion) {
    throw "Wails runtime mismatch; expected ${expectedRuntimeVersion}: frontend/package.json=$($frontend.dependencies.'@wailsio/runtime')"
}
$lockText = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath "frontend/pnpm-lock.yaml"))
$lockVersions = @(
    [regex]::Matches($lockText, "'@wailsio/runtime@([^']+)'?") |
        ForEach-Object { $_.Groups[1].Value } |
        Sort-Object -Unique
)
if ($lockVersions.Count -ne 1 -or $lockVersions[0] -ne $expectedRuntimeVersion) {
    throw "Wails runtime lock mismatch; expected ${expectedRuntimeVersion}: $($lockVersions -join ', ')"
}

Write-Host "Wails toolchain sync OK: Go/CLI $ExpectedVersion, runtime $expectedRuntimeVersion"

if ($CheckInstalled) {
    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = "wails3"
    $startInfo.Arguments = "version"
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    try {
        $process = [System.Diagnostics.Process]::Start($startInfo)
        $stdout = $process.StandardOutput.ReadToEnd()
        $stderr = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
    } catch {
        throw "Unable to execute installed wails3 CLI: $($_.Exception.Message)"
    }
    $installed = "${stdout}`n${stderr}".Trim()
    if ($process.ExitCode -ne 0) {
        throw "Installed wails3 exited with code $($process.ExitCode): ${installed}"
    }
    $versionPattern = '(?<![0-9A-Za-z.-])' + [regex]::Escape($ExpectedVersion) + '(?![0-9A-Za-z.-])'
    $match = [regex]::Match($installed, $versionPattern)
    if (-not $match.Success) {
        throw "Installed wails3 output does not contain expected version ${ExpectedVersion}: ${installed}"
    }
    Write-Host "installed Wails CLI OK: $($match.Value)"
}
