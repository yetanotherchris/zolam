# Tasks: Switch to local embeddings

- [x] Update `ingest.py`: remove `get_embedding_function()`, OpenRouter imports, embedding env vars from docstring
- [x] Update `Dockerfile`: remove `openai` dep, add model pre-download step
- [x] Update `docker-compose.yml`: remove embedding env vars from ingest service
- [x] Update `src/internal/docker/compose.yml`: remove embedding env vars from ingest service
- [x] Update `src/internal/domain/config.go`: remove embedding fields, MergeFlags, validation
- [x] Update `src/internal/domain/config_test.go`: remove embedding-related tests and assertions
- [x] Update `src/internal/zolam/stats.go`: remove EmbeddingType field and logic
- [x] Update `src/internal/tui/app.go`: remove embedding display from settings and stats views
- [x] Update `src/cmd/zolam/main.go`: remove embedding display from stats and config commands
- [x] Update `README.md`: remove OpenRouter references, update chroma-mcp command
- [x] Verify with review agents
