package zolam

import (
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/daulet/tokenizers"
	ort "github.com/yalue/onnxruntime_go"

	"github.com/yetanotherchris/zolam/internal/domain"
)

var ortInitOnce sync.Once
var ortInitErr error

// maxSeqLen is bge-small-en-v1.5's max_position_embeddings. The ONNX
// graph's positional-embedding Add node is sized against this, so any
// input longer than this many tokens fails at Run with a broadcast error
// rather than a clean truncation.
const maxSeqLen = 512

// Embedder turns text into normalised embedding vectors using the ONNX
// export of the project's embedding model (BAAI/bge-small-en-v1.5, CLS
// pooling, 384 dims). Safe for concurrent use from multiple goroutines:
// the underlying onnxruntime session supports concurrent Run calls, and
// the tokenizer has no mutable per-call state.
type Embedder struct {
	tokenizer *tokenizers.Tokenizer
	session   *ort.DynamicAdvancedSession
	dims      int
}

// NewEmbedder downloads embedding assets on first use (see
// EnsureEmbeddingAssets) and loads the tokenizer and ONNX session.
func NewEmbedder(outputFn func(string)) (*Embedder, error) {
	assets, err := EnsureEmbeddingAssets(outputFn)
	if err != nil {
		return nil, err
	}

	ortInitOnce.Do(func() {
		ort.SetSharedLibraryPath(assets.OnnxLibPath)
		ortInitErr = ort.InitializeEnvironment()
	})
	if ortInitErr != nil {
		return nil, fmt.Errorf("initialising onnxruntime: %w", ortInitErr)
	}

	tokenizerData, err := os.ReadFile(assets.TokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("reading tokenizer: %w", err)
	}
	tk, err := tokenizers.FromBytesWithTruncation(tokenizerData, maxSeqLen, tokenizers.TruncationDirectionRight)
	if err != nil {
		return nil, fmt.Errorf("loading tokenizer: %w", err)
	}

	session, err := ort.NewDynamicAdvancedSession(assets.ModelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		nil,
	)
	if err != nil {
		tk.Close()
		return nil, fmt.Errorf("loading onnx model: %w", err)
	}

	return &Embedder{tokenizer: tk, session: session, dims: domain.DefaultEmbeddingDims}, nil
}

// Close releases the tokenizer and ONNX session.
func (e *Embedder) Close() {
	if e.tokenizer != nil {
		e.tokenizer.Close()
	}
	if e.session != nil {
		e.session.Destroy()
	}
}

// Embed returns one normalised 384-dim vector per input text (CLS pooling
// over the model's last_hidden_state, then L2 normalisation, matching how
// sentence-transformers wraps this model).
func (e *Embedder) Embed(texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, text := range texts {
		v, err := e.embedOne(text)
		if err != nil {
			return nil, fmt.Errorf("embedding text %d: %w", i, err)
		}
		out[i] = v
	}
	return out, nil
}

func (e *Embedder) embedOne(text string) ([]float32, error) {
	ids, _ := e.tokenizer.Encode(text, true)
	n := len(ids)
	if n == 0 {
		return make([]float32, e.dims), nil
	}

	inputIDs := make([]int64, n)
	attnMask := make([]int64, n)
	tokenType := make([]int64, n)
	for i, id := range ids {
		inputIDs[i] = int64(id)
		attnMask[i] = 1
	}

	shape := ort.NewShape(1, int64(n))
	inputIDsTensor, err := ort.NewTensor(shape, inputIDs)
	if err != nil {
		return nil, err
	}
	defer inputIDsTensor.Destroy()
	attnTensor, err := ort.NewTensor(shape, attnMask)
	if err != nil {
		return nil, err
	}
	defer attnTensor.Destroy()
	tokenTypeTensor, err := ort.NewTensor(shape, tokenType)
	if err != nil {
		return nil, err
	}
	defer tokenTypeTensor.Destroy()

	outputShape := ort.NewShape(1, int64(n), int64(e.dims))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, err
	}
	defer outputTensor.Destroy()

	if err := e.session.Run(
		[]ort.Value{inputIDsTensor, attnTensor, tokenTypeTensor},
		[]ort.Value{outputTensor},
	); err != nil {
		return nil, err
	}

	// CLS pooling: the first token's vector is the sentence embedding.
	data := outputTensor.GetData()
	cls := make([]float32, e.dims)
	copy(cls, data[:e.dims])
	return normalizeVector(cls), nil
}

func normalizeVector(v []float32) []float32 {
	var sumSq float64
	for _, x := range v {
		sumSq += float64(x) * float64(x)
	}
	if sumSq == 0 {
		return v
	}
	norm := math.Sqrt(sumSq)
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out
}
