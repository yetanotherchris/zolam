package zolam

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yetanotherchris/zolam/internal/domain"
)

// onnxRuntimeVersion pins the onnxruntime release downloaded on first run.
// Must match the ORT_API_VERSION baked into github.com/yalue/onnxruntime_go
// (see its onnxruntime_c_api.h): onnxruntime's GetApi() rejects requests for
// an API version higher than the shared library itself implements, so an
// older runtime than the one the Go bindings target fails to initialise.
// Bump this alongside the onnxruntime_go dependency in go.mod.
const onnxRuntimeVersion = "1.26.0"

// EmbeddingAssetsDir returns <data-dir>/models, where the tokenizer and
// ONNX model weights are cached.
func EmbeddingAssetsDir() (string, error) {
	dataDir, err := domain.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "models"), nil
}

// OnnxRuntimeDir returns <data-dir>/onnxruntime, where the platform's
// onnxruntime shared library is cached.
func OnnxRuntimeDir() (string, error) {
	dataDir, err := domain.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "onnxruntime"), nil
}

// EmbeddingAssets is the set of local paths needed to run the embedding
// model, all downloaded/cached on first use.
type EmbeddingAssets struct {
	TokenizerPath string
	ModelPath     string
	OnnxLibPath   string
}

// EnsureEmbeddingAssets downloads (if not already cached) the tokenizer
// config, ONNX model weights, and the onnxruntime shared library for the
// current OS/arch, reporting progress via outputFn. Mirrors what
// fastembed/uv used to fetch transparently on first ingest.
func EnsureEmbeddingAssets(outputFn func(string)) (*EmbeddingAssets, error) {
	modelsDir, err := EmbeddingAssetsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating models directory: %w", err)
	}

	tokenizerPath := filepath.Join(modelsDir, "tokenizer.json")
	modelPath := filepath.Join(modelsDir, "model.onnx")
	base := "https://huggingface.co/" + domain.DefaultEmbeddingModel + "/resolve/main/"

	if err := downloadIfMissing(tokenizerPath, base+"tokenizer.json", outputFn); err != nil {
		return nil, fmt.Errorf("fetching tokenizer: %w", err)
	}
	if err := downloadIfMissing(modelPath, base+"onnx/model.onnx", outputFn); err != nil {
		return nil, fmt.Errorf("fetching embedding model: %w", err)
	}

	onnxLibPath, err := ensureOnnxRuntime(outputFn)
	if err != nil {
		return nil, fmt.Errorf("fetching onnxruntime: %w", err)
	}

	return &EmbeddingAssets{
		TokenizerPath: tokenizerPath,
		ModelPath:     modelPath,
		OnnxLibPath:   onnxLibPath,
	}, nil
}

func downloadIfMissing(path, url string, outputFn func(string)) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if outputFn != nil {
		outputFn(fmt.Sprintf("Downloading %s...", filepath.Base(path)))
	}
	return downloadFile(path, url)
}

func downloadFile(destPath, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	tmp := destPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, destPath)
}

// onnxRuntimeLibName returns the expected shared-library filename for the
// current OS.
func onnxRuntimeLibName() string {
	switch runtime.GOOS {
	case "windows":
		return "onnxruntime.dll"
	case "darwin":
		return "libonnxruntime.dylib"
	default:
		return "libonnxruntime.so"
	}
}

// onnxRuntimeReleaseAsset returns the Microsoft onnxruntime release archive
// name for the current OS/arch.
func onnxRuntimeReleaseAsset() (string, error) {
	return onnxRuntimeReleaseAssetFor(runtime.GOOS, runtime.GOARCH)
}

