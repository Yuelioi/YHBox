param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$arguments = @("run", "./cmd/yotta-versions", "bump")
if ($DryRun) {
    $arguments += "--dry-run"
}
$arguments += $Version

& go @arguments
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
