## ADDED Requirements

### Requirement: Multi-stage Docker build
The Dockerfile SHALL use a multi-stage build with a builder stage for pip installation and a runtime stage containing only the virtual environment, embedding model cache, and application code.

#### Scenario: Builder stage installs dependencies
- **WHEN** the Docker image is built
- **THEN** pip packages are installed in an isolated virtual environment in the builder stage

#### Scenario: Runtime stage contains only runtime files
- **WHEN** the final image is produced
- **THEN** it SHALL contain only the Python runtime, the virtual environment from the builder stage, the embedding model cache, and `ingest.py`

### Requirement: Use chromadb-client package
The Dockerfile SHALL install `chromadb-client` instead of the full `chromadb` package, with `onnxruntime` and `tokenizers` as explicit dependencies for client-side embedding.

#### Scenario: Ingest connects to remote ChromaDB
- **WHEN** the ingest container runs with `chromadb-client` installed
- **THEN** `ingest.py` SHALL successfully connect to the ChromaDB server using `chromadb.HttpClient` and perform all collection operations (get, create, upsert, delete, count)

#### Scenario: Client-side embedding works
- **WHEN** the ingest container generates embeddings
- **THEN** the ONNXMiniLM_L6_V2 embedding function SHALL produce valid 384-dimension vectors

### Requirement: Reduced image size
The final Docker image SHALL be smaller than the current 1 GB image.

#### Scenario: Image size reduced
- **WHEN** the Docker image is built and inspected with `docker images`
- **THEN** the reported size SHALL be under 800 MB
