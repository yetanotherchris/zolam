# TUI Settings Editor

## Overview

Transform the read-only Settings view in the TUI into an editable form that allows users to modify configuration values and persist them to `~/.zolam/config.json`.

## Editable Fields

The following fields are editable via text input:
- Collection Name
- Data Dir
- Rclone Source
- Rclone Config Dir

## Directory Management

The ingested directories list supports:
- Deleting a directory entry (removes it from config, does not affect files on disk)
- Directories are added via the Ingest workflow (not duplicated here)

## Interaction

### Navigation
- Up/Down arrows move between fields
- Enter on a field opens it for editing (shows a text input pre-filled with current value)
- Enter again confirms the edit
- Esc while editing cancels the edit and reverts to the previous value
- Esc while navigating returns to the main menu

### Directory deletion
- When cursor is on a directory entry, pressing `d` or `delete` removes it
- A confirmation is not needed (the change isn't persisted until save)

### Saving
- Each field edit is saved to config.json immediately when confirmed with Enter
- If the save fails, an error message is shown at the bottom of the screen
- Directory deletions are also saved immediately

## Menu Item Update

Change the menu description from "View current configuration" to "Edit configuration".

## Non-requirements

- No adding directories from settings (use Ingest for that)
- No editing per-directory extensions from settings (use Ingest with --extensions)
- No validation of paths (user responsibility)
