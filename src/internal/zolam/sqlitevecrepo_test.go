package zolam

import (
	"testing"
)

func TestSQLiteRepo_InsertSearchDeleteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	repo, err := OpenSQLiteRepo(dir, "test-model", 3)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer repo.Close()

	page1 := 1
	records := []ChunkRecord{
		{Path: "a.txt", ChunkNum: 0, Page: nil, Text: "the quick brown fox", Embedding: []float32{1, 0, 0}},
		{Path: "b.txt", ChunkNum: 0, Page: &page1, Text: "lazy dog sleeps", Embedding: []float32{0, 1, 0}},
	}
	if err := repo.InsertChunks(records); err != nil {
		t.Fatalf("insert: %v", err)
	}

	hits, err := repo.Search([]float32{1, 0, 0}, 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	if hits[0].Path != "a.txt" || hits[0].Score == nil || *hits[0].Score < 0.99 {
		t.Errorf("expected a.txt as top hit with score~1, got %+v", hits[0])
	}
	if hits[1].Page == nil || *hits[1].Page != 1 {
		t.Errorf("expected b.txt to carry page=1, got %+v", hits[1])
	}

	kwHits, err := repo.KeywordSearch("lazy", 5)
	if err != nil {
		t.Fatalf("keyword search: %v", err)
	}
	if len(kwHits) != 1 || kwHits[0].Path != "b.txt" {
		t.Fatalf("expected 1 keyword hit for b.txt, got %+v", kwHits)
	}

	if err := repo.DeletePaths([]string{"a.txt"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	hits, err = repo.Search([]float32{1, 0, 0}, 5)
	if err != nil {
		t.Fatalf("search after delete: %v", err)
	}
	if len(hits) != 1 || hits[0].Path != "b.txt" {
		t.Fatalf("expected only b.txt to remain, got %+v", hits)
	}
}

func TestSQLiteRepo_ReopenPreservesData(t *testing.T) {
	dir := t.TempDir()
	repo, err := OpenSQLiteRepo(dir, "test-model", 3)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := repo.InsertChunks([]ChunkRecord{
		{Path: "a.txt", ChunkNum: 0, Text: "hello", Embedding: []float32{1, 2, 3}},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	repo2, err := OpenSQLiteRepo(dir, "test-model", 3)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer repo2.Close()
	hits, err := repo2.KeywordSearch("hello", 5)
	if err != nil {
		t.Fatalf("keyword search after reopen: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected data to survive reopen, got %+v", hits)
	}
}
