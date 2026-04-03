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

	if i.config.UseLocalEmbeddings {
		stats.EmbeddingType = "Local"
	} else {
		stats.EmbeddingType = "OpenRouter"
	}

	outputFn("Querying Docker for ChromaDB status...")
	running, err := i.docker.ChromaDBStatus()
	if err != nil {
		outputFn(fmt.Sprintf("Error checking ChromaDB status: %v", err))
	}
	stats.ChromaDBRunning = running

	outputFn("Running ingest container with --stats...")
	cmd, err := i.docker.ComposeRun("ingest", nil, []string{"--stats"})
	if err != nil {
		return stats, fmt.Errorf("creating stats command: %w", err)
	}

	var buf bytes.Buffer
	if err := i.docker.StreamOutput(cmd, &buf); err != nil {
		outputFn(fmt.Sprintf("Error retrieving stats from container: %v", err))
		output := strings.TrimSpace(buf.String())
		if output != "" {
			outputFn(fmt.Sprintf("Container output: %s", output))
		}
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
