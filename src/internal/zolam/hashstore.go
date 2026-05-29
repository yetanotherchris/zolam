package zolam

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// HashStore persists file hashes to a SQLite database in ~/.zolam/hashes.db.
type HashStore struct {
	db         *sql.DB
	collection string
}

// OpenHashStore opens (or creates) the hash store for a given collection.
func OpenHashStore(collection string) (*HashStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}
	dir := filepath.Join(homeDir, ".zolam")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating ~/.zolam: %w", err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "hashes.db"))
	if err != nil {
		return nil, fmt.Errorf("opening hash store: %w", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS file_hashes (
			collection TEXT NOT NULL,
			file_path  TEXT NOT NULL,
			hash       TEXT NOT NULL,
			PRIMARY KEY (collection, file_path)
		)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialising hash store: %w", err)
	}
	return &HashStore{db: db, collection: collection}, nil
}

func (s *HashStore) Close() error {
	return s.db.Close()
}

// GetAll returns all stored path→hash entries for the collection.
func (s *HashStore) GetAll() (map[string]string, error) {
	rows, err := s.db.Query(
		`SELECT file_path, hash FROM file_hashes WHERE collection = ?`,
		s.collection,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			return nil, err
		}
		result[path] = hash
	}
	return result, rows.Err()
}

// Set stores or updates the hash for a file path.
func (s *HashStore) Set(filePath, hash string) error {
	_, err := s.db.Exec(
		`INSERT INTO file_hashes (collection, file_path, hash) VALUES (?, ?, ?)
		 ON CONFLICT(collection, file_path) DO UPDATE SET hash = excluded.hash`,
		s.collection, filePath, hash,
	)
	return err
}

// DeleteCollection removes all hash entries for the collection.
func (s *HashStore) DeleteCollection() error {
	_, err := s.db.Exec(`DELETE FROM file_hashes WHERE collection = ?`, s.collection)
	return err
}

// DeleteFile removes any hash entries whose path ends with the given file name.
func (s *HashStore) DeleteFile(fileName string) error {
	_, err := s.db.Exec(
		`DELETE FROM file_hashes WHERE collection = ? AND (file_path LIKE ? OR file_path LIKE ?)`,
		s.collection, "%/"+fileName, "%\\"+fileName,
	)
	return err
}
