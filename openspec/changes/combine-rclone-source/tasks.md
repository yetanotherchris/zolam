# Tasks: Combined RCLONE_SOURCE

- [x] Remove `RcloneRemote` from `Config` struct
- [x] Remove `RCLONE_REMOTE` loading and validation warning from `LoadConfig`/`Validate`
- [x] Remove `rclone-remote` from `MergeFlags`
- [x] Update `RcloneCopy` (renamed from `RcloneSync`) to accept single `source` instead of `remote` + `source`
- [x] Update download command: remove `--remote` flag, update error message
- [x] Update TUI: remove Rclone Remote from settings, update `RcloneCopy` call
- [x] Update config command output
- [x] Update tests
- [x] Update `RCLONE_CONFIG_DIR` default to `~/.rclone`
- [x] Build and test
