param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

$repo = "yetanotherchris/ingester"
$platforms = @("darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64")
$formulaPath = "$PSScriptRoot/Formula/ingester.rb"

# Read the template
$formula = Get-Content -Path $formulaPath -Raw

# Replace VERSION placeholders
$formula = $formula -replace 'VERSION', $Version

foreach ($platform in $platforms) {
    $url = "https://github.com/$repo/releases/download/v$Version/ingester-$platform.tar.gz"
    $tempFile = Join-Path ([System.IO.Path]::GetTempPath()) "ingester-$platform.tar.gz"

    Write-Host "Downloading $url ..."
    Invoke-WebRequest -Uri $url -OutFile $tempFile

    $hash = (Get-FileHash -Path $tempFile -Algorithm SHA256).Hash.ToLower()
    Write-Host "SHA256 for ${platform}: $hash"

    Remove-Item $tempFile

    # Replace the first remaining SHA256 placeholder that corresponds to this platform
    $formula = $formula -replace "(?<=ingester-$platform\.tar\.gz`"`n\s+sha256 `")SHA256", $hash
}

Set-Content -Path $formulaPath -Value $formula -NoNewline
Write-Host "Updated $formulaPath with version $Version"
