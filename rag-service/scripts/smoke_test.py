"""
End-to-end smoke test for the Voyage retrieval pipeline — no Postgres required.

Usage:
    VOYAGE_API_KEY=pa-xxx python scripts/smoke_test.py <pdf_path> "query 1" "query 2" ...

What it does:
    1. Extracts text per page from the PDF (PyMuPDF).
    2. Sentence-aware chunks the text (Arabic + Latin punctuation).
    3. Embeds all chunks in batches via voyage-context-3 (contextualized).
    4. For each query: embeds the query, cosine ranks the top-20, reranks via rerank-2.5,
       prints top-5 with page numbers + scores + a short snippet.

Cost guardrail: prints estimated input tokens before sending to Voyage.
"""
from __future__ import annotations

import math
import os
import re
import sys
import time
from pathlib import Path

import httpx

# Allow `import voyage` from the rag-service root.
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

import fitz  # PyMuPDF
import voyage  # noqa: E402

CHUNK_TARGET_CHARS = 1200    # ~300-400 tokens for Arabic; comfortable for context-3
CHUNK_OVERLAP_SENTS = 1
EMBED_BATCH = int(os.getenv("EMBED_BATCH", "30"))  # chunks per contextualized request
DENSE_K = 20                 # cosine top-k before rerank
RERANK_K = 5

# Sentence ends: Arabic full stop ۔, Arabic question ؟, Arabic semicolon ؛, Latin . ! ?
SENT_SPLIT = re.compile(r"(?<=[\.\?\!۔؟؛])\s+|\n{2,}")


def extract_pages(path: str) -> list[tuple[int, str]]:
    doc = fitz.open(path)
    pages = []
    for i, page in enumerate(doc, start=1):
        text = page.get_text("text").strip()
        if text:
            pages.append((i, text))
    doc.close()
    return pages


def chunk_pages(pages: list[tuple[int, str]]) -> list[dict]:
    """Sentence-aware chunks. Each chunk records the start/end page it spans."""
    chunks: list[dict] = []
    buf: list[str] = []
    buf_len = 0
    buf_start_page: int | None = None
    last_page = pages[0][0] if pages else 0

    def flush(end_page: int):
        nonlocal buf, buf_len, buf_start_page
        if not buf:
            return
        chunks.append({
            "content": " ".join(buf).strip(),
            "start_page": buf_start_page,
            "end_page": end_page,
        })
        # Carry overlap sentences for context continuity.
        overlap = buf[-CHUNK_OVERLAP_SENTS:] if CHUNK_OVERLAP_SENTS > 0 else []
        buf = list(overlap)
        buf_len = sum(len(s) for s in buf)
        buf_start_page = end_page if overlap else None

    for page_no, text in pages:
        last_page = page_no
        sents = [s.strip() for s in SENT_SPLIT.split(text) if s.strip()]
        for s in sents:
            if buf_start_page is None:
                buf_start_page = page_no
            buf.append(s)
            buf_len += len(s)
            if buf_len >= CHUNK_TARGET_CHARS:
                flush(page_no)
    flush(last_page)
    return chunks


def cosine(a: list[float], b: list[float]) -> float:
    dot = sum(x * y for x, y in zip(a, b))
    na = math.sqrt(sum(x * x for x in a))
    nb = math.sqrt(sum(y * y for y in b))
    if na == 0 or nb == 0:
        return 0.0
    return dot / (na * nb)


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)
    pdf_path = sys.argv[1]
    queries = sys.argv[2:]
    max_chunks_env = os.getenv("MAX_CHUNKS")
    max_chunks = int(max_chunks_env) if max_chunks_env else None

    print(f"\n=== Extracting {pdf_path}")
    pages = extract_pages(pdf_path)
    print(f"  pages with text: {len(pages)}")

    chunks = chunk_pages(pages)
    if max_chunks:
        # Sample evenly across the book so retrieval covers different topics.
        step = max(1, len(chunks) // max_chunks)
        chunks = chunks[::step][:max_chunks]
    total_chars = sum(len(c["content"]) for c in chunks)
    print(f"  chunks: {len(chunks)}  (~{total_chars // 4} tokens estimated)")
    if chunks:
        sample = chunks[0]
        print(f"  first chunk pp.{sample['start_page']}-{sample['end_page']}: "
              f"{sample['content'][:120]}…")

    print(f"\n=== Embedding {len(chunks)} chunks via {voyage.VOYAGE_EMBED_MODEL}")
    vectors: list[list[float]] = []
    for i in range(0, len(chunks), EMBED_BATCH):
        window = chunks[i : i + EMBED_BATCH]
        texts = [c["content"] for c in window]
        vecs = None
        for attempt in range(5):
            try:
                vecs = voyage.embed_document(texts)
                break
            except httpx.HTTPStatusError as e:
                body = e.response.text[:300]
                if e.response.status_code == 429 and attempt < 4:
                    wait = 2 ** (attempt + 2)  # 4, 8, 16, 32
                    print(f"  [429] body={body!r} backing off {wait}s (attempt {attempt+1}/5)")
                    time.sleep(wait)
                    continue
                print(f"  [HTTP {e.response.status_code}] body={body!r}")
                raise
        if vecs is None or len(vecs) != len(texts):
            raise RuntimeError(f"voyage returned {len(vecs) if vecs else 0} vectors for {len(texts)} chunks")
        vectors.extend(vecs)
        print(f"  batch {i // EMBED_BATCH + 1}: {len(vecs)} vectors  "
              f"(dim={len(vecs[0]) if vecs else 0})")
        time.sleep(8)  # ~7-8 RPM safe pace for free tier

    for q in queries:
        print(f"\n=== Query: {q!r}")
        qvec = voyage.embed_query(q)

        scored = sorted(
            ((cosine(qvec, v), idx) for idx, v in enumerate(vectors)),
            reverse=True,
        )[:DENSE_K]
        cand_idxs = [idx for _, idx in scored]

        docs = [chunks[i]["content"] for i in cand_idxs]
        ranked = voyage.rerank(q, docs, top_n=RERANK_K)

        print(f"  top-{RERANK_K} after rerank-2.5:")
        for rank, (local_i, score) in enumerate(ranked, 1):
            ch = chunks[cand_idxs[local_i]]
            snippet = ch["content"].replace("\n", " ")[:160]
            print(f"   {rank}. pp.{ch['start_page']}-{ch['end_page']}  "
                  f"score={score:.4f}\n      {snippet}…")


if __name__ == "__main__":
    main()
