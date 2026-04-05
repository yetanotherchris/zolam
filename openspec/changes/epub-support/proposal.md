# Proposal: Add EPUB File Support

## Why

Zolam currently supports markdown, PDF, DOCX, plain text, and code files. EPUB is a common format for ebooks and technical documentation. Adding EPUB support allows users to ingest their ebook collections for semantic search.

## What Changes

- **Add** `ebooklib` to Dockerfile pip dependencies
- **Add** EPUB extraction case in `ingest.py` `extract_text()` function
- **Add** `.epub` to `SUPPORTED_EXTENSIONS` in `ingest.py`
- **Add** `.epub` to `SupportedFileExtensions` in `src/internal/domain/config.go`

## Impact

- Users can ingest `.epub` files alongside existing supported formats
- Minimal Docker image size increase (ebooklib is lightweight)
- No changes to chunking, embedding, or ChromaDB interaction
