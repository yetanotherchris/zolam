package zolam

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yetanotherchris/zolam/internal/domain"
)

// binaryExtractedExtensions mirrors pyscripts/ingest.py's BINARY_EXTENSIONS:
// formats whose text lives in an extracted/ sidecar rather than the
// original file itself.
var binaryExtractedExtensions = map[string]bool{
	".pdf":  true,
	".docx": true,
}

// stripFrontMatter removes a leading "---\n...\n---\n" YAML block, if present,
// so it isn't mistaken for document body text.
func stripFrontMatter(text string) string {
	const marker = "---"
	if !strings.HasPrefix(text, marker+"\n") {
		return text
	}
	rest := text[len(marker)+1:]
	idx := strings.Index(rest, "\n"+marker)
	if idx == -1 {
		return text
	}
	after := rest[idx+len(marker)+1:]
	return strings.TrimPrefix(after, "\n")
}

// isPageHeading matches the "Page N" headings ingest.py inserts into PDF
// sidecars, which are structural boilerplate rather than real content.
func isPageHeading(heading string) bool {
	fields := strings.Fields(heading)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "page") {
		return false
	}
	for _, r := range fields[1] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// summarizeText derives a heuristic title, up to two extra heading names,
// and the first ~30 words of body text from a markdown or plain-text
// document, falling back to fallbackTitle when no heading is present.
func summarizeText(fallbackTitle, text string) (title string, headings []string, excerpt string) {
	var bodyWords []string

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if heading == "" {
				continue
			}
			if title == "" {
				title = heading
				continue
			}
			if len(headings) < 2 && !isPageHeading(heading) {
				headings = append(headings, heading)
			}
			continue
		}
		if len(bodyWords) < 30 {
			words := strings.Fields(trimmed)
			if remaining := 30 - len(bodyWords); len(words) > remaining {
				words = words[:remaining]
			}
			bodyWords = append(bodyWords, words...)
		}
	}

	if title == "" {
		title = fallbackTitle
	}
	excerpt = strings.Join(bodyWords, " ")
	return title, headings, excerpt
}

// summaryLine joins title/headings/excerpt into one descriptive string.
func summaryLine(title string, headings []string, excerpt string) string {
	parts := []string{title}
	if len(headings) > 0 {
		parts = append(parts, strings.Join(headings, ", "))
	}
	if excerpt != "" {
		parts = append(parts, excerpt)
	}
	return strings.Join(parts, " — ")
}

// fileSummary reads the appropriate source (the sidecar for binary formats,
// the original file otherwise) and produces a heuristic index.md entry.
// Missing/unreadable sources degrade to a filename-only entry rather than
// failing the whole index.md generation.
func fileSummary(projectDir, path string) (summary string, extractedRel string) {
	name := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	var text string
	if binaryExtractedExtensions[ext] {
		extractedRel = filepath.Join("extracted", name+".md")
		data, err := os.ReadFile(filepath.Join(projectDir, extractedRel))
		if err != nil {
			return name, ""
		}
		text = stripFrontMatter(string(data))
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return name, ""
		}
		text = string(data)
	}

	title, headings, excerpt := summarizeText(name, text)
	return summaryLine(title, headings, excerpt), extractedRel
}

// sourceDirLabel returns the configured source directory a file falls
// under (matched by longest path prefix), or "Other" if none match.
func sourceDirLabel(path string, sourceDirs []string) string {
	best := ""
	for _, dir := range sourceDirs {
		if path == dir || strings.HasPrefix(path, dir+string(filepath.Separator)) {
			if len(dir) > len(best) {
				best = dir
			}
		}
	}
	if best == "" {
		return "Other"
	}
	return filepath.Base(best)
}

// GenerateIndexMD regenerates <projectDir>/index.md from the files
// currently on disk: plain-text sources are read directly, and binary
// formats are read from their extracted/ sidecar. currentFiles is the
// full set of indexed paths (e.g. from file-hashes.json) after this run's
// adds/changes/removals have been applied.
func GenerateIndexMD(project *domain.Project, projectName, projectDir string, currentFiles map[string]string) error {
	groups := make(map[string][]string)
	for path := range currentFiles {
		label := sourceDirLabel(path, project.SourceDirs)
		groups[label] = append(groups[label], path)
	}

	var labels []string
	for label := range groups {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", projectName)
	fmt.Fprintf(&b, "> %d files indexed from %s. Last updated %s.\n",
		len(currentFiles), strings.Join(project.SourceDirs, ", "), project.LastIngest.Format("2006-01-02"))

	for _, label := range labels {
		paths := groups[label]
		sort.Strings(paths)
		fmt.Fprintf(&b, "\n## %s\n", label)
		for _, path := range paths {
			summary, extractedRel := fileSummary(projectDir, path)
			name := filepath.Base(path)
			line := fmt.Sprintf("- [%s](%s): %s", name, path, summary)
			if extractedRel != "" {
				line += " — extracted: " + filepath.ToSlash(extractedRel)
			}
			b.WriteString(line + "\n")
		}
	}

	path := filepath.Join(projectDir, "index.md")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing index.md: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("finalising index.md: %w", err)
	}
	return nil
}
