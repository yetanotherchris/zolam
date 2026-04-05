# Tasks: Simplify ChromaDB Embeddings

## Findings

- [x] Investigate whether ChromaDB server generates embeddings — Result: NO, client-side only
- [x] Investigate Go PDF/DOCX libraries as Python replacements — Result: inferior quality
- [x] Investigate ChromaDB REST API endpoints — Result: upsert requires pre-computed embeddings
- [x] Revert incorrect commit that removed onnxruntime/tokenizers from Dockerfile

## No-op

No implementation changes needed. The current architecture is correct:
- Python client computes embeddings client-side (required)
- PyMuPDF/python-docx provide superior text extraction (preferred)
- ChromaDB server stores and queries vectors (unchanged)
