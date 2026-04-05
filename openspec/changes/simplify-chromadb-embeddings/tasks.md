# Tasks: Simplify ChromaDB Embeddings

## Investigation

- [x] Investigate whether ChromaDB server generates embeddings — Result: NO, client-side only
- [x] Investigate Go PDF/DOCX libraries as Python replacements — Result: inferior quality
- [x] Investigate ChromaDB REST API endpoints — Result: upsert requires pre-computed embeddings
- [x] Verify findings against ChromaDB source code — Confirmed: zero server-side embedding logic
- [x] Revert incorrect commit that removed onnxruntime/tokenizers from Dockerfile
- [x] Document findings in OpenSpec
