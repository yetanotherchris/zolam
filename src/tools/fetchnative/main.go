// Command fetchnative downloads the prebuilt libtokenizers static library
// (github.com/daulet/tokenizers has no pure-Go option, and its Go module
// doesn't vendor the library the way go-fitz/duckdb-go do) into native/,
// where embedder.go's cgo directive points the linker. Run this once
// before building zolam with CGO enabled; CI runs it as a build step.
package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// tokenizersVersion pins the daulet/tokenizers release this fetches.
// Bump alongside the github.com/daulet/tokenizers Go module version.
const tokenizersVersion = "v1.27.0"

func releaseAssetName(goos, goarch string) (string, error) {
	var platform string
	switch goos {
	case "linux":
		m := map[string]string{"amd64": "linux-amd64", "arm64": "linux-arm64"}
		platform = m[goarch]
	case "darwin":
		m := map[string]string{"amd64": "darwin-x86_64", "arm64": "darwin-arm64"}
		platform = m[goarch]
	case "windows":
		m := map[string]string{"amd64": "windows-amd64"}
		platform = m[goarch]
	}
	if platform == "" {
		return "", fmt.Errorf("no prebuilt libtokenizers for %s/%s", goos, goarch)
	}
	return fmt.Sprintf("libtokenizers.%s.tar.gz", platform), nil
}

func main() {
	outDir := flag.String("out", "native/tokenizers", "directory to extract libtokenizers.a into")
	flag.Parse()

	asset, err := releaseAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetchnative:", err)
		os.Exit(1)
	}

	destLib := filepath.Join(*outDir, "libtokenizers.a")
	if _, err := os.Stat(destLib); err == nil {
		fmt.Println("fetchnative: libtokenizers.a already present, skipping download")
		return
	}

	url := fmt.Sprintf("https://github.com/daulet/tokenizers/releases/download/%s/%s", tokenizersVersion, asset)
	fmt.Println("fetchnative: downloading", url)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "fetchnative:", err)
		os.Exit(1)
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetchnative:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "fetchnative: GET %s: unexpected status %s\n", url, resp.Status)
		os.Exit(1)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetchnative:", err)
		os.Exit(1)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			fmt.Fprintln(os.Stderr, "fetchnative: libtokenizers.a not found in archive")
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "fetchnative:", err)
			os.Exit(1)
		}
		if filepath.Base(hdr.Name) != "libtokenizers.a" {
			continue
		}
		tmp := destLib + ".tmp"
		out, err := os.Create(tmp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fetchnative:", err)
			os.Exit(1)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			os.Remove(tmp)
			fmt.Fprintln(os.Stderr, "fetchnative:", err)
			os.Exit(1)
		}
		out.Close()
		if err := os.Rename(tmp, destLib); err != nil {
			fmt.Fprintln(os.Stderr, "fetchnative:", err)
			os.Exit(1)
		}
		fmt.Println("fetchnative: wrote", destLib)
		return
	}
}
