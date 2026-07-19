package zolam

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	docx "github.com/fumiama/go-docx"
	"github.com/gen2brain/go-fitz"
	"github.com/otiai10/gosseract/v2"
)

var binaryExtensions = map[string]bool{".pdf": true, ".docx": true}

// PendingChunk is one chunk of extracted text awaiting embedding.
type PendingChunk struct {
	Page int // 0 means "no page" (non-paginated source)
	Text string
}

// ExtractAndChunk reads path, extracts its text (with OCR fallback for
// scanned PDF pages), writes a sidecar for binary formats, and splits the
// result into chunks. It does not embed — that's the caller's job, so it
// can batch embedding calls across files.
func ExtractAndChunk(path, projectDir string) ([]PendingChunk, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".pdf":
		pages, err := extractPDFPages(path)
		if err != nil {
			return nil, err
		}
		if err := writePDFSidecar(projectDir, path, pages); err != nil {
			return nil, err
		}
		var chunks []PendingChunk
		for i, pageText := range pages {
			for _, c := range ChunkText(pageText) {
				chunks = append(chunks, PendingChunk{Page: i + 1, Text: c})
			}
		}
		return chunks, nil
	case ".docx":
		text, err := extractDocx(path)
		if err != nil {
			return nil, err
		}
		if err := writeTextSidecar(projectDir, path, text); err != nil {
			return nil, err
		}
		return chunksNoPage(text), nil
	case ".csv":
		text, err := extractCSV(path)
		if err != nil {
			return nil, err
		}
		return chunksNoPage(text), nil
	default:
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return chunksNoPage(string(data)), nil
	}
}

func chunksNoPage(text string) []PendingChunk {
	var chunks []PendingChunk
	for _, c := range ChunkText(text) {
		chunks = append(chunks, PendingChunk{Text: c})
	}
	return chunks
}

// RemoveSidecar deletes a binary-format file's extracted sidecar, if any.
func RemoveSidecar(projectDir, sourcePath string) error {
	if !binaryExtensions[strings.ToLower(filepath.Ext(sourcePath))] {
		return nil
	}
	err := os.Remove(sidecarPath(projectDir, sourcePath))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func sidecarPath(projectDir, sourcePath string) string {
	return filepath.Join(projectDir, "extracted", filepath.Base(sourcePath)+".md")
}

func frontMatter(sourcePath string) (string, error) {
	hash, err := ComputeHash(sourcePath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("---\nsource: %s\nextracted: %s\nsha256: %s\n---\n",
		sourcePath, time.Now().UTC().Format(time.RFC3339), hash), nil
}

func writePDFSidecar(projectDir, sourcePath string, pages []string) error {
	sidecar := sidecarPath(projectDir, sourcePath)
	if err := os.MkdirAll(filepath.Dir(sidecar), 0o755); err != nil {
		return err
	}
	fm, err := frontMatter(sourcePath)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(fm)
	fmt.Fprintf(&b, "# %s\n", filepath.Base(sourcePath))
	for i, text := range pages {
		fmt.Fprintf(&b, "\n## Page %d\n", i+1)
		if strings.TrimSpace(text) != "" {
			b.WriteString(strings.TrimSpace(text))
		} else {
			b.WriteString("*(no extractable text)*")
		}
		b.WriteString("\n")
	}
	return os.WriteFile(sidecar, []byte(b.String()), 0o644)
}

func writeTextSidecar(projectDir, sourcePath, text string) error {
	sidecar := sidecarPath(projectDir, sourcePath)
	if err := os.MkdirAll(filepath.Dir(sidecar), 0o755); err != nil {
		return err
	}
	fm, err := frontMatter(sourcePath)
	if err != nil {
		return err
	}
	body := fmt.Sprintf("# %s\n\n%s", filepath.Base(sourcePath), text)
	return os.WriteFile(sidecar, []byte(fm+body), 0o644)
}

// extractDocx pulls paragraph and table text out of a .docx file, matching
// python-docx's paragraph-then-table-rows ordering and " | "-joined cells.
func extractDocx(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// go-docx assumes every relationship ID is "rId<number>" and hard-fails
	// (strconv.ParseUint) otherwise. Word features like "Insert Signature
	// Line" generate non-numeric IDs (e.g. "rIdSig100"), which otherwise
	// make an affected docx unparseable. Sanitizing is best-effort: if it
	// fails, fall through and let docx.Parse report on the original bytes.
	if sanitized, err := sanitizeDocxRelationIDs(data); err == nil {
		data = sanitized
	}

	doc, err := docx.Parse(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("parsing docx %s: %w", path, err)
	}

	var parts []string
	for _, item := range doc.Document.Body.Items {
		switch v := item.(type) {
		case *docx.Paragraph:
			if text := v.String(); strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		case *docx.Table:
			for _, row := range v.TableRows {
				cells := make([]string, 0, len(row.TableCells))
				for _, cell := range row.TableCells {
					var cellText strings.Builder
					for _, p := range cell.Paragraphs {
						cellText.WriteString(p.String())
					}
					cells = append(cells, strings.TrimSpace(cellText.String()))
				}
				parts = append(parts, strings.Join(cells, " | "))
			}
		}
	}
	return strings.Join(parts, "\n\n"), nil
}

// extractCSV renders each data row as "header: value | header: value" pairs
// (falling back to "colN" for a missing/blank header), one row per
// blank-line-separated paragraph, so ChunkText packs whole rows into a chunk
// instead of hard-splitting through the middle of a row.
func extractCSV(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // tolerate ragged rows rather than failing the whole file
	r.LazyQuotes = true

	header, err := r.Read()
	if err == io.EOF {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading csv header %s: %w", path, err)
	}

	var rows []string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading csv %s: %w", path, err)
		}

		var pairs []string
		for i, value := range record {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			name := fmt.Sprintf("col%d", i+1)
			if i < len(header) && strings.TrimSpace(header[i]) != "" {
				name = strings.TrimSpace(header[i])
			}
			pairs = append(pairs, fmt.Sprintf("%s: %s", name, value))
		}
		if len(pairs) > 0 {
			rows = append(rows, strings.Join(pairs, " | "))
		}
	}
	return strings.Join(rows, "\n\n"), nil
}

