package zolam

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestFileHashesRoundTrip(t *testing.T) {
	dir := t.TempDir()

	hashes, err := LoadFileHashes(dir)
	if err != nil {
		t.Fatalf("LoadFileHashes() on missing file returned error: %v", err)
	}
	if len(hashes) != 0 {
		t.Fatalf("expected empty map for missing file, got %v", hashes)
	}

	want := map[string]string{
		filepath.Join(dir, "a.md"): "hash-a",
		filepath.Join(dir, "b.md"): "hash-b",
	}
	if err := SaveFileHashes(dir, want); err != nil {
		t.Fatalf("SaveFileHashes() returned error: %v", err)
	}

	got, err := LoadFileHashes(dir)
	if err != nil {
		t.Fatalf("LoadFileHashes() returned error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadFileHashes() = %v, want %v", got, want)
	}
}

func sortedStrings(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func TestDiffHashes(t *testing.T) {
	old := map[string]string{
		"a.md": "h1",
		"b.md": "h2",
		"c.md": "h3",
	}
	next := map[string]string{
		"a.md": "h1",      // unchanged
		"b.md": "h2-new",  // changed
		"d.md": "h4",      // added
	}

	r := DiffHashes(old, next)

	if got, want := sortedStrings(r.Added), []string{"d.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Added = %v, want %v", got, want)
	}
	if got, want := sortedStrings(r.Changed), []string{"b.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Changed = %v, want %v", got, want)
	}
	if got, want := sortedStrings(r.Removed), []string{"c.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Removed = %v, want %v", got, want)
	}
	if got, want := sortedStrings(r.Unchanged), []string{"a.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Unchanged = %v, want %v", got, want)
	}
}

func TestDiffHashes_Empty(t *testing.T) {
	r := DiffHashes(map[string]string{}, map[string]string{})
	if len(r.Added)+len(r.Changed)+len(r.Removed)+len(r.Unchanged) != 0 {
		t.Errorf("expected no diffs for two empty maps, got %+v", r)
	}
}
