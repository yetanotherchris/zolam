package zolam

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed skillfiles/SKILL.md
var claudeSkillFS embed.FS

//go:embed skillfiles/AGENTS.md
var opencodeInstructionsFS embed.FS

// ClaudeSkillSnippet is a suggested CLAUDE.md addition printed by
// `zolam init claude` after the skill file is installed.
const ClaudeSkillSnippet = `## Personal document search

This project has the zolam skill installed. When the user asks about the
contents of their own files (notes, contracts, manuals, PDFs), use the
zolam skill: run ` + "`zolam query \"<question>\"`" + ` from the project's
directory (run ` + "`zolam ingest`" + ` there first if it hasn't been indexed
yet), or read its ` + "`.zolam/index.md`" + ` manifest directly.
`

// WriteClaudeSkill installs ~/.claude/skills/zolam/SKILL.md, overwriting any
// existing copy (idempotent). Returns the path written.
func WriteClaudeSkill() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	dir := filepath.Join(home, ".claude", "skills", "zolam")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating skill directory: %w", err)
	}
	content, err := claudeSkillFS.ReadFile("skillfiles/SKILL.md")
	if err != nil {
		return "", fmt.Errorf("reading embedded skill: %w", err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}
	return path, nil
}

// WriteOpencodeInstructions installs ~/.config/opencode/AGENTS.md,
// overwriting any existing copy (idempotent). Returns the path written.
func WriteOpencodeInstructions() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating opencode config directory: %w", err)
	}
	content, err := opencodeInstructionsFS.ReadFile("skillfiles/AGENTS.md")
	if err != nil {
		return "", fmt.Errorf("reading embedded instructions: %w", err)
	}
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}
	return path, nil
}
