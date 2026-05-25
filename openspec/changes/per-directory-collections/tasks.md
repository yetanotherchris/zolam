## 1. Config Rename and Migration

- [ ] 1.1 Rename `Config.CollectionName` to `Config.DefaultCollectionName` in `internal/domain/config.go`
- [ ] 1.2 Add `CollectionNameLegacy` field to `configJSON` tagged `json:"collectionName,omitempty"` for migration
- [ ] 1.3 Rename `configJSON.CollectionName` to `configJSON.DefaultCollectionName` tagged `json:"defaultCollectionName,omitempty"`
- [ ] 1.4 Update `loadConfigJSON`/`LoadConfig` to migrate legacy `collectionName` to `defaultCollectionName`
- [ ] 1.5 Update `SaveConfig` to write only `defaultCollectionName` (do not populate legacy field)
- [ ] 1.6 Update `MergeFlags` to use `DefaultCollectionName`
- [ ] 1.7 Update all references to `Config.CollectionName` across the codebase (`app.go`, `ingester.go`, `settings.go`, CLI commands)
- [ ] 1.8 Update `COLLECTION_NAME` env var override in `LoadConfig` to set `DefaultCollectionName`

## 2. Per-Directory Collection on DirectoryEntry

- [ ] 2.1 Add `CollectionName string json:"collectionName,omitempty"` to `DirectoryEntry`
- [ ] 2.2 Add `EffectiveCollection(defaultName string) string` method on `DirectoryEntry`
- [ ] 2.3 Update `AddOrUpdateDirectory` to accept and store collection name

## 3. TUI Ingest Flow

- [ ] 3.1 Add `collectionInput` text field to `IngestModel` and initialise with `DefaultCollectionName`
- [ ] 3.2 Add step 2 (collection name) between extensions (step 1) and confirm (now step 3)
- [ ] 3.3 Update step numbering: 0=directories, 1=extensions, 2=collection name, 3=confirm
- [ ] 3.4 Tab on step 2 skips to confirm; Enter on step 2 accepts value and moves to confirm
- [ ] 3.5 Add `CollectionName` field to `StartIngestMsg`
- [ ] 3.6 Display collection name on the confirm step view

## 4. TUI Settings

- [ ] 4.1 Change settings field label from `"Collection Name"` to `"Default Collection Name"`
- [ ] 4.2 Update the settings field getter/setter to use `DefaultCollectionName`

## 5. Ingester Changes

- [ ] 5.1 Update `runIngest` in `app.go` to pass `StartIngestMsg.CollectionName` into `IngestOptions`
- [ ] 5.2 Update `runIngest` in `app.go` to save collection name on directory entry after successful ingest (omit if matches default)
- [ ] 5.3 Update `RunUpdateOnly` to resolve effective collection per directory and group by collection
- [ ] 5.4 Update `RunUpdateOnly` to call `Run` once per collection group with the correct `CollectionName`

## 6. Clear Collection

- [ ] 6.1 Add "Clear Collection" menu item to the TUI menu
- [ ] 6.2 Create a `ClearCollectionModel` with text input (default: `DefaultCollectionName`) and confirmation prompt
- [ ] 6.3 Add `clearCollectionView` state to `AppModel` and wire up navigation
- [ ] 6.4 Implement `runClearCollection` command in `app.go` that calls `Ingester.Run` with `Reset: true` and the specified collection name
- [ ] 6.5 Display result/error to user via progress view

## 7. Tests

- [ ] 7.1 Unit tests for config migration: legacy key loaded, new key preferred, round-trip
- [ ] 7.2 Unit tests for `DirectoryEntry.EffectiveCollection`
- [ ] 7.3 Unit tests for `AddOrUpdateDirectory` with collection name
- [ ] 7.4 Build and run `go vet ./...`
