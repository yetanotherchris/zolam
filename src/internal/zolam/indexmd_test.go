package zolam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetanotherchris/zolam/internal/domain"
)

func TestGenerateIndexMD(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()

	mdPath := filepath.Join(sourceDir, "notes.md")
	mdContent := "# My Notes\n\n## Background\n\n## Details\n\nThis is the body text of the notes file used for the excerpt heuristic in index generation testing.\n"
	if err := os.WriteFile(mdPath, []byte(mdContent), 0o644); err != nil {
		t.Fatalf("writing fixture md: %v", err)
	}

	pdfPath := filepath.Join(sourceDir, "report.pdf")
	if err := os.WriteFile(pdfPath, []byte("binary-pdf-bytes"), 0o644); err != nil {
		t.Fatalf("writing fixture pdf: %v", err)
	}
	sidecarDir := filepath.Join(projectDir, "extracted")
	if err := os.MkdirAll(sidecarDir, 0o755); err != nil {
		t.Fatalf("creating sidecar dir: %v", err)
	}
	sidecar := "---\nsource: " + pdfPath + "\nextracted: 2026-01-01T00:00:00Z\nsha256: abc\n---\n" +
		"# report.pdf\n\n## Page 1\n\nContract terms and obligations discussed across several paragraphs of legal text.\n"
	if err := os.WriteFile(filepath.Join(sidecarDir, "report.pdf.md"), []byte(sidecar), 0o644); err != nil {
		t.Fatalf("writing sidecar: %v", err)
	}

	project := &domain.Project{
		SourceDirs: []string{sourceDir},
		LastIngest: time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
	}
	currentFiles := map[string]string{
		"notes.md":   "hash1",
		"report.pdf": "hash2",
	}

	if err := GenerateIndexMD(project, "my-project", projectDir, sourceDir, currentFiles); err != nil {
		t.Fatalf("GenerateIndexMD() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "index.md"))
	if err != nil {
		t.Fatalf("reading generated index.md: %v", err)
	}
	got := string(data)

	if !strings.HasPrefix(got, "# my-project\n") {
		t.Errorf("expected index.md to start with project heading, got:\n%s", got)
	}
	if !strings.Contains(got, "2 files indexed") {
		t.Errorf("expected file count in summary line, got:\n%s", got)
	}
	if !strings.Contains(got, "My Notes") {
		t.Errorf("expected notes.md title 'My Notes' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "Background, Details") {
		t.Errorf("expected extra headings joined, got:\n%s", got)
	}
	if !strings.Contains(got, "[report.pdf]") {
		t.Errorf("expected report.pdf entry, got:\n%s", got)
	}
	if !strings.Contains(got, "extracted: extracted/report.pdf.md") {
		t.Errorf("expected extracted sidecar reference for pdf, got:\n%s", got)
	}
	if strings.Contains(got, "Page 1") {
		t.Errorf("expected 'Page 1' boilerplate heading to be filtered out, got:\n%s", got)
	}
	if strings.Contains(got, "source: "+pdfPath) {
		t.Errorf("expected YAML front matter to be stripped from summary, got:\n%s", got)
	}
}

func TestGenerateIndexMD_MissingSourceDegradesGracefully(t *testing.T) {
	projectDir := t.TempDir()
	root := t.TempDir() // "gone.md" is never actually written here.

	project := &domain.Project{
		SourceDirs: []string{root},
		LastIngest: time.Now(),
	}
	currentFiles := map[string]string{"gone.md": "hash"}

	if err := GenerateIndexMD(project, "proj", projectDir, root, currentFiles); err != nil {
		t.Fatalf("GenerateIndexMD() returned error for missing file: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "index.md"))
	if err != nil {
		t.Fatalf("reading index.md: %v", err)
	}
	if !strings.Contains(string(data), "gone.md") {
		t.Errorf("expected filename fallback entry for unreadable file, got:\n%s", string(data))
	}
}

func TestSummarizeText_FallbackTitle(t *testing.T) {
	title, headings, excerpt := summarizeText("fallback.txt", "just some plain body text with no headings at all")
	if title != "fallback.txt" {
		t.Errorf("title = %q, want fallback filename", title)
	}
	if len(headings) != 0 {
		t.Errorf("headings = %v, want none", headings)
	}
	if excerpt == "" {
		t.Errorf("expected non-empty excerpt")
	}
}
