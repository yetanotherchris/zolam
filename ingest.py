"""
ChromaDB Vault Ingestion Script (Docker CLI)

Ingests markdown, PDF, and DOCX files into ChromaDB for semantic search
via the chroma-mcp server.

Default: uses OpenRouter for embeddings (requires OPENROUTER_API_KEY).
Optional: set USE_LOCAL_EMBEDDINGS=1 for offline sentence-transformers.

Usage (Docker):
    docker run -v /chromadb/data:/data -v /obsidian:/sources/obsidian ingest --source obsidian
    docker run -v /chromadb/data:/data -v /gdrive:/sources/gdrive ingest --source gdrive
    docker run -v /chromadb/data:/data -v /mydir:/sources/mydir ingest --directory /sources/mydir
    docker run -v /chromadb/data:/data -v ... ingest --directory /sources/dir1 /sources/dir2
    docker run -v /chromadb/data:/data -v ... ingest --directory /sources/mydir --extensions .md .txt
    docker run -v /chromadb/data:/data -v ... ingest --reset
    docker run -v /chromadb/data:/data ingest --stats

Mount points:
    /data              -> ChromaDB persistent storage
    /sources/obsidian  -> Obsidian vault
    /sources/gdrive    -> rclone'd Google Drive
    /sources/repos     -> GitHub repos

Environment variables:
    OPENROUTER_API_KEY   -> API key for OpenRouter embeddings (required unless local)
    OPENROUTER_MODEL     -> (optional) embedding model, default: openai/text-embedding-3-small
    USE_LOCAL_EMBEDDINGS -> set to 1 to use local sentence-transformers instead
"""

import argparse
import hashlib
import os
from pathlib import Path

import chromadb
from tqdm import tqdm

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
    parser.add_argument("--source", choices=list(SOURCES.keys()), help="Ingest only this source")
    parser.add_argument("--directory", nargs="+", metavar="DIR",
                        help="Ingest files from one or more directories (recursively)")
    parser.add_argument("--extensions", nargs="+", metavar="EXT",
                        help="File extensions to include with --directory (e.g. .md .txt .pdf). "
                             "Default: all supported extensions")
    parser.add_argument("--reset", action="store_true", help="Delete collection and re-ingest")
    parser.add_argument("--stats", action="store_true", help="Show collection stats and exit")
    args = parser.parse_args()

    client = chromadb.PersistentClient(path=CHROMA_DATA_DIR)

    ef = get_embedding_function()

    if args.stats:
        sources = {args.source: SOURCES[args.source]} if args.source else SOURCES
        for name in sources:
            try:
                collection = client.get_collection(name)
                print(f"Collection '{name}' has {collection.count()} documents.")
            except Exception:
                print(f"No '{name}' collection found.")
        return

    if args.directory:
        all_supported = list({ext for s in SOURCES.values() for ext in s["extensions"]})
        extensions = args.extensions if args.extensions else all_supported
        # Normalize extensions to include leading dot
        extensions = [ext if ext.startswith(".") else f".{ext}" for ext in extensions]

        total = 0
        for dir_path in args.directory:
            resolved = Path(dir_path).resolve()
            collection_name = resolved.name
            if args.reset:
                try:
                    client.delete_collection(collection_name)
                    print(f"Deleted existing collection '{collection_name}'.")
                except Exception:
                    pass
            kwargs = {"name": collection_name}
            if ef:
                kwargs["embedding_function"] = ef
            collection = client.get_or_create_collection(**kwargs)
            config = {"path": str(resolved), "extensions": extensions}
            print(f"\nIngesting [{collection_name}] from {resolved}")
            count = ingest_source(collection, collection_name, config)
            print(f"  Ingested {count} chunks")
            total += count
        print(f"\nDone. Total chunks ingested: {total}")
    else:
        sources = {args.source: SOURCES[args.source]} if args.source else SOURCES
        total = 0
        for name, config in sources.items():
            if args.reset:
                try:
                    client.delete_collection(name)
                    print(f"Deleted existing collection '{name}'.")
                except Exception:
                    pass
            kwargs = {"name": name}
            if ef:
                kwargs["embedding_function"] = ef
            collection = client.get_or_create_collection(**kwargs)
            print(f"\nIngesting [{name}] from {config['path']}")
            count = ingest_source(collection, name, config)
            print(f"  Ingested {count} chunks")
            total += count
        print(f"\nDone. Total chunks ingested: {total}")


if __name__ == "__main__":
    main()
