FROM python:3.12-slim AS builder

RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

RUN pip install --no-cache-dir \
    chromadb-client \
    onnxruntime \
    tokenizers \
    pymupdf \
    python-docx \
    tqdm \
    && pip uninstall -y pip \
    && find /opt/venv -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null; true

# Pre-download the embedding model so it's available at runtime without network access.
RUN python -c "\
from chromadb.utils.embedding_functions.onnx_mini_lm_l6_v2 import ONNXMiniLM_L6_V2; \
ef = ONNXMiniLM_L6_V2(); \
ef(['warmup'])" \
    && rm -f /root/.cache/chroma/onnx_models/all-MiniLM-L6-v2/onnx.tar.gz

FROM python:3.12-slim

COPY --from=builder /opt/venv /opt/venv
COPY --from=builder /root/.cache/chroma /root/.cache/chroma
ENV PATH="/opt/venv/bin:$PATH"

WORKDIR /app
COPY ingest.py .

ENTRYPOINT ["python", "ingest.py"]
