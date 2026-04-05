FROM python:3.12-slim AS builder

RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

RUN pip install --no-cache-dir \
    chromadb-client \
    pymupdf \
    python-docx \
    tqdm \
    && pip uninstall -y pip \
    && find /opt/venv -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null; true

FROM python:3.12-slim

COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

WORKDIR /app
COPY ingest.py .

ENTRYPOINT ["python", "ingest.py"]
