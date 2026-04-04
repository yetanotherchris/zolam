"""
ChromaDB Ingestion Script (Docker CLI)

Ingests markdown, PDF, and DOCX files into ChromaDB for semantic search
via the chroma-mcp server. Connects to ChromaDB over HTTP using HttpClient.

Usage (Docker Compose):
    docker compose --profile ingest run --rm \
      -v /path/to/notes:/sources/notes \
      ingest --directory /sources/notes

    docker compose --profile ingest run --rm \
      -v /path/to/notes:/sources/notes \
      -v /path/to/docs:/sources/docs \
      ingest --directory /sources/notes /sources/docs

    docker compose --profile ingest run --rm ingest --stats
    docker compose --profile ingest run --rm ingest --reset --directory /sources/notes

Environment variables:
    COLLECTION_NAME      -> ChromaDB collection name (default: my notes)
    CHROMA_HOST          -> ChromaDB server hostname (default: chromadb)
    CHROMA_PORT          -> ChromaDB server port (default: 8000)
    OPENROUTER_API_KEY   -> API key for OpenRouter embeddings (required unless local)
    OPENROUTER_MODEL     -> (optional) embedding model, default: openai/text-embedding-3-small
    USE_LOCAL_EMBEDDINGS -> set to 1 to use local sentence-transformers instead
"""

import argparse
import hashlib
import os
import re
from pathlib import Path

import chromadb
from tqdm import tqdm

# ---------------------------------------------------------------------------
# CONFIG
# ---------------------------------------------------------------------------

CHROMA_HOST = os.environ.get("CHROMA_HOST", "chromadb")
CHROMA_PORT = int(os.environ.get("CHROMA_PORT", "8000"))

SUPPORTED_EXTENSIONS = [".md", ".pdf", ".docx", ".txt", ".py", ".cs", ".js", ".ts", ".json", ".yml", ".yaml"]

CHUNK_SIZE = 2000
CHUNK_OVERLAP = 200

# ---------------------------------------------------------------------------
# EMBEDDING FUNCTION
# ---------------------------------------------------------------------------

def get_embedding_function():
    """Use OpenRouter by default, or local sentence-transformers if USE_LOCAL_EMBEDDINGS is set."""
    if os.environ.get("USE_LOCAL_EMBEDDINGS"):
        print("Using local embeddings (all-MiniLM-L6-v2)")
        return None  # ChromaDB default

    api_key = os.environ.get("OPENROUTER_API_KEY")
    if not api_key:
        raise SystemExit(
            "Error: OPENROUTER_API_KEY is required. "
            "Set USE_LOCAL_EMBEDDINGS=1 to use local sentence-transformers instead."
        )

    from chromadb.utils.embedding_functions import OpenAIEmbeddingFunction
    model = os.environ.get("OPENROUTER_MODEL") or "openai/text-embedding-3-small"
    print(f"Using OpenRouter embeddings: {model}")
    return OpenAIEmbeddingFunction(
        api_key=api_key,
        api_base="https://openrouter.ai/api/v1",
        model_name=model,
    )


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
            tqdm.write(f"  SKIP {filepath}: {e}")
            return None

    if ext == ".pdf":
        try:
            import fitz  # pymupdf
            doc = fitz.open(str(filepath))
            text = "\n".join(page.get_text() for page in doc)
            doc.close()
            return text if text.strip() else None
        except Exception as e:
            tqdm.write(f"  SKIP {filepath}: {e}")
            return None

    if ext == ".docx":
        try:
            from docx import Document
            doc = Document(str(filepath))
            text = "\n".join(p.text for p in doc.paragraphs)
            return text if text.strip() else None
        except Exception as e:
            tqdm.write(f"  SKIP {filepath}: {e}")
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
    files = []
    for ext in tqdm(extensions, desc="  Scanning extensions", unit="ext", leave=False):
        files.extend(base_path.rglob(f"*{ext}"))
    print(f"  Found {len(files)} files")

    for filepath in tqdm(files, desc="  Ingesting files", unit="file"):
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
        try:
            for b in range(0, len(chunks), batch_size):
                collection.upsert(
                    documents=chunks[b:b + batch_size],
                    ids=ids[b:b + batch_size],
                    metadatas=metadatas[b:b + batch_size],
                )
        except Exception as e:
            tqdm.write(f"  ERROR upserting {relative}: {e}")
            continue

        count += len(chunks)

    return count


def main():
    parser = argparse.ArgumentParser(description="Ingest files into ChromaDB")
    parser.add_argument("--directory", nargs="+", metavar="DIR",
                        help="One or more directories to ingest (recursively)")
    parser.add_argument("--extensions", nargs="+", metavar="EXT",
                        help="File extensions to include (e.g. .md .txt .pdf). "
                             "Default: all supported extensions")
    parser.add_argument("--reset", action="store_true", help="Delete collection and re-ingest")
    parser.add_argument("--stats", action="store_true", help="Show collection stats and exit")
    args = parser.parse_args()

    collection_name = os.environ.get("COLLECTION_NAME", "my-notes")
    # Sanitize: replace invalid chars with hyphens, strip leading/trailing hyphens
    collection_name = re.sub(r'[^a-zA-Z0-9._-]', '-', collection_name).strip('-')
    client = chromadb.HttpClient(host=CHROMA_HOST, port=CHROMA_PORT)
    ef = get_embedding_function()

    if args.stats:
        try:
            collection = client.get_collection(collection_name)
            print(f"Collection '{collection_name}' has {collection.count()} documents.")
        except Exception:
            print(f"No '{collection_name}' collection found.")
        return

    if not args.directory and not args.reset:
        parser.error("--directory is required (unless using --stats or --reset)")

    if args.reset:
        try:
            client.delete_collection(collection_name)
            print(f"Deleted existing collection '{collection_name}'.")
        except Exception:
            pass
        if not args.directory:
            print("Collection reset complete.")
            return

    extensions = args.extensions if args.extensions else SUPPORTED_EXTENSIONS
    extensions = [ext if ext.startswith(".") else f".{ext}" for ext in extensions]

    kwargs = {"name": collection_name}
    if ef:
        kwargs["embedding_function"] = ef
    collection = client.get_or_create_collection(**kwargs)

    total = 0
    for dir_path in args.directory:
        resolved = Path(dir_path).resolve()
        source_label = resolved.name
        config = {"path": str(resolved), "extensions": extensions}
        print(f"\nIngesting [{source_label}] into collection '{collection_name}' from {resolved}")
        count = ingest_source(collection, source_label, config)
        print(f"  Ingested {count} chunks")
        total += count

    print(f"\nDone. Total chunks ingested: {total}")


if __name__ == "__main__":
    main()
