## ADDED Requirements

### Requirement: claude-json extension registration
The system SHALL register `claude-json` as a supported file extension in `SupportedFileExtensions`, distinct from the existing `.json` entry. The extension SHALL appear in the TUI extension selector and be available via CLI `--extensions` flag.

#### Scenario: Extension appears in TUI
- **WHEN** a user opens the TUI ingest view
- **THEN** `claude-json` SHALL appear as a selectable extension option alongside existing extensions

#### Scenario: Extension available via CLI
- **WHEN** a user runs `zolam ingest --extensions claude-json`
- **THEN** the ingest pipeline SHALL accept and process `claude-json` as a valid extension

### Requirement: claude-json pseudo-extension file discovery
Since `claude-json` is not a real file suffix, the system SHALL treat it as a special case in all file discovery paths. When `claude-json` is in the extensions list, file scanning SHALL glob for `*.json` files (not `*.claude-json`). In `hasher.go`, `HashDirectory` SHALL map `claude-json` to `.json` when walking the filesystem for differential ingest.

#### Scenario: File globbing for claude-json
- **WHEN** `claude-json` is in the extensions list passed to `ingest.py`
- **THEN** the system SHALL scan for files matching `*.json` (not `*.claude-json`)
- **THEN** the matched files SHALL be routed through the Claude export conversion pipeline instead of plain text extraction

#### Scenario: Differential ingest with claude-json
- **WHEN** `claude-json` is configured for a directory and `RunUpdateOnly` is called
- **THEN** `HashDirectory` SHALL match files with `.json` extension for hashing

### Requirement: Claude export JSON detection
When `claude-json` is selected, `ingest.py` SHALL scan the source directory for `.json` files and attempt to parse each as a Claude chat export. The top-level JSON value may be either an array of conversation objects (bulk export) or a single conversation object. A valid conversation object MUST contain `uuid` (string) and `chat_messages` (array) fields.

#### Scenario: Valid Claude export file with array of conversations
- **WHEN** a `.json` file contains a top-level JSON array where elements have `uuid` and `chat_messages` fields
- **THEN** each conversation object in the array SHALL be parsed and converted to a separate markdown file

#### Scenario: Valid Claude export file with single conversation
- **WHEN** a `.json` file contains a top-level JSON object with `uuid` and `chat_messages` fields
- **THEN** the single conversation SHALL be parsed and converted to markdown

#### Scenario: Conversation with empty chat_messages
- **WHEN** a conversation object has `uuid` and `chat_messages` present but `chat_messages` is an empty array
- **THEN** the conversation SHALL be skipped (no markdown file generated)

#### Scenario: Invalid JSON file when claude-json selected
- **WHEN** a `.json` file does not contain valid conversation objects (missing `uuid` or `chat_messages`)
- **THEN** the file SHALL be skipped with a warning logged to stdout
- **THEN** the ingest run SHALL continue processing remaining files

#### Scenario: File is not valid JSON
- **WHEN** a file with `.json` extension contains invalid JSON
- **THEN** the file SHALL be skipped with a warning logged to stdout
- **THEN** the ingest run SHALL continue processing remaining files

#### Scenario: Non-empty attachments warning
- **WHEN** a message contains non-empty `attachments` or `files` arrays
- **THEN** a warning SHALL be logged indicating that attachments were skipped

### Requirement: Conversation to markdown conversion
Each valid Claude export conversation SHALL be converted to a single markdown file. The markdown file SHALL be written to a temporary directory created via `tempfile.mkdtemp()`.

#### Scenario: Conversation with name
- **WHEN** a Claude export has a non-empty `name` field
- **THEN** the markdown file SHALL use a sanitised version of the name (replacing non-alphanumeric characters with hyphens, truncated to 100 characters) followed by `-{first 8 chars of uuid}` with `.md` extension, to prevent filename collisions
- **THEN** the markdown SHALL begin with `# {name}` as the H1 header

