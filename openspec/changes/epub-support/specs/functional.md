# Functional Spec: EPUB File Support

## Requirement 1: EPUB Text Extraction

### Scenario: Valid EPUB file
- **WHEN** a `.epub` file is processed by `extract_text()`
- **THEN** text SHALL be extracted from all content documents (chapters)
- **AND** HTML tags SHALL be stripped, returning plain text
- **AND** chapter text SHALL be concatenated with newline separators

### Scenario: Empty EPUB
- **WHEN** a `.epub` file contains no text content
- **THEN** it SHALL be skipped (return `None`)

### Scenario: DRM-protected EPUB
- **WHEN** a `.epub` file cannot be read (DRM or corruption)
- **THEN** it SHALL be skipped with a `SKIP` message via `tqdm.write()`
- **AND** ingestion SHALL continue with remaining files

## Requirement 2: Extension Registration

### Scenario: Supported extensions list
- **WHEN** the `SUPPORTED_EXTENSIONS` list is checked
- **THEN** `.epub` SHALL be included

### Scenario: Go config
- **WHEN** `SupportedFileExtensions` in `domain/config.go` is checked
- **THEN** `.epub` SHALL be included

## Requirement 3: Docker Dependencies

### Scenario: Docker image build
- **WHEN** the Docker image is built
- **THEN** `ebooklib`, `beautifulsoup4`, and `lxml` SHALL be installed in the virtual environment
