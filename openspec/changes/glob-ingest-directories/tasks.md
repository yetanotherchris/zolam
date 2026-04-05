## 1. Dependencies

- [ ] 1.1 Add `github.com/bmatcuk/doublestar/v4` to `go.mod`

## 2. Glob Resolution

- [ ] 2.1 Create `ResolveDirectories(entries []DirectoryEntry, warnFn func(string)) []DirectoryEntry` in the `zolam` package
- [ ] 2.2 Detect glob patterns by checking for `*`, `?`, `[`, `{` meta-characters; literal paths pass through unchanged
- [ ] 2.3 Filter resolved paths to directories only (exclude files)
- [ ] 2.4 Inherit extensions from the original pattern entry to each resolved entry
- [ ] 2.5 Normalize all resolved paths: strip trailing slashes, apply `filepath.ToSlash`
- [ ] 2.6 Deduplicate resolved directories (first occurrence wins, preserving its extensions)
- [ ] 2.7 Log a warning via `warnFn` and continue when a pattern matches zero directories

## 3. Docker Volume Mount Collision

- [ ] 3.1 Update `Ingester.Run` to detect duplicate `filepath.Base` names among resolved directories and generate unique mount suffixes (e.g. `/sources/Draft`, `/sources/Draft_1`)

## 4. Integrate into Ingest Pipeline

- [ ] 4.1 Call `ResolveDirectories` at the start of `Ingester.Run` before building volume mounts
- [ ] 4.2 Call `ResolveDirectories` at the start of `Ingester.RunUpdateOnly` before hashing
- [ ] 4.3 Refactor `RunUpdateOnly` to use resolved `DirectoryEntry` extensions directly instead of matching back against `config.Directories` (which stores the unresolved glob pattern)

## 5. CLI Commands

- [ ] 5.1 Update `newIngestCmd` to resolve glob patterns in CLI directory args before passing to `Ingester.Run`
- [ ] 5.2 Update `newUpdateCmd` to resolve glob patterns in CLI directory args before passing to `Ingester.RunUpdateOnly`

## 6. TUI

- [ ] 6.1 Allow glob patterns to be entered in the TUI directory-add flow (patterns stored as-is in config)

## 7. Tests

- [ ] 7.1 Unit tests for `ResolveDirectories`: literal passthrough, simple glob, `**` recursive glob, zero-match warning, files excluded, deduplication, trailing slash normalization
- [ ] 7.2 Unit test for volume mount uniqueness when multiple resolved dirs share a base name
- [ ] 7.3 Unit test for `RunUpdateOnly` extension lookup using resolved entries (not config pattern match)
- [ ] 7.4 Build and run `go vet ./...`
