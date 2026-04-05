# Tasks: Rclone Config Password Prompt

- [x] Add `configPass` parameter to `RcloneCopy` in `docker/rclone.go`
- [x] Pass password as `-e RCLONE_CONFIG_PASS` to Docker when non-empty
- [x] Create `tui/password.go` with masked input view
- [x] Add `passwordView` state to `tui/app.go`
- [x] Wire "Download (rclone)" menu item to show password prompt first
- [x] Handle `PasswordSubmitMsg` to start rclone with password
- [x] Add `golang.org/x/term` dependency
- [x] Add env var check and stdin password prompt to CLI `download` command
- [x] Build and test
