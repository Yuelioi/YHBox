param(
    [string]$ExpectedVersion
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

if (-not [string]::IsNullOrWhiteSpace($ExpectedVersion)) {
    $actual = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath "VERSION")).Trim()
    if ($actual -ne $ExpectedVersion) {
        throw "VERSION mismatch; expected ${ExpectedVersion}, got ${actual}"
    }
}

& go run ./cmd/yotta-versions check
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
