#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<EOF
Usage: ./ingest.sh [OPTIONS] <host-directory> [host-directory...]

Ingest local directories into ChromaDB for semantic search.

Options:
  --extensions EXT,EXT,...    Filter by file extensions (e.g. .md,.txt)
  --reset                     Wipe the collection before ingesting
  --stats                     Show collection statistics (no directories needed)
  --collection NAME           Set the collection name (default: from .env or "my notes")
  -h, --help                  Show this help message

Examples:
  ./ingest.sh ~/notes
  ./ingest.sh ~/notes ~/docs
  ./ingest.sh --extensions .md,.txt ~/docs
  ./ingest.sh --reset ~/notes
  ./ingest.sh --stats
EOF
  exit 0
}

DIRS=()
EXTENSIONS=()
RESET=false
STATS=false
COLLECTION=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      ;;
    --stats)
      STATS=true
      shift
      ;;
    --reset)
      RESET=true
      shift
      ;;
    --extensions)
      shift
      IFS=',' read -ra EXTENSIONS <<< "$1"
      shift
      ;;
    --collection)
      shift
      COLLECTION="$1"
      shift
      ;;
    -*)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
    *)
      DIRS+=("$1")
      shift
      ;;
  esac
done

# Ensure ChromaDB is running
docker compose up -d chromadb

if $STATS; then
  docker compose --profile ingest run --rm ingest --stats
  exit 0
fi

if [[ ${#DIRS[@]} -eq 0 ]]; then
  echo "Error: at least one directory is required (or use --stats)" >&2
  echo "Run './ingest.sh --help' for usage." >&2
  exit 1
fi

# Build volume mounts and container paths
VOLUME_ARGS=()
CONTAINER_DIRS=()
for dir in "${DIRS[@]}"; do
  abs_dir="$(cd "$dir" && pwd)"
  name="$(basename "$abs_dir")"
  VOLUME_ARGS+=("-v" "${abs_dir}:/sources/${name}")
  CONTAINER_DIRS+=("/sources/${name}")
done

# Build ingest arguments
INGEST_ARGS=()
if $RESET; then
  INGEST_ARGS+=("--reset")
fi
INGEST_ARGS+=("--directory" "${CONTAINER_DIRS[@]}")
if [[ ${#EXTENSIONS[@]} -gt 0 ]]; then
  INGEST_ARGS+=("--extensions" "${EXTENSIONS[@]}")
fi

# Build extra docker args
DOCKER_ARGS=()
if [[ -n "$COLLECTION" ]]; then
  DOCKER_ARGS+=("-e" "COLLECTION_NAME=${COLLECTION}")
fi

docker compose --profile ingest run --rm \
  "${DOCKER_ARGS[@]}" \
  "${VOLUME_ARGS[@]}" \
  ingest "${INGEST_ARGS[@]}"
