---
name: zolam
description: Search the user's personal document indexes (PDFs, notes, contracts, manuals, code docs) ingested with the zolam CLI. Use when the user asks about the contents of their own files or documents.
---

# Zolam document search

A zolam project is just a directory: run `zolam ingest <dirs> [--extensions
...]` inside it, naming one or more subdirectories to scope/filter what's
indexed (`zolam ingest .` indexes the directory itself, including
dotfiles/dirs like `.git`/`.claude`/`.agents` — usually not what you want).
A directory is required on `zolam ingest <dirs>` every time, not just the
first. To re-sync without naming directories again, use `zolam ingest
update` instead — it reuses the source directories already recorded in
`project.json`. Either way, incremental behaviour (only added/changed/
removed files reprocessed) comes from the stored file hashes, not from
which form of the command you use. Ingesting creates a hidden `.zolam/` folder
(`.zolam/project.json`, `.zolam/index.md`,
`.zolam/index.duckdb` or `.zolam/index.jsonl`,
`.zolam/extracted/` for PDF/DOCX sidecars). There is no separate global
project registry — a directory containing `.zolam/project.json` is an
indexed project.

Workflow for answering a question from the user's documents:

1. Identify the relevant directory (from context, or ask the user). Check for `.zolam/project.json` there; if missing, tell the user to run `zolam ingest <subdir>` in that directory first, naming the subdirectory that actually holds their documents (not `.`, which would also sweep up tool/config directories like `.git`, `.claude`, `.agents`).
2. Read `.zolam/index.md` in that directory — a summary of every indexed file. Often this alone identifies the right document.
3. Always run `zolam query "<question>"` from that directory at least once — this is the tool the skill exists to invoke, and confirms the index is actually live. Each result is printed as `N. [score] path (page P, chunk C)` followed by the matching chunk text — the score is a semantic-similarity number (higher is a better match) shown only for the default embedding search, and omitted with `--keyword`. Paths are relative to the project directory (the one containing `.zolam/`), so resolve them against it, not against your current working directory.
4. Supplement with grep for precise keyword/date lookups the embedding search might miss: `grep -ri "<term>" .zolam/extracted/` (binary docs) and the original source dirs listed in `.zolam/index.md` (plain-text docs).
5. Always open the full extracted file (`.zolam/extracted/<file>.md`) or the original source file to read surrounding context before answering — do not answer from an isolated chunk if the question depends on context.
6. Cite the source file (and page for PDFs) in your answer.

If `zolam query` reports a missing index or model mismatch, tell the user to run `zolam ingest update --reset` in that directory (reuses the recorded source directories; no need to name them again).
