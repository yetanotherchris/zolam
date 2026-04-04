# Design: Switch to local embeddings

## Changes

### Python ingester (`ingest.py`)
- Remove `get_embedding_function()` entirely
- Remove all OpenRouter/embedding env var references from docstring and code
- Pass no `embedding_function` to `get_or_create_collection()` (ChromaDB defaults to all-MiniLM-L6-v2)

### Dockerfile
- Remove `openai` from pip install
- Add a build step that pre-downloads the all-MiniLM-L6-v2 model into the image
- The model is Apache 2.0 licensed and can be freely distributed

### Docker Compose files (`docker-compose.yml`, `src/internal/docker/compose.yml`)
- Remove `OPENROUTER_API_KEY`, `OPENROUTER_MODEL`, `USE_LOCAL_EMBEDDINGS` env vars from ingest service

### Go config (`src/internal/domain/config.go`)
- Remove `OpenRouterAPIKey`, `OpenRouterModel`, `UseLocalEmbeddings` from Config struct
- Remove related MergeFlags handling
- Remove API key validation (no longer needed)
- Remove OpenRouter model warning from Validate()

### Go stats (`src/internal/zolam/stats.go`)
- Remove `EmbeddingType` field and conditional logic

### Go TUI (`src/internal/tui/app.go`)
- Remove OpenRouter model, local embeddings, and API key from settings view
- Remove embedding type from stats summary

### Go CLI (`src/cmd/zolam/main.go`)
- Remove embedding display from stats and config commands

### Tests (`src/internal/domain/config_test.go`)
- Remove embedding-related test cases and assertions
- Simplify default config test (no longer needs USE_LOCAL_EMBEDDINGS=1 to pass)

### README
- Remove OpenRouter/embedding prerequisites and env vars
- Update chroma-mcp command to use HTTP client: `claude mcp add chroma -- uvx chroma-mcp --client-type http --host localhost --port 8000`
