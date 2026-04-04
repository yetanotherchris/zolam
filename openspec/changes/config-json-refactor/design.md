# Design

## config.json Schema

```json
{
  "collectionName": "my-notes",
  "rcloneSource": "",
  "rcloneConfigDir": "~/.config/rclone",
  "dataDir": "~/.zolam/chromadb-data",
  "directories": [
    {
      "path": "/home/user/notes",
      "extensions": [".md", ".txt"]
    },
    {
      "path": "/home/user/docs",
      "extensions": [".pdf", ".docx"]
    }
  ]
}
```

## Config Loading Order

1. Set defaults
2. Read `~/.zolam/config.json` (if exists)
3. Override with env vars (COLLECTION_NAME, RCLONE_SOURCE, RCLONE_CONFIG_DIR, ZOLAM_DATA_DIR)
4. Override with CLI flags

## Components Modified

- `internal/domain/config.go` - Add config.json loading, SupportedFileExtensions constant, directory tracking types, save function
- `internal/domain/config_test.go` - Update tests for new loading logic
- `internal/tui/ingest.go` - Use SupportedFileExtensions instead of local allExtensions var
- `internal/tui/app.go` - Remove Extensions from settings view, save directories after ingest
- `internal/zolam/stats.go` - Print SupportedFileExtensions on stats page
- `cmd/zolam/main.go` - Update update command (optional args), ingest saves to config, stats prints extensions, config command updated
- `README.md` - Simplified structure
