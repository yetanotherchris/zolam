# Zolam document search

A zolam project is just a directory: run `zolam ingest` inside it once (or
`zolam ingest <dirs> [--extensions ...]` to scope/filter what's indexed),
and it gets hidden `.zolam.*` files (`.zolam.project.json`,
`.zolam.index.md`, `.zolam.duckdb`/`.zolam.jsonl`, `.zolam.extracted/` for
PDF/DOCX sidecars). There is no separate global project registry — a
directory containing `.zolam.project.json` is an indexed project.

Workflow for answering a question from the user's documents:

1. Identify the relevant directory (from context, or ask the user). Check for `.zolam.project.json` there; if missing, tell the user to run `zolam ingest` in that directory first.
2. Read `.zolam.index.md` in that directory — a summary of every indexed file. Often this alone identifies the right document.
3. For keyword-shaped questions, grep: `grep -ri "<term>" .zolam.extracted/` (binary docs) and the original source dirs listed in `.zolam.index.md` (plain-text docs). On Windows, use `findstr` or ripgrep (`rg`) instead of `grep`.
4. For conceptual questions, run `zolam query "<question>"` from that directory. Results include file path, page, and the matching chunk.
5. Always open the full extracted file (`.zolam.extracted/<file>.md`) or the original source file to read surrounding context before answering — do not answer from an isolated chunk if the question depends on context.
6. Cite the source file (and page for PDFs) in your answer.

If `zolam query` reports a missing index or model mismatch, tell the user to run `zolam ingest --reset` in that directory.
