## Context

Zolam currently has a single `CollectionName` field on `Config` (default: `"my-notes"`). All directories ingest into this one collection. The ingester passes `COLLECTION_NAME` as an env var to the Docker container. The `ingest.py` script already supports `--reset` to delete a collection.

The TUI ingest flow has 3 steps: add directories, select extensions, confirm. The settings screen allows editing `CollectionName` directly.

## Goals / Non-Goals

**Goals:**
- Each `DirectoryEntry` can optionally specify a collection name
- A default collection name applies when a directory has no override
- The TUI ingest flow lets the user set the collection for the current run
- Users can clear (delete all data from) a collection from the TUI
- Existing configs with `collectionName` are migrated to `defaultCollectionName` on load

**Non-Goals:**
- Querying across multiple collections (that is a chroma-mcp concern)
- Per-file collection assignment (too granular)
- Collection management beyond clearing (renaming, listing, merging)

## Decisions

### Rename CollectionName to DefaultCollectionName

`Config.CollectionName` becomes `Config.DefaultCollectionName`. The JSON key changes from `"collectionName"` to `"defaultCollectionName"`.

Migration: `loadConfigJSON` checks for the old `"collectionName"` key. If `"defaultCollectionName"` is absent but `"collectionName"` is present, use its value. This is a one-time read-time migration; the next `SaveConfig` writes the new key only.

Alternative considered: keeping both fields and treating `collectionName` as an alias. Rejected because it adds ambiguity about which takes precedence.

### Add CollectionName to DirectoryEntry

`DirectoryEntry` gains `CollectionName string json:"collectionName,omitempty"`. When empty, the directory uses `Config.DefaultCollectionName`.

The effective collection for a directory is resolved in the ingester, not stored redundantly. A helper `(d DirectoryEntry).EffectiveCollection(defaultName string) string` returns `d.CollectionName` if non-empty, else `defaultName`.

### TUI ingest flow: optional collection name step

A new step 2 (between extensions and confirm) shows a text input pre-filled with `DefaultCollectionName`. The user can edit it or press Tab to skip (keeping the default). The confirm step shows the chosen collection name.

`StartIngestMsg` gains a `CollectionName string` field. `app.go` uses this value when building `IngestOptions`, and when saving the directory entry to config after a successful ingest.

### Per-directory collection in the ingester

`Ingester.Run` already accepts `IngestOptions.CollectionName` and passes it as `COLLECTION_NAME` env var. No change needed to `Run` itself.

The change is in the callers: `runIngest` in `app.go` sets `opts.CollectionName` from `StartIngestMsg.CollectionName`. `RunUpdateOnly` resolves the effective collection per directory and calls `Run` once per unique collection, grouping directories that share the same collection.

### Clear collection via TUI

A new menu item "Clear Collection" prompts for a collection name (defaulting to `DefaultCollectionName`), then runs the ingest container with `--reset` and the specified `COLLECTION_NAME`. This reuses the existing `ingest.py --reset` mechanism.

Alternative considered: adding a direct ChromaDB HTTP delete call from Go. Rejected because `ingest.py` already handles this and the Docker networking is already set up for it.

### Config migration strategy

`configJSON` gains a `CollectionNameLegacy` field tagged `json:"collectionName,omitempty"` and `DefaultCollectionName` tagged `json:"defaultCollectionName,omitempty"`. On load, if `DefaultCollectionName` is empty and `CollectionNameLegacy` is non-empty, the legacy value is used. On save, only `DefaultCollectionName` is written (the legacy field is not populated in `SaveConfig`).

## Risks / Trade-offs

- [Breaking config key rename] -> Mitigated by read-time migration. Old key is silently upgraded on next save. Users who downgrade to an older zolam version would lose the setting, but that is unlikely for a personal tool.
- [RunUpdateOnly grouping by collection] -> Adds complexity to the update-only path. If a directory's collection changes between runs, old data stays in the old collection. This is acceptable; the user can clear the old collection manually.
- [Clear collection is destructive] -> The TUI should show a confirmation prompt before clearing. A simple "Are you sure? Press Enter to confirm, Esc to cancel" is sufficient.
