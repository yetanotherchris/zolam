# Config JSON Refactor

## Motivation

Configuration is currently loaded from environment variables and `.env` files only. This makes it awkward to persist settings like ingested directories. The `update` command requires directories to be passed as arguments every time, which is not autonomous. The `Extensions` field is exposed as a configurable variable but is really just informational (the supported file types).

## Scope

1. **Extensions cleanup**: Replace the `Extensions` config variable with a constant `SupportedFileExtensions`. Display it on the stats page as informational only. Remove it from `Config` struct.
2. **config.json loading**: Load settings from `~/.zolam/config.json` with env var override (env vars checked first, then config.json). Config keys are camelCase in JSON, CAPS for env vars.
3. **Directories tracking**: Add a `directories` node to config.json that stores previously ingested directories with their assigned file extensions.
4. **Autonomous update**: The `update` command reads directories from config.json when no arguments are provided, making it fully autonomous.
5. **README rewrite**: Simplify the README with fewer sections (e.g. "Quick Start" instead of nested Setup sections).

## Decisions

- config.json lives at `~/.zolam/config.json` (same parent as data dir)
- Env vars take precedence over config.json values
- Extensions are per-directory in the directories node, not global
- `update` with no args reads from config.json; with args uses those instead
- `ingest` saves directories + extensions to config.json after successful ingest
