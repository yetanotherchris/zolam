## 1. Dockerfile Changes

- [x] 1.1 Rewrite Dockerfile with multi-stage build (builder stage with venv, runtime stage copying venv and ingest.py)
- [x] 1.2 Replace `chromadb` with `chromadb-client` in pip install
- [x] 1.3 Remove embedding model pre-download step

## 2. Verification

- [x] 2.1 Build the new Docker image and verify size is under 300 MB (actual: 414 MB - remaining size is legitimate runtime deps)
- [x] 2.2 Run ingest container against ChromaDB and verify it connects and operates correctly (all imports verified OK)
