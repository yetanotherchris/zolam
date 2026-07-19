package zolam

import (
	"archive/zip"
	"bytes"
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

func TestExtractAndChunk_CSV(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, ".zolam")
	path := filepath.Join(dir, "contacts.csv")
	csvData := "name,email,notes\nAda Lovelace,ada@example.com,first programmer\nAlan Turing,alan@example.com,\n"
	if err := os.WriteFile(path, []byte(csvData), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	chunks, err := ExtractAndChunk(path, projectDir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected a single chunk, got %d: %+v", len(chunks), chunks)
	}
	text := chunks[0].Text
	if !strings.Contains(text, "name: Ada Lovelace | email: ada@example.com | notes: first programmer") {
		t.Errorf("expected first row rendered with headers, got: %q", text)
	}
	if !strings.Contains(text, "name: Alan Turing | email: alan@example.com") {
		t.Errorf("expected second row rendered with headers, got: %q", text)
	}
	if strings.Contains(text, "Alan Turing | notes:") {
		t.Errorf("expected blank notes field to be dropped, got: %q", text)
	}
	if chunks[0].Page != 0 {
		t.Errorf("expected no page for csv, got %d", chunks[0].Page)
	}

	// CSV files are already plain text; no sidecar should be written.
	sidecar := sidecarPath(projectDir, path)
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Errorf("expected no sidecar for csv, got: %v", err)
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

// TestSanitizeDocxRelationIDs guards against a regression on Windows-authored
// docx files containing a "Insert Signature Line" field: Word gives that
// relationship a non-numeric ID (e.g. "rIdSig100"), which go-docx's
// parseDocRelation can't parse (strconv.ParseUint on the "Sig100" suffix),
// making the whole file fail to extract.
func TestSanitizeDocxRelationIDs(t *testing.T) {
	const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="t1" Target="a"/>
<Relationship Id="rIdSig100" Type="tSig" Target="signatureLine1.wmf"/>
</Relationships>`
	const docXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document><w:body>
<w:p><w:r r:id="rId1">hello</w:r></w:p>
<w:pict><v:shape o:relid="rIdSig100"></v:shape></w:pict>
</w:body></w:document>`

	orig := buildTestZip(t, map[string]string{
		"word/_rels/document.xml.rels": relsXML,
		"word/document.xml":            docXML,
	})

	sanitized, err := sanitizeDocxRelationIDs(orig)
	if err != nil {
		t.Fatalf("sanitizeDocxRelationIDs: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(sanitized), int64(len(sanitized)))
	if err != nil {
		t.Fatalf("re-reading sanitized zip: %v", err)
	}
	for _, f := range zr.File {
		content, err := readZipFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f.Name, err)
		}
		if strings.Contains(string(content), "rIdSig100") {
			t.Errorf("expected rIdSig100 to be rewritten, still present in %s", f.Name)
		}
	}
}

func TestSanitizeDocxRelationIDs_NoOpWhenAllNumeric(t *testing.T) {
	const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="t1" Target="a"/>
</Relationships>`

	orig := buildTestZip(t, map[string]string{
		"word/_rels/document.xml.rels": relsXML,
	})

	sanitized, err := sanitizeDocxRelationIDs(orig)
	if err != nil {
		t.Fatalf("sanitizeDocxRelationIDs: %v", err)
	}
	if !bytes.Equal(orig, sanitized) {
		t.Errorf("expected no-op when all relationship IDs are already numeric")
	}
}

func buildTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
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
