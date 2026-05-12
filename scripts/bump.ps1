$file = "cmd/yhbox/main.go"
$content = Get-Content $file -Raw -Encoding UTF8
if ($content -match 'version\s*=\s*"(\d+)\.(\d+)\.(\d+)"') {
    $maj = [int]$Matches[1]
    $min = [int]$Matches[2]
    $pat = [int]$Matches[3] + 1
    $new = "$maj.$min.$pat"
    $content = $content -replace 'version\s*=\s*"\d+\.\d+\.\d+"', "version = `"$new`""
    [System.IO.File]::WriteAllText($file, $content, [System.Text.UTF8Encoding]::new($false))
    Write-Host "main.go      -> $new"
} else {
    Write-Error "version not found in $file"
    exit 1
}

$winres = "cmd/yhbox/winres/winres.json"
$wc = Get-Content $winres -Raw -Encoding UTF8
$wc = $wc -replace '("version"\s*:\s*")\d+\.\d+\.\d+\.\d+(")', "`${1}$new.0`${2}"
$wc = $wc -replace '("file_version"\s*:\s*")\d+\.\d+\.\d+\.\d+(")', "`${1}$new.0`${2}"
$wc = $wc -replace '("product_version"\s*:\s*")\d+\.\d+\.\d+\.\d+(")', "`${1}$new.0`${2}"
$wc = $wc -replace '("FileVersion"\s*:\s*")\d+\.\d+\.\d+(")', "`${1}$new`${2}"
$wc = $wc -replace '("ProductVersion"\s*:\s*")\d+\.\d+\.\d+(")', "`${1}$new`${2}"
[System.IO.File]::WriteAllText($winres, $wc, [System.Text.UTF8Encoding]::new($false))
Write-Host "winres.json  -> $new"
