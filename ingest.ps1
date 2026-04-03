<#
.SYNOPSIS
  Ingest local directories into ChromaDB for semantic search.

.EXAMPLE
  ./ingest.ps1 ~/notes
  ./ingest.ps1 ~/notes ~/docs
  ./ingest.ps1 -Extensions .md,.txt ~/docs
  ./ingest.ps1 -Reset ~/notes
  ./ingest.ps1 -Stats
#>
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Directories,

    [string]$Extensions,

    [switch]$Reset,

    [switch]$Stats,

    [string]$Collection,

    [switch]$Help
)

if ($Help) {
    Get-Help $MyInvocation.MyCommand.Path -Detailed
    return
}

# Ensure ChromaDB is running
docker compose up -d chromadb

if ($Stats) {
    docker compose --profile ingest run --rm ingest --stats
    return
}

if (-not $Directories -or $Directories.Count -eq 0) {
    Write-Error "At least one directory is required (or use -Stats)."
    Write-Host "Run './ingest.ps1 -Help' for usage."
    return
}

# Build volume mounts and container paths
$volumeArgs = @()
$containerDirs = @()
foreach ($dir in $Directories) {
    $absDir = (Resolve-Path $dir).Path
    $name = Split-Path $absDir -Leaf
    $volumeArgs += "-v"
    $volumeArgs += "${absDir}:/sources/${name}"
    $containerDirs += "/sources/${name}"
}

# Build ingest arguments
$ingestArgs = @()
if ($Reset) {
    $ingestArgs += "--reset"
}
$ingestArgs += "--directory"
$ingestArgs += $containerDirs
if ($Extensions) {
    $extList = $Extensions -split ','
    $ingestArgs += "--extensions"
    $ingestArgs += $extList
}

# Build extra docker args
$dockerArgs = @()
if ($Collection) {
    $dockerArgs += "-e"
    $dockerArgs += "COLLECTION_NAME=$Collection"
}

docker compose --profile ingest run --rm `
    @dockerArgs `
    @volumeArgs `
    ingest @ingestArgs
