## 1. Dependencies

- [ ] 1.1 Add `github.com/bmatcuk/doublestar/v4` to `go.mod`

## 2. Glob Resolution (Go)

- [ ] 2.1 Create `ResolveDirectories(entries []DirectoryEntry, warnFn func(string)) []DirectoryEntry` in the `zolam` package
- [ ] 2.2 Detect glob patterns by checking for `*`, `?`, `[`, `{` meta-characters; literal paths pass through unchanged
- [ ] 2.3 Filter resolved paths to directories only (exclude files)
- [ ] 2.4 Inherit extensions from the original pattern entry to each resolved entry
- [ ] 2.5 Normalize all resolved paths: strip trailing slashes, apply `filepath.ToSlash`
- [ ] 2.6 Deduplicate resolved directories (first occurrence wins, preserving its extensions)
- [ ] 2.7 Log a warning via `warnFn` and continue when a pattern matches zero directories

## 3. File Discovery (Go)

- [ ] 3.1 Create a function that takes resolved `[]DirectoryEntry`, walks each directory with extension filtering, and returns a list of absolute file paths
- [ ] 3.2 Write discovered file paths to a JSON manifest file (`{"files": [...]}`) in a temp location
- [ ] 3.3 Clean up the manifest temp file after the container exits

## 4. Docker Mount Strategy (Go)

- [ ] 4.1 Group resolved file paths by drive letter (Windows) or single root (Unix)
- [ ] 4.2 Compute common parent per group and mount each as a volume (e.g. `-v c:/myfolder:/sources/c`)
- [ ] 4.3 Write manifest with absolute container paths
- [ ] 4.4 Mount the manifest file into the container and pass `--manifest <container-path>` to `ingest.py`

## 5. Python Ingest Container

- [ ] 5.1 Add `--manifest <path>` argument to `ingest.py` that reads a JSON file of file paths
- [ ] 5.2 Add `--file-path <path> [<path>...]` argument to `ingest.py` for individual files
- [ ] 5.3 When `--manifest` or `--file-path` is used, process exactly those files (no directory walking)
- [ ] 5.4 Derive `source` metadata from the file's parent directory name when using `--manifest` or `--file-path`
- [ ] 5.5 Keep existing `--directory` flag working for backward compatibility

## 6. Integrate into Ingest Pipeline (Go)

- [ ] 6.1 Update `Ingester.Run` to use glob resolution -> file discovery -> manifest -> single mount flow
- [ ] 6.2 Update `Ingester.RunUpdateOnly` to use resolved entries directly for extension lookup instead of matching back against `config.Directories`
- [ ] 6.3 Update CLI `ingest` and `update` commands to resolve glob patterns in CLI directory args

## 7. TUI

- [ ] 7.1 Allow glob patterns to be entered in the TUI directory-add flow (patterns stored as-is in config)

## 8. Tests

- [ ] 8.1 Unit tests for `ResolveDirectories`: literal passthrough, simple glob, `**` recursive glob, zero-match warning, files excluded, deduplication, trailing slash normalization
- [ ] 8.2 Unit tests for file discovery: extension filtering, manifest JSON format
- [ ] 8.3 Unit tests for common parent computation and mount path conversion
- [ ] 8.4 Tests for `ingest.py`: `--manifest`, `--file-path`, and `--directory` backward compatibility
- [ ] 8.5 Build and run `go vet ./...`
