# Proposal: Rclone Config Password Prompt

## Motivation

Rclone configs can be encrypted with a password. The rclone Docker container needs `RCLONE_CONFIG_PASS` set as an environment variable to decrypt the config. Previously there was no way to provide this, causing rclone downloads to fail with encrypted configs.

## Scope

- TUI: prompt for the password (masked input) before running rclone download
- CLI: read `RCLONE_CONFIG_PASS` env var if set, otherwise prompt from stdin with hidden input
- Pass the password to the Docker container via `-e RCLONE_CONFIG_PASS`

## Changes

- Add `configPass` parameter to `RcloneCopy` in `docker/rclone.go`
- Add password prompt TUI view (`tui/password.go`)
- Wire password prompt into the rclone download flow in `tui/app.go`
- Add `golang.org/x/term` dependency for hidden password input in CLI
- Prompt for password in CLI `download` command (`cmd/zolam/main.go`)
