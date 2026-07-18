package zolam

import (
	"encoding/json"
	"os"
	"testing"
)

// TestChunkText_MatchesPythonReference compares ChunkText's output against
// the original Python pipeline's chunk_text() for a range of inputs
// (short/long, headings, oversized paragraphs, unicode), to verify the Go
// port is a faithful behavioural match, not just "looks right."
func TestChunkText_MatchesPythonReference(t *testing.T) {
	casesData, err := os.ReadFile("testdata/chunk_cases.json")
	if err != nil {
		t.Fatalf("reading cases: %v", err)
	}
	expectedData, err := os.ReadFile("testdata/chunk_expected.json")
	if err != nil {
		t.Fatalf("reading expected: %v", err)
	}

	var cases []string
	if err := json.Unmarshal(casesData, &cases); err != nil {
		t.Fatalf("parsing cases: %v", err)
	}
	var expected [][]string
	if err := json.Unmarshal(expectedData, &expected); err != nil {
		t.Fatalf("parsing expected: %v", err)
	}
	if len(cases) != len(expected) {
		t.Fatalf("cases/expected length mismatch: %d vs %d", len(cases), len(expected))
	}

	for i, c := range cases {
		got := ChunkText(c)
		want := expected[i]
		if len(got) != len(want) {
			t.Errorf("case %d: got %d chunks, want %d", i, len(got), len(want))
			continue
		}
		for j := range got {
			if got[j] != want[j] {
				t.Errorf("case %d chunk %d: mismatch\ngot:  %q\nwant: %q", i, j, got[j], want[j])
			}
		}
	}
}

func TestChunkText_Empty(t *testing.T) {
	if got := ChunkText("   \n\n  "); got != nil {
		t.Errorf("expected nil for blank input, got %v", got)
	}
}
