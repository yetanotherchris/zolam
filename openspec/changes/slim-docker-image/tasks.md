## 1. Dockerfile Changes

- [x] 1.1 Rewrite Dockerfile with multi-stage build (builder stage with venv, runtime stage copying venv, model cache, and ingest.py)
- [x] 1.2 Replace `chromadb` with `chromadb-client` plus explicit `onnxruntime` and `tokenizers` dependencies
- [x] 1.3 Keep embedding model pre-download step (embeddings run client-side)

## 2. Verification

- [x] 2.1 Build the new Docker image and verify size is reduced (result: 757 MB, down from 1.01 GB)
- [x] 2.2 Verify all ingest.py imports work (chromadb, fitz, docx, tqdm)
- [x] 2.3 Verify client-side embedding function produces valid vectors
