# Scenarios: Go TUI CLI

## Scenario 1: First-Time User

1. User installs via `brew install yetanotherchris/tap/ingester` or downloads binary
2. User sets `OPENROUTER_API_KEY` env var (or creates `.env` file)
3. User runs `ingester` with no arguments
4. TUI launches, shows main menu
5. TUI shows green checkmarks for configured vars, yellow warnings for missing optional vars
6. User selects "Ingest", enters directory path
7. App auto-starts ChromaDB, waits for health check, runs ingestion
8. Progress view shows real-time Docker output
9. Returns to menu with summary (X files, Y chunks ingested)

## Scenario 2: Scripted/CI Usage

1. User runs: `ingester ingest ~/notes ~/docs --extensions .md .txt --collection my-docs`
2. App validates env vars, errors if `OPENROUTER_API_KEY` missing and local embeddings not enabled
3. Starts ChromaDB if not running
4. Runs ingestion in non-interactive mode
5. Prints summary to stdout
6. Exits with code 0 on success, non-zero on failure

## Scenario 3: Update Only

1. User has previously ingested a set of directories
2. User runs `ingester update ~/notes ~/docs`
3. App loads `.ingester-hashes.json` from data directory
4. Scans all files, computes SHA-256 hashes
5. Identifies: 2 new files, 3 changed files, 1 deleted file, 50 unchanged
6. Ingests only the 5 new/changed files
7. Removes chunks for the deleted file from ChromaDB
8. Updates manifest file
9. Prints summary

## Scenario 4: Google Drive Download + Ingest

1. User has rclone configured with a `gdrive` remote
2. User selects "Download (rclone)" from TUI
3. Enters source: `gdrive:Documents/notes` and destination: `~/notes`
4. App runs rclone via Docker, streams download progress
5. User then selects "Ingest" to ingest the downloaded files

## Scenario 5: Docker Not Installed

1. User runs `ingester`
2. App checks for Docker availability
3. Displays clear error: "Docker is required but not found. Please install Docker Desktop or Docker Engine."
4. Exits with non-zero code

## Scenario 6: ChromaDB Management

1. User selects "Start ChromaDB" from TUI
2. App runs `docker compose up -d chromadb`
3. Waits for health check (HTTP 200 on port 8000)
4. Shows "ChromaDB is running" with green indicator
5. Later, user selects "Stop ChromaDB"
6. App runs `docker compose down`
7. Shows "ChromaDB stopped"
