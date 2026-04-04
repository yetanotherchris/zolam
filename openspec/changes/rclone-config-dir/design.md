# Design: Configurable Rclone Config Directory

## Architecture

Follows the existing config pattern used by `RCLONE_REMOTE`, `ZOLAM_DATA_DIR`, etc.

### Config Flow

1. `.env` file loaded (if present)
2. `RCLONE_CONFIG_DIR` env var read with default `~/.config/rclone`
3. CLI flags can override via `MergeFlags`
4. Value passed to `RcloneSync` as a parameter

### Components Modified

- `internal/domain/config.go` - Add field, loading, validation warning
- `internal/domain/config_test.go` - Test default and override
- `internal/docker/rclone.go` - Accept config dir as parameter instead of hardcoding
- `internal/tui/app.go` - Pass config dir to `RcloneSync`
