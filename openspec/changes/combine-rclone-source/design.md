# Design: Combine RCLONE_REMOTE and RCLONE_SOURCE

## Changes

### Config (`internal/domain/config.go`)
- Remove `RcloneRemote` field from `Config`
- `RcloneSource` loaded from `RCLONE_SOURCE` env var, no default
- Remove `rclone-remote` from `MergeFlags`
- Remove `RCLONE_REMOTE` validation warning

### Docker (`internal/docker/rclone.go`)
- `RcloneSync` takes `source` directly instead of `remote` + `source`
- Source string passed straight to rclone (e.g. `gdrive:/docs`)

### CLI (`cmd/zolam/main.go`)
- Remove `--remote` flag from download command
- `--source` flag overrides `RCLONE_SOURCE`
- Error message updated

### TUI (`internal/tui/app.go`)
- Remove "Rclone Remote" from settings view
- Pass `RcloneSource` directly to `RcloneSync`