// onnxRuntimeReleaseAssetFor returns the Microsoft onnxruntime release
// archive name for the given OS/arch, split out from onnxRuntimeReleaseAsset
// so the platform mapping can be table-tested independent of the host the
// tests run on.
func onnxRuntimeReleaseAssetFor(goos, arch string) (string, error) {
	switch goos {
	case "linux":
		goarch := map[string]string{"amd64": "x64", "arm64": "aarch64"}[arch]
		if goarch == "" {
			return "", fmt.Errorf("unsupported linux arch %s", arch)
		}
		return fmt.Sprintf("onnxruntime-linux-%s-%s.tgz", goarch, onnxRuntimeVersion), nil
	case "darwin":
		if arch == "amd64" {
			// Upstream onnxruntime stopped publishing osx-x86_64 (and
			// osx-universal2) archives after v1.23.0, so Intel Macs can't
			// get a build new enough to satisfy onnxruntime_go's required
			// API version. Only Apple Silicon is downloadable from here.
			return "", fmt.Errorf("onnxruntime %s no longer publishes macOS Intel (x86_64) binaries; Apple Silicon (arm64) is required", onnxRuntimeVersion)
		}
		if arch != "arm64" {
			return "", fmt.Errorf("unsupported darwin arch %s", arch)
		}
		return fmt.Sprintf("onnxruntime-osx-arm64-%s.tgz", onnxRuntimeVersion), nil
	case "windows":
		goarch := map[string]string{"amd64": "x64", "arm64": "arm64"}[arch]
		if goarch == "" {
			return "", fmt.Errorf("unsupported windows arch %s", arch)
		}
		return fmt.Sprintf("onnxruntime-win-%s-%s.zip", goarch, onnxRuntimeVersion), nil
	default:
		return "", fmt.Errorf("unsupported OS %s", goos)
	}
}

func ensureOnnxRuntime(outputFn func(string)) (string, error) {
	baseDir, err := OnnxRuntimeDir()
	if err != nil {
		return "", err
	}
	// Cache under a version-specific subdirectory so bumping
	// onnxRuntimeVersion (e.g. to fix an API version mismatch) always
	// downloads the new shared library instead of silently reusing a stale
	// one left over from a previous zolam version at the old flat path.
	dir := filepath.Join(baseDir, onnxRuntimeVersion)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating onnxruntime directory: %w", err)
	}

	libPath := filepath.Join(dir, onnxRuntimeLibName())
	if _, err := os.Stat(libPath); err == nil {
		return libPath, nil
	}

	asset, err := onnxRuntimeReleaseAsset()
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/%s", onnxRuntimeVersion, asset)

	if outputFn != nil {
		outputFn("Downloading onnxruntime...")
	}
	archivePath := filepath.Join(dir, asset)
	if err := downloadFile(archivePath, url); err != nil {
		return "", err
	}
	defer os.Remove(archivePath)

	if err := extractOnnxRuntimeLib(archivePath, libPath); err != nil {
		return "", err
	}
	return libPath, nil
}

// extractOnnxRuntimeLib pulls just the shared library out of the
// onnxruntime release archive (.tgz on linux/macOS, .zip on Windows) and
// writes it to destPath, discarding the rest of the archive (headers,
// static libs, license files).
func extractOnnxRuntimeLib(archivePath, destPath string) error {
	wantName := filepath.Base(destPath)

	if filepath.Ext(archivePath) == ".zip" {
		r, err := zip.OpenReader(archivePath)
		if err != nil {
			return err
		}
		defer r.Close()
		for _, f := range r.File {
			if filepath.Base(f.Name) == wantName {
				rc, err := f.Open()
				if err != nil {
					return err
				}
				defer rc.Close()
				return writeExtracted(destPath, rc)
			}
		}
		return fmt.Errorf("%s not found in %s", wantName, archivePath)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// Skip versioned symlinks/duplicates (e.g. libonnxruntime.so.1),
		// matching only the exact unversioned shared-library name.
		if filepath.Base(hdr.Name) == wantName && hdr.Typeflag == tar.TypeReg {
			return writeExtracted(destPath, tr)
		}
	}
	return fmt.Errorf("%s not found in %s", wantName, archivePath)
}

func writeExtracted(destPath string, r io.Reader) error {
	tmp := destPath + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, destPath)
}