var (
	relIDAttrRe    = regexp.MustCompile(`Id="(rId[^"]*)"`)
	numericRelIDRe = regexp.MustCompile(`^rId[0-9]+$`)
)

// sanitizeDocxRelationIDs rewrites word/_rels/document.xml.rels and
// word/document.xml inside a docx zip, replacing any relationship ID that
// doesn't match go-docx's assumed "rId<number>" format with a synthetic
// numeric one. Returns the original bytes unchanged (with a nil error) if
// there's nothing to fix.
func sanitizeDocxRelationIDs(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	const relsName = "word/_rels/document.xml.rels"
	var relsFile *zip.File
	for _, f := range zr.File {
		if f.Name == relsName {
			relsFile = f
			break
		}
	}
	if relsFile == nil {
		return data, nil
	}
	relsBytes, err := readZipFile(relsFile)
	if err != nil {
		return nil, err
	}

	replacements := map[string]string{}
	next := 9000001
	for _, m := range relIDAttrRe.FindAllStringSubmatch(string(relsBytes), -1) {
		id := m[1]
		if numericRelIDRe.MatchString(id) {
			continue
		}
		if _, ok := replacements[id]; ok {
			continue
		}
		replacements[id] = fmt.Sprintf("rId%d", next)
		next++
	}
	if len(replacements) == 0 {
		return data, nil
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		content, err := readZipFile(f)
		if err != nil {
			return nil, err
		}
		if f.Name == relsName || f.Name == "word/document.xml" {
			s := string(content)
			for old, new := range replacements {
				s = strings.ReplaceAll(s, `"`+old+`"`, `"`+new+`"`)
			}
			content = []byte(s)
		}
		hdr := f.FileHeader
		w, err := zw.CreateHeader(&hdr)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// extractPDFPages extracts each page's text layer, falling back to OCR
// (rendering the page via go-fitz, then recognising it via gosseract) for
// pages with no embedded text, e.g. scanned PDFs. Rendering and OCR are
// deliberately separate calls (rather than a single combined "OCR this PDF
// page" API) so a renderer failure never corrupts the source document.
func extractPDFPages(path string) ([]string, error) {
	doc, err := fitz.New(path)
	if err != nil {
		if !hasPDFHeader(path) {
			return nil, fmt.Errorf("opening pdf %s: %w (file has no %%PDF- header — it may be corrupted, misnamed, or not actually a PDF)", path, err)
		}
		return nil, fmt.Errorf("opening pdf %s: %w", path, err)
	}
	defer doc.Close()

	var ocr *ocrEngine
	pages := make([]string, doc.NumPage())
	for i := 0; i < doc.NumPage(); i++ {
		text, err := doc.Text(i)
		if err != nil {
			return nil, fmt.Errorf("extracting text from %s page %d: %w", path, i+1, err)
		}
		if strings.TrimSpace(text) == "" {
			if ocr == nil {
				ocr, err = newOCREngine()
				if err != nil {
					fmt.Fprintf(os.Stderr, "  OCR unavailable for %s: %v\n", path, err)
					ocr = &ocrEngine{unavailable: true}
				}
			}
			if !ocr.unavailable {
				if ocrText, err := ocr.recognizePage(doc, i); err != nil {
					fmt.Fprintf(os.Stderr, "  OCR failed %s page %d: %v\n", path, i+1, err)
				} else {
					text = ocrText
				}
			}
		}
		pages[i] = text
	}
	if ocr != nil && !ocr.unavailable {
		ocr.close()
	}
	return pages, nil
}

// hasPDFHeader reports whether path starts with a "%PDF-" marker somewhere
// in its first 1024 bytes, per the PDF spec's allowance for leading junk
// before the header. Used only to give a clearer error when go-fitz/MuPDF
// refuses to open a file, distinguishing "not a PDF at all" from a PDF
// MuPDF genuinely can't parse.
func hasPDFHeader(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 1024)
	n, _ := io.ReadFull(f, buf)
	return bytes.Contains(buf[:n], []byte("%PDF-"))
}

// ocrEngine wraps a single gosseract client. gosseract clients aren't safe
// for concurrent use, so each worker in the ingest pool creates its own.
type ocrEngine struct {
	client      *gosseract.Client
	unavailable bool
	mu          sync.Mutex
}

func newOCREngine() (*ocrEngine, error) {
	if dir := findTessdataDir(); dir != "" {
		os.Setenv("TESSDATA_PREFIX", dir)
	}
	client := gosseract.NewClient()
	if err := client.SetLanguage("eng"); err != nil {
		client.Close()
		return nil, fmt.Errorf("tesseract not available (install it, or set TESSDATA_PREFIX to a tessdata folder — see README): %w", err)
	}
	return &ocrEngine{client: client}, nil
}

func (o *ocrEngine) recognizePage(doc *fitz.Document, page int) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	img, err := doc.Image(page)
	if err != nil {
		return "", fmt.Errorf("rendering page: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("encoding rendered page: %w", err)
	}
	if err := o.client.SetImageFromBytes(buf.Bytes()); err != nil {
		return "", fmt.Errorf("loading rendered page into tesseract: %w", err)
	}
	return o.client.Text()
}

func (o *ocrEngine) close() {
	if o.client != nil {
		o.client.Close()
	}
}

// findTessdataDir locates a Tesseract `tessdata` folder (the language files
// OCR needs), since Tesseract's own installers don't always set
// TESSDATA_PREFIX. Checked in order: TESSDATA_PREFIX, then well-known
// per-OS default install paths.
func findTessdataDir() string {
	if env := os.Getenv("TESSDATA_PREFIX"); env != "" {
		return env
	}

	var candidates []string
	scoopDir := os.Getenv("SCOOP")
	if scoopDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			scoopDir = filepath.Join(home, "scoop")
		}
	}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join(scoopDir, "apps", "tesseract", "current", "tessdata"),
			filepath.Join(os.Getenv("ProgramFiles"), "Tesseract-OCR", "tessdata"),
		)
	}
	candidates = append(candidates,
		"/usr/share/tesseract-ocr/5/tessdata",
		"/usr/share/tesseract-ocr/4.00/tessdata",
		"/usr/share/tessdata",
		"/opt/homebrew/share/tessdata",
		"/usr/local/share/tessdata",
	)

	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".traineddata") {
				return dir
			}
		}
	}
	return ""
}
