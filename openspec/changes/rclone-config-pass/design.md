# Design: Rclone Config Password Prompt

## Architecture

### TUI Flow

1. User selects "Download (rclone)" from menu
2. App transitions to `passwordView` showing a masked text input
3. On enter, `PasswordSubmitMsg` is sent containing the password
4. App transitions to `progressView` and runs rclone with the password
5. On esc, returns to menu without running rclone

### CLI Flow

1. Check `RCLONE_CONFIG_PASS` environment variable
2. If not set, prompt on stderr with hidden input via `golang.org/x/term`
3. Pass password to `RcloneCopy`

### Components Modified

- `internal/docker/rclone.go` - Accept password parameter, pass as `-e RCLONE_CONFIG_PASS` to Docker
- `internal/tui/password.go` - New reusable password prompt view using `bubbles/textinput` with `EchoPassword` mode
- `internal/tui/app.go` - Add `passwordView` state, wire `PasswordSubmitMsg` to `runRclone`
- `cmd/zolam/main.go` - Add env var check and stdin prompt for password
