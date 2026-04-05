## 1. Go Extension Registration and File Discovery

- [ ] 1.1 Add `claude-json` to `SupportedFileExtensions` in `src/internal/domain/config.go`
- [ ] 1.2 Update `HashDirectory` in `src/internal/zolam/hasher.go` to map `claude-json` to `.json` when walking the filesystem, so differential ingest correctly finds and hashes `.json` files

## 2. Python Extension Handling

- [ ] 2.1 Update `ingest.py` `SUPPORTED_EXTENSIONS` list to include `claude-json`
- [ ] 2.2 Intercept `claude-json` in the `main()` extension normalisation logic (before the dot-prefix step) so it does not become `.claude-json`
- [ ] 2.3 Update file scanning in `ingest_source()` so that when `claude-json` is in the extensions list, it globs for `*.json` and routes matched files through the Claude export conversion pipeline instead of `extract_text()`

## 3. Filename Sanitisation

- [ ] 3.1 Add `sanitize_filename(name, uuid)` function that converts conversation name to a safe filename (replace non-alphanumeric with hyphens, truncate to 100 chars, append first 8 chars of UUID to prevent collisions, fall back to full UUID if name is empty)

## 4. Claude Export JSON Parser

- [ ] 4.1 Add `is_claude_export(data)` function in `ingest.py` that checks for required `uuid` and `chat_messages` fields in a parsed JSON object
- [ ] 4.2 Add `parse_claude_file(filepath)` function that reads a JSON file, handles both top-level array (bulk export) and single object formats, validates each conversation, and returns a list of valid conversation objects. Skip conversations with empty `chat_messages`. Log warnings for invalid entries.
- [ ] 4.3 Add `sanitize_text(text)` function that removes private-use Unicode characters (U+E000-U+F8FF)
- [ ] 4.4 Add `extract_content_text(message)` function that extracts text from content blocks or falls back to message-level `text` field
- [ ] 4.5 Add `format_thinking_block(block)` function that renders thinking blocks as `<details>` HTML with duration calculation from timestamps (minimum 1s), and appends "(truncated)" to summary if `cut_off` is true
- [ ] 4.6 Add `convert_conversation_to_markdown(conversation)` function that produces the full markdown string for one conversation, with H1 title, created date, H2 sender headers, message text, thinking blocks, and `---` separators. Log warning when messages have non-empty `attachments` or `files`.

## 5. Temp Directory and Ingestion Integration

- [ ] 5.1 Add `convert_claude_exports(source_dir, temp_dir)` function that scans for `.json` files, calls `parse_claude_file()` for each, converts valid conversations to `.md` in temp_dir using `sanitize_filename()`, and returns a list of conversion results (temp md path, original json filename, conversation_id, conversation_name)
- [ ] 5.2 Create a metadata lookup dict (keyed by temp `.md` filename) from the conversion results, and modify `ingest_source()` to accept an optional metadata override dict that merges `source`, `conversation_id`, and `conversation_name` into chunk metadata when processing temp files
- [ ] 5.3 Modify the main ingestion flow in `ingest.py` to detect when `claude-json` is in the extensions list: create temp dir, run conversion, ingest the temp markdown files with metadata overrides, then clean up temp dir in a try/finally block

## 6. Testing

- [ ] 6.1 Add unit tests for `is_claude_export()` with valid and invalid JSON structures
- [ ] 6.2 Add unit tests for `parse_claude_file()` covering: JSON array of conversations, single conversation object, empty chat_messages, invalid JSON, missing fields
- [ ] 6.3 Add unit tests for `convert_conversation_to_markdown()` covering: named/unnamed conversations, human/assistant/system messages, thinking blocks with duration and cut_off, empty messages, unicode sanitisation, attachments warning
- [ ] 6.4 Add integration test for the full `convert_claude_exports()` flow with a sample Claude export JSON file containing multiple conversations
- [ ] 6.5 Update `TestSupportedFileExtensions` in `src/internal/domain/config_test.go` to verify `claude-json` is in the list
