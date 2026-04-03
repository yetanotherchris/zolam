package ingester

import (
	"bytes"
	"fmt"
	"strings"
)

// Stats holds summary information about the ingester environment.
type Stats struct {
	CollectionName  string
	ChromaDBRunning bool
	EmbeddingType   string // "OpenRouter" or "Local"
}

// GetStats checks the ChromaDB status and retrieves collection statistics by
// running the ingest container with the --stats flag.
func (i *Ingester) GetStats(outputFn func(string)) (*Stats, error) {
	stats := &Stats{
		CollectionName: i.config.CollectionName,
	}

	// Determine embedding type from config.
	if i.config.UseLocalEmbeddings {
		stats.EmbeddingType = "Local"
	} else {
		stats.EmbeddingType = "OpenRouter"
	}

	// Check if ChromaDB is running.
	running, _ := i.docker.ChromaDBStatus()
	stats.ChromaDBRunning = running

	// Run the ingest container with --stats to retrieve collection info.
	cmd, err := i.docker.ComposeRun("ingest", "--stats", "--collection", i.config.CollectionName)
	if err != nil {
		return stats, fmt.Errorf("creating stats command: %w", err)
	}

	var buf bytes.Buffer
	if err := i.docker.StreamOutput(cmd, &buf); err != nil {
		// Non-fatal: we still return what we know
		outputFn(fmt.Sprintf("Warning: could not retrieve stats from container: %v", err))
		return stats, nil
	}

	for _, line := range strings.Split(buf.String(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			outputFn(line)
		}
	}

	return stats, nil
}
