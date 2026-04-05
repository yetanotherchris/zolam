## Context

Zolam's ingest pipeline accepts a list of directory paths from `config.json`. Each `DirectoryEntry.Path` is a literal filesystem path. Users with structured file trees (e.g. year/topic folders) must manually add each directory. Go's `filepath.Glob` does not support `**` for recursive matching.

## Goals / Non-Goals

**Goals:**
- Allow glob patterns in `DirectoryEntry.Path` that resolve to concrete directories at ingest time
- Support `**` for recursive directory matching (e.g. `c:/notes/**/Draft/`)
- Existing literal paths continue to work without changes
- Clear feedback when a pattern matches zero directories

**Non-Goals:**
- File-level glob patterns (globs target directories, not individual files)
- Pattern validation in the config save path - patterns are resolved at ingest time only
- Watching for new directories that match patterns after ingest starts

## Decisions

### Use `doublestar` library for glob resolution

Go's `filepath.Glob` does not support `**`. The `github.com/bmatcuk/doublestar/v4` library supports full glob syntax including `**` and is widely used.

Alternative considered: custom recursive walk with `filepath.Match` per segment. Rejected because it reimplements what doublestar already handles correctly, including edge cases around symlinks and path separators on Windows.

### Resolve globs in a new function, not inline in Ingester

A `ResolveDirectories(entries []DirectoryEntry) []DirectoryEntry` function in the `domain` or `zolam` package takes the configured entries and returns expanded entries. Literal paths pass through unchanged. Glob patterns expand to one entry per matched directory, each inheriting the extensions from the original pattern entry.

This keeps the Ingester unchanged - it still receives concrete directory lists.

### Detect glob vs literal by checking for glob meta-characters

If a path contains `*`, `?`, `[`, or `{` it is treated as a glob pattern. Otherwise it is a literal path. No new config field needed.

Alternative considered: adding an `IsGlob bool` field to `DirectoryEntry`. Rejected because it adds config complexity for no benefit - the presence of glob characters is unambiguous.

### Glob resolution happens at ingest time, not config save time

Patterns are stored as-is in `config.json`. Resolution happens when `Run` or `RunUpdateOnly` is called. This means the set of matched directories can change between runs as the filesystem changes, which is the expected behavior.

## Risks / Trade-offs

- [Glob matches zero directories] -> Log a warning and continue. Do not fail the entire ingest for one empty pattern.
- [Pattern matches unexpected directories] -> User responsibility. The TUI could show a preview of matched directories before ingesting, but this is not in initial scope.
- [Windows path separator issues] -> `doublestar` handles both `/` and `\`. Paths in config.json already use forward slashes (via `filepath.ToSlash`).
- [Performance on deep trees] -> `doublestar.Glob` walks the filesystem. For very deep trees this could be slow. Acceptable for a local tool; not a concern at personal-use scale.
