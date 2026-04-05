# Functional Spec: Simplify ChromaDB Embeddings

## Finding: No Changes Required

Investigation determined that the proposed simplification is not feasible:

### Requirement 1: Embeddings must be computed client-side
- **WHEN** documents are sent to ChromaDB via `POST /api/v1/collections/{id}/upsert`
- **THEN** the `embeddings` field must contain pre-computed vectors
- **AND** the ChromaDB server will NOT generate embeddings from `documents` alone

### Requirement 2: Python embedding dependencies are required
- **WHEN** the ingest container runs
- **THEN** `onnxruntime` and `tokenizers` must be installed
- **AND** the all-MiniLM-L6-v2 model must be available at `/root/.cache/chroma/`
- **BECAUSE** the `chromadb` Python client calls the embedding function locally before HTTP calls

### Requirement 3: Python text extraction is preferred
- **WHEN** comparing Go vs Python PDF extraction libraries
- **THEN** PyMuPDF (Python) provides significantly better quality than ledongthuc/pdf (Go)
- **AND** python-docx provides better DOCX handling than available Go libraries
