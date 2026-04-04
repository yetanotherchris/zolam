# Proposal: Configurable Rclone Config Directory

## Motivation

The rclone config directory is currently hardcoded to `~/.config/rclone` in `internal/docker/rclone.go`. Users who store their rclone config in a non-standard location have no way to override this.

## Scope

Add a `RCLONE_CONFIG_DIR` environment variable (and corresponding CLI flag) to allow users to specify where their rclone configuration lives.

- Default: `~/.config/rclone` (preserves current behaviour)
- Follows the same pattern as existing config fields (env var, CLI flag, .env file support)

## Changes

- Add `RcloneConfigDir` field to `Config` struct
- Wire it through `LoadConfig`, `MergeFlags`, and `Validate`
- Pass it into `RcloneSync` instead of hardcoding
- Update tests
