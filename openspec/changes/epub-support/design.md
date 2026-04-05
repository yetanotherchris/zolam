# Design: Add EPUB File Support

## Context

EPUB files are ZIP archives containing XHTML content, metadata, and optional assets (images, CSS). Text extraction requires parsing the XHTML documents within the archive.

## Approach

Use `ebooklib` (Python) — the standard library for EPUB parsing. It reads the EPUB structure and provides access to individual chapters/documents.

### Text Extraction Flow

1. Open EPUB with `epub.read_epub(filepath)`
2. Iterate over items of type `EpubHtml` (content documents)
3. Parse HTML content with `BeautifulSoup` to extract plain text
4. Concatenate all chapter text with newline separators

### Dependencies

- `ebooklib` — EPUB parsing
- `beautifulsoup4` + `lxml` — HTML-to-text extraction from EPUB chapters

Note: `ebooklib` returns chapter content as raw HTML/XHTML. `BeautifulSoup` is needed to strip tags and extract clean text.

### Integration Points

- `ingest.py:extract_text()` — Add `.epub` case alongside existing `.pdf` and `.docx` handlers
- `ingest.py:SUPPORTED_EXTENSIONS` — Add `.epub`
- `Dockerfile` — Add `ebooklib`, `beautifulsoup4`, `lxml` to pip install
- `src/internal/domain/config.go` — Add `.epub` to `SupportedFileExtensions`

## Risks

- Large EPUB files (full books) may produce many chunks. This is consistent with current behavior for large PDFs.
- Some EPUBs contain DRM — `ebooklib` cannot read DRM-protected files. These will be skipped with an error message, matching existing behavior for unreadable files.
