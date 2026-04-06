## ADDED Requirements

### Requirement: Default collection name in config
The system SHALL store a `defaultCollectionName` field in `config.json` with a default value of `"my-notes"`. This replaces the previous `collectionName` field.

#### Scenario: Fresh config uses default
- **WHEN** no config.json exists
- **THEN** `DefaultCollectionName` SHALL be `"my-notes"`

#### Scenario: Legacy collectionName is migrated
- **WHEN** config.json contains `"collectionName": "work-docs"` but no `"defaultCollectionName"`
- **THEN** `DefaultCollectionName` SHALL be `"work-docs"`
- **AND** the next `SaveConfig` SHALL write `"defaultCollectionName": "work-docs"` and omit `"collectionName"`

#### Scenario: Both keys present
- **WHEN** config.json contains both `"collectionName"` and `"defaultCollectionName"`
- **THEN** `DefaultCollectionName` SHALL use the `"defaultCollectionName"` value

### Requirement: Per-directory collection name
Each `DirectoryEntry` in config SHALL support an optional `collectionName` field. When set, that directory's ingest SHALL target the specified collection instead of the default.

#### Scenario: Directory with explicit collection
- **WHEN** a directory entry has `"collectionName": "work-docs"`
- **THEN** ingesting that directory SHALL use the `"work-docs"` collection

#### Scenario: Directory without collection name uses default
- **WHEN** a directory entry has no `"collectionName"` field
- **AND** `DefaultCollectionName` is `"my-notes"`
- **THEN** ingesting that directory SHALL use the `"my-notes"` collection

#### Scenario: Config round-trip preserves per-directory collection
- **WHEN** a directory entry has `"collectionName": "code-snippets"`
- **AND** config is saved and reloaded
- **THEN** the directory entry SHALL still have `"collectionName": "code-snippets"`

### Requirement: TUI ingest step for collection name
The TUI ingest flow SHALL include an optional step (after extension selection, before confirmation) where the user can set the collection name for the current ingest run. The field SHALL default to `DefaultCollectionName`.

#### Scenario: User sets custom collection in TUI
- **WHEN** the user reaches the collection name step
- **AND** types `"work-docs"` and presses Enter or Tab
- **THEN** the ingest SHALL use the `"work-docs"` collection
- **AND** the confirmation step SHALL display the chosen collection name

#### Scenario: User skips collection step
- **WHEN** the user reaches the collection name step
- **AND** presses Tab without editing
- **THEN** the ingest SHALL use `DefaultCollectionName`

#### Scenario: Collection name shown on confirm step
- **WHEN** the user reaches the confirmation step
- **THEN** the chosen collection name SHALL be displayed alongside directories and extensions

### Requirement: Ingester passes per-directory collection name
The ingester SHALL pass the effective collection name (per-directory override or default) as the `COLLECTION_NAME` environment variable to the ingest container for each directory.

#### Scenario: Two directories with different collections
- **WHEN** directory `/notes` has collection `"my-notes"`
- **AND** directory `/code` has collection `"code-snippets"`
- **THEN** the ingester SHALL run the container twice, once with `COLLECTION_NAME=my-notes` and once with `COLLECTION_NAME=code-snippets`

#### Scenario: Update-only respects per-directory collection
- **WHEN** `RunUpdateOnly` processes directories with different collection names
- **THEN** each directory group SHALL be ingested with its effective collection name

### Requirement: Settings screen reflects rename
The settings screen SHALL display `"Default Collection Name"` as the label for the default collection field, replacing the previous `"Collection Name"` label.

#### Scenario: Settings label updated
- **WHEN** the user opens the settings screen
- **THEN** the field SHALL be labelled `"Default Collection Name"`

### Requirement: Saved directory entries include collection name
After a successful ingest, the directory entry saved to config SHALL include the collection name used for that ingest.

#### Scenario: Ingest saves collection to directory entry
- **WHEN** the user ingests directory `/notes` with collection `"work-docs"`
- **AND** the ingest completes successfully
- **THEN** the directory entry in config.json SHALL have `"collectionName": "work-docs"`

#### Scenario: Default collection not stored on directory entry
- **WHEN** the user ingests directory `/notes` using the default collection
- **THEN** the directory entry in config.json SHALL omit the `"collectionName"` field (or store it empty)
