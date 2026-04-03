FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir \
    chromadb \
    openai \
    pymupdf \
    python-docx \
    tqdm

COPY ingest.py .

ENTRYPOINT ["python", "ingest.py"]
