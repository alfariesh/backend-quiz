"""
A/B retrieval quality test: compare voyage-context-3 vs voyage-4 (and -large) on the same kitab.

Usage:
    VOYAGE_API_KEY=pa-xxx python scripts/ab_test.py <pdf> "q1" "q2" ...

Embeds the same chunks with each model, runs the same queries, reranks with rerank-2.5,
and prints a side-by-side comparison: top-5 per model + avg rerank score per query.

Endpoints:
  voyage-context-3     → /contextualizedembeddings   (chunks aware of siblings)
  voyage-4, -4-large   → /embeddings                 (standard, per-text)
"""
from __future__ import annotations

import math
import os
import re
import sys
import time
from pathlib import Path

import httpx

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

import fitz  # noqa: E402

API_KEY = os.environ["VOYAGE_API_KEY"]
BASE_URL = os.getenv("VOYAGE_BASE_URL", "https://api.voyageai.com/v1")

MODELS = [
    ("voyage-context-3", "contextualized"),
    ("voyage-4", "standard"),
    ("voyage-4-large", "standard"),
]

CHUNK_TARGET_CHARS = 1200
EMBED_BATCH = 25       # chunks per request (fits 32K context window for Arabic)
DENSE_K = 20
RERANK_K = 5
EMBED_DIM = 1024

SENT_SPLIT = re.compile(r"(?<=[\.\?\!۔؟؛])\s+|\n{2,}")
TIMEOUT = httpx.Timeout(connect=10.0, read=120.0, write=30.0, pool=10.0)


def headers() -> dict:
    return {"Authorization": f"Bearer {API_KEY}", "Content-Type": "application/json"}


# ─── chunking (same as smoke_test) ─────────────────────────────────

def extract_pages(path: str) -> list[tuple[int, str]]:
    doc = fitz.open(path)
    out = [(i, p.get_text("text").strip()) for i, p in enumerate(doc, 1)]
    doc.close()
    return [(i, t) for i, t in out if t]


def chunk_pages(pages: list[tuple[int, str]]) -> list[dict]:
    chunks, buf, buf_len, start_pg = [], [], 0, None
    last_pg = pages[0][0] if pages else 0

    def flush(end):
        nonlocal buf, buf_len, start_pg
        if not buf:
            return
        chunks.append({"content": " ".join(buf).strip(),
                       "start_page": start_pg, "end_page": end})
        overlap = buf[-1:]
        buf = list(overlap)
        buf_len = sum(len(s) for s in buf)
        start_pg = end if overlap else None

    for pg, text in pages:
        last_pg = pg
        for s in (s.strip() for s in SENT_SPLIT.split(text) if s.strip()):
            if start_pg is None:
                start_pg = pg
            buf.append(s)
            buf_len += len(s)
            if buf_len >= CHUNK_TARGET_CHARS:
                flush(pg)
    flush(last_pg)
    return chunks


# ─── voyage calls ──────────────────────────────────────────────────

def post_with_retry(url: str, payload: dict) -> dict:
    for attempt in range(5):
        try:
            r = httpx.post(url, headers=headers(), json=payload, timeout=TIMEOUT)
            r.raise_for_status()
            return r.json()
        except httpx.HTTPStatusError as e:
            if e.response.status_code == 429 and attempt < 4:
                wait = 2 ** (attempt + 2)
                print(f"   [429] backoff {wait}s")
                time.sleep(wait)
                continue
            print(f"   [HTTP {e.response.status_code}] {e.response.text[:200]}")
            raise


def embed_contextualized(chunks: list[str], model: str, input_type: str) -> list[list[float]]:
    body = post_with_retry(f"{BASE_URL.rstrip('/')}/contextualizedembeddings", {
        "inputs": [chunks], "model": model, "input_type": input_type,
        "output_dimension": EMBED_DIM,
    })
    return [c["embedding"] for c in body["data"][0]["data"]]


