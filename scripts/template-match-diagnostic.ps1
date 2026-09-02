param(
    [string]$Workflow = "异环 看电影",
    [Parameter(Mandatory = $true)][string]$ResourceId,
    [string]$VariantId = "default",
    [string]$TargetSlot = "window-target",
    [string]$CaptureBackend = "",
    [string]$Root = "$env:LOCALAPPDATA\Yotta\Yotta",
    [string]$Output = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path

if ([string]::IsNullOrWhiteSpace($Output)) {
    $Output = Join-Path $Root ("diagnostics\captures\template-match-" + (Get-Date -Format "yyyyMMdd-HHmmss"))
}
$settings = (Get-Content -Raw -LiteralPath (Join-Path $Root "config\settings.json") | ConvertFrom-Json).payload
$target = @($settings.automation.targets | Where-Object slot -EQ $TargetSlot)
if ($target.Count -ne 1) { throw "Configured target slot '$TargetSlot' resolved $($target.Count) entries" }
$application = @($settings.applications.profiles | Where-Object slot -EQ $target[0].profile.applicationSlot)
if ($application.Count -ne 1) { throw "Application slot '$($target[0].profile.applicationSlot)' resolved $($application.Count) entries" }
if ([string]::IsNullOrWhiteSpace($CaptureBackend)) { $CaptureBackend = $target[0].profile.captureBackend }

$previous = @{}
$previousPath = $env:PATH
foreach ($name in @(
    "YOTTA_TEMPLATE_DIAGNOSTIC", "YOTTA_DIAGNOSTIC_ROOT", "YOTTA_DIAGNOSTIC_WORKFLOW",
    "YOTTA_DIAGNOSTIC_RESOURCE", "YOTTA_DIAGNOSTIC_VARIANT", "YOTTA_DIAGNOSTIC_OUTPUT",
    "YOTTA_DIAGNOSTIC_EXECUTABLE", "YOTTA_DIAGNOSTIC_WINDOW_TITLE", "YOTTA_DIAGNOSTIC_TITLE_MATCH",
    "YOTTA_DIAGNOSTIC_WINDOW_CLASS", "YOTTA_DIAGNOSTIC_CAPTURE_BACKEND"
)) { $previous[$name] = [Environment]::GetEnvironmentVariable($name, "Process") }

try {
    $env:YOTTA_TEMPLATE_DIAGNOSTIC = "1"
    $env:YOTTA_DIAGNOSTIC_ROOT = $Root
    $env:YOTTA_DIAGNOSTIC_WORKFLOW = $Workflow
    $env:YOTTA_DIAGNOSTIC_RESOURCE = $ResourceId
    $env:YOTTA_DIAGNOSTIC_VARIANT = $VariantId
    $env:YOTTA_DIAGNOSTIC_OUTPUT = $Output
    $env:YOTTA_DIAGNOSTIC_EXECUTABLE = $application[0].executable
    $env:YOTTA_DIAGNOSTIC_WINDOW_TITLE = $target[0].profile.windowTitle
    $env:YOTTA_DIAGNOSTIC_TITLE_MATCH = $target[0].profile.windowTitleMatch
    $env:YOTTA_DIAGNOSTIC_WINDOW_CLASS = $target[0].profile.windowClass
    $env:YOTTA_DIAGNOSTIC_CAPTURE_BACKEND = $CaptureBackend
    $env:PATH = (Join-Path $repoRoot "bin") + [IO.Path]::PathSeparator + $env:PATH
    & go test ./internal/noderuntime -run '^TestNativeTemplateDiagnostic$' -count=1 -v
    if ($LASTEXITCODE -ne 0) { throw "Template diagnostic failed with exit code $LASTEXITCODE; evidence: $Output" }
} finally {
    $env:PATH = $previousPath
    foreach ($entry in $previous.GetEnumerator()) {
        [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, "Process")
    }
}

Write-Host "Template diagnostic evidence: $Output"
