param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

$exePath = Join-Path $PSScriptRoot "artifacts" "zolam-windows-amd64.exe"

if (-not (Test-Path $exePath)) {
    throw "Unable to locate zolam-windows-amd64.exe at $exePath"
}

$hash = (Get-FileHash -Path $exePath -Algorithm SHA256).Hash.ToLower()
Write-Host "Hash: $hash"

$url = "https://github.com/yetanotherchris/zolam/releases/download/v$Version/zolam-windows-amd64.exe"
$manifestPath = Join-Path $PSScriptRoot "zolam.json"

$manifest = Get-Content -Path $manifestPath -Raw | ConvertFrom-Json
$manifest.version = $Version
$manifest.architecture."64bit".url = $url
$manifest.architecture."64bit".hash = $hash

$manifest | ConvertTo-Json -Depth 10 | Set-Content -Path $manifestPath -NoNewline
Write-Host "Updated zolam.json to v$Version"
