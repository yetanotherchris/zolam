package zolam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractAndChunk_PlainText(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, ".zolam")
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("hello world, this is a short note."), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	chunks, err := ExtractAndChunk(path, projectDir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(chunks) != 1 || chunks[0].Text != "hello world, this is a short note." {
		t.Fatalf("unexpected chunks: %+v", chunks)
	}
	if chunks[0].Page != 0 {
		t.Errorf("expected no page for plain text, got %d", chunks[0].Page)
	}
}

func TestExtractAndChunk_DOCX(t *testing.T) {
	src := "/tmp/sample.docx"
	if _, err := os.Stat(src); err != nil {
		t.Skip("sample.docx fixture not present")
	}
	dir := t.TempDir()
	projectDir := filepath.Join(dir, ".zolam")

	chunks, err := ExtractAndChunk(src, projectDir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	joined := chunks[0].Text
	if !strings.Contains(joined, "Hello world") || !strings.Contains(joined, "A1 | B1") {
		t.Errorf("expected paragraph + table text, got: %q", joined)
	}

	sidecar := sidecarPath(projectDir, src)
	if _, err := os.Stat(sidecar); err != nil {
		t.Errorf("expected sidecar to be written at %s: %v", sidecar, err)
	}
}

func TestExtractAndChunk_PDF(t *testing.T) {
	src := "/root/.claude/uploads/82c8e12b-f48e-5b07-8e11-d86772157afd/67528296-Family_court__CB7.pdf"
	if _, err := os.Stat(src); err != nil {
		t.Skip("sample PDF fixture not present")
	}
	dir := t.TempDir()
	projectDir := filepath.Join(dir, ".zolam")

	chunks, err := ExtractAndChunk(src, projectDir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks from an 18-page PDF with real text")
	}
	maxPage := 0
	for _, c := range chunks {
		if c.Page > maxPage {
			maxPage = c.Page
		}
	}
	if maxPage != 18 {
		t.Errorf("expected chunks spanning 18 pages, max page seen = %d", maxPage)
	}

	sidecar := sidecarPath(projectDir, src)
	data, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatalf("expected sidecar to be written: %v", err)
	}
	if !strings.Contains(string(data), "## Page 1") {
		t.Errorf("expected sidecar to contain page markers")
	}
}

func TestExtractAndChunk_PDF_OCRFallback(t *testing.T) {
	src := "/tmp/ocr_only.pdf"
	if _, err := os.Stat(src); err != nil {
		t.Skip("synthetic OCR-only PDF fixture not present")
	}
	dir := t.TempDir()
	projectDir := filepath.Join(dir, ".zolam")

	chunks, err := ExtractAndChunk(src, projectDir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected OCR to recover text from an image-only PDF page")
	}
	joined := strings.ToUpper(chunks[0].Text)
	if !strings.Contains(joined, "SYNTHETIC") || !strings.Contains(joined, "OCR") {
		t.Errorf("expected OCR'd text to contain the synthetic phrase, got: %q", chunks[0].Text)
	}
}

func TestRemoveSidecar(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, ".zolam")
	sidecar := sidecarPath(projectDir, filepath.Join(dir, "doc.pdf"))
	if err := os.MkdirAll(filepath.Dir(sidecar), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(sidecar, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := RemoveSidecar(projectDir, filepath.Join(dir, "doc.pdf")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Errorf("expected sidecar to be removed")
	}

	// Plain text files never had a sidecar; removing must be a no-op, not
	// an error.
	if err := RemoveSidecar(projectDir, filepath.Join(dir, "note.txt")); err != nil {
		t.Errorf("expected no-op for non-binary source, got: %v", err)
	}
}
