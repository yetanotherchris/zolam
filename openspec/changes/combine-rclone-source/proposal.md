# Proposal: Combine RCLONE_REMOTE and RCLONE_SOURCE into single RCLONE_SOURCE

## Motivation

Currently rclone remote and source path are separate config fields (`RCLONE_REMOTE` and `RCLONE_SOURCE`). Since rclone natively accepts combined strings like `gdrive:/path/to/folder` or `gdrive:`, there's no reason to split them. A single `RCLONE_SOURCE` simplifies configuration.

## Scope

- Remove `RCLONE_REMOTE` and the `RcloneRemote` config field
- Repurpose `RCLONE_SOURCE` to hold the full rclone source string (e.g. `gdrive:/path/to/folder`, `gdrive:`)
- Remove `--remote` CLI flag from the download command
- Update `RcloneSync` to accept a single source string instead of remote + source
- No default value - `RCLONE_SOURCE` is user-provided
