# Functional Spec: Rclone Config Password

## Requirements

1. TUI prompts for the rclone config password (masked with `*`) before starting a download
2. Pressing esc on the password prompt returns to the menu without running rclone
3. The password is passed to the rclone Docker container via `RCLONE_CONFIG_PASS` env var
4. CLI checks the `RCLONE_CONFIG_PASS` env var first; if not set, prompts from stdin with hidden input
5. If the password is empty (user just presses enter), it is still passed through (rclone handles validation)
