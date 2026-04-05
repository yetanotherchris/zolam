## Why

Ingest directories in `config.json` only accept literal paths. Users with files spread across many similarly-named subdirectories (e.g. `c:/notes/2024/Draft/`, `c:/notes/2025/Draft/`) must add each one individually. Glob patterns let a single entry like `c:/notes/**/Draft/` resolve to all matching directories at ingest time, reducing config churn and making the tool more practical for large, structured file trees.

## What Changes

- `DirectoryEntry.Path` in config accepts glob patterns (e.g. `c:/myfolder/**/D*/`) in addition to literal paths
- A new glob-resolution step runs before ingest, expanding patterns into concrete directories
- File discovery moves to Go: globs resolve to directories, directories are walked with extension filtering, and individual file paths are passed to the Python container
- `ingest.py` gains `--manifest` (JSON file of paths) and `--file-path` (CLI args) as new input modes; existing `--directory` is preserved for backward compatibility
- A single Docker volume mount of the common parent directory replaces per-directory mounts, eliminating mount collision issues
- Invalid or zero-match patterns produce clear warnings rather than silent failures
- The TUI directory-add flow accepts glob patterns

## Capabilities

### New Capabilities
- `glob-directories`: Resolve glob patterns in directory entries to concrete paths at ingest time

### Modified Capabilities

## Impact

- `internal/zolam/ingester.go` - `Run` and `RunUpdateOnly` gain glob resolution and file discovery steps
- `ingest.py` - new `--manifest` and `--file-path` arguments
- `internal/tui/ingest.go` - TUI directory-add accepts glob patterns
- New Go dependency: `github.com/bmatcuk/doublestar/v4` for `**` glob support
- No breaking changes to config.json format - existing literal paths continue to work unchanged
- `ingest.py` `--directory` flag preserved for backward compatibility
