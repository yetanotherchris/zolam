# zolam

Ingest your personal files (PDF, Markdown, Docx, Txt, code) into a local flat-file/DuckDB/ChromaDb index for semantic search in Claude Code / OpenCode. 

Zolam is a Go CLI that walks your directories, extracting and chunking text (via Python), hashes files for incremental updates, and generates a human-readable `index.md` summary.  It stores what it needs in `.zolam` directory.


## Quick Start

```bash
# Install uv (required)
curl -LsSf https://astral.sh/uv/install.sh | sh
brew install uv             # macOS/Linux
winget install astral-sh.uv # Windows
scoop install uv            # Windows

# Optional: Tesseract, for OCR on scanned PDFs with no text layer
brew install tesseract       # macOS/Linux
apt install tesseract-ocr    # Linux
scoop install tesseract              # Windows
scoop install tesseract-languages    # Windows: language data (eng.traineddata etc.)
```

### Example usage

Subdirectories are scanned recursively. Binary formats (PDF, DOCX) get a markdown version under `.zolam/extracted/`;

```
# Ingest files into the current directory's project (defaults to the duckdb backend)
cd ~/notes
zolam ingest --extensions .md,.pdf
zolam ingest ./my-sub-dir

# Safe to re-run any time — only added/changed/removed files are reprocessed
zolam ingest

# Ask a question — semantic search over the indexed chunks
zolam query "what did we agree on renewal terms?"

# Install the Claude Code skill so Claude can search your files itself
zolam init claude
```

The following are stored in the `.zolam` directory:

```
~/notes/.zolam/
  project.json         # metadata for the ingestion
  index.duckdb         # (or index.jsonl) the vector index
  index.md             # human-readable summary of every indexed file
  extracted/           # markdown sidecars for PDFs/DOCX (grep-able text)
  file-hashes.json     # incremental-update state
```

### Installation

```bash
# MacOS/Linux
brew install yetanotherchris/tap/zolam

# Windows
scoop bucket add zolam https://github.com/yetanotherchris/zolam
scoop install zolam
```

## Index backends

Set with `--backend` on first `ingest` (recorded thereafter in `project.json`):

| Backend | When to use |
|---|---|
| `duckdb` (default) | General use — SQL-queryable, supports keyword (`ILIKE`) search alongside semantic search. |
| `jsonl` | You want the index itself to be plain-text: greppable, diffable, easy to inspect or version. |
| `chroma` (legacy) | You're already using the pre-v3 ChromaDB/Docker/MCP workflow and want to keep doing so. |

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ZOLAM_DATA_DIR` | `~/.zolam` | Cache directory for the embedded Python pipeline script (project data itself lives in each project's own `.zolam/` folder) |
| `ZOLAM_CHROMADB_DATA_DIR` | `~/.zolam/chromadb` | Legacy `--backend chroma` ChromaDB persistent storage path |

## Supported File Extensions

`.md`, `.pdf`, `.docx`, `.txt`, `.py`, `.cs`, `.js`, `.ts`, `.json`, `.yml`, `.yaml`, `.csv`, `.html`, `.htm`

## Deprecated: ChromaDB / Docker / MCP workflow

Before v3, zolam ran a ChromaDB server in Docker and required registering
a `chroma-mcp` MCP server with Claude Code/OpenCode. It's deprecated in
favor of the daemon-free `duckdb`/`jsonl` workflow above — it requires
Docker Desktop, a background container, and an extra MCP registration
step that the v3 flow doesn't need. `zolam ingest`/`zolam query` no
longer support this backend at all (`--backend chroma` is rejected);
container management and MCP registration live under `zolam chromadb`
for existing ChromaDB data only:

```bash
zolam chromadb start                                              # start the container
zolam chromadb mcp claude                                         # register chroma-mcp
zolam chromadb collections list                                   # list chroma collections
```

Ingesting new data into ChromaDB requires the standalone
`docker-compose.yml` in this repo (bypassing the `zolam` CLI entirely):

```bash
docker compose --profile ingest run --rm \
  -v /path/to/notes:/sources/notes \
  ingest --directory /sources/notes
```

## Why Python?

The ingest pipeline (text extraction, chunking, embedding) runs as an
embedded Python script rather than pure Go, mainly because of the embedding
step: generating vectors with `BAAI/bge-small-en-v1.5` needs an ONNX
inference runtime plus a matching tokenizer, and Python's
[`fastembed`](https://github.com/qdrant/fastembed) package bundles both,
mature and battle-tested.

A Go-only pipeline is possible but not yet a clean swap:
- [`onnxruntime.ai`](https://onnxruntime.ai/docs/) (what `fastembed` runs on under the hood) has no official Go API — you'd rely on a community CGo binding like [`yalue/onnxruntime_go`](https://github.com/yalue/onnxruntime_go), trading a static Go binary for CGo plus a bundled native `onnxruntime` shared library per platform.
- Go still lacks a mature, bit-exact tokenizer matching `fastembed`'s output for `BAAI/bge-small-en-v1.5`, so tokenization would need to be built and verified against the Python output to avoid silently different embeddings.
- PDF/DOCX extraction, however, has solid pure-Go options — [`gopdf`](https://github.com/razvandimescu/gopdf) and [`go-docx`](https://github.com/fumiama/go-docx) — so that half of the pipeline could move to Go today.

`uv` keeps the Python dependency painless (no manual install, auto-provisioned
on first run), so it stays for now.

## Building from Source

```bash
cd src
go build -o zolam ./cmd/zolam/
go test ./...
```

## Name

The name zolam comes from Midazolam, a sedative most people take for an endoscopy or other procedures. It induces sleepiness, decreases anxiety, and causes anterograde amnesia.
