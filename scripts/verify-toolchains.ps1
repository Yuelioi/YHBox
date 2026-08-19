param(
    [string]$ManifestPath = "toolchains.json",
    [switch]$CheckInstalled,
    [switch]$CheckReleaseTools
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Read-Text {
    param([string]$Path)
    return [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath $Path))
}

function Match-One {
    param([string]$Path, [string]$Pattern, [string]$Label)
    $matches = [regex]::Matches((Read-Text $Path), $Pattern)
    $values = @($matches | ForEach-Object { $_.Groups[1].Value } | Sort-Object -Unique)
    if ($values.Count -ne 1) {
        throw "expected one $Label in ${Path}, found: $($values -join ', ')"
    }
    return $values[0]
}

function Assert-Equal {
    param([string]$Label, [string]$Actual, [string]$Expected)
    if ($Actual -ne $Expected) {
        throw "$Label mismatch: expected $Expected, found $Actual"
    }
}

function Invoke-Version {
    param([string]$Label, [string]$FileName, [string[]]$Arguments, [string]$Pattern, [string]$Expected)
    try {
        $command = Get-Command -Name $FileName -CommandType Application -ErrorAction Stop | Select-Object -First 1
        $previousErrorActionPreference = $ErrorActionPreference
        try {
            $ErrorActionPreference = "Continue"
            $outputLines = @(& $command.Source @Arguments 2>&1)
            $exitCode = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $previousErrorActionPreference
        }
    } catch {
        throw "unable to execute ${Label}: $($_.Exception.Message)"
    }
    $output = ($outputLines | ForEach-Object { [string]$_ }) -join "`n"
    $output = $output.Trim()
    if ($exitCode -ne 0) {
        throw "$Label exited with code ${exitCode}: $output"
    }
    $match = [regex]::Match($output, $Pattern, [System.Text.RegularExpressions.RegexOptions]::Multiline)
    if (-not $match.Success) {
        throw "unable to parse $Label version from: $output"
    }
    Assert-Equal -Label $Label -Actual $match.Groups[1].Value -Expected $Expected
}

$manifest = Read-Text $ManifestPath | ConvertFrom-Json
if ($manifest.schemaVersion -ne 1) {
    throw "unsupported toolchain manifest schema in $ManifestPath"
}

$frontend = Read-Text "frontend/package.json" | ConvertFrom-Json
$goVersion = Match-One "go.mod" '(?m)^go\s+([^\s]+)$' "Go version"
$wailsGoVersion = Match-One "go.mod" 'github\.com/wailsapp/wails/v3\s+(v[^\s]+)' "Wails Go version"
$rustVersion = Match-One "rust-toolchain.toml" '(?m)^channel\s*=\s*"([^"]+)"$' "Rust version"
$nodeVersion = (Read-Text ".node-version").Trim()
$packageManagerVersion = [regex]::Match([string]$frontend.packageManager, '^pnpm@(.+)$').Groups[1].Value

Assert-Equal "go.mod Go" $goVersion ([string]$manifest.go)
Assert-Equal ".node-version" $nodeVersion ([string]$manifest.node)
Assert-Equal "frontend engines.node" ([string]$frontend.engines.node) ([string]$manifest.node)
Assert-Equal "frontend packageManager" $packageManagerVersion ([string]$manifest.pnpm)
Assert-Equal "rust-toolchain.toml" $rustVersion ([string]$manifest.rust)
Assert-Equal "Wails Go library" $wailsGoVersion ([string]$manifest.wails.go)
Assert-Equal "Wails Go/CLI compatibility" ([string]$manifest.wails.cli) ([string]$manifest.wails.go)
Assert-Equal "Wails frontend runtime" ([string]$frontend.dependencies.'@wailsio/runtime') ([string]$manifest.wails.runtime)

$workflowSyftVersions = @(
    [regex]::Matches((Read-Text ".github/workflows/release.yml"), '(?m)^\s*syft-version:\s*([^\s#]+)') |
        ForEach-Object { $_.Groups[1].Value } |
        Sort-Object -Unique
)
if ($workflowSyftVersions.Count -ne 1) {
    throw "expected one exact Syft version in release workflow, found: $($workflowSyftVersions -join ', ')"
}
Assert-Equal "release Syft" $workflowSyftVersions[0] ([string]$manifest.syft)

