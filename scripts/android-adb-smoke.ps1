param(
    [string]$Serial = '',
    [string]$Package = 'com.android.settings'
)

$ErrorActionPreference = 'Stop'
$previousSmoke = $env:YOTTA_ADB_SMOKE
$previousSerial = $env:YOTTA_ADB_SERIAL
$previousPackage = $env:YOTTA_ADB_SMOKE_PACKAGE

try {
    $env:YOTTA_ADB_SMOKE = '1'
    $env:YOTTA_ADB_SMOKE_PACKAGE = $Package
    if ([string]::IsNullOrWhiteSpace($Serial)) {
        Remove-Item Env:YOTTA_ADB_SERIAL -ErrorAction SilentlyContinue
    }
    else {
        $env:YOTTA_ADB_SERIAL = $Serial
    }

    & go test ./internal/appbootstrap -run '^TestAndroidADBWorkflowSmoke$' -count=1 -v
    if ($LASTEXITCODE -ne 0) {
        throw "Android ADB smoke exited $LASTEXITCODE"
    }
}
finally {
    if ($null -eq $previousSmoke) { Remove-Item Env:YOTTA_ADB_SMOKE -ErrorAction SilentlyContinue } else { $env:YOTTA_ADB_SMOKE = $previousSmoke }
    if ($null -eq $previousSerial) { Remove-Item Env:YOTTA_ADB_SERIAL -ErrorAction SilentlyContinue } else { $env:YOTTA_ADB_SERIAL = $previousSerial }
    if ($null -eq $previousPackage) { Remove-Item Env:YOTTA_ADB_SMOKE_PACKAGE -ErrorAction SilentlyContinue } else { $env:YOTTA_ADB_SMOKE_PACKAGE = $previousPackage }
}
