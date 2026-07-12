param(
    [string]$WorkflowDirectory = ".github/workflows"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$workflowRoot = Resolve-Path -LiteralPath $WorkflowDirectory
$workflowFiles = @(
    Get-ChildItem -LiteralPath $workflowRoot -File |
        Where-Object { $_.Extension -in ".yml", ".yaml" } |
        Sort-Object FullName
)
if ($workflowFiles.Count -eq 0) {
    throw "no GitHub Actions workflows found under $WorkflowDirectory"
}

$violations = @()
$references = 0
foreach ($file in $workflowFiles) {
    $lineNumber = 0
    foreach ($line in Get-Content -LiteralPath $file.FullName) {
        $lineNumber++
        $match = [regex]::Match($line, '^\s*uses:\s*["'']?([^\s"''#]+)["'']?(?:\s+#\s*(\S.*))?\s*$')
        if (-not $match.Success) {
            if ($line -match '^\s*uses:') {
                $violations += "$($file.Name):${lineNumber}: cannot parse uses reference"
            }
            continue
        }

        $references++
        $reference = $match.Groups[1].Value
        $comment = $match.Groups[2].Value
        if ($reference.StartsWith("./", [System.StringComparison]::Ordinal)) {
            continue
        }
        if ($reference.StartsWith("docker://", [System.StringComparison]::OrdinalIgnoreCase)) {
            if ($reference -notmatch '^docker://[^@\s]+@sha256:[0-9a-f]{64}$') {
                $violations += "$($file.Name):${lineNumber}: Docker action must use a sha256 digest: $reference"
            }
            continue
        }
        if ($reference -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.\/-]+)?@[0-9a-f]{40}$') {
            $violations += "$($file.Name):${lineNumber}: external action must use a full 40-character commit SHA: $reference"
            continue
        }
        if ([string]::IsNullOrWhiteSpace($comment)) {
            $violations += "$($file.Name):${lineNumber}: pinned action must retain its upstream version in a comment"
        }
    }
}

if ($references -eq 0) {
    throw "no uses references found under $WorkflowDirectory"
}
if ($violations.Count -gt 0) {
    throw "mutable GitHub Action references found:`n$($violations -join "`n")"
}

Write-Host "GitHub Actions pinning OK: $references references across $($workflowFiles.Count) workflows"
