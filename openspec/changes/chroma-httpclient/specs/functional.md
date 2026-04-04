# Functional Spec: ChromaDB HttpClient

## Requirements

1. The ingester connects to ChromaDB via HTTP using `chromadb.HttpClient`
2. `CHROMA_HOST` environment variable sets the server hostname (default: `chromadb`)
3. `CHROMA_PORT` environment variable sets the server port (default: `8000`)
4. The ingester no longer needs direct access to the ChromaDB data directory
5. All existing operations (ingest, stats, reset) work identically over HTTP
6. The `depends_on: chromadb` relationship ensures the server is available before ingestion starts
