# Tasks

- [x] Add SupportedFileExtensions as a package-level slice (not in Config struct), remove Extensions from Config
- [x] Add DirectoryEntry type and Directories field to config.json schema
- [x] Add config.json load/save functions in domain/config.go
- [x] Update LoadConfig to read config.json then overlay env vars
- [x] Add SaveConfig function to persist config.json
- [x] Update ingest command to save directories+extensions to config.json after ingest
- [x] Update update command to read directories from config.json when no args given
- [x] Update stats command/page to print SupportedFileExtensions
- [x] Update TUI ingest.go to use SupportedFileExtensions
- [x] Update TUI settings view (remove Extensions line)
- [x] Update TUI app.go config command output
- [x] Update config_test.go for new config.json loading
- [x] Update README.md with simplified structure
- [x] Run tests and verify
