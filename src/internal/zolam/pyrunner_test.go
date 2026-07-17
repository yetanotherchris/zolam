package zolam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureScripts_WritesAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZOLAM_DATA_DIR", dir)

	scriptPath, err := EnsureScripts()
	if err != nil {
		t.Fatalf("EnsureScripts() returned error: %v", err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("expected script to exist at %s: %v", scriptPath, err)
	}

	versionPath := filepath.Join(dir, "scripts", ".version")
	firstVersion, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("reading version marker: %v", err)
	}

	// A second call should not rewrite the script (idempotent).
	info1, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}

	if _, err := EnsureScripts(); err != nil {
		t.Fatalf("EnsureScripts() second call returned error: %v", err)
	}

	info2, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat script after second call: %v", err)
	}
	if info1.ModTime() != info2.ModTime() {
		t.Errorf("expected script to be untouched on second call, mtime changed")
	}

	secondVersion, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("reading version marker after second call: %v", err)
	}
	if string(firstVersion) != string(secondVersion) {
		t.Errorf("version marker changed unexpectedly")
	}
}

func TestEnsureScripts_RewritesOnStaleVersion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZOLAM_DATA_DIR", dir)

	scriptPath, err := EnsureScripts()
	if err != nil {
		t.Fatalf("EnsureScripts() returned error: %v", err)
	}

	// Simulate a stale/corrupted script + version marker.
	if err := os.WriteFile(scriptPath, []byte("# stale"), 0o644); err != nil {
		t.Fatalf("writing stale script: %v", err)
	}
	versionPath := filepath.Join(dir, "scripts", ".version")
	if err := os.WriteFile(versionPath, []byte("stale-hash"), 0o644); err != nil {
		t.Fatalf("writing stale version: %v", err)
	}

	if _, err := EnsureScripts(); err != nil {
		t.Fatalf("EnsureScripts() returned error: %v", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("reading script: %v", err)
	}
	if string(data) == "# stale" {
		t.Errorf("expected stale script to be rewritten")
	}
}

func TestLastJSONLine(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`{"a":1}`, `{"a":1}`},
		{"progress\n{\"a\":1}\n", `{"a":1}`},
		{"{\"a\":1}\n\n", `{"a":1}`},
	}
	for _, c := range cases {
		got := string(lastJSONLine([]byte(c.in)))
		if got != c.want {
			t.Errorf("lastJSONLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
