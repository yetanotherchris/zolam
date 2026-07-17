# Functional spec: Zolam v3

## Requirement: Daemon-free ingestion

Zolam SHALL support ingesting files into a per-project flat-file index
without any running service, using `uv run` to invoke an embedded Python
script for extraction/chunking/embedding.

### Scenario: First ingest creates a project

- **GIVEN** `~/.zolam/my-project/` does not exist
- **WHEN** the user runs `zolam ingest ~/notes --project my-project --extensions .md,.pdf`
- **THEN** `~/.zolam/my-project/project.json`, `index.duckdb`, `index.md`,
  and `file-hashes.json` are created, and PDFs get sidecars under
  `extracted/`.

### Scenario: uv missing

- **GIVEN** `uv` is not on PATH
- **WHEN** any v3 command that needs the Python script runs
- **THEN** zolam fails with a one-line remedy naming brew/winget/scoop and
  Astral's installer.

## Requirement: Incremental update

### Scenario: Update only reprocesses changed files

- **GIVEN** a project already ingested, and one file modified since
- **WHEN** the user runs `zolam update --project my-project` (no dirs)
- **THEN** only the changed file is re-extracted/re-chunked/re-embedded;
  unchanged files are skipped; `index.md` is regenerated to reflect the
  current file set.

### Scenario: Removed file is cleaned up

- **GIVEN** a previously-ingested file no longer exists on disk
- **WHEN** `zolam update` runs
- **THEN** its rows are deleted from the index, its sidecar (if any) is
  removed, and it disappears from `index.md`.

## Requirement: Query

### Scenario: Semantic query

- **WHEN** the user runs `zolam query "<text>" --project my-project`
- **THEN** the query is embedded with the project's recorded model and
  the top-k most similar chunks are printed with path, page/chunk, score,
  and matching text.

### Scenario: Keyword query

- **WHEN** the user runs `zolam query "<text>" --project my-project --keyword`
- **THEN** matching is done by substring/ILIKE without invoking the
  embedding model.

### Scenario: Model mismatch

- **GIVEN** a project's recorded `embedding_model` differs from zolam's
  current default
- **WHEN** `query`/`ingest`/`update` runs against it
- **THEN** zolam refuses with a message instructing `--reset`.

## Requirement: Pluggable backends

Zolam SHALL support `duckdb` (default), `jsonl`, and `chroma` (legacy)
backends selected via `--backend`, recorded in `project.json`, and
enforced consistently on later commands against the same project.

## Requirement: Claude Code / OpenCode integration without MCP

### Scenario: Install the skill

- **WHEN** the user runs `zolam init claude`
- **THEN** `~/.claude/skills/zolam/SKILL.md` is written (idempotent on
  re-run) and a suggested `CLAUDE.md` snippet is printed.

## Requirement: Legacy backend preserved

### Scenario: Existing chroma workflow keeps working

- **GIVEN** a user who ran `zolam ingest ... --collection X` before v3
- **WHEN** they run the same command (or `--backend chroma`) after
  upgrading
- **THEN** the Docker/ChromaDB pipeline behaves exactly as before,
  with a printed deprecation notice.
