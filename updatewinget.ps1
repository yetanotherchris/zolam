param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

$repo = "yetanotherchris/ingester"
$wingetDir = "$PSScriptRoot/winget"
$manifestFiles = @(
    "yetanotherchris.zolam.yaml",
    "yetanotherchris.zolam.installer.yaml",
    "yetanotherchris.zolam.locale.en-US.yaml"
)

$url = "https://github.com/$repo/releases/download/v$Version/ingester-windows-amd64.exe"
$tempFile = Join-Path ([System.IO.Path]::GetTempPath()) "ingester-windows-amd64.exe"

Write-Host "Downloading $url ..."
Invoke-WebRequest -Uri $url -OutFile $tempFile

$hash = (Get-FileHash -Path $tempFile -Algorithm SHA256).Hash.ToUpper()
Write-Host "SHA256 for windows-amd64: $hash"

Remove-Item $tempFile

$releaseDate = (Get-Date).ToString("yyyy-MM-dd")

foreach ($file in $manifestFiles) {
    $path = Join-Path $wingetDir $file
    $content = Get-Content -Path $path -Raw
    $content = $content -replace 'VERSION', $Version
    $content = $content -replace 'SHA256', $hash
    $content = $content -replace 'RELEASEDATE', $releaseDate

    Set-Content -Path $path -Value $content -NoNewline
    Write-Host "Updated $path with version $Version"
}

Write-Host ""
Write-Host "Winget manifests updated. To submit to winget-pkgs:"
Write-Host "  1. Fork https://github.com/microsoft/winget-pkgs"
Write-Host "  2. Copy winget/*.yaml to manifests/y/yetanotherchris/zolam/$Version/"
Write-Host "  3. Validate with: winget validate manifests/y/yetanotherchris/zolam/$Version/"
Write-Host "  4. Submit a pull request"
