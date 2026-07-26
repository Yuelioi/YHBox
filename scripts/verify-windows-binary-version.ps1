param(
    [string]$Path = "bin/Yotta.exe",
    [string]$ExpectedVersion
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
    throw "Windows binary not found: $Path"
}
if ([string]::IsNullOrWhiteSpace($ExpectedVersion)) {
    $ExpectedVersion = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath "VERSION")).Trim()
}
if ($ExpectedVersion -notmatch '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$') {
    throw "expected product version is not numeric SemVer: $ExpectedVersion"
}

$expectedWindowsVersion = "${ExpectedVersion}.0"
$item = Get-Item -LiteralPath $Path
$versionInfo = $item.VersionInfo
if ($versionInfo.FileVersionRaw.ToString() -ne $expectedWindowsVersion) {
    throw "fixed FileVersion mismatch in ${Path}: expected ${expectedWindowsVersion}, got $($versionInfo.FileVersionRaw)"
}
if ($versionInfo.ProductVersionRaw.ToString() -ne $expectedWindowsVersion) {
    throw "fixed ProductVersion mismatch in ${Path}: expected ${expectedWindowsVersion}, got $($versionInfo.ProductVersionRaw)"
}
if ($versionInfo.FileVersion -ne $ExpectedVersion) {
    throw "string FileVersion mismatch in ${Path}: expected ${ExpectedVersion}, got $($versionInfo.FileVersion)"
}
if ($versionInfo.ProductVersion -ne $ExpectedVersion) {
    throw "string ProductVersion mismatch in ${Path}: expected ${ExpectedVersion}, got $($versionInfo.ProductVersion)"
}

$bytes = [System.IO.File]::ReadAllBytes($item.FullName)
if ($bytes.Length -lt 256 -or $bytes[0] -ne 0x4D -or $bytes[1] -ne 0x5A) {
    throw "not a valid PE image: $Path"
}
$peOffset = [System.BitConverter]::ToInt32($bytes, 0x3C)
if ($peOffset -lt 0 -or $peOffset + 94 -gt $bytes.Length -or
    $bytes[$peOffset] -ne 0x50 -or $bytes[$peOffset + 1] -ne 0x45 -or
    $bytes[$peOffset + 2] -ne 0 -or $bytes[$peOffset + 3] -ne 0) {
    throw "invalid PE header: $Path"
}
$optionalHeaderOffset = $peOffset + 24
$subsystem = [System.BitConverter]::ToUInt16($bytes, $optionalHeaderOffset + 68)
if ($subsystem -ne 2) {
    throw "PE subsystem mismatch in ${Path}: expected WINDOWS_GUI (2), got ${subsystem}"
}

Write-Host "Windows binary metadata OK: $Path, product $ExpectedVersion, subsystem WINDOWS_GUI"
