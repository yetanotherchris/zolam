# Design: Simplify ChromaDB Embeddings

## Context

Investigated simplifying the ingestion pipeline by offloading embedding generation to the ChromaDB server.

## How ChromaDB Embeddings Actually Work

1. The Python `chromadb` client assigns a default `EmbeddingFunction` (ONNXMiniLM_L6_V2) to each collection
2. When `collection.upsert(documents=...)` is called, the client:
   - Calls `embedding_function(documents)` locally to compute 384-dim vectors
   - POSTs the computed `embeddings` + `documents` + `ids` + `metadatas` to the server
3. The ChromaDB server stores and indexes vectors — it never generates them
4. The REST API endpoint `POST /api/v1/collections/{id}/upsert` expects pre-computed embeddings
5. Sending documents without embeddings results in `None` vectors — data is stored but not similarity-searchable

## Why Python Stays

- Embeddings must be computed client-side (ChromaDB architectural constraint)
- PyMuPDF has best-in-class PDF text extraction
- python-docx is mature for DOCX handling
- Go PDF libraries (ledongthuc/pdf) are significantly inferior for complex documents

## Decision

No architectural changes. Current design is correct given ChromaDB's constraints.
