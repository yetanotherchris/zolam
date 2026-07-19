package zolam

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const lockFileName = "ingest.lock"

// acquireLock takes an exclusive, filesystem-based lock on projectDir for
// the lifetime of a single ingest/query invocation. Both backends
// (SQLiteRepo, JsonlRepo) are kept to a single open connection/writer by
// design, so without this, running 'zolam ingest' or 'zolam query' against
// the same project concurrently (e.g. from two terminals) risks a
// corrupted or confusingly-locked index instead of a clear error. Stale
// locks left behind by a crashed/killed process are detected via PID
// liveness and reclaimed automatically.
func acquireLock(projectDir string) (release func(), err error) {
	path := filepath.Join(projectDir, lockFileName)

	for attempts := 0; attempts < 2; attempts++ {
		f, ferr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if ferr == nil {
			fmt.Fprintf(f, "%d", os.Getpid())
			f.Close()
			return func() { os.Remove(path) }, nil
		}
		if !os.IsExist(ferr) {
			return nil, fmt.Errorf("creating lock file %s: %w", path, ferr)
		}

		if pid, perr := readLockPID(path); perr == nil && isProcessAlive(pid) {
			return nil, fmt.Errorf(
				"another zolam ingest/query is already running against this project (pid %d); wait for it to finish and try again",
				pid,
			)
		}
		// The owning process is gone (crash, kill -9, power loss), so the
		// lock is stale. Remove it and retry once rather than blocking
		// this run forever.
		os.Remove(path)
	}

	return nil, fmt.Errorf("could not acquire lock at %s", path)
}

func readLockPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
