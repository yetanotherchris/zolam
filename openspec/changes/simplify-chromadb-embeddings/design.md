# Design: Simplify ChromaDB Embeddings

## Context

Investigation into simplifying the ChromaDB ingestion pipeline revealed that the original premise — that ChromaDB server generates embeddings — was incorrect.

## How ChromaDB Embeddings Actually Work

1. The Python `chromadb` client library assigns a default `EmbeddingFunction` (ONNXMiniLM_L6_V2) to each collection
2. When `collection.upsert(documents=...)` is called, the client library:
   - Calls `embedding_function(documents)` locally to compute 384-dim vectors
   - POSTs the computed `embeddings` + `documents` + `ids` + `metadatas` to the server
3. The ChromaDB server stores and indexes vectors — it never generates them
4. The REST API endpoint `POST /api/v1/collections/{id}/upsert` expects pre-computed embeddings

## Implications

- `onnxruntime` and `tokenizers` are required in the Docker image — they power the embedding function
- The pre-downloaded model (`all-MiniLM-L6-v2`) is required — it's used at runtime by the embedding function
- Removing these dependencies would break ingestion
- The `chroma-mcp` server used by Claude must also compute embeddings client-side using the same model

## Why Python Stays

- PyMuPDF has best-in-class PDF text extraction (handles complex layouts, encodings, scanned docs)
- python-docx is mature and handles all DOCX edge cases
- Go PDF libraries (ledongthuc/pdf) are adequate for simple PDFs but significantly inferior for complex documents
- The embedding function (onnxruntime + all-MiniLM-L6-v2) must run client-side, and the Python chromadb library handles this transparently

## Decisions

No architectural changes. The current design is correct given the constraints. Document findings for future reference.
