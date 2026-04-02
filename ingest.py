"""
ChromaDB Vault Ingestion Script (Docker CLI)

Ingests markdown, PDF, and DOCX files into ChromaDB for semantic search
via the chroma-mcp server.

Default: uses sentence-transformers locally (no API keys needed).
Optional: set OPENROUTER_API_KEY env var for better embeddings.

Usage (Docker):
    docker run -v C:\chromadb\data:/data -v C:\obsidian:/sources/obsidian ingest --source obsidian
    docker run -v C:\chromadb\data:/data -v C:\gdrive:/sources/gdrive ingest --source gdrive
    docker run -v C:\chromadb\data:/data -v ... ingest --reset
    docker run -v C:\chromadb\data:/data ingest --stats

Mount points:
    /data              -> ChromaDB persistent storage
    /sources/obsidian  -> Obsidian vault
    /sources/gdrive    -> rclone'd Google Drive
    /sources/repos     -> GitHub repos

Environment variables:
    OPENROUTER_API_KEY -> (optional) use OpenRouter for embeddings
    OPENROUTER_MODEL   -> (optional) embedding model, default: openai/text-embedding-3-small
"""

import argparse
import hashlib
import os
from pathlib import Path

import chromadb

# ---------------------------------------------------------------------------
# CONFIG
# ---------------------------------------------------------------------------

CHROMA_DATA_DIR = "/data"

SOURCES = {
    "obsidian": {
        "path": "/sources/obsidian",
        "extensions": [".md"],
    },
    "gdrive": {
        "path": "/sources/gdrive",
        "extensions": [".md", ".pdf", ".docx", ".txt"],
    },
    "repos": {
        "path": "/sources/repos",
        "extensions": [".md", ".py", ".cs", ".js", ".ts", ".json", ".yml", ".yaml"],
    },
}

CHUNK_SIZE = 2000
CHUNK_OVERLAP = 200

# ---------------------------------------------------------------------------
# EMBEDDING FUNCTION
# ---------------------------------------------------------------------------

def get_embedding_function():
    """Use OpenRouter if key is set, otherwise default sentence-transformers."""
    api_key = os.environ.get("OPENROUTER_API_KEY")
    if api_key:
        from chromadb.utils.embedding_functions import OpenAIEmbeddingFunction
        model = os.environ.get("OPENROUTER_MODEL", "openai/text-embedding-3-small")
        print(f"Using OpenRouter embeddings: {model}")
        return OpenAIEmbeddingFunction(
            api_key=api_key,
            api_base="https://openrouter.ai/api/v1",
            model_name=model,
        )
    else:
        print("Using default local embeddings (all-MiniLM-L6-v2)")
        return None  # ChromaDB default


# ---------------------------------------------------------------------------
# TEXT EXTRACTION
# ---------------------------------------------------------------------------

def extract_text(filepath: Path) -> str | None:
    """Extract text from a file based on its extension."""
    ext = filepath.suffix.lower()

    if ext in (".md", ".txt", ".py", ".cs", ".js", ".ts", ".json", ".yml", ".yaml"):
        try:
            return filepath.read_text(encoding="utf-8", errors="replace")
        except Exception as e:
            print(f"  SKIP {filepath}: {e}")
            return None

    if ext == ".pdf":
        try:
            import fitz  # pymupdf
            doc = fitz.open(str(filepath))
            text = "\n".join(page.get_text() for page in doc)
            doc.close()
            return text if text.strip() else None
        except Exception as e:
            print(f"  SKIP {filepath}: {e}")
            return None

    if ext == ".docx":
        try:
            from docx import Document
            doc = Document(str(filepath))
            text = "\n".join(p.text for p in doc.paragraphs)
            return text if text.strip() else None
        except Exception as e:
            print(f"  SKIP {filepath}: {e}")
            return None

    return None


# ---------------------------------------------------------------------------
# CHUNKING
# ---------------------------------------------------------------------------

def chunk_text(text: str, chunk_size: int = CHUNK_SIZE, overlap: int = CHUNK_OVERLAP) -> list[str]:
    """Split text into overlapping chunks. Returns whole text if short enough."""
    if len(text) <= chunk_size:
        return [text]

    chunks = []
    start = 0
    while start < len(text):
        end = start + chunk_size
        chunk = text[start:end]
        if chunk.strip():
            chunks.append(chunk)
        start = end - overlap

    return chunks


def make_chunk_id(source: str, filepath: str, chunk_index: int) -> str:
    """Deterministic ID for a chunk so re-ingestion is idempotent."""
    raw = f"{source}:{filepath}:{chunk_index}"
    return hashlib.sha256(raw.encode()).hexdigest()[:16]


# ---------------------------------------------------------------------------
# INGESTION
# ---------------------------------------------------------------------------

def ingest_source(collection, source_name: str, source_config: dict) -> int:
    """Ingest all matching files from a source directory. Returns chunk count."""
    base_path = Path(source_config["path"])
    extensions = source_config["extensions"]

    if not base_path.exists():
        print(f"  WARNING: {base_path} not mounted, skipping.")
        return 0

    count = 0
    files = [f for ext in extensions for f in base_path.rglob(f"*{ext}")]
    print(f"  Found {len(files)} files")

    for filepath in files:
        text = extract_text(filepath)
        if not text:
            continue

        relative = str(filepath.relative_to(base_path))
        chunks = chunk_text(text)

        ids = [make_chunk_id(source_name, relative, i) for i in range(len(chunks))]
        metadatas = [
            {
                "source": source_name,
                "file": relative,
                "chunk": i,
                "total_chunks": len(chunks),
            }
            for i in range(len(chunks))
        ]

        batch_size = 50
        for b in range(0, len(chunks), batch_size):
            collection.upsert(
                documents=chunks[b:b + batch_size],
                ids=ids[b:b + batch_size],
                metadatas=metadatas[b:b + batch_size],
            )

        count += len(chunks)

    return count


def main():
    parser = argparse.ArgumentParser(description="Ingest files into ChromaDB")
    parser.add_argument("--source", choices=list(SOURCES.keys()), help="Ingest only this source")
    parser.add_argument("--reset", action="store_true", help="Delete collection and re-ingest")
    parser.add_argument("--stats", action="store_true", help="Show collection stats and exit")
    args = parser.parse_args()

    client = chromadb.PersistentClient(path=CHROMA_DATA_DIR)

    if args.stats:
        try:
            collection = client.get_collection("vault")
            print(f"Collection 'vault' has {collection.count()} documents.")
        except Exception:
            print("No 'vault' collection found.")
        return

    if args.reset:
        try:
            client.delete_collection("vault")
            print("Deleted existing collection.")
        except Exception:
            pass

    ef = get_embedding_function()
    kwargs = {"name": "vault"}
    if ef:
        kwargs["embedding_function"] = ef
    collection = client.get_or_create_collection(**kwargs)

    sources = {args.source: SOURCES[args.source]} if args.source else SOURCES
    total = 0

    for name, config in sources.items():
        print(f"\nIngesting [{name}] from {config['path']}")
        count = ingest_source(collection, name, config)
        print(f"  Ingested {count} chunks")
        total += count

    print(f"\nDone. Total chunks ingested: {total}")
    print(f"Collection now has {collection.count()} documents.")


if __name__ == "__main__":
    main()
