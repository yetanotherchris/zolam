# Proposal: Simplify ChromaDB Embeddings — Keep Python, Explore Options

## Why

The current architecture uses a separate Python Docker container (`ingest.py`) to extract text, chunk it, compute embeddings client-side (all-MiniLM-L6-v2 via onnxruntime), and send vectors to ChromaDB. This results in a ~750MB+ Docker image.

Initial investigation revealed that **ChromaDB does NOT generate embeddings server-side** — the Python `chromadb` client library computes them locally before sending vectors via HTTP. This means `onnxruntime` and `tokenizers` cannot simply be removed.

However, Python remains the right choice for text extraction — PyMuPDF and python-docx are significantly superior to Go alternatives for PDF/DOCX parsing quality.

## Options Investigated

1. **"Let the server embed"** — INVALID. ChromaDB server does not compute embeddings. The Python client does it client-side before making HTTP calls.
2. **Eliminate Python entirely (Go + HTTP)** — Would require computing embeddings in Go (hard) or calling an external embedding API. Go PDF/DOCX libraries are inferior to Python's.
3. **Keep Python for extraction + embeddings** — Current approach. The Dockerfile size is driven by onnxruntime + model weights, which are required.

## What Changes

Given the constraints, the scope is reduced to documentation corrections and minor cleanup:

- **Correct** `CLAUDE.md` to clarify that embeddings are computed by the Python client library, not by the ChromaDB server
- **Update** OpenSpec to document findings for future reference

## Key Finding

The `chromadb.HttpClient` Python class wraps HTTP calls but the `Collection` object runs the embedding function **client-side** in `Collection.upsert()` before POSTing vectors to the server. A raw HTTP client (Go or otherwise) sending `documents` without `embeddings` will fail.
