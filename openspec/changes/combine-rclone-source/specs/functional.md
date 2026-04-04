# Functional Spec: Combined RCLONE_SOURCE

## Requirements

1. `RCLONE_SOURCE` accepts any rclone source string (e.g. `gdrive:`, `gdrive:/path/to/folder`)
2. `RCLONE_REMOTE` is removed entirely
3. The `--source` CLI flag overrides `RCLONE_SOURCE`
4. If `RCLONE_SOURCE` is empty and no `--source` flag given, error with clear message
5. The source string is passed directly to rclone without modification
