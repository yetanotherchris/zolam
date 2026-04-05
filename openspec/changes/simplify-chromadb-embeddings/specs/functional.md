# Functional Spec: Simplify ChromaDB Embeddings

## Finding: No Changes Required

### Constraint 1: Embeddings must be computed client-side
- **WHEN** documents are sent to ChromaDB via `POST /api/v1/collections/{id}/upsert`
- **THEN** the `embeddings` field must contain pre-computed vectors
- **BECAUSE** the ChromaDB server never generates embeddings

### Constraint 2: Python embedding dependencies are required
- **WHEN** the ingest container runs
- **THEN** `onnxruntime` and `tokenizers` must be installed
- **AND** the all-MiniLM-L6-v2 model must be available at `/root/.cache/chroma/`
- **BECAUSE** the `chromadb` Python client calls the embedding function locally before HTTP calls

### Constraint 3: Python text extraction is preferred
- **WHEN** comparing Go vs Python PDF/DOCX extraction libraries
- **THEN** PyMuPDF (Python) provides significantly better quality than Go alternatives
- **AND** python-docx provides better DOCX handling than available Go libraries