#### Scenario: Conversation without name
- **WHEN** a Claude export has an empty or missing `name` field
- **THEN** the markdown file SHALL use the conversation `uuid` as the filename with `.md` extension
- **THEN** the markdown SHALL begin with `# Untitled Conversation` as the H1 header

#### Scenario: Conversation metadata
- **WHEN** a conversation is converted to markdown
- **THEN** the markdown SHALL include `Created: {created_at}` below the H1 header

### Requirement: Message rendering in markdown
Each message in the `chat_messages` array SHALL be rendered in the markdown output with sender identification and content.

#### Scenario: Human message
- **WHEN** a message has `sender` value `human`
- **THEN** the message SHALL be rendered with an `## Human` H2 header followed by the message text content

#### Scenario: Assistant message with text only
- **WHEN** a message has `sender` value `assistant` and contains only `text` type content blocks
- **THEN** the message SHALL be rendered with an `## Assistant` H2 header followed by the concatenated text content

#### Scenario: Assistant message with thinking blocks
- **WHEN** a message has `sender` value `assistant` and contains `thinking` type content blocks
- **THEN** each thinking block SHALL be rendered as a collapsed `<details>` element with a `<summary>` containing "Thinking" and the duration in seconds (calculated from `stop_timestamp - start_timestamp`, minimum 1 second)
- **THEN** if the thinking block has `cut_off` set to `true`, the summary SHALL append "(truncated)"
- **THEN** the thinking block text SHALL appear inside the `<details>` element
- **THEN** text content blocks SHALL follow after the thinking blocks

#### Scenario: System message
- **WHEN** a message has `sender` value `system`
- **THEN** the message SHALL be rendered with an `## System` H2 header followed by the message text content

#### Scenario: Message separator
- **WHEN** multiple messages exist in a conversation
- **THEN** each message SHALL be separated by a horizontal rule (`---`)

### Requirement: Content block text extraction
The system SHALL extract text from message `content` arrays. If a message has a non-empty `text` field at the message level and no `content` array, the `text` field SHALL be used. If a `content` array exists, text SHALL be extracted from content blocks.

#### Scenario: Content from content array
- **WHEN** a message has a `content` array with `text` type blocks
- **THEN** the text from all `text` type blocks SHALL be concatenated with newlines

#### Scenario: Content from message text field
- **WHEN** a message has a non-empty `text` field and no `content` array
- **THEN** the `text` field value SHALL be used as the message content

#### Scenario: Empty message
- **WHEN** a message has no `text` field and no `content` array (or empty content)
- **THEN** the message SHALL be skipped in the markdown output

### Requirement: Temp directory lifecycle
The temporary directory used for markdown conversion SHALL be created before conversion begins and cleaned up after ingestion completes.

#### Scenario: Successful ingestion
- **WHEN** all Claude export files have been converted and the resulting markdown has been ingested
- **THEN** the temporary directory and all its contents SHALL be deleted

#### Scenario: Ingestion failure
- **WHEN** an error occurs during ingestion after markdown files have been written
- **THEN** the temporary directory SHALL still be cleaned up (using try/finally)

### Requirement: ChromaDB metadata for Claude export chunks
Chunks generated from Claude export markdown SHALL include metadata that traces back to the original conversation.

#### Scenario: Chunk metadata fields
- **WHEN** a chunk is created from Claude export markdown
- **THEN** the chunk metadata SHALL include `source` set to the original `.json` filename
- **THEN** the chunk metadata SHALL include `conversation_id` set to the conversation UUID
- **THEN** the chunk metadata SHALL include `conversation_name` set to the conversation name (or empty string if unnamed)

### Requirement: Unicode sanitisation
The system SHALL sanitise text content by removing private-use Unicode characters (U+E000 to U+F8FF range) from message text before writing to markdown.

#### Scenario: Text with private-use characters
- **WHEN** message text contains characters in the U+E000 to U+F8FF Unicode range
- **THEN** those characters SHALL be removed from the output markdown
