## Why

Zolam currently ingests `.json` files as plain text, which is not useful for Claude chat export files. Claude's export feature produces JSON files with a structured schema containing conversations, messages, thinking blocks, and metadata. To make these conversations searchable via semantic search, zolam needs to parse the Claude export format and convert it into readable markdown before ingestion.

## What Changes

- Add a new file extension option `claude-json` (displayed as "JSON (Claude export)") distinct from the existing plain `.json` support
- Add a Python conversion step in `ingest.py` that detects Claude export JSON files, parses conversations and messages, and writes one `.md` file per conversation to a temp directory
- The markdown output preserves: conversation title, message sender labels (Human/Assistant), message text content, thinking blocks (as collapsed `<details>` sections), and timestamps
- Ingest the generated markdown files through the existing chunking/embedding pipeline
- Clean up the temp directory after ingestion completes

## Capabilities

### New Capabilities
- `claude-export-parsing`: Parsing Claude chat export JSON files and converting them to markdown for ingestion

### Modified Capabilities

None.

## Impact

- **`src/internal/domain/config.go`**: Add `claude-json` to `SupportedFileExtensions`
- **`ingest.py`**: Add Claude export JSON detection, markdown conversion logic, temp directory management
- **`src/internal/tui/ingest.go`**: The new extension appears in the TUI extension selector (automatic from config change)
- **`src/internal/zolam/hasher.go`**: `HashDirectory` must map `claude-json` to `.json` when scanning files for differential ingest
- **No new Python dependencies required**: JSON parsing and temp file handling use Python stdlib
- **No Dockerfile changes**: No additional pip packages needed
- **No breaking changes**: Existing `.json` plain-text ingestion is unaffected
- **Note**: Claude exports are JSON arrays of conversation objects, not single objects. Each file may contain multiple conversations.
