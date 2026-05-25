## ADDED Requirements

### Requirement: Clear collection from TUI
The TUI SHALL provide a way to clear (delete all data from) a named ChromaDB collection. This SHALL reuse the existing `ingest.py --reset` mechanism.

#### Scenario: User clears a collection
- **WHEN** the user selects "Clear Collection" from the menu
- **AND** enters or confirms a collection name
- **AND** confirms the destructive action
- **THEN** the system SHALL run the ingest container with `--reset` and `COLLECTION_NAME` set to the specified name
- **AND** display the result to the user

#### Scenario: Clear collection prompts for confirmation
- **WHEN** the user selects "Clear Collection"
- **AND** enters a collection name
- **THEN** the system SHALL display a confirmation prompt before proceeding
- **AND** pressing Esc SHALL cancel without clearing

#### Scenario: Clear collection defaults to default collection name
- **WHEN** the user selects "Clear Collection"
- **THEN** the collection name input SHALL default to `DefaultCollectionName`

#### Scenario: Clear collection requires ChromaDB running
- **WHEN** the user attempts to clear a collection
- **AND** ChromaDB is not running
- **THEN** the system SHALL start ChromaDB before attempting the reset (consistent with ingest behaviour)
