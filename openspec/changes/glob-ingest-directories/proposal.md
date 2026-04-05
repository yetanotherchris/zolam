## Why

Ingest directories in `config.json` only accept literal paths. Users with files spread across many similarly-named subdirectories (e.g. `c:/notes/2024/Draft/`, `c:/notes/2025/Draft/`) must add each one individually. Glob patterns let a single entry like `c:/notes/**/Draft/` resolve to all matching directories at ingest time, reducing config churn and making the tool more practical for large, structured file trees.

## What Changes

- `DirectoryEntry.Path` in config accepts glob patterns (e.g. `c:/myfolder/**/D*/`) in addition to literal paths
- A new glob-resolution step runs before ingest, expanding patterns into concrete directories
- The rest of the pipeline (volume mounts, hashing, Docker ingest) continues to receive resolved literal paths
- Invalid or zero-match patterns produce clear warnings rather than silent failures
- The TUI directory-add flow accepts glob patterns

## Capabilities

### New Capabilities
- `glob-directories`: Resolve glob patterns in directory entries to concrete paths at ingest time

### Modified Capabilities

## Impact

- `internal/domain/config.go` - `DirectoryEntry` may need a flag or detection for glob vs literal paths
- `internal/zolam/ingester.go` - `Run` and `RunUpdateOnly` need a resolution step before iterating directories
- `internal/tui/ingest.go` - TUI ingest flow needs to handle patterns that resolve to multiple directories
- Go stdlib `filepath.Glob` handles simple patterns but not `**` (recursive); will need `doublestar` library or custom implementation
- No breaking changes to config.json format - existing literal paths continue to work unchanged
