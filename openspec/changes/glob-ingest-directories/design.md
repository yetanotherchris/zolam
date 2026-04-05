## Context

Zolam's ingest pipeline accepts a list of directory paths from `config.json`. Each `DirectoryEntry.Path` is a literal filesystem path. Users with structured file trees (e.g. year/topic folders) must manually add each directory. Go's `filepath.Glob` does not support `**` for recursive matching.

Currently the responsibility for file discovery is split: Go picks directories, then `ingest.py` walks them to find files. This change consolidates file discovery in Go and changes `ingest.py` to accept individual file paths.

## Goals / Non-Goals

**Goals:**
- Allow glob patterns in `DirectoryEntry.Path` that resolve to concrete directories at ingest time
- Support `**` for recursive directory matching (e.g. `c:/notes/**/Draft/`)
- Existing literal paths continue to work without changes
- Clear feedback when a pattern matches zero directories
- Move file discovery responsibility to Go, pass resolved file paths to the Python container

**Non-Goals:**
- File-level glob patterns in config (globs target directories, not individual files)
- Pattern validation in the config save path - patterns are resolved at ingest time only
- Watching for new directories that match patterns after ingest starts

## Decisions

### Use `doublestar` library for glob resolution

Go's `filepath.Glob` does not support `**`. The `github.com/bmatcuk/doublestar/v4` library supports full glob syntax including `**` and is widely used.

Alternative considered: custom recursive walk with `filepath.Match` per segment. Rejected because it reimplements what doublestar already handles correctly, including edge cases around symlinks and path separators on Windows.

### Resolve globs in a new function in the `zolam` package

A `ResolveDirectories(entries []DirectoryEntry, warnFn func(string)) []DirectoryEntry` function in the `zolam` package takes the configured entries and returns expanded entries. Literal paths pass through unchanged. Glob patterns expand to one entry per matched directory, each inheriting the extensions from the original pattern entry. The `warnFn` callback is used to report zero-match patterns.

All resolved paths are normalized with `filepath.ToSlash` and deduplicated. When overlapping patterns or a pattern-plus-literal resolve to the same directory, the first occurrence wins (preserving its extensions).

This function is called:
- In `Ingester.Run` and `Ingester.RunUpdateOnly` before iterating directories
- In the CLI `ingest` and `update` commands when directories come from CLI args

### Detect glob vs literal by checking for glob meta-characters

If a path contains `*`, `?`, `[`, or `{` it is treated as a glob pattern. Otherwise it is a literal path. No new config field needed.

### Glob resolution happens at ingest time, not config save time

Patterns are stored as-is in `config.json`. Resolution happens when `Run` or `RunUpdateOnly` is called. This means the set of matched directories can change between runs as the filesystem changes, which is the expected behavior.

### Go does file discovery, Python processes individual files

Currently `ingest.py` takes `--directory` args and walks each directory to find files. This mixes file discovery (which depends on glob resolution, extension filtering) with file processing (text extraction, chunking, embedding).

New approach: Go resolves globs to directories, walks them with extension filtering, and produces a list of absolute file paths. These are passed to the Python container which processes them without needing to do any directory traversal.

This eliminates the Docker volume mount collision problem entirely. Instead of mounting each resolved directory separately, Go mounts the common parent directory once and passes file paths relative to that mount.

### File path passing: manifest file and `--file-path` flag

`ingest.py` gains two ways to receive file paths:

1. **`--manifest <path>`** - a JSON file containing a list of file paths. Go writes this file, mounts it into the container. Used for bulk ingest operations where the file list could be large (avoids command-line length limits).
2. **`--file-path <path> [<path>...]`** - one or more file paths as CLI args. For small/ad-hoc operations.

The existing `--directory` flag is kept for backward compatibility but the primary path from Go becomes `--manifest`.

Manifest format (paths are absolute container paths):
```json
{
  "files": [
    "/sources/root/2024/Draft/notes.md",
    "/sources/root/2024/Draft/todo.txt",
    "/sources/root/2025/Draft/readme.md"
  ]
}
```

### Mount strategy

Go groups resolved file paths by drive letter (Windows) or assumes a single root (Unix). For each drive group, it finds the common parent and mounts it as a volume (e.g. `-v c:/myfolder:/sources/c`). In practice, most users will have all files on one drive so there will be a single mount. File paths in the manifest use absolute container paths.

### Source label for chunk metadata

`ingest.py` currently uses the directory name as the `source` metadata field in ChromaDB chunks. When switching to manifest/file-path mode, the directory context is lost. Fix: when processing individual files, derive `source` from the file's parent directory name. This preserves the existing metadata convention without requiring Go to pass source labels.

### Pass resolved extensions through the ingest pipeline

`RunUpdateOnly` currently looks up per-directory extensions by matching `d.Path` against config entries. After glob resolution, the resolved paths won't match the stored pattern. Fix: `ResolveDirectories` produces `DirectoryEntry` values with the resolved path and inherited extensions. `RunUpdateOnly` should use the resolved entries directly for extension lookup rather than matching back against the config.

### Normalize trailing slashes and path separators

`ResolveDirectories` strips trailing slashes and applies `filepath.ToSlash` to all resolved paths before deduplication and before returning. This keeps behavior consistent with existing config path handling.

## Risks / Trade-offs

- [Glob matches zero directories] -> Log a warning and continue. Do not fail the entire ingest for one empty pattern.
- [Pattern matches unexpected directories] -> User responsibility. The TUI could show a preview of matched directories before ingesting, but this is not in initial scope.
- [Windows path separator issues] -> `doublestar` handles both `/` and `\`. Paths in config.json already use forward slashes (via `filepath.ToSlash`).
- [Performance on deep trees] -> `doublestar.Glob` walks the filesystem. For very deep trees this could be slow. Acceptable for a local tool; not a concern at personal-use scale.
- [Manifest file cleanup] -> The manifest is written to a temp file and mounted. Go should clean it up after the container exits.
- [Backward compatibility] -> `--directory` is kept on `ingest.py` so existing Docker Compose usage still works. The Go orchestrator switches to `--manifest`.
