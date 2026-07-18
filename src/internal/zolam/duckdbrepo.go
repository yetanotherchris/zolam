package zolam

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb/v2"
)

// ChunkRecord is a single embedded chunk ready for insertion.
type ChunkRecord struct {
	Path      string
	ChunkNum  int
	Page      *int
	Text      string
	Embedding []float32
}

// SearchHit is a single ranked or keyword-matched result.
type SearchHit struct {
	Path  string
	Chunk int
	Page  *int
	Text  string
	Score *float64
}

// DuckDBRepo owns the single connection to a project's index.duckdb: schema
// setup, incremental writes (delete/insert), and both vector and keyword
// search. Only one process may ever have this file open at a time (a
// DuckDB constraint, not a choice made here) — callers must hold the
// project's lock file (see lock.go) for the lifetime of a DuckDBRepo.
type DuckDBRepo struct {
	db *sql.DB
}

// OpenDuckDBRepo opens (creating if needed) index.duckdb in projectDir and
// ensures its schema and meta row are up to date.
func OpenDuckDBRepo(projectDir, model string, dims int) (*DuckDBRepo, error) {
	path := filepath.Join(projectDir, "index.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	// DuckDB only supports a single writer connection at a time in the
	// database/sql pool; keeping the pool at 1 avoids the driver silently
	// opening a second one under concurrent use from within this process.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)"); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating meta table: %w", err)
	}
	if _, err := db.Exec(fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS chunks (path VARCHAR, chunk_num INTEGER, page INTEGER, text VARCHAR, embedding FLOAT[%d])",
		dims,
	)); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating chunks table: %w", err)
	}

	meta := map[string]string{
		"model":         model,
		"dims":          fmt.Sprintf("%d", dims),
		"zolam_version": "3",
	}
	for k, v := range meta {
		if _, err := db.Exec(
			"INSERT INTO meta VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value",
			k, v,
		); err != nil {
			db.Close()
			return nil, fmt.Errorf("writing meta %s: %w", k, err)
		}
	}

	return &DuckDBRepo{db: db}, nil
}

// DeletePaths removes every chunk belonging to any of the given file paths
// (used both for changed files, before re-inserting, and for removed
// files).
func (r *DuckDBRepo) DeletePaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("DELETE FROM chunks WHERE path = ?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, p := range paths {
		if _, err := stmt.Exec(p); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// InsertChunks writes a batch of chunk records in a single transaction.
func (r *DuckDBRepo) InsertChunks(records []ChunkRecord) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO chunks VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, rec := range records {
		var page any
		if rec.Page != nil {
			page = *rec.Page
		}
		if _, err := stmt.Exec(rec.Path, rec.ChunkNum, page, rec.Text, rec.Embedding); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// Search runs a cosine-similarity nearest-neighbour query against the
// query embedding.
func (r *DuckDBRepo) Search(queryEmbedding []float32, topK int) ([]SearchHit, error) {
	dims := len(queryEmbedding)
	rows, err := r.db.Query(
		fmt.Sprintf(
			"SELECT path, chunk_num, page, text, array_cosine_similarity(embedding, ?::FLOAT[%d]) AS score "+
				"FROM chunks ORDER BY score DESC LIMIT ?",
			dims,
		),
		queryEmbedding, topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows, true)
}

// KeywordSearch runs a case-insensitive substring match over chunk text.
func (r *DuckDBRepo) KeywordSearch(term string, topK int) ([]SearchHit, error) {
	rows, err := r.db.Query(
		"SELECT path, chunk_num, page, text FROM chunks WHERE text ILIKE ? LIMIT ?",
		"%"+term+"%", topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows, false)
}

func scanHits(rows *sql.Rows, withScore bool) ([]SearchHit, error) {
	var hits []SearchHit
	for rows.Next() {
		var (
			path     string
			chunkNum int
			page     sql.NullInt64
			text     string
			score    float64
		)
		var scanErr error
		if withScore {
			scanErr = rows.Scan(&path, &chunkNum, &page, &text, &score)
		} else {
			scanErr = rows.Scan(&path, &chunkNum, &page, &text)
		}
		if scanErr != nil {
			return nil, scanErr
		}
		hit := SearchHit{Path: path, Chunk: chunkNum, Text: text}
		if page.Valid {
			p := int(page.Int64)
			hit.Page = &p
		}
		if withScore {
			hit.Score = &score
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}

// Close closes the underlying DuckDB connection.
func (r *DuckDBRepo) Close() error {
	return r.db.Close()
}
