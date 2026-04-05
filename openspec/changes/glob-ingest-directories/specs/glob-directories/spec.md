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

### Requirement: Duplicate directories are deduplicated
When multiple patterns (or a pattern and a literal path) resolve to the same directory, the system SHALL include it only once. The first occurrence wins, preserving its extensions.

#### Scenario: Overlapping glob and literal path
- **WHEN** config has a literal entry `c:/notes/2024/Draft` with extensions `[".md"]`
- **AND** config has a glob entry `c:/notes/*/Draft` with extensions `[".txt"]`
- **THEN** the resolved list contains `c:/notes/2024/Draft` only once, with extensions `[".md"]` (from the first entry)

#### Scenario: Two globs resolve to same directory
- **WHEN** config has `c:/notes/*/Draft` and `c:/notes/2024/*`
- **AND** both resolve to include `c:/notes/2024/Draft`
- **THEN** the resolved list contains `c:/notes/2024/Draft` only once

### Requirement: Paths are normalized before comparison
All resolved paths SHALL have trailing slashes stripped and use forward slashes (`/`) before deduplication and before being passed to the ingest pipeline. This is consistent with existing `filepath.ToSlash` usage in the codebase.

#### Scenario: Trailing slash stripped
- **WHEN** a glob pattern `c:/notes/*/Draft/` resolves to `c:/notes/2024/Draft/`
- **THEN** the resolved path is `c:/notes/2024/Draft` (no trailing slash)

#### Scenario: Backslashes normalized
- **WHEN** a glob resolves to `c:\notes\2024\Draft`
- **THEN** the resolved path is `c:/notes/2024/Draft`

### Requirement: Go performs file discovery and passes file paths to the container
The Go side SHALL resolve globs to directories, walk them with extension filtering, and pass individual file paths to the Python ingest container. The Python container SHALL NOT perform directory traversal for glob-resolved ingests.

#### Scenario: Glob-resolved ingest passes file paths
- **WHEN** a glob pattern resolves to directories containing `notes.md`, `todo.txt`, and `readme.md`
- **THEN** Go discovers the files, writes them to a manifest, and passes the manifest to the container

### Requirement: Ingest container accepts a manifest file
`ingest.py` SHALL accept a `--manifest <path>` argument pointing to a JSON file containing a list of file paths to process. The manifest format SHALL be `{"files": ["path1", "path2", ...]}`.

#### Scenario: Manifest-based ingest
- **WHEN** the container is invoked with `--manifest /tmp/manifest.json`
- **AND** the manifest contains `{"files": ["/sources/root/notes.md", "/sources/root/readme.md"]}`
- **THEN** the container processes exactly those two files

### Requirement: Ingest container accepts individual file paths
`ingest.py` SHALL accept a `--file-path <path> [<path>...]` argument for passing one or more file paths directly as CLI args.

#### Scenario: File path args
- **WHEN** the container is invoked with `--file-path /sources/root/notes.md /sources/root/readme.md`
- **THEN** the container processes exactly those two files

### Requirement: Existing --directory flag is preserved
The existing `--directory` flag on `ingest.py` SHALL continue to work for backward compatibility. The Go orchestrator switches to `--manifest` as the primary interface.

#### Scenario: Directory flag still works
- **WHEN** the container is invoked with `--directory /sources/notes`
- **THEN** the container walks the directory and processes files as before

### Requirement: Single volume mount using common parent
When passing file paths to the container, Go SHALL mount the common parent directory of all resolved files as a single Docker volume. File paths in the manifest SHALL be relative to this container mount point.

#### Scenario: Common parent mounted
- **WHEN** resolved files are under `c:/myfolder/2024/Draft/` and `c:/myfolder/2025/Draft/`
- **THEN** Go mounts `c:/myfolder` as `/sources/root` and manifest paths are relative to `/sources/root`