$lockRuntimeVersions = @(
    [regex]::Matches((Read-Text "frontend/pnpm-lock.yaml"), "'@wailsio/runtime@([^']+)'?") |
        ForEach-Object { $_.Groups[1].Value } |
        Sort-Object -Unique
)
if ($lockRuntimeVersions.Count -ne 1) {
    throw "expected one @wailsio/runtime resolution in pnpm lock, found: $($lockRuntimeVersions -join ', ')"
}
Assert-Equal "pnpm lock Wails runtime" $lockRuntimeVersions[0] ([string]$manifest.wails.runtime)

$workflowText = @(
    Get-ChildItem -LiteralPath ".github/workflows" -File |
        Where-Object { $_.Extension -in ".yml", ".yaml" } |
        ForEach-Object { [System.IO.File]::ReadAllText($_.FullName) }
) -join "`n"
$workflowWailsVersions = @(
    [regex]::Matches($workflowText, 'github\.com/wailsapp/wails/v3/cmd/wails3@(v[^\s]+)') |
        ForEach-Object { $_.Groups[1].Value } |
        Sort-Object -Unique
)
if ($workflowWailsVersions.Count -ne 1) {
    throw "expected one Wails CLI version in workflows, found: $($workflowWailsVersions -join ', ')"
}
Assert-Equal "workflow Wails CLI" $workflowWailsVersions[0] ([string]$manifest.wails.cli)

$workflowNodeVersions = @(
    [regex]::Matches($workflowText, '(?m)^\s*node-version:\s*([^\s#]+)') |
        ForEach-Object { $_.Groups[1].Value } |
        Sort-Object -Unique
)
if ($workflowNodeVersions.Count -gt 0) {
    if ($workflowNodeVersions.Count -ne 1) {
        throw "conflicting Node versions in workflows: $($workflowNodeVersions -join ', ')"
    }
    Assert-Equal "workflow Node" $workflowNodeVersions[0] ([string]$manifest.node)
}

$workflowTaskVersions = @(
    [regex]::Matches($workflowText, '(?m)^\s*version:\s*(3\.[0-9]+\.[0-9]+)\s*$') |
        ForEach-Object { $_.Groups[1].Value } |
        Sort-Object -Unique
)
if ($workflowTaskVersions.Count -ne 1) {
    throw "expected one exact Task version in workflows, found: $($workflowTaskVersions -join ', ')"
}
Assert-Equal "workflow Task" $workflowTaskVersions[0] ([string]$manifest.task)

foreach ($runner in $manifest.githubRunners.PSObject.Properties) {
    if ($workflowText -notmatch [regex]::Escape([string]$runner.Value)) {
        throw "workflow runner pin is unused: $($runner.Name)=$($runner.Value)"
    }
}
if ($workflowText -match '(?m)^\s*(?:runs-on:\s*)?(?:-|)\s*(?:ubuntu|macos|windows)-latest\s*$') {
    throw "mutable GitHub-hosted runner label found"
}

if ($CheckInstalled) {
    Invoke-Version "Go" "go" @("version") 'go version go([^\s]+)' ([string]$manifest.go)
    Invoke-Version "Node" "node" @("--version") '^v([^\s]+)' ([string]$manifest.node)
    Invoke-Version "pnpm" "pnpm" @("--version") '^([^\s]+)' ([string]$manifest.pnpm)
    Invoke-Version "Task" "task" @("--version") '^(?:Task version:\s*v?)?([^\s]+)' ([string]$manifest.task)
    Invoke-Version "Rust" "rustc" @("--version") '^rustc\s+([^\s]+)' ([string]$manifest.rust)
    Invoke-Version "Wails CLI" "wails3" @("version") '(v3\.0\.0-[0-9A-Za-z.-]+)' ([string]$manifest.wails.cli)
    Invoke-Version "protoc" "protoc" @("--version") '^libprotoc\s+([^\s]+)' ([string]$manifest.protoc)
    Invoke-Version "protoc-gen-go" "protoc-gen-go" @("--version") '^protoc-gen-go(?:\.exe)?\s+v?([^\s]+)' ([string]$manifest.protocGenGo)
}

if ($CheckReleaseTools) {
    Invoke-Version "NSIS" "makensis" @("/VERSION") '^v?([^\s]+)' ([string]$manifest.nsis)
}

Write-Host "toolchain contract OK: Go $($manifest.go), Node $($manifest.node), pnpm $($manifest.pnpm), Task $($manifest.task), Rust $($manifest.rust), Wails $($manifest.wails.go)/runtime $($manifest.wails.runtime), protoc $($manifest.protoc), protoc-gen-go $($manifest.protocGenGo)"
