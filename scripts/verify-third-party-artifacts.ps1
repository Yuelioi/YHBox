param(
    [string]$ManifestPath = "third_party/artifacts.json"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Get-Sha256 {
    param([string]$LiteralPath)
    $stream = [System.IO.File]::OpenRead((Resolve-Path -LiteralPath $LiteralPath))
    $hasher = [System.Security.Cryptography.SHA256]::Create()
    try { return ([BitConverter]::ToString($hasher.ComputeHash($stream)) -replace '-', '').ToLowerInvariant() }
    finally { $hasher.Dispose(); $stream.Dispose() }
}

$manifest = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath $ManifestPath)) | ConvertFrom-Json
if ($manifest.schemaVersion -ne 1 -or -not $manifest.artifacts) {
    throw "unsupported third-party artifact manifest: $ManifestPath"
}

$seen = @{}
foreach ($artifact in $manifest.artifacts) {
    $path = [string]$artifact.path
    if ([string]::IsNullOrWhiteSpace($path) -or $seen.ContainsKey($path)) {
        throw "invalid or duplicate third-party artifact path: $path"
    }
    $seen[$path] = $true
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "third-party artifact is missing: $path"
    }
    if ([string]::IsNullOrWhiteSpace([string]$artifact.version) -or
        [string]::IsNullOrWhiteSpace([string]$artifact.source) -or
        [string]::IsNullOrWhiteSpace([string]$artifact.license)) {
        throw "third-party artifact metadata is incomplete: $path"
    }
    $actual = Get-Sha256 -LiteralPath $path
    $expected = ([string]$artifact.sha256).ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "third-party artifact hash mismatch: $path expected=$expected actual=$actual"
    }
}

Write-Host "third-party artifact contract OK: $($manifest.artifacts.Count) files"
