param(
    [string]$ExpectedVersion,
    [switch]$CheckInstalled
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

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
    $ExpectedVersion = $moduleVersion
}

$pins = [ordered]@{
    "go.mod"                        = $moduleVersion
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

Write-Host "Wails toolchain sync OK: $ExpectedVersion"

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
