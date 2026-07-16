param(
    [Parameter()]
    [string]$CoverageFile = "coverage.out",

    [Parameter()]
    [string[]]$AdditionalCoverageFiles = @(),

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

$coverageFiles = @($CoverageFile) + @($AdditionalCoverageFiles)
foreach ($file in $coverageFiles) {
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw "coverage profile not found: $file"
    }
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
$blockPattern = '^(.+?):(\d+\.\d+,\d+\.\d+)\s+(\d+)\s+(\d+)$'
$blocks = @{}
$generated = @{}

foreach ($coveragePath in $coverageFiles) {
    Get-Content -LiteralPath $coveragePath | Select-Object -Skip 1 | ForEach-Object {
        $match = [regex]::Match($_, $blockPattern)
        if (-not $match.Success) {
            throw "invalid coverage profile line: $_"
        }

        $path = $match.Groups[1].Value
        if (-not $path.StartsWith($modulePrefix, [System.StringComparison]::Ordinal)) {
            throw "coverage path is outside module $($budget.module): $path"
        }
        $relativePath = $path.Substring($modulePrefix.Length)
        if (-not $generated.ContainsKey($relativePath)) {
            $sourcePath = Join-Path (Split-Path -Parent $PSScriptRoot) $relativePath
            $firstLine = if (Test-Path -LiteralPath $sourcePath -PathType Leaf) { Get-Content -LiteralPath $sourcePath -TotalCount 1 } else { "" }
            $generated[$relativePath] = $firstLine -match '^// Code generated .* DO NOT EDIT\.$'
        }
        if ($generated[$relativePath]) {
            return
        }

        $statements = [int]$match.Groups[3].Value
        $count = [int]$match.Groups[4].Value
        $key = "$path`:$($match.Groups[2].Value)"
        if (-not $blocks.ContainsKey($key)) {
            $blocks[$key] = @{ Path = $relativePath; Statements = $statements; Covered = $false }
        } elseif ($blocks[$key].Statements -ne $statements) {
            throw "coverage profiles disagree about statement count: $key"
        }
        if ($count -gt 0) {
            $blocks[$key].Covered = $true
        }
    }
}

foreach ($block in $blocks.Values) {
    $relativePath = $block.Path
    $package = [System.IO.Path]::GetDirectoryName($relativePath).Replace('\', '/')
    if (-not $package) {
        $package = "."
    }
    if (-not $packages.ContainsKey($package)) {
        $packages[$package] = @{ Covered = 0; Total = 0 }
    }

    $packages[$package].Total += $block.Statements
    $global.Total += $block.Statements
    if ($block.Covered) {
        $packages[$package].Covered += $block.Statements
        $global.Covered += $block.Statements
    }
}

Assert-Coverage -Name "global" -Stats $global -Minimum ([double]$budget.globalMinimum)
foreach ($entry in $budget.packageMinimums.PSObject.Properties | Sort-Object Name) {
    if (-not $packages.ContainsKey($entry.Name)) {
        throw "coverage profile does not contain package: $($entry.Name)"
    }
    Assert-Coverage -Name $entry.Name -Stats $packages[$entry.Name] -Minimum ([double]$entry.Value)
}
