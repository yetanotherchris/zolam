param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

$repo = "yetanotherchris/ingester"
$manifestPath = "$PSScriptRoot/ingester.json"

$url = "https://github.com/$repo/releases/download/v$Version/ingester-windows-amd64.exe"
$tempFile = Join-Path ([System.IO.Path]::GetTempPath()) "ingester-windows-amd64.exe"

Write-Host "Downloading $url ..."
Invoke-WebRequest -Uri $url -OutFile $tempFile

$hash = (Get-FileHash -Path $tempFile -Algorithm SHA256).Hash.ToLower()
Write-Host "SHA256 for windows-amd64: $hash"

Remove-Item $tempFile

# Read and update the manifest
$manifest = Get-Content -Path $manifestPath -Raw
$manifest = $manifest -replace 'VERSION', $Version
$manifest = $manifest -replace 'SHA256', $hash

Set-Content -Path $manifestPath -Value $manifest -NoNewline
Write-Host "Updated $manifestPath with version $Version"
