# zolam

Ingest your personal files (PDF, Markdown, Docx, Txt, code) into a local flat-file/SQLite/ChromaDb index for semantic search in Claude Code / OpenCode. 

Zolam is a single Go binary that walks subdirectories, extracting and chunking text and creating embeddings natively (no separate runtime to install). It hashes files for incremental updates, and generates a human-readable `index.md` summary.  It stores what it needs in `.zolam` directory.

## Quick Start

```bash
# MacOS/Linux
brew install yetanotherchris/tap/zolam

# Windows
scoop bucket add zolam https://github.com/yetanotherchris/zolam
scoop install zolam

# Optional: Tesseract, for OCR on scanned PDFs with no text layer
brew install tesseract       # macOS/Linux
apt install tesseract-ocr    # Linux
scoop install tesseract              # Windows
scoop install tesseract-languages    # Windows: language data (eng.traineddata etc.)
```

### Example usage

Subdirectories are scanned recursively. Binary formats (PDF, DOCX) get a markdown version under `.zolam/extracted/`.

```
# Ingest files into the current directory's project (defaults to the sqlite backend)
# A directory is always required, to scope what gets indexed
cd ~/notes
zolam ingest . --extensions .md,.pdf
zolam ingest ./my-sub-dir

# Safe to re-run any time — only added/changed/removed files are reprocessed,
# based on stored file hashes, but the directory still needs to be named
zolam ingest ./my-sub-dir

# ...or re-sync without naming directories again, using the ones already
# recorded in project.json
zolam ingest update

# Ask a question — semantic search over the indexed chunks
zolam query "what did we agree on renewal terms?"
```

### Claude Code / OpenCode skill

Install the zolam skill so Claude Code or OpenCode can search your files
itself, using [`npx skills`](https://github.com/vercel-labs/skills):

```bash
npx skills add https://github.com/yetanotherchris/zolam
```

This installs `skills/zolam/SKILL.md` from this repo into your agent's
skill directory (e.g. `~/.claude/skills/` or `~/.config/opencode/skills/`).


## .zolam directory

The first time you run "ingest", zolam creates a `.zolam` subdirectory in your current path (so `.zolam` is per-project or per-directory essentially). The following are stored in this `.zolam` directory:

```
  project.json         # metadata for the ingestion
  index.db             # (or index.jsonl) the vector index
  index.md             # human-readable summary of every indexed file
  extracted/           # markdown sidecars for PDFs/DOCX (grep-able text)
  file-hashes.json     # incremental-update state
```

You can change the way the text and embeds are stored with `--backend` on first `ingest` (recorded thereafter in `project.json`):

| Backend | When to use |
|---|---|
| `sqlite` (default) | General use — SQLite + [sqlite-vec](https://github.com/asg017/sqlite-vec) for vector search, SQL-queryable, supports keyword (`LIKE`) search alongside semantic search. |
| `jsonl` | You want the index itself to be plain-text: greppable, diffable, easy to inspect or version. |
| `chroma` (legacy) | You're already using the pre-v3 ChromaDB/Docker/MCP workflow and want to keep doing so. |

## Supported File Extensions

`.md`, `.pdf`, `.docx`, `.txt`, `.py`, `.cs`, `.js`, `.ts`, `.json`, `.yml`, `.yaml`, `.csv`, `.html`, `.htm`

## Deprecated: ChromaDB / Docker / MCP workflow

Before v3, zolam ran a ChromaDB server in Docker and required registering
a `chroma-mcp` MCP server with Claude Code/OpenCode. It's deprecated in
favor of the daemon-free `sqlite`/`jsonl` workflow above — it requires
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

## Architecture notes

The ingest/query pipeline (extraction, chunking, embedding, and the SQLite
index itself) is pure Go — there's no separate Python or Node runtime to
install. This needs CGO enabled at build time, since several of the pipeline's
dependencies wrap native libraries: [`mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3)
and [`asg017/sqlite-vec-go-bindings`](https://github.com/asg017/sqlite-vec-go-bindings)
(SQLite + the sqlite-vec extension, both compiled directly into the binary —
no separate library to install), [`gen2brain/go-fitz`](https://github.com/gen2brain/go-fitz) (MuPDF,
for PDF extraction/rendering — statically bundled, no extra install needed),
and [`otiai10/gosseract`](https://github.com/otiai10/gosseract) (Tesseract,
for OCR — needs Tesseract installed on the host, same as the "Optional:
Tesseract" step above). Embeddings run via [`yalue/onnxruntime_go`](https://github.com/yalue/onnxruntime_go)
and [`daulet/tokenizers`](https://github.com/daulet/tokenizers) (the real
HuggingFace tokenizers library, for byte-exact compatibility with the
`BAAI/bge-small-en-v1.5` model this project uses); the onnxruntime shared
library and model weights are downloaded once into `~/.zolam` on first use,
same as before.

Released binaries are unaffected by any of this — CGO only matters at
*build* time (it needs a real C compiler present), not at runtime, so
prebuilt `zolam` downloads need nothing extra installed beyond Tesseract for
OCR.

## Building from Source

Building from source needs a C compiler (CGO is enabled) and, once, a
prebuilt `libtokenizers.a` fetched into `native/tokenizers/`:

```bash
cd src
go run ./tools/fetchnative   # fetches native/tokenizers/libtokenizers.a (one-time)
go build -o zolam ./cmd/zolam/
go test ./...
```

## Name

The name zolam comes from Midazolam, a sedative most people take for an endoscopy or other procedures. It induces sleepiness, decreases anxiety, and causes anterograde amnesia.
