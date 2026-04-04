# Functional Spec: Local Embeddings

## Requirements

1. The ingester uses ChromaDB's default embedding function (all-MiniLM-L6-v2, 384 dimensions)
2. No API key is required for embedding generation
3. The Docker image contains the pre-downloaded model, no network access needed at runtime
4. The chroma-mcp server and ingester use the same default model, eliminating dimension mismatches
5. All existing operations (ingest, stats, reset, update) work without embedding configuration
6. The `OPENROUTER_API_KEY`, `OPENROUTER_MODEL`, and `USE_LOCAL_EMBEDDINGS` env vars are removed
7. Existing collections with 1536-dim embeddings must be reset and re-ingested after this change
