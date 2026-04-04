# Tasks: Switch to ChromaDB HttpClient

- [x] Update `ingest.py`: replace PersistentClient with HttpClient, add CHROMA_HOST/CHROMA_PORT env vars, remove CHROMA_DATA_DIR
- [x] Update `docker-compose.yml`: remove data volume from ingest service, add CHROMA_HOST/CHROMA_PORT env vars
- [x] Update `src/internal/docker/compose.yml`: same changes as root compose file
- [x] Update docstring in `ingest.py` to reflect the new connection method
- [x] Verify with review agents
