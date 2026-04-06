## Why

The collection name is currently a global setting (`CollectionName` in config). All ingested directories go into the same ChromaDB collection. Users who want to separate their data (e.g. work notes vs personal notes vs code) have no way to target different collections per directory. There is also no way to clear a collection's data from the TUI without using the CLI `--reset` flag.

## What Changes

- Rename `Config.CollectionName` to `Config.DefaultCollectionName` (config key `defaultCollectionName`). The default remains `"my-notes"`. **BREAKING**: existing `collectionName` key in config.json is migrated on load.
- Add an optional `CollectionName` field to `DirectoryEntry`. When set, that directory's ingest targets the specified collection instead of the default.
- Add a new optional step in the TUI ingest flow (between extension selection and confirmation) where the user can set the collection name for this ingest run. Defaults to `DefaultCollectionName`. Pressing Tab skips the step.
- The ingester passes a per-directory collection name to `ingest.py` when the directory entry specifies one.
- Add a "Clear Collection" option in the TUI that deletes all data from a named ChromaDB collection via the `--reset` flag on `ingest.py`.
- The settings screen label changes from "Collection Name" to "Default Collection Name".

## Capabilities

### New Capabilities
- `per-directory-collection`: Assign a ChromaDB collection name per directory entry, falling back to the default collection name
- `clear-collection`: Clear all data from a named ChromaDB collection via the TUI

### Modified Capabilities

## Impact

- `internal/domain/config.go` - Rename `CollectionName` to `DefaultCollectionName`, add `CollectionName` to `DirectoryEntry`, migration logic for old config key
- `internal/domain/config.go` (`configJSON`) - Rename JSON key, add directory-level collection field
- `internal/tui/ingest.go` - New optional step for collection name input
- `internal/tui/settings.go` - Label change, potential "Clear Collection" action
- `internal/tui/app.go` - Pass collection name from `StartIngestMsg`, handle clear-collection flow
- `internal/zolam/ingester.go` - Per-directory collection name in `Run` and `RunUpdateOnly`
- `ingest.py` - Already supports `COLLECTION_NAME` env var; no changes needed for per-directory since the Go side calls `Run` per directory group
- `internal/tui/menu.go` - New "Clear Collection" menu item (or settings action)
