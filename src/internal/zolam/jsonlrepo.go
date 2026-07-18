package zolam

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type jsonlRecord struct {
	Path      string    `json:"path"`
	Chunk     int       `json:"chunk"`
	Page      *int      `json:"page,omitempty"`
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding"`
}

type jsonlMeta struct {
	Meta    bool   `json:"_meta"`
	Version int    `json:"version"`
	Model   string `json:"model"`
	Dims    int    `json:"dims"`
}

// JsonlRepo is the dependency-free fallback index backend: all records
// live in memory and are rewritten to a single index.jsonl file on Close.
// Search is a plain linear scan, which is fine at personal-file scale (the
// same approach the DuckDB backend's array_cosine_similarity uses under
// the hood, just without a query-planner in front of it).
type JsonlRepo struct {
	path    string
	model   string
	dims    int
	records []jsonlRecord
}

// OpenJsonlRepo loads (or creates) index.jsonl in projectDir.
func OpenJsonlRepo(projectDir, model string, dims int) (*JsonlRepo, error) {
	path := filepath.Join(projectDir, "index.jsonl")
	repo := &JsonlRepo{path: path, model: model, dims: dims}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return repo, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if first {
			first = false
			continue // skip the _meta header line, matching the Python backend
		}
		var rec jsonlRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		repo.records = append(repo.records, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *JsonlRepo) DeletePaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	remove := make(map[string]bool, len(paths))
	for _, p := range paths {
		remove[p] = true
	}
	kept := r.records[:0]
	for _, rec := range r.records {
		if !remove[rec.Path] {
			kept = append(kept, rec)
		}
	}
	r.records = kept
	return nil
}

func (r *JsonlRepo) InsertChunks(records []ChunkRecord) error {
	for _, rec := range records {
		r.records = append(r.records, jsonlRecord{
			Path:      rec.Path,
			Chunk:     rec.ChunkNum,
			Page:      rec.Page,
			Text:      rec.Text,
			Embedding: rec.Embedding,
		})
	}
	return nil
}

func (r *JsonlRepo) Search(queryEmbedding []float32, topK int) ([]SearchHit, error) {
	type scored struct {
		hit   SearchHit
		score float64
	}
	qNorm := norm(queryEmbedding)
	scoredHits := make([]scored, 0, len(r.records))
	for _, rec := range r.records {
		score := cosine(rec.Embedding, queryEmbedding, norm(rec.Embedding), qNorm)
		scoredHits = append(scoredHits, scored{
			hit:   SearchHit{Path: rec.Path, Chunk: rec.Chunk, Page: rec.Page, Text: rec.Text, Score: &score},
			score: score,
		})
	}
	sort.Slice(scoredHits, func(i, j int) bool { return scoredHits[i].score > scoredHits[j].score })
	if topK < len(scoredHits) {
		scoredHits = scoredHits[:topK]
	}
	hits := make([]SearchHit, len(scoredHits))
	for i, s := range scoredHits {
		hits[i] = s.hit
	}
	return hits, nil
}

func (r *JsonlRepo) KeywordSearch(term string, topK int) ([]SearchHit, error) {
	termLower := strings.ToLower(term)
	var hits []SearchHit
	for _, rec := range r.records {
		if strings.Contains(strings.ToLower(rec.Text), termLower) {
			hits = append(hits, SearchHit{Path: rec.Path, Chunk: rec.Chunk, Page: rec.Page, Text: rec.Text})
			if len(hits) >= topK {
				break
			}
		}
	}
	return hits, nil
}

// Close rewrites index.jsonl atomically: a _meta header line followed by
// one line per record, matching the Python backend's on-disk format.
func (r *JsonlRepo) Close() error {
	tmp := r.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)

	meta := jsonlMeta{Meta: true, Version: 3, Model: r.model, Dims: r.dims}
	metaLine, err := json.Marshal(meta)
	if err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if _, err := w.Write(metaLine); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	w.WriteByte('\n')

	for _, rec := range r.records {
		line, err := json.Marshal(rec)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		if _, err := w.Write(line); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		w.WriteByte('\n')
	}

	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, r.path)
}

func norm(v []float32) float64 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	return math.Sqrt(sum)
}

func cosine(a, b []float32, normA, normB float64) float64 {
	if len(a) != len(b) || normA == 0 || normB == 0 {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot / (normA * normB)
}
