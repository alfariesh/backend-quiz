"""
Voyage AI client — contextualized embeddings (voyage-context-3) + reranker (rerank-2.5).

API schema verified against https://docs.voyageai.com/reference/contextualized-embeddings-api
and https://docs.voyageai.com/reference/reranker-api

Voyage's contextualized embeddings differ from regular embeddings: chunks belonging
to the same document are submitted together (`inputs: [[chunkA1, chunkA2, ...], [docB1, ...]]`)
so each chunk's vector is informed by neighboring chunks in the same document. This
replaces manual tricks like contextual prefix injection or late chunking.

For queries (single short string), we pass as a 1-document, 1-chunk input with
input_type="query". Voyage handles symmetric retrieval internally.
"""
from __future__ import annotations

import os
import httpx

VOYAGE_API_KEY = os.environ["VOYAGE_API_KEY"]
VOYAGE_EMBED_MODEL = os.getenv("VOYAGE_EMBED_MODEL", "voyage-context-3")
VOYAGE_EMBED_DIM = int(os.getenv("VOYAGE_EMBED_DIM", "1024"))
VOYAGE_RERANK_MODEL = os.getenv("VOYAGE_RERANK_MODEL", "rerank-2.5")
VOYAGE_BASE_URL = os.getenv("VOYAGE_BASE_URL", "https://api.voyageai.com/v1")

_timeout_embed = httpx.Timeout(connect=10.0, read=120.0, write=30.0, pool=10.0)
_timeout_rerank = httpx.Timeout(connect=10.0, read=60.0, write=30.0, pool=10.0)


def _headers() -> dict:
    return {
        "Authorization": f"Bearer {VOYAGE_API_KEY}",
        "Content-Type": "application/json",
    }


# ─── embeddings ───────────────────────────────────────────────────

def _post_context_embed(
    inputs: list[list[str]], input_type: str,
) -> list[list[list[float]]]:
    """
    Call voyage contextualized embeddings. Returns [doc_idx][chunk_idx] → vector.
    """
    payload = {
        "inputs": inputs,
        "model": VOYAGE_EMBED_MODEL,
        "input_type": input_type,
        "output_dimension": VOYAGE_EMBED_DIM,
    }
    resp = httpx.post(
        f"{VOYAGE_BASE_URL.rstrip('/')}/contextualizedembeddings",
        headers=_headers(), json=payload, timeout=_timeout_embed,
    )
    resp.raise_for_status()
    body = resp.json()
    out: list[list[list[float]]] = []
    for doc in body.get("data", []):
        doc_chunks = doc.get("data", [])
        out.append([ch["embedding"] for ch in doc_chunks])
    return out


def embed_query(text: str) -> list[float]:
    """Embed a single query string. Returns one vector of VOYAGE_EMBED_DIM floats."""
    result = _post_context_embed([[text]], input_type="query")
    return result[0][0]


def embed_document(chunks: list[str]) -> list[list[float]]:
    """
    Embed all chunks of ONE document. Each chunk's embedding is
    contextually aware of sibling chunks in the same document.

    Use for ingestion: call once per kitab with all its chunks.
    """
    if not chunks:
        return []
    result = _post_context_embed([chunks], input_type="document")
    return result[0]


def embed_documents_batch(docs: list[list[str]]) -> list[list[list[float]]]:
    """Embed multiple documents in one API call. Returns [doc][chunk] → vector."""
    if not docs:
        return []
    return _post_context_embed(docs, input_type="document")


# ─── reranker ─────────────────────────────────────────────────────

def rerank(query: str, documents: list[str], top_n: int) -> list[tuple[int, float]]:
    """
    Return [(original_index, relevance_score), ...] sorted by score desc, len=top_n.
    rerank-2.5 supports 32k context — no need to truncate short chunks.
    """
    if not documents:
        return []
    payload = {
        "query": query,
        "documents": documents,
        "model": VOYAGE_RERANK_MODEL,
        "top_k": min(top_n, len(documents)),
        "return_documents": False,
    }
    resp = httpx.post(
        f"{VOYAGE_BASE_URL.rstrip('/')}/rerank",
        headers=_headers(), json=payload, timeout=_timeout_rerank,
    )
    resp.raise_for_status()
    results = resp.json().get("data", [])
    return [(r["index"], float(r["relevance_score"])) for r in results]
