# Tasks: EPUB File Support

## 1. Python Changes

- [ ] Add `ebooklib`, `beautifulsoup4`, `lxml` to Dockerfile pip install
- [ ] Add `.epub` to `SUPPORTED_EXTENSIONS` in `ingest.py`
- [ ] Add EPUB extraction case in `extract_text()` using ebooklib + BeautifulSoup
- [ ] Handle errors (DRM, corrupt files) with skip + warning message

## 2. Go Changes

- [ ] Add `.epub` to `SupportedFileExtensions` in `src/internal/domain/config.go`

## 3. Verification

- [ ] Test with a sample EPUB file
- [ ] Verify chunks are created and upserted to ChromaDB
- [ ] Run `go build ./cmd/zolam/` — verify build succeeds
- [ ] Run `go test ./...` — verify all tests pass
