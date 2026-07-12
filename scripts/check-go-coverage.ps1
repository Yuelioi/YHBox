param(
    [Parameter()]
    [string]$CoverageFile = "coverage.out",

    [Parameter()]
    [string]$BudgetFile = "scripts/go-coverage-budgets.json"
)

$ErrorActionPreference = "Stop"

function Assert-Coverage {
    param(
        [string]$Name,
        [hashtable]$Stats,
        [double]$Minimum
    )

    if ($Stats.Total -le 0) {
        throw "coverage scope has no statements: $Name"
    }
    $percentage = 100.0 * $Stats.Covered / $Stats.Total
    if ($percentage -lt $Minimum) {
        throw "$Name statement coverage $($percentage.ToString('F1'))% is below the required $Minimum%"
    }
    Write-Host "$Name coverage OK: $($percentage.ToString('F1'))% (minimum $Minimum%)"
}

if (-not (Test-Path -LiteralPath $CoverageFile -PathType Leaf)) {
    throw "coverage profile not found: $CoverageFile"
}
if (-not (Test-Path -LiteralPath $BudgetFile -PathType Leaf)) {
    throw "coverage budget not found: $BudgetFile"
}

$budget = Get-Content -Raw -LiteralPath $BudgetFile | ConvertFrom-Json
if ($budget.schemaVersion -ne 1 -or -not $budget.module -or -not $budget.packageMinimums) {
    throw "unsupported coverage budget schema: $BudgetFile"
}

$modulePrefix = "$($budget.module)/"
$packages = @{}
$global = @{ Covered = 0; Total = 0 }
$linePattern = '^(.+?):\d+\.\d+,\d+\.\d+\s+(\d+)\s+(\d+)$'

Get-Content -LiteralPath $CoverageFile | Select-Object -Skip 1 | ForEach-Object {
    $match = [regex]::Match($_, $linePattern)
    if (-not $match.Success) {
        throw "invalid coverage profile line: $_"
    }

    $path = $match.Groups[1].Value
    if (-not $path.StartsWith($modulePrefix, [System.StringComparison]::Ordinal)) {
        throw "coverage path is outside module $($budget.module): $path"
    }

    $statements = [int]$match.Groups[2].Value
    $count = [int]$match.Groups[3].Value
    $relativePath = $path.Substring($modulePrefix.Length)
    $package = [System.IO.Path]::GetDirectoryName($relativePath).Replace('\', '/')
    if (-not $package) {
        $package = "."
    }
    if (-not $packages.ContainsKey($package)) {
        $packages[$package] = @{ Covered = 0; Total = 0 }
    }

    $packages[$package].Total += $statements
    $global.Total += $statements
    if ($count -gt 0) {
        $packages[$package].Covered += $statements
        $global.Covered += $statements
    }
}

Assert-Coverage -Name "global" -Stats $global -Minimum ([double]$budget.globalMinimum)
foreach ($entry in $budget.packageMinimums.PSObject.Properties | Sort-Object Name) {
    if (-not $packages.ContainsKey($entry.Name)) {
        throw "coverage profile does not contain package: $($entry.Name)"
    }
    Assert-Coverage -Name $entry.Name -Stats $packages[$entry.Name] -Minimum ([double]$entry.Value)
}
