## 1. Dependencies

- [ ] 1.1 Add `github.com/bmatcuk/doublestar/v4` to `go.mod`

## 2. Glob Resolution

- [ ] 2.1 Create `ResolveDirectories` function that expands glob patterns in `[]DirectoryEntry` to concrete directories, passing literal paths through unchanged
- [ ] 2.2 Detect glob patterns by checking for `*`, `?`, `[`, `{` meta-characters
- [ ] 2.3 Filter resolved paths to directories only (exclude files)
- [ ] 2.4 Inherit extensions from the original pattern entry to each resolved entry
- [ ] 2.5 Log a warning and continue when a pattern matches zero directories

## 3. Integrate into Ingest Pipeline

- [ ] 3.1 Call `ResolveDirectories` at the start of `Ingester.Run` before building volume mounts
- [ ] 3.2 Call `ResolveDirectories` at the start of `Ingester.RunUpdateOnly` before hashing

## 4. TUI

- [ ] 4.1 Allow glob patterns to be entered in the TUI directory-add flow (no validation changes needed since patterns are stored as-is)

## 5. Tests

- [ ] 5.1 Unit tests for `ResolveDirectories` - literal path passthrough, simple glob, `**` recursive glob, zero-match warning, files excluded
- [ ] 5.2 Integration test: ingest with a glob pattern that resolves to multiple directories
- [ ] 5.3 Build and run `go vet ./...`
