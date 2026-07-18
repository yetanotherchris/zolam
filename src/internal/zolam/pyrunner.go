package zolam

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/yetanotherchris/zolam/internal/domain"
)

//go:embed pyscripts/ingest.py
var scriptFS embed.FS

const scriptName = "ingest.py"

// EnsureUV verifies that uv can be found (via PATH or a well-known install
// location), returning a one-line remedy if not.
func EnsureUV() error {
	_, err := findUV()
	return err
}

// findUV locates the uv executable. It checks PATH first, then falls back
// to uv's well-known install directories: some launchers (editors, agent
// runners like OpenCode/Claude Code) spawn subprocesses with a PATH that
// doesn't include the same directories a login shell would, even though
// `uv` itself is installed and works fine interactively.
func findUV() (string, error) {
	if path, err := exec.LookPath("uv"); err == nil {
		return path, nil
	}

	name := "uv"
	if runtime.GOOS == "windows" {
		name = "uv.exe"
	}
	for _, dir := range uvFallbackDirs() {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("uv is required but not found on PATH: install it with " +
		"'brew install uv' (macOS/Linux), 'winget install astral-sh.uv' or " +
		"'scoop install uv' (Windows), or see https://docs.astral.sh/uv/getting-started/installation/")
}

// uvFallbackDirs lists directories uv's official installers commonly place
// the binary in, beyond whatever a subprocess's PATH happens to contain.
func uvFallbackDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		return []string{
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Links"),
			filepath.Join(home, "scoop", "shims"),
		}
	}
	return []string{
		filepath.Join(home, ".local", "bin"), // astral.sh install script default
		filepath.Join(home, ".cargo", "bin"), // older uv installs
		"/opt/homebrew/bin",                  // Homebrew on Apple Silicon
		"/usr/local/bin",                     // Homebrew on Intel macOS / Linux
		"/home/linuxbrew/.linuxbrew/bin",     // Homebrew on Linux
	}
}

// ScriptsDir returns <data-dir>/scripts.
func ScriptsDir() (string, error) {
	dataDir, err := domain.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "scripts"), nil
}

// EnsureScripts writes the embedded ingest.py to <data-dir>/scripts,
// rewriting it whenever the embedded content's hash differs from the
// ".version" marker left on disk. Returns the path to the script.
func EnsureScripts() (string, error) {
	dir, err := ScriptsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating scripts directory: %w", err)
	}

	content, err := scriptFS.ReadFile("pyscripts/" + scriptName)
	if err != nil {
		return "", fmt.Errorf("reading embedded script: %w", err)
	}
	sum := sha256.Sum256(content)
	wantHash := hex.EncodeToString(sum[:])

	scriptPath := filepath.Join(dir, scriptName)
	versionPath := filepath.Join(dir, ".version")

	current, _ := os.ReadFile(versionPath)
	if string(current) != wantHash {
		if err := os.WriteFile(scriptPath, content, 0o644); err != nil {
			return "", fmt.Errorf("writing %s: %w", scriptPath, err)
		}
		if err := os.WriteFile(versionPath, []byte(wantHash), 0o644); err != nil {
			return "", fmt.Errorf("writing script version marker: %w", err)
		}
	}

	return scriptPath, nil
}

// IngestSummary is the final JSON object ingest.py prints to stdout for
// --mode ingest/update.
type IngestSummary struct {
	FilesProcessed int      `json:"files_processed"`
	FilesErrored   int      `json:"files_errored"`
	FilesRemoved   int      `json:"files_removed"`
	ChunksWritten  int      `json:"chunks_written"`
	Errors         []string `json:"errors"`
}

// QueryHit is a single ranked result from --mode query.
type QueryHit struct {
	Path  string   `json:"path"`
	Chunk int      `json:"chunk"`
	Page  *int     `json:"page"`
	Text  string   `json:"text"`
	Score *float64 `json:"score"`
}

// QueryResponse is the final JSON object ingest.py prints to stdout for
// --mode query.
type QueryResponse struct {
	Results []QueryHit `json:"results"`
}

