---
name: zolam
description: Search the user's personal document indexes (PDFs, notes, contracts, manuals, code docs) ingested with the zolam CLI. Use when the user asks about the contents of their own files or documents.
---

# Zolam document search

Indexes live in `~/.zolam/<project>/`. Discover projects with `zolam projects list`.

Workflow for answering a question from the user's documents:

1. Run `zolam projects list` and pick the relevant project(s).
2. Read `~/.zolam/<project>/index.md` — a summary of every indexed file. Often this alone identifies the right document.
3. For keyword-shaped questions, grep: `grep -ri "<term>" ~/.zolam/<project>/extracted/` (binary docs) and the original source dirs listed in index.md (plain-text docs).
4. For conceptual questions, semantic search: `zolam query "<question>" --project <name>`. Results include file path, page, and the matching chunk.
5. Always open the full extracted file (`~/.zolam/<project>/extracted/<file>.md`) or the original source file to read surrounding context before answering — do not answer from an isolated chunk if the question depends on context.
6. Cite the source file (and page for PDFs) in your answer.

If `zolam query` reports a missing index or model mismatch, tell the user to run `zolam ingest`/`zolam update`.