def embed_standard(texts: list[str], model: str, input_type: str) -> list[list[float]]:
    body = post_with_retry(f"{BASE_URL.rstrip('/')}/embeddings", {
        "input": texts, "model": model, "input_type": input_type,
        "output_dimension": EMBED_DIM,
    })
    return [d["embedding"] for d in body["data"]]


def rerank(query: str, docs: list[str], top_n: int) -> list[tuple[int, float]]:
    body = post_with_retry(f"{BASE_URL.rstrip('/')}/rerank", {
        "query": query, "documents": docs, "model": "rerank-2.5",
        "top_k": min(top_n, len(docs)), "return_documents": False,
    })
    return [(r["index"], float(r["relevance_score"])) for r in body["data"]]


def embed_doc(chunks: list[str], model: str, kind: str) -> list[list[float]]:
    """Batch embed all chunks of one doc using the right endpoint."""
    out = []
    for i in range(0, len(chunks), EMBED_BATCH):
        window = chunks[i : i + EMBED_BATCH]
        if kind == "contextualized":
            vecs = embed_contextualized(window, model, "document")
        else:
            vecs = embed_standard(window, model, "document")
        out.extend(vecs)
        print(f"   batch {i // EMBED_BATCH + 1}: {len(vecs)} vectors")
    return out


def embed_q(text: str, model: str, kind: str) -> list[float]:
    if kind == "contextualized":
        return embed_contextualized([text], model, "query")[0]
    return embed_standard([text], model, "query")[0]


def cosine(a, b):
    dot = sum(x * y for x, y in zip(a, b))
    na = math.sqrt(sum(x * x for x in a))
    nb = math.sqrt(sum(y * y for y in b))
    return dot / (na * nb) if na and nb else 0.0


# ─── main ──────────────────────────────────────────────────────────

def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)
    pdf, queries = sys.argv[1], sys.argv[2:]

    pages = extract_pages(pdf)
    chunks = chunk_pages(pages)
    print(f"PDF: {len(pages)} pages, {len(chunks)} chunks "
          f"(~{sum(len(c['content']) for c in chunks) // 4} tokens est.)\n")

    # model_name → list[vector]
    embeddings: dict[str, list[list[float]]] = {}
    for model, kind in MODELS:
        print(f"=== Embedding with {model} ({kind})")
        try:
            embeddings[model] = embed_doc([c["content"] for c in chunks], model, kind)
        except Exception as e:
            print(f"   FAILED: {e}\n")
            continue
        print()

    # Per query: rank with each model, rerank with rerank-2.5, compare top-5.
    for q in queries:
        print(f"\n{'='*70}\nQUERY: {q!r}\n{'='*70}")
        results_per_model: dict[str, list[tuple[int, float]]] = {}
        for model, kind in MODELS:
            if model not in embeddings:
                continue
            qv = embed_q(q, model, kind)
            scored = sorted(((cosine(qv, v), i) for i, v in enumerate(embeddings[model])),
                            reverse=True)[:DENSE_K]
            cand_idxs = [i for _, i in scored]
            ranked = rerank(q, [chunks[i]["content"] for i in cand_idxs], top_n=RERANK_K)
            results_per_model[model] = [(cand_idxs[li], score) for li, score in ranked]

        # Side-by-side print
        for model in [m for m, _ in MODELS if m in results_per_model]:
            avg = sum(s for _, s in results_per_model[model]) / len(results_per_model[model])
            print(f"\n--- {model}  (avg rerank score: {avg:.4f})")
            for r, (idx, score) in enumerate(results_per_model[model], 1):
                ch = chunks[idx]
                snippet = ch["content"].replace("\n", " ")[:130]
                print(f"  {r}. pp.{ch['start_page']}-{ch['end_page']}  "
                      f"score={score:.4f}\n     {snippet}…")


if __name__ == "__main__":
    main()
