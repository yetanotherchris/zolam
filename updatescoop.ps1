param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

$repo = "yetanotherchris/zolam"
$manifestPath = "$PSScriptRoot/zolam.json"

$url = "https://github.com/$repo/releases/download/v$Version/zolam-windows-amd64.exe"
$tempFile = Join-Path ([System.IO.Path]::GetTempPath()) "zolam-windows-amd64.exe"

Write-Host "Downloading $url ..."
Invoke-WebRequest -Uri $url -OutFile $tempFile

$hash = (Get-FileHash -Path $tempFile -Algorithm SHA256).Hash.ToLower()
Write-Host "SHA256 for windows-amd64: $hash"

Remove-Item $tempFile

$manifest = Get-Content -Path $manifestPath -Raw | ConvertFrom-Json
$manifest.version = $Version
$manifest.architecture."64bit".url = $url
$manifest.architecture."64bit".hash = $hash

$manifest | ConvertTo-Json -Depth 10 | Set-Content -Path $manifestPath -NoNewline
Write-Host "Updated $manifestPath with version $Version"
