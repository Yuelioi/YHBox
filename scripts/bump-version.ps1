param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Fail($Message) {
    Write-Error $Message
    exit 1
}

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    Fail "Version must be numeric semver, for example 3.1.1"
}

$repo = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($repo)) {
    Fail "Not inside a git repository"
}
$repo = $repo.Trim()
Set-Location $repo

$tag = "v$Version"
& git rev-parse -q --verify "refs/tags/$tag" *> $null
if ($LASTEXITCODE -eq 0) {
    Fail "Tag $tag already exists"
}

if (-not $DryRun) {
    $dirty = (& git status --short)
    if (-not [string]::IsNullOrWhiteSpace(($dirty -join "`n"))) {
        Fail "Working tree must be clean before bumping version"
    }
}

$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$changes = @(
    @{
        Path = "pkg/version/version.go"
        Pattern = 'const Version = "[^"]+"'
        Replacement = "const Version = `"$Version`""
    },
    @{
        Path = "build/config.yml"
        Pattern = '(?m)^  version: "[^"]+"'
        Replacement = "  version: `"$Version`""
    },
    @{
        Path = "build/windows/info.json"
        Pattern = '"file_version": "[^"]+"'
        Replacement = "`"file_version`": `"$Version`""
    },
    @{
        Path = "build/windows/info.json"
        Pattern = '"ProductVersion": "[^"]+"'
        Replacement = "`"ProductVersion`": `"$Version`""
    },
    @{
        Path = "build/windows/nsis/wails_tools.nsh"
        Pattern = '(!define INFO_PRODUCTVERSION ")[^"]+"'
        Replacement = "`${1}$Version`""
    },
    @{
        Path = "frontend/package.json"
        Pattern = '("version":\s*)"[^"]+"'
        Replacement = "`${1}`"$Version`""
    },
    @{
        Path = "build/windows/wails.exe.manifest"
        Pattern = '(name="com\.yottaapp\.yotta" version=)"[^"]+"'
        Replacement = "`${1}`"$Version`""
    }
)

$changedFiles = New-Object System.Collections.Generic.HashSet[string]
foreach ($change in $changes) {
    $path = [string]$change.Path
    if (-not (Test-Path -LiteralPath $path)) {
        Fail "Missing version file: $path"
    }
    $resolved = (Resolve-Path -LiteralPath $path).Path
    $text = [System.IO.File]::ReadAllText($resolved)
    if ($text -notmatch [string]$change.Pattern) {
        Fail "Pattern not found in ${path}: $($change.Pattern)"
    }
    $next = [regex]::Replace($text, [string]$change.Pattern, [string]$change.Replacement)
    if ($next -ne $text) {
        [void]$changedFiles.Add($path)
        if (-not $DryRun) {
            [System.IO.File]::WriteAllText($resolved, $next, $utf8NoBom)
        }
    }
}

if ($DryRun) {
    Write-Host "dry-run: would bump to $Version"
    Write-Host "dry-run: would update files:"
    foreach ($file in ($changedFiles | Sort-Object)) {
        Write-Host "  $file"
    }
    Write-Host "dry-run: would commit chore(release): bump version to $tag"
    Write-Host "dry-run: would create tag $tag"
    exit 0
}

if ($changedFiles.Count -eq 0) {
    Fail "All version files already contain $Version"
}

& git diff --check
if ($LASTEXITCODE -ne 0) {
    Fail "git diff --check failed"
}

foreach ($file in ($changedFiles | Sort-Object)) {
    & git add -- $file
    if ($LASTEXITCODE -ne 0) {
        Fail "git add failed for $file"
    }
}

& git commit -m "chore(release): bump version to $tag"
if ($LASTEXITCODE -ne 0) {
    Fail "git commit failed"
}

& git tag $tag
if ($LASTEXITCODE -ne 0) {
    Fail "git tag failed"
}

Write-Host "bumped version to $Version and created tag $tag"
