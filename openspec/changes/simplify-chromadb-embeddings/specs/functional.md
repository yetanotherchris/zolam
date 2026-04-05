# Functional Spec: Simplify ChromaDB Embeddings

## Requirement 1: ChromaDB REST API Client

### Scenario: Health check
- **WHEN** `Heartbeat()` is called
- **THEN** it SHALL return nil error if ChromaDB responds with HTTP 200 at `/api/v1/heartbeat`

### Scenario: Create or get collection
- **WHEN** `GetOrCreateCollection(name)` is called
- **THEN** it SHALL return the collection (with its server-assigned ID) or create it if it doesn't exist

### Scenario: Delete collection
- **WHEN** `DeleteCollection(name)` is called
- **THEN** it SHALL delete the collection from ChromaDB

### Scenario: Upsert documents
- **WHEN** `Upsert(collectionID, documents, ids, metadatas)` is called
- **THEN** it SHALL POST documents to `/api/v1/collections/{id}/upsert`
- **AND** ChromaDB SHALL generate embeddings server-side
- **AND** batch size SHALL NOT exceed 50 documents per request

### Scenario: Count documents
- **WHEN** `Count(collectionID)` is called
- **THEN** it SHALL return the number of documents in the collection

## Requirement 2: Text Extraction

### Scenario: Plain text files
- **WHEN** a file with extension .md, .txt, .py, .cs, .js, .ts, .json, .yml, or .yaml is processed
- **THEN** its contents SHALL be read as UTF-8 text

### Scenario: PDF files
- **WHEN** a .pdf file is processed
- **THEN** text SHALL be extracted from all pages
- **AND** empty PDFs SHALL be skipped

### Scenario: DOCX files
- **WHEN** a .docx file is processed
- **THEN** paragraph text SHALL be extracted
- **AND** empty documents SHALL be skipped

### Scenario: Unsupported files
- **WHEN** a file with an unsupported extension is encountered
- **THEN** it SHALL be skipped without error

## Requirement 3: Text Chunking

### Scenario: Short text
- **WHEN** text is shorter than or equal to 2000 characters
- **THEN** it SHALL be returned as a single chunk

### Scenario: Long text
- **WHEN** text is longer than 2000 characters
- **THEN** it SHALL be split into chunks of 2000 characters with 200 character overlap
- **AND** empty chunks SHALL be excluded

## Requirement 4: Deterministic IDs

### Scenario: Chunk ID generation
- **WHEN** a chunk is created from source `S`, file path `F`, and index `I`
- **THEN** its ID SHALL be the first 16 hex characters of SHA256(`S:F:I`)
- **AND** re-ingesting the same file SHALL produce the same IDs (idempotent upsert)

## Requirement 5: Metadata Schema

### Scenario: Document metadata
- **WHEN** a chunk is upserted to ChromaDB
- **THEN** its metadata SHALL include: `source` (directory name), `file` (relative path), `chunk` (index), `total_chunks` (count)

## Requirement 6: CLI/TUI Interface

### Scenario: Ingest command
- **WHEN** `zolam ingest [directories...]` is run
- **THEN** it SHALL ingest files directly via HTTP (no Docker ingest container)
- **AND** progress output SHALL be sent via `outputFn` callback

### Scenario: Stats command
- **WHEN** `zolam stats` is run
- **THEN** it SHALL query ChromaDB directly via HTTP for collection count

### Scenario: Reset command
- **WHEN** `zolam reset` is run
- **THEN** it SHALL delete the collection via HTTP
