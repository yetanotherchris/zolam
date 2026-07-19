package zolam

import (
	"database/sql"
	"fmt"
	"path/filepath"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	// Registers sqlite-vec's functions/virtual tables (vec_version(),
	// vec0, ...) on every SQLite connection opened in this process from
	// here on, via sqlite3_auto_extension. mattn/go-sqlite3 compiles its
	// own sqlite3.c into this binary, and sqlite-vec.c (compiled with
	// -DSQLITE_CORE, see the cgo package) links directly against those
	// same symbols — no separate loadable-extension file involved.
	sqlite_vec.Auto()
}

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

// SQLiteRepo owns the single connection to a project's index.db: schema
// setup, incremental writes (delete/insert), and both vector and keyword
// search. Chunk metadata lives in a regular "chunks" table; embeddings
// live in a parallel sqlite-vec "chunks_vec" virtual table sharing the
// same rowid, joined on read — the pattern sqlite-vec itself recommends,
// since vec0 tables are best kept to just the vector column. Callers must
// hold the project's lock file (see lock.go) for the lifetime of a
// SQLiteRepo.
type SQLiteRepo struct {
	db *sql.DB
}

// OpenSQLiteRepo opens (creating if needed) index.db in projectDir and
// ensures its schema and meta row are up to date.
func OpenSQLiteRepo(projectDir, model string, dims int) (*SQLiteRepo, error) {
	path := filepath.Join(projectDir, "index.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	// Keep a single connection: simpler transaction semantics, and
	// matches this project's existing design of serializing all access
	// to a project's index through the ingest.lock file (see lock.go)
	// rather than relying on SQLite's own concurrency control.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT)"); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating meta table: %w", err)
	}
	if _, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS chunks (" +
			"id INTEGER PRIMARY KEY, path TEXT NOT NULL, chunk_num INTEGER NOT NULL, " +
			"page INTEGER, text TEXT NOT NULL)",
	); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating chunks table: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_chunks_path ON chunks (path)"); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating chunks path index: %w", err)
	}
	if _, err := db.Exec(fmt.Sprintf(
		"CREATE VIRTUAL TABLE IF NOT EXISTS chunks_vec USING vec0(embedding FLOAT[%d] distance_metric=cosine)",
		dims,
	)); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating chunks_vec table: %w", err)
	}

	meta := map[string]string{
		"model":         model,
		"dims":          fmt.Sprintf("%d", dims),
		"zolam_version": "3",
	}
	for k, v := range meta {
		if _, err := db.Exec(
			"INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value",
			k, v,
		); err != nil {
			db.Close()
			return nil, fmt.Errorf("writing meta %s: %w", k, err)
		}
	}

	return &SQLiteRepo{db: db}, nil
}

// DeletePaths removes every chunk belonging to any of the given file paths
// (used both for changed files, before re-inserting, and for removed
// files).
func (r *SQLiteRepo) DeletePaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	delVec, err := tx.Prepare("DELETE FROM chunks_vec WHERE rowid IN (SELECT id FROM chunks WHERE path = ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer delVec.Close()
	delChunks, err := tx.Prepare("DELETE FROM chunks WHERE path = ?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer delChunks.Close()
	for _, p := range paths {
		if _, err := delVec.Exec(p); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := delChunks.Exec(p); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// InsertChunks writes a batch of chunk records in a single transaction.
func (r *SQLiteRepo) InsertChunks(records []ChunkRecord) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	insChunk, err := tx.Prepare("INSERT INTO chunks (path, chunk_num, page, text) VALUES (?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer insChunk.Close()
	insVec, err := tx.Prepare("INSERT INTO chunks_vec (rowid, embedding) VALUES (?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer insVec.Close()

	for _, rec := range records {
		var page any
		if rec.Page != nil {
			page = *rec.Page
		}
		res, err := insChunk.Exec(rec.Path, rec.ChunkNum, page, rec.Text)
		if err != nil {
			tx.Rollback()
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}
		blob, err := sqlite_vec.SerializeFloat32(rec.Embedding)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("serializing embedding for %s#%d: %w", rec.Path, rec.ChunkNum, err)
		}
		if _, err := insVec.Exec(id, blob); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// Search runs a cosine-similarity nearest-neighbour query against the
// query embedding.
func (r *SQLiteRepo) Search(queryEmbedding []float32, topK int) ([]SearchHit, error) {
	blob, err := sqlite_vec.SerializeFloat32(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("serializing query embedding: %w", err)
	}
	rows, err := r.db.Query(
		"SELECT c.path, c.chunk_num, c.page, c.text, m.distance "+
			"FROM (SELECT rowid, distance FROM chunks_vec WHERE embedding MATCH ? AND k = ? ORDER BY distance) m "+
			"JOIN chunks c ON c.id = m.rowid "+
			"ORDER BY m.distance",
		blob, topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows, true)
}

// KeywordSearch runs a case-insensitive substring match over chunk text.
func (r *SQLiteRepo) KeywordSearch(term string, topK int) ([]SearchHit, error) {
	rows, err := r.db.Query(
		"SELECT path, chunk_num, page, text FROM chunks WHERE text LIKE ? LIMIT ?",
		"%"+term+"%", topK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows, false)
}

func scanHits(rows *sql.Rows, withDistance bool) ([]SearchHit, error) {
	var hits []SearchHit
	for rows.Next() {
		var (
			path     string
			chunkNum int
			page     sql.NullInt64
			text     string
			distance float64
		)
		var scanErr error
		if withDistance {
			scanErr = rows.Scan(&path, &chunkNum, &page, &text, &distance)
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
		if withDistance {
			// chunks_vec is declared with distance_metric=cosine, so
			// distance is cosine distance (0 = identical, 2 =
			// opposite); report similarity (1 = identical) to match
			// the score semantics callers/tests expect.
			score := 1 - distance
			hit.Score = &score
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}

// Close closes the underlying SQLite connection.
func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}
