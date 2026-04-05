## Context

Zolam ingests files into ChromaDB for semantic search. Files are processed by `ingest.py` inside a Docker container: text is extracted, chunked, embedded, and stored. Currently `.json` files are read as plain text, which produces poor results for structured data like Claude chat exports.

Claude's chat export produces JSON files containing a top-level array of conversation objects. Each conversation has `uuid`, `name`, `created_at`, `updated_at`, `account`, and a `chat_messages` array. Each message has a `sender` (human/assistant/system), `content` blocks (text and thinking types), and timestamps. A single export file typically contains many conversations. The reference implementation at `github.com/yetanotherchris/openwebui-importer` demonstrates parsing this format.

The ingestion pipeline currently works as: Go CLI orchestrates Docker Compose -> `ingest.py` extracts text -> chunks -> embeds -> upserts to ChromaDB. File extensions are registered in `domain/config.go` and mirrored in `ingest.py`.

## Goals / Non-Goals

**Goals:**
- Parse Claude export JSON files and convert them to readable markdown
- Preserve conversation structure: sender labels, message content, thinking blocks, timestamps
- Integrate with the existing ingestion pipeline (chunking, embedding, ChromaDB upsert)
- Keep `claude-json` distinct from plain `.json` so users can choose which behaviour they want

**Non-Goals:**
- Supporting other chat export formats (ChatGPT, Gemini, etc.)
- Preserving file attachments or images from Claude exports
- Supporting the older/alternative Claude export schemas (only the current schema with `chat_messages`)
- Modifying the ChromaDB query side or chroma-mcp configuration

## Decisions

### 1. Use `claude-json` as the extension identifier

Register `claude-json` in `SupportedFileExtensions` rather than overloading `.json`. Since `claude-json` is a pseudo-extension (not a real file suffix), three places need special handling:

1. **`ingest.py` extension normalisation** (line ~207): Intercept `claude-json` before the dot-prefix logic. Instead of globbing for `*.claude-json`, map it to scan for `*.json` files and route them through Claude export parsing.
2. **`ingest.py` `ingest_source()`**: When processing `claude-json`, skip the normal `extract_text()` path and instead use the Claude export conversion pipeline.
3. **`hasher.go` `HashDirectory()`**: Map `claude-json` to `.json` when walking the filesystem, so differential ingest correctly hashes and detects changes in the source JSON files.

**Rationale**: Users may want to ingest plain JSON files (config files, data files) as-is. A distinct extension identifier lets them choose. The TUI and CLI already support per-directory extension selection.

**Alternative considered**: Auto-detecting Claude exports by inspecting JSON structure. Rejected because it adds complexity and ambiguity - a JSON file might coincidentally have similar keys.

### 2. Convert to markdown in a temp directory, then ingest the markdown

When `claude-json` is selected, `ingest.py` will:
1. Scan for `.json` files in the source directory
2. Parse each JSON file. The top-level value may be an array of conversation objects (bulk export) or a single conversation object.
3. Validate each conversation object has required fields (`uuid`, `chat_messages`)
4. Convert each valid conversation to a `.md` file in a temp directory (`tempfile.mkdtemp()`)
5. Ingest the temp markdown files through the existing text extraction path
6. Clean up the temp directory

**Rationale**: Reuses the existing markdown ingestion path completely. Markdown is the natural format for readable conversation text. The temp directory avoids polluting the user's source directory.

**Alternative considered**: Extracting text directly without the intermediate markdown step. Rejected because markdown conversion produces better chunking boundaries (headers, paragraphs) and the resulting ChromaDB entries are more readable in search results.

### 3. Markdown format for converted conversations

Each conversation produces one `.md` file named `{sanitised_name}-{short_uuid}.md` (the short UUID suffix prevents filename collisions when two conversations share the same name):

```markdown
# Conversation: {name or "Untitled"}
Created: {created_at}

---

## Human
{message text}

## Assistant

<details>
<summary>Thinking ({duration}s)</summary>

{thinking text}

</details>

{response text}

---

## Human
{next message}
```

**Rationale**: H2 headers for sender labels create natural chunk boundaries. The `---` separators between message pairs help the chunker split at conversation turns. Thinking blocks use HTML `<details>` to keep them present but visually secondary.

### 4. Handle files that aren't valid Claude exports gracefully

If a `.json` file in the source directory doesn't match the Claude export schema (missing `uuid` or `chat_messages`), skip it with a warning log. Don't fail the entire ingest run.

**Rationale**: Users might have mixed JSON files in a directory. Failing silently would be confusing, but failing hard would be worse.

### 5. Metadata in ChromaDB

The chunk metadata will include:
- `source`: The original `.json` filename (not the temp `.md` path)
- `conversation_id`: The conversation UUID from the export
- `conversation_name`: The conversation name (if present)

**Mechanism**: The `convert_claude_exports()` function returns a list of conversion results, each containing the temp `.md` path, original source filename, conversation_id, and conversation_name. A metadata lookup dict (keyed by temp `.md` filename) is passed to a modified `ingest_source()` which merges these fields into chunk metadata when processing temp files.

**Rationale**: Preserving the original source filename and conversation ID helps users trace search results back to specific conversations.

## Risks / Trade-offs

- **[Large export files]** A user with hundreds of conversations in a single export could generate many temp markdown files. -> Mitigation: Files are processed sequentially and temp dir is cleaned up after. Memory usage is bounded by one conversation at a time.
- **[Schema evolution]** Claude may change their export format in future. -> Mitigation: Validate required fields before parsing. Unknown fields are ignored. If the schema changes significantly, the parser will skip unrecognised files with a warning rather than crashing.
- **[Temp directory cleanup on failure]** If ingest crashes mid-way, the temp directory may not be cleaned up. -> Mitigation: Use Python's `tempfile.mkdtemp()` in the system temp location. OS-level cleanup will handle it eventually. Could also wrap in try/finally.
- **[Citations and attachments ignored]** Claude exports may contain `citations` on text blocks and `attachments`/`files` on messages. These are intentionally not rendered in markdown. If non-empty attachments are present, a warning is logged so users know content was skipped.
