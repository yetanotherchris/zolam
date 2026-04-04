## ADDED Requirements

### Requirement: Multi-stage Docker build
The Dockerfile SHALL use a multi-stage build with a builder stage for pip installation and a runtime stage containing only the virtual environment and application code.

#### Scenario: Builder stage installs dependencies
- **WHEN** the Docker image is built
- **THEN** pip packages are installed in an isolated virtual environment in the builder stage

#### Scenario: Runtime stage contains only runtime files
- **WHEN** the final image is produced
- **THEN** it SHALL contain only the Python runtime, the virtual environment from the builder stage, and `ingest.py`

### Requirement: Use chromadb-client package
The Dockerfile SHALL install `chromadb-client` instead of the full `chromadb` package.

#### Scenario: Ingest connects to remote ChromaDB
- **WHEN** the ingest container runs with `chromadb-client` installed
- **THEN** `ingest.py` SHALL successfully connect to the ChromaDB server using `chromadb.HttpClient` and perform all collection operations (get, create, upsert, delete, count)

### Requirement: No local embedding model
The Dockerfile SHALL NOT pre-download or bake in the embedding model, as embeddings are handled by the ChromaDB server.

#### Scenario: No embedding model in image
- **WHEN** the Docker image is inspected
- **THEN** there SHALL be no model files in `/root/.cache/chroma/`

### Requirement: Reduced image size
The final Docker image SHALL be significantly smaller than the current 1 GB image.

#### Scenario: Image size under 300 MB
- **WHEN** the Docker image is built and inspected with `docker images`
- **THEN** the reported size SHALL be under 300 MB
