# Functional Spec: RCLONE_CONFIG_DIR

## Requirements

1. When `RCLONE_CONFIG_DIR` is not set, default to `~/.config/rclone`
2. When `RCLONE_CONFIG_DIR` is set, use that path for the rclone Docker volume mount
3. The `--rclone-config-dir` CLI flag overrides the env var
4. A validation warning is emitted when the env var is not set (matching other config fields)
