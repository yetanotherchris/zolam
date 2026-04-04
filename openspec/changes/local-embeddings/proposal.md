# Proposal: Switch to local sentence-transformers embeddings

## Motivation

The ingester currently supports two embedding modes: OpenRouter (OpenAI text-embedding-3-small, 1536 dims) and local sentence-transformers (all-MiniLM-L6-v2, 384 dims). The chroma-mcp server used by Claude defaults to the local model. This causes a dimension mismatch (1536 vs 384) when querying via Claude.

Rather than configuring chroma-mcp to use OpenAI (which it doesn't easily support), remove the OpenRouter dependency entirely. For ~1000 personal notes, local embeddings are sufficient and eliminate the API key requirement.

## Scope

- Remove `OPENROUTER_API_KEY`, `OPENROUTER_MODEL`, `USE_LOCAL_EMBEDDINGS` env vars from all code
- Remove `get_embedding_function()` from `ingest.py`, use ChromaDB's default embedding (all-MiniLM-L6-v2)
- Remove `openai` pip dependency from Dockerfile
- Pre-download the sentence-transformers model in the Docker image so it's available at runtime without network access
- Remove embedding-related fields from Go config, TUI, CLI, stats, and tests
- Remove embedding env vars from both docker-compose files
- Update README: remove OpenRouter references, update chroma-mcp command to use HTTP client
