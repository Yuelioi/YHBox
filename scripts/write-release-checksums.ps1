param(
    [Parameter(Mandatory = $true)]
    [string[]]$Path,
    [string]$OutputPath = "artifacts/SHA256SUMS"
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

$files = @($Path | ForEach-Object { Get-Item -LiteralPath $_ } | Sort-Object Name)
if ($files.Count -eq 0) { throw "no release artifacts supplied" }
$lines = @($files | ForEach-Object {
    $hash = Get-Sha256 -LiteralPath $_.FullName
    "$hash *$($_.Name)"
})
$parent = Split-Path -Parent $OutputPath
if ($parent) { [System.IO.Directory]::CreateDirectory([System.IO.Path]::GetFullPath($parent)) | Out-Null }
[System.IO.File]::WriteAllLines([System.IO.Path]::GetFullPath($OutputPath), $lines, [System.Text.UTF8Encoding]::new($false))
Write-Host "wrote $OutputPath"
