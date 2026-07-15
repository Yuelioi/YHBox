param(
    [string]$BinDirectory = "bin",
    [string]$OutputDirectory = "artifacts",
    [string]$Target = "windows-amd64"
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

$root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$outputRoot = [System.IO.Path]::GetFullPath((Join-Path $root $OutputDirectory))
$artifactsRoot = [System.IO.Path]::GetFullPath((Join-Path $root "artifacts"))
$artifactsPrefix = $artifactsRoot.TrimEnd([System.IO.Path]::DirectorySeparatorChar) + [System.IO.Path]::DirectorySeparatorChar
if ($outputRoot -ne $artifactsRoot -and -not $outputRoot.StartsWith($artifactsPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "release output must stay under $artifactsRoot"
}

$versionSource = [System.IO.File]::ReadAllText((Join-Path $root "pkg/version/version.go"))
$versionMatch = [regex]::Match($versionSource, 'const Version = "([^"]+)"')
if (-not $versionMatch.Success) { throw "unable to read application version" }
$version = $versionMatch.Groups[1].Value

$stageRoot = Join-Path $outputRoot "staging/Yotta"
if (Test-Path -LiteralPath $stageRoot) {
    Remove-Item -LiteralPath $stageRoot -Recurse -Force
}
[System.IO.Directory]::CreateDirectory($stageRoot) | Out-Null

$payload = @(
    @{ Source = "$BinDirectory/Yotta.exe"; Destination = "Yotta.exe"; Origin = "project-build"; Signing = "unsigned-candidate" },
    @{ Source = "$BinDirectory/Yotta.ScriptWorker.exe"; Destination = "Yotta.ScriptWorker.exe"; Origin = "project-build"; Signing = "unsigned-candidate" },
    @{ Source = "$BinDirectory/capture_wgc.dll"; Destination = "capture_wgc.dll"; Origin = "project-build"; Signing = "unsigned-candidate" },
    @{ Source = "$BinDirectory/platform-tools/adb.exe"; Destination = "platform-tools/adb.exe"; Origin = "bundled-google-platform-tools"; Signing = "upstream-signed" },
    @{ Source = "$BinDirectory/platform-tools/AdbWinApi.dll"; Destination = "platform-tools/AdbWinApi.dll"; Origin = "bundled-google-platform-tools"; Signing = "upstream-signed" },
    @{ Source = "$BinDirectory/platform-tools/AdbWinUsbApi.dll"; Destination = "platform-tools/AdbWinUsbApi.dll"; Origin = "bundled-google-platform-tools"; Signing = "upstream-signed" },
    @{ Source = "$BinDirectory/platform-tools/ADBUTILS_LICENSE"; Destination = "platform-tools/ADBUTILS_LICENSE"; Origin = "bundled-google-platform-tools"; Signing = "not-applicable" },
    @{ Source = "LICENSE"; Destination = "LICENSE"; Origin = "repository"; Signing = "not-applicable" },
    @{ Source = "THIRD_PARTY_NOTICES.txt"; Destination = "THIRD_PARTY_NOTICES.txt"; Origin = "repository"; Signing = "not-applicable" }
)

$fileRecords = @()
foreach ($item in $payload | Sort-Object { [string]$_['Destination'] }) {
    $source = Join-Path $root $item.Source
    if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
        throw "required release payload is missing: $($item.Source)"
    }
    $destination = Join-Path $stageRoot $item.Destination
    [System.IO.Directory]::CreateDirectory([System.IO.Path]::GetDirectoryName($destination)) | Out-Null
    Copy-Item -LiteralPath $source -Destination $destination
    $file = Get-Item -LiteralPath $destination
    $fileRecords += [ordered]@{
        path = $item.Destination.Replace("\", "/")
        size = $file.Length
        sha256 = Get-Sha256 -LiteralPath $destination
        origin = $item.Origin
        signing = $item.Signing
    }
}

$commit = (& git -C $root rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0) { throw "unable to resolve source commit" }
$manifest = [ordered]@{
    schemaVersion = 1
    product = "Yotta"
    version = $version
    target = $Target
    sourceCommit = $commit
    toolchains = Get-Content -LiteralPath (Join-Path $root "toolchains.json") -Raw | ConvertFrom-Json
    files = $fileRecords
}
$manifestPath = Join-Path $stageRoot "artifact-manifest.json"
$manifestJson = $manifest | ConvertTo-Json -Depth 10
[System.IO.File]::WriteAllText($manifestPath, $manifestJson, [System.Text.UTF8Encoding]::new($false))

$zipPath = Join-Path $outputRoot "Yotta-$version-$Target.zip"
if (Test-Path -LiteralPath $zipPath) { Remove-Item -LiteralPath $zipPath -Force }
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem
$stream = [System.IO.File]::Open($zipPath, [System.IO.FileMode]::CreateNew)
try {
    $archive = [System.IO.Compression.ZipArchive]::new($stream, [System.IO.Compression.ZipArchiveMode]::Create, $false)
    try {
        $epoch = [System.DateTimeOffset]::new(1980, 1, 1, 0, 0, 0, [System.TimeSpan]::Zero)
        Get-ChildItem -LiteralPath $stageRoot -File -Recurse |
            Sort-Object { $_.FullName.Substring($stageRoot.Length).TrimStart([char[]]"\/") } |
            ForEach-Object {
                $relative = $_.FullName.Substring($stageRoot.Length).TrimStart([char[]]"\/").Replace("\", "/")
                $entry = $archive.CreateEntry("Yotta/$relative", [System.IO.Compression.CompressionLevel]::Optimal)
                $entry.LastWriteTime = $epoch
                $input = [System.IO.File]::OpenRead($_.FullName)
                $output = $entry.Open()
                try { $input.CopyTo($output) } finally { $output.Dispose(); $input.Dispose() }
            }
    } finally { $archive.Dispose() }
} finally { $stream.Dispose() }

Write-Host "staged Yotta $version at $stageRoot"
Write-Host "created $zipPath"