// runPythonScript invokes `uv run <script> <args...>`, streaming stderr
// lines live to outputFn while capturing stdout in full. On failure the
// returned error includes the last few lines of stderr for context.
func runPythonScript(args []string, outputFn func(string)) ([]byte, error) {
	uvBin, err := findUV()
	if err != nil {
		return nil, err
	}
	scriptPath, err := EnsureScripts()
	if err != nil {
		return nil, err
	}

	cmdArgs := append([]string{"run", scriptPath}, args...)
	cmd := exec.Command(uvBin, cmdArgs...)
	// Windows without Developer Mode/admin can't create symlinks, so the
	// huggingface_hub cache falls back to copies; that's expected here and
	// not worth surfacing as a warning on every ingest.
	cmd.Env = append(os.Environ(), "HF_HUB_DISABLE_SYMLINKS_WARNING=1")
	setProcessGroup(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting uv run %s: %w", scriptPath, err)
	}

	// uv run spawns python (and its own subprocesses) as a genuine child
	// tree; without this, Ctrl+C only kills the zolam process and leaves
	// ingest.py running as an orphan.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	done := make(chan struct{})
	go func() {
		select {
		case <-sigCh:
			killProcessTree(cmd)
		case <-done:
		}
	}()
	defer func() {
		signal.Stop(sigCh)
		close(done)
	}()

	var outBuf bytes.Buffer
	var lastLines []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(&outBuf, stdout)
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			mu.Lock()
			if outputFn != nil {
				outputFn(line)
			}
			lastLines = append(lastLines, line)
			if len(lastLines) > 20 {
				lastLines = lastLines[1:]
			}
			mu.Unlock()
		}
	}()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		mu.Lock()
		detail := strings.Join(lastLines, "\n")
		mu.Unlock()
		if detail != "" {
			return nil, fmt.Errorf("ingest.py failed: %w\n%s", err, detail)
		}
		return nil, fmt.Errorf("ingest.py failed: %w", err)
	}

	return outBuf.Bytes(), nil
}

// lastJSONLine returns the last non-empty line of stdout, which by
// contract is always the script's single JSON result object.
func lastJSONLine(out []byte) []byte {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return []byte(lines[i])
		}
	}
	return out
}

// RunIngest invokes ingest.py in ingest/update mode for the given project.
func RunIngest(project *domain.Project, projectDir string, files, removed []string, outputFn func(string)) (*IngestSummary, error) {
	args := []string{
		"--mode", "ingest",
		"--project-dir", projectDir,
		"--backend", project.Backend,
		"--embedding-model", project.EmbeddingModel,
		"--embedding-dims", strconv.Itoa(project.EmbeddingDims),
	}
	if len(files) > 0 {
		args = append(args, "--files")
		args = append(args, files...)
	}
	if len(removed) > 0 {
		args = append(args, "--removed")
		args = append(args, removed...)
	}

	out, err := runPythonScript(args, outputFn)
	if err != nil {
		return nil, err
	}

	var summary IngestSummary
	if err := json.Unmarshal(lastJSONLine(out), &summary); err != nil {
		return nil, fmt.Errorf("parsing ingest.py output: %w", err)
	}
	return &summary, nil
}

// RunQuery invokes ingest.py in query mode for the given project.
func RunQuery(project *domain.Project, projectDir, queryText string, topK int, keyword bool, outputFn func(string)) (*QueryResponse, error) {
	args := []string{
		"--mode", "query",
		"--project-dir", projectDir,
		"--backend", project.Backend,
		"--embedding-model", project.EmbeddingModel,
		"--embedding-dims", strconv.Itoa(project.EmbeddingDims),
		"--query", queryText,
		"--top-k", strconv.Itoa(topK),
	}
	if keyword {
		args = append(args, "--keyword")
	}

	out, err := runPythonScript(args, outputFn)
	if err != nil {
		return nil, err
	}

	var resp QueryResponse
	if err := json.Unmarshal(lastJSONLine(out), &resp); err != nil {
		return nil, fmt.Errorf("parsing ingest.py output: %w", err)
	}
	return &resp, nil
}
