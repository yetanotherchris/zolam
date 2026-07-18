package zolam

import (
	"regexp"
	"strings"
)

const (
	// ChunkSize is the target chunk length in runes, matching the size the
	// embedding model was tuned/evaluated against by the original Python
	// pipeline.
	ChunkSize = 2000
	// ChunkOverlap keeps ~15% continuity between adjacent hard-split chunks.
	ChunkOverlap = int(ChunkSize * 0.15)
)

var paragraphSplitRe = regexp.MustCompile(`\n\s*\n`)

func splitParagraphs(text string) []string {
	return paragraphSplitRe.Split(text, -1)
}

func isHeading(paragraph string) bool {
	return strings.HasPrefix(strings.TrimLeft(paragraph, " \t\r\n"), "#")
}

// hardSplit splits a single oversized unit into fixed-size, overlapping
// pieces, operating on runes so multi-byte characters aren't split mid-code
// point.
func hardSplit(s []rune, size, overlap int) [][]rune {
	if len(s) <= size {
		return [][]rune{s}
	}
	var chunks [][]rune
	start := 0
	n := len(s)
	for start < n {
		end := start + size
		if end > n {
			end = n
		}
		chunks = append(chunks, s[start:end])
		if end >= n {
			break
		}
		start = end - overlap
	}
	return chunks
}

// ChunkText splits text into ~size-rune chunks with ~overlap-rune
// continuity, preferring markdown heading boundaries, then paragraph
// (blank-line) boundaries, then hard-splitting any single paragraph too
// large to fit. Faithfully mirrors the original Python pipeline's chunker
// so ingested content is comparable across the rewrite.
func ChunkText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len([]rune(text)) <= ChunkSize {
		return []string{text}
	}

	var paragraphs []string
	for _, p := range splitParagraphs(text) {
		if strings.TrimSpace(p) != "" {
			paragraphs = append(paragraphs, p)
		}
	}

	var chunks []string
	buffer := ""

	startBuffer := func(seedPara string) string {
		if len(chunks) == 0 {
			return seedPara
		}
		last := []rune(chunks[len(chunks)-1])
		tailLen := ChunkOverlap
		if tailLen > len(last) {
			tailLen = len(last)
		}
		tail := string(last[len(last)-tailLen:])
		if tail == "" {
			return seedPara
		}
		return tail + "\n\n" + seedPara
	}

	for _, para := range paragraphs {
		paraRunes := []rune(para)

		if len(paraRunes) > ChunkSize {
			if strings.TrimSpace(buffer) != "" {
				chunks = append(chunks, strings.TrimSpace(buffer))
				buffer = ""
			}
			for _, piece := range hardSplit(paraRunes, ChunkSize, ChunkOverlap) {
				chunks = append(chunks, string(piece))
			}
			continue
		}

		if isHeading(para) && strings.TrimSpace(buffer) != "" {
			chunks = append(chunks, strings.TrimSpace(buffer))
			buffer = startBuffer(para)
			continue
		}

		candidate := para
		if buffer != "" {
			candidate = buffer + "\n\n" + para
		}
		if len([]rune(candidate)) > ChunkSize {
			chunks = append(chunks, strings.TrimSpace(buffer))
			buffer = startBuffer(para)
		} else {
			buffer = candidate
		}
	}

	if strings.TrimSpace(buffer) != "" {
		chunks = append(chunks, strings.TrimSpace(buffer))
	}

	if len(chunks) == 0 {
		return []string{text}
	}
	return chunks
}
