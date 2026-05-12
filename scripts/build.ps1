[CmdletBinding()]
param(
    [string]$Version,
    [switch]$NoUpx,
    [switch]$NoWinres,
    [string]$Out = "YHBox.exe"
)

$ErrorActionPreference = "Stop"
$repo = Split-Path -Parent $PSScriptRoot
Set-Location $repo

if (-not $Version) {
    $main = Get-Content "cmd/yhbox/main.go" -Raw -Encoding UTF8
    if ($main -match 'version\s*=\s*"([^"]+)"') { $Version = $Matches[1] } else { $Version = "dev" }
}
Write-Host "Version: $Version"

if (-not $NoWinres) {
    if (-not (Get-Command go-winres -ErrorAction SilentlyContinue)) {
        Write-Host "Installing go-winres..."
        & go install github.com/tc-hib/go-winres@latest
        if ($LASTEXITCODE -ne 0) { throw "go install go-winres failed" }
    }
    Push-Location "cmd/yhbox"
    try {
        & go-winres make
        if ($LASTEXITCODE -ne 0) { throw "go-winres make failed" }
    } finally { Pop-Location }
}

Write-Host "Building $Out ..."
# -H=windowsgui: GUI 子系统，双击启动不带 cmd 黑框（walk GUI 必需）
$ldflags = "-s -w -H=windowsgui -X main.version=$Version"
& go build -ldflags="$ldflags" -o $Out ./cmd/yhbox
if ($LASTEXITCODE -ne 0) { throw "go build failed" }

$sizeBefore = (Get-Item $Out).Length
Write-Host ("Built  : {0}  ({1:N2} MB)" -f $Out, ($sizeBefore / 1MB))

if (-not $NoUpx) {
    if (-not (Get-Command upx -ErrorAction SilentlyContinue)) {
        Write-Host "upx not found. Install with: choco install upx -y   (or scoop install upx)" -ForegroundColor Yellow
        Write-Host "Skipping compression. Pass -NoUpx to silence this." -ForegroundColor Yellow
        return
    }
    Write-Host "Compressing with UPX..."
    & upx --best $Out
    if ($LASTEXITCODE -ne 0) {
        Write-Host "upx returned $LASTEXITCODE (continuing)" -ForegroundColor Yellow
    }
    $sizeAfter = (Get-Item $Out).Length
    $ratio = if ($sizeBefore -gt 0) { ($sizeAfter / $sizeBefore) * 100 } else { 0 }
    Write-Host ("Packed : {0}  ({1:N2} MB, {2:N1}% of original)" -f $Out, ($sizeAfter / 1MB), $ratio)
}
