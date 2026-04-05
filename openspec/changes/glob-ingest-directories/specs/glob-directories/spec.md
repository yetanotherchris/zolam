## ADDED Requirements

### Requirement: Glob patterns resolve to directories at ingest time
The system SHALL accept glob patterns (containing `*`, `?`, `[`, or `{`) in `DirectoryEntry.Path` and resolve them to concrete directory paths before ingesting. Literal paths (no glob characters) SHALL pass through unchanged.

#### Scenario: Literal path unchanged
- **WHEN** a directory entry has path `c:/notes/docs`
- **THEN** the path is used as-is for ingestion

#### Scenario: Simple wildcard pattern
- **WHEN** a directory entry has path `c:/notes/*/Draft`
- **THEN** the system resolves to all directories matching that pattern (e.g. `c:/notes/2024/Draft`, `c:/notes/2025/Draft`)

#### Scenario: Recursive wildcard pattern
- **WHEN** a directory entry has path `c:/myfolder/**/D*/`
- **THEN** the system resolves to all subdirectories at any depth inside `c:/myfolder/` whose name starts with `D`

### Requirement: Resolved directories inherit extensions from pattern entry
Each directory resolved from a glob pattern SHALL inherit the `extensions` list from the original `DirectoryEntry` that contained the pattern.

#### Scenario: Extensions carried forward
- **WHEN** a directory entry has path `c:/notes/*/Draft` with extensions `[".md", ".txt"]`
- **AND** the pattern resolves to `c:/notes/2024/Draft` and `c:/notes/2025/Draft`
- **THEN** both resolved entries have extensions `[".md", ".txt"]`

### Requirement: Zero-match patterns produce a warning
When a glob pattern matches zero directories, the system SHALL log a warning and continue processing remaining directories. It SHALL NOT fail the ingest run.

#### Scenario: Pattern matches nothing
- **WHEN** a directory entry has path `c:/empty/**/Docs` and no directories match
- **THEN** a warning is logged indicating the pattern matched no directories
- **AND** ingestion continues with any remaining directory entries

### Requirement: Glob patterns stored as-is in config
Glob patterns SHALL be stored in `config.json` without modification. Resolution happens at ingest time only, not at config save time.

#### Scenario: Config round-trip preserves pattern
- **WHEN** a user adds a directory with path `c:/notes/**/Draft`
- **AND** the config is saved and reloaded
- **THEN** the path in the loaded config is still `c:/notes/**/Draft`

### Requirement: Only directories are matched
Glob resolution SHALL only return paths that are directories. Files matching the pattern SHALL be excluded.

#### Scenario: File matches pattern but is excluded
- **WHEN** a glob pattern `c:/notes/D*` matches both a directory `c:/notes/Docs` and a file `c:/notes/Data.txt`
- **THEN** only `c:/notes/Docs` is included in the resolved list
