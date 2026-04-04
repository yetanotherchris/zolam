FROM python:3.12-slim

WORKDIR /app

RUN pip install --no-cache-dir \
    chromadb \
    pymupdf \
    python-docx \
    tqdm

# Pre-download and extract the embedding model (all-MiniLM-L6-v2) so it's
# available at runtime without network access or re-extraction.
RUN python -c "\
from chromadb.utils.embedding_functions.onnx_mini_lm_l6_v2 import ONNXMiniLM_L6_V2; \
ef = ONNXMiniLM_L6_V2(); \
ef(['warmup'])" \
    && rm -f /root/.cache/chroma/onnx_models/all-MiniLM-L6-v2/onnx.tar.gz

COPY ingest.py .

ENTRYPOINT ["python", "ingest.py"]
