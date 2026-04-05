# Proposal: Simplify ChromaDB Embeddings — Investigation

## Why

Investigated whether the ChromaDB server could generate embeddings server-side, which would allow removing onnxruntime, tokenizers, and the pre-downloaded model from the Python ingest container — significantly reducing Docker image size.

## Finding

**ChromaDB server does NOT generate embeddings.** The Python `chromadb` client library computes embeddings client-side using `ONNXMiniLM_L6_V2` before sending vectors via HTTP. The server only stores and queries pre-computed vectors.

Evidence from ChromaDB source code:
- Server API accepts `embeddings: Optional` but never generates them
- Zero `embedding_function` or `generate_embedding` references in `chromadb/server/`
- Embedding generation happens in client-side `Collection._validate_and_prepare_add_request()`
- `SegmentAPI._add()` expects pre-computed `Embeddings` as a required parameter

## Outcome

No changes. The current architecture is correct:
- Python client computes embeddings client-side (required by ChromaDB design)
- onnxruntime + tokenizers + model weights must remain in the Docker image
- PyMuPDF and python-docx provide superior text extraction vs Go alternatives
