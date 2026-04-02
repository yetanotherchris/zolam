FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir \
    chromadb \
    pymupdf \
    python-docx

COPY ingest.py .

ENTRYPOINT ["python", "ingest.py"]
