# Design: Zolam v3

## Directory layout

```
~/.zolam/                           (override: ZOLAM_DATA_DIR)
  scripts/
    ingest.py                       # go:embed'd, rewritten on content-hash change
    .version                        # sha256 of the embedded script content
  <project-name>/
    project.json
    index.duckdb                    # backend=duckdb (default)
    index.jsonl                     # backend=jsonl
    index.md                        # summary index, always regenerated
    extracted/<name>.<ext>.md       # sidecars for binary formats only
    file-hashes.json                # path -> sha256, incremental update state
  chromadb/                         # legacy backend=chroma data dir (unchanged)
```

`project.json` mirrors the spec's schema (version, backend, embedding_model,
embedding_dims, source_dirs, extensions, created, last_ingest).

## Go/Python boundary

One embedded script, not two. The spec's directory sketch lists both
`ingest.py` and `embed_query.py`; this implementation folds query-time
embedding and search into `ingest.py` via `--mode ingest|update|query`,
matching the spec's own "Script invocation contract" section literally
("`uv run ingest.py --mode ingest|update|query`"). Two independently
PEP-723-declared scripts would duplicate the dependency list and the
duckdb/jsonl reader code for no benefit — one owner per backend, as the
spec requires, is simpler with one file.

Go owns: CLI, file walking, hashing (`file-hashes.json`), diffing
(added/changed/removed), orchestration, and `index.md` generation.
Python owns: extraction, chunking, embedding, and all reads/writes of
`index.duckdb`/`index.jsonl`.

`index.md` generation needs no round-trip through Python: plain-text
source files are read directly by Go, and binary-format sidecars
(`extracted/*.md`) are left on disk by Python across runs, so Go can
re-derive every heuristic summary (title/headings/excerpt) straight from
files already on disk. The stdout JSON contract from Python is therefore
just a run summary (`files_processed`, `chunks_written`, `files_removed`,
`errors`) plus, for `--mode query`, the ranked result list. Progress text
goes to stderr and is streamed live; the final JSON line is the only
stdout Go parses.

## Chunking

Target ~2,000 chars (~500 tokens) with ~15% overlap (~300 chars).
Non-PDF text: split on blank-line paragraph boundaries; a markdown
heading line always starts a new chunk if the buffer is non-empty.
Hard-split only when a single paragraph exceeds the chunk size.

PDFs are chunked **per page** (not spanning pages): each page's text is
run through the same paragraph/heading chunker independently. This is a
deliberate deviation from a single global heading/paragraph pass over the
whole document — it is the only way to guarantee every chunk carries an
exact, correct page number, which the spec requires for PDF results.

## Keyword search

The spec allows creating a DuckDB FTS index and degrading to `ILIKE` if
the extension fails to load. This implementation always uses `ILIKE`
(duckdb) / substring scan (jsonl) for `--keyword` and does not attempt
FTS: maintaining an FTS index correctly across incremental
add/change/remove cycles is nontrivial, and the spec explicitly treats
the degraded path as acceptable. FTS may be revisited post-v3.0.0.

## Embedding model mismatch

`project.json` records `embedding_model`/`embedding_dims`. If a project's
recorded model differs from zolam's current default
(`BAAI/bge-small-en-v1.5`, 384 dims), `ingest`/`update`/`query` refuse
with a one-line remedy pointing at `--reset`.

## `migrate`

Best-effort: pulls documents + metadata from a running ChromaDB
collection over its existing HTTP API, reconstructs each file's text by
concatenating its chunks in stored chunk order (original PDF/DOCX bytes
are not recoverable from Chroma, so re-extraction is not attempted — only
re-embedding), writes the reconstructed text as `.md` under a temp
staging dir, and then runs it through the normal v3 ingest path so
chunking/embedding/index-writing code is not duplicated.

## Legacy path

`--backend chroma` (and the bare pre-v3 `zolam ingest --collection ...`
invocation) is unchanged: Docker Compose, the `ingest` container image,
and the SQLite-backed hash store keep working exactly as before.
`zolam chromadb`, `zolam mcp`, and `zolam collections` gain a one-line
deprecation notice pointing at the v3 commands but are not removed.
