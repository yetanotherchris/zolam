package zolam

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// prepareCachedEmbeddingAssets points ZOLAM_DATA_DIR at a fresh temp dir
// pre-populated with already-downloaded assets, so tests never hit the
// network (EnsureEmbeddingAssets skips downloading whatever's already
// present).
func prepareCachedEmbeddingAssets(t *testing.T) {
	t.Helper()
	const cacheDir = "/tmp/tokenizerlib"
	required := []string{"tokenizer.json", "model.onnx", "libonnxruntime.so"}
	for _, f := range required {
		if _, err := os.Stat(filepath.Join(cacheDir, f)); err != nil {
			t.Skipf("cached embedding asset %s not present, skipping", f)
		}
	}

	dataDir := t.TempDir()
	t.Setenv("ZOLAM_DATA_DIR", dataDir)

	modelsDir := filepath.Join(dataDir, "models")
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ortDir := filepath.Join(dataDir, "onnxruntime")
	if err := os.MkdirAll(ortDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	copyFile(t, filepath.Join(cacheDir, "tokenizer.json"), filepath.Join(modelsDir, "tokenizer.json"))
	copyFile(t, filepath.Join(cacheDir, "model.onnx"), filepath.Join(modelsDir, "model.onnx"))
	copyFile(t, filepath.Join(cacheDir, "libonnxruntime.so"), filepath.Join(ortDir, onnxRuntimeLibName()))
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", dst, err)
	}
}

func TestEmbedder_SemanticSimilaritySanity(t *testing.T) {
	prepareCachedEmbeddingAssets(t)

	e, err := NewEmbedder(nil)
	if err != nil {
		t.Fatalf("NewEmbedder: %v", err)
	}
	defer e.Close()

	vecs, err := e.Embed([]string{
		"The cat sat on the mat.",
		"A feline rested on the rug.",
		"Quantum entanglement violates local realism.",
	})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
	for i, v := range vecs {
		if len(v) != 384 {
			t.Errorf("vector %d: expected 384 dims, got %d", i, len(v))
		}
	}

	simAB := cosineSimilarity(vecs[0], vecs[1])
	simAC := cosineSimilarity(vecs[0], vecs[2])
	t.Logf("sim(cat/feline)=%.4f sim(cat/quantum)=%.4f", simAB, simAC)
	if simAB <= simAC {
		t.Errorf("expected paraphrase to score higher than unrelated sentence: simAB=%.4f simAC=%.4f", simAB, simAC)
	}
}

func cosineSimilarity(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
