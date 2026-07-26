$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$status = @(& git -C $root status --porcelain --untracked-files=all)
if ($LASTEXITCODE -ne 0) { throw "unable to inspect Git worktree" }
if ($status.Count -gt 0) {
    throw "release candidates require a clean index and worktree:`n$($status -join "`n")"
}
Write-Host "Git index and worktree are clean"
