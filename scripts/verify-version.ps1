param(
    [string]$ExpectedVersion
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Get-RegexVersion {
    param(
        [string]$Path,
        [string]$Pattern
    )

    $text = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath $Path))
    $match = [regex]::Match($text, $Pattern)
    if (-not $match.Success) {
        throw "Version pattern not found in ${Path}: ${Pattern}"
    }
    return $match.Groups[1].Value
}

$sourceVersion = Get-RegexVersion "pkg/version/version.go" 'const Version = "([^"]+)"'
if ([string]::IsNullOrWhiteSpace($ExpectedVersion)) {
    $ExpectedVersion = $sourceVersion
}
if ($ExpectedVersion -notmatch '^\d+\.\d+\.\d+$') {
    throw "Expected version must be numeric semver, got ${ExpectedVersion}"
}

$infoText = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath "build/windows/info.json"))
$frontendText = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath "frontend/package.json"))
$info = $infoText | ConvertFrom-Json
$frontend = $frontendText | ConvertFrom-Json
$versions = [ordered]@{
    "pkg/version/version.go"                 = $sourceVersion
    "build/config.yml"                       = Get-RegexVersion "build/config.yml" '(?m)^  version: "([^"]+)"'
    "build/windows/info.json:file_version"   = [string]$info.fixed.file_version
    "build/windows/info.json:ProductVersion" = [string]$info.info.'0000'.ProductVersion
    "build/windows/nsis/wails_tools.nsh"     = Get-RegexVersion "build/windows/nsis/wails_tools.nsh" '!define INFO_PRODUCTVERSION "([^"]+)"'
    "frontend/package.json"                  = [string]$frontend.version
    "build/windows/wails.exe.manifest"       = Get-RegexVersion "build/windows/wails.exe.manifest" 'name="com\.yottaapp\.yotta" version="([^"]+)"'
}

$mismatches = @()
foreach ($entry in $versions.GetEnumerator()) {
    if ($entry.Value -ne $ExpectedVersion) {
        $mismatches += "$($entry.Key)=$($entry.Value)"
    }
}
if ($mismatches.Count -gt 0) {
    throw "Version mismatch; expected ${ExpectedVersion}: $($mismatches -join ', ')"
}

Write-Host "version sync OK: $ExpectedVersion"
