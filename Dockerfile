FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir \
    chromadb \
    pymupdf \
    python-docx \
    tqdm

# Pre-download the default embedding model (all-MiniLM-L6-v2) so it's
# available at runtime without network access.
RUN python -c "from chromadb.utils.embedding_functions import DefaultEmbeddingFunction; DefaultEmbeddingFunction()"

COPY ingest.py .

ENTRYPOINT ["python", "ingest.py"]
