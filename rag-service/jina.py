import os

import httpx

JINA_API_KEY = os.getenv("JINA_API_KEY", "").strip()
JINA_MODEL = os.getenv("JINA_MODEL", "jina-reranker-v2-base-multilingual")

_enabled = bool(JINA_API_KEY)


def enabled() -> bool:
    return _enabled


def rerank(query: str, documents: list[str], top_n: int) -> list[tuple[int, float]]:
    """Return [(original_index, relevance_score), ...] sorted by score desc, len=top_n."""
    if not _enabled or not documents:
        return [(i, 0.0) for i in range(min(top_n, len(documents)))]
    resp = httpx.post(
        "https://api.jina.ai/v1/rerank",
        headers={
            "Authorization": f"Bearer {JINA_API_KEY}",
            "Content-Type": "application/json",
        },
        json={
            "model": JINA_MODEL,
            "query": query,
            "documents": documents,
            "top_n": top_n,
            "return_documents": False,
        },
        timeout=30.0,
    )
    resp.raise_for_status()
    data = resp.json()
    return [(r["index"], r["relevance_score"]) for r in data.get("results", [])]
