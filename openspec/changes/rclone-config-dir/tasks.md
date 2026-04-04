# Tasks: RCLONE_CONFIG_DIR

- [x] Add `RcloneConfigDir` field to `Config` struct
- [x] Load from env var with default in `LoadConfig`
- [x] Add validation warning when env var not set
- [x] Add `rclone-config-dir` to `MergeFlags`
- [x] Update `RcloneSync` signature to accept config dir parameter
- [x] Update `RcloneSync` callers in `tui/app.go` and `cmd/zolam/main.go`
- [x] Add/update tests in `config_test.go`
- [x] Run tests to verify
