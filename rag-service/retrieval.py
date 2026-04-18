"""
Hybrid retrieval: dense (voyage-context-3 embeddings) + lexical (tsvector ts_rank),
fused via Reciprocal Rank Fusion in SQL, then reranked with voyage rerank-2.5.

Tree reasoning has been removed from the retrieval path — kitab_tree remains in
the DB for UI navigation only. The hybrid path handles structural and semantic
queries alike without a per-query LLM call.

Two public functions:
    retrieve_per_kitab(kitab_id, question) — scoped to one kitab
    retrieve_general(question)             — across all published+embedded kitab
"""
from __future__ import annotations

import os
from dataclasses import dataclass, field

import voyage
from db import conn

# Candidate counts before rerank. 40 each gives ~50-70 unique after RRF fusion.
DENSE_CANDIDATES = int(os.getenv("DENSE_CANDIDATES", "40"))
LEXICAL_CANDIDATES = int(os.getenv("LEXICAL_CANDIDATES", "40"))
RRF_K = int(os.getenv("RRF_K", "60"))          # RRF smoothing constant
RERANK_TOPK = int(os.getenv("RERANK_TOPK", "8"))
RERANK_ENABLED = os.getenv("RERANK_ENABLED", "true").lower() == "true"


@dataclass
class RetrievedChunk:
    chunk_id: int
    kitab_id: str
    kitab_title: str
    content: str
    start_page: int
    end_page: int
    tree_node_db_id: str
    tree_node_id: str
    tree_title: str
    score: float | None = None


@dataclass
class RetrievalResult:
    strategy: str
    reasoning: str
    picked_tree_node_ids: list[str] = field(default_factory=list)
    chunks: list[RetrievedChunk] = field(default_factory=list)


# ─── helpers ──────────────────────────────────────────────────────

def _pick_text(j) -> str:
    if not j:
        return ""
    if isinstance(j, str):
        return j
    for key in ("id", "en", "ar"):
        if j.get(key):
            return j[key]
    return next(iter(j.values()), "") if j else ""


def _row_to_chunk(r, score: float | None = None) -> RetrievedChunk:
    """
    Row shape (10 cols): chunk_id, kitab_id, kitab_title, content,
    start_page, end_page, tree_uuid, tree_node_id, tree_title, fused_score.
    tree_* may be NULL for flat (tree-less) chunks.
    """
    return RetrievedChunk(
        chunk_id=r[0],
        kitab_id=str(r[1]),
        kitab_title=_pick_text(r[2]),
        content=r[3],
        start_page=r[4],
        end_page=r[5],
        tree_node_db_id=str(r[6]) if r[6] is not None else "",
        tree_node_id=r[7] or "",
        tree_title=_pick_text(r[8]) if r[8] is not None else "",
        score=score if score is not None else (float(r[9]) if r[9] is not None else None),
    )


# ─── core hybrid query ────────────────────────────────────────────

# Shared tail: join fused chunk ids back to full metadata.
_SELECT_COLUMNS = """
    kc.id, kc.kitab_id, k.title, kc.content, kc.start_page, kc.end_page,
    t.id, t.node_id, t.title, f.rrf_score
"""


def _hybrid_per_kitab(kitab_id: str, query_vec: list[float], question: str) -> list[RetrievedChunk]:
    sql = f"""
    WITH
    dense AS (
      SELECT id,
             ROW_NUMBER() OVER (ORDER BY embedding_voyage <=> %(qvec)s::vector) AS rank
      FROM kitab_chunks
      WHERE kitab_id = %(kid)s
        AND embedding_voyage IS NOT NULL
      ORDER BY embedding_voyage <=> %(qvec)s::vector
      LIMIT %(dense_n)s
    ),
    lex AS (
      SELECT id,
             ROW_NUMBER() OVER (ORDER BY ts_rank(content_tsv, tsq) DESC) AS rank
      FROM (
        SELECT id, content_tsv, plainto_tsquery('simple', %(q)s) AS tsq
        FROM kitab_chunks
        WHERE kitab_id = %(kid)s
      ) s
      WHERE content_tsv @@ tsq
      ORDER BY ts_rank(content_tsv, tsq) DESC
      LIMIT %(lex_n)s
    ),
    fused AS (
      SELECT id, SUM(1.0 / (%(k)s + rank)) AS rrf_score
      FROM (SELECT id, rank FROM dense UNION ALL SELECT id, rank FROM lex) u
      GROUP BY id
      ORDER BY rrf_score DESC
      LIMIT %(fuse_n)s
    )
    SELECT {_SELECT_COLUMNS}
    FROM fused f
    JOIN kitab_chunks kc ON kc.id = f.id
    JOIN kitab k ON k.id = kc.kitab_id
    LEFT JOIN kitab_tree t ON t.id = kc.tree_node_id
    ORDER BY f.rrf_score DESC
    """
    params = {
        "qvec": query_vec, "kid": kitab_id, "q": question,
        "dense_n": DENSE_CANDIDATES, "lex_n": LEXICAL_CANDIDATES,
        "k": RRF_K, "fuse_n": DENSE_CANDIDATES + LEXICAL_CANDIDATES,
    }
    with conn() as c, c.cursor() as cur:
        cur.execute(sql, params)
        return [_row_to_chunk(r) for r in cur.fetchall()]


def _hybrid_general(query_vec: list[float], question: str) -> list[RetrievedChunk]:
    sql = f"""
    WITH
    dense AS (
      SELECT kc.id,
             ROW_NUMBER() OVER (ORDER BY kc.embedding_voyage <=> %(qvec)s::vector) AS rank
      FROM kitab_chunks kc
      JOIN kitab k ON k.id = kc.kitab_id
      WHERE k.is_published = true
        AND k.embeddings_processed = true
        AND kc.embedding_voyage IS NOT NULL
      ORDER BY kc.embedding_voyage <=> %(qvec)s::vector
      LIMIT %(dense_n)s
    ),
    lex AS (
      SELECT id,
             ROW_NUMBER() OVER (ORDER BY ts_rank(content_tsv, tsq) DESC) AS rank
      FROM (
        SELECT kc.id, kc.content_tsv, plainto_tsquery('simple', %(q)s) AS tsq
        FROM kitab_chunks kc
        JOIN kitab k ON k.id = kc.kitab_id
        WHERE k.is_published = true AND k.embeddings_processed = true
      ) s
      WHERE content_tsv @@ tsq
      ORDER BY ts_rank(content_tsv, tsq) DESC
      LIMIT %(lex_n)s
    ),
    fused AS (
      SELECT id, SUM(1.0 / (%(k)s + rank)) AS rrf_score
      FROM (SELECT id, rank FROM dense UNION ALL SELECT id, rank FROM lex) u
      GROUP BY id
      ORDER BY rrf_score DESC
      LIMIT %(fuse_n)s
    )
    SELECT {_SELECT_COLUMNS}
    FROM fused f
    JOIN kitab_chunks kc ON kc.id = f.id
    JOIN kitab k ON k.id = kc.kitab_id
    LEFT JOIN kitab_tree t ON t.id = kc.tree_node_id
    ORDER BY f.rrf_score DESC
    """
    params = {
        "qvec": query_vec, "q": question,
        "dense_n": DENSE_CANDIDATES, "lex_n": LEXICAL_CANDIDATES,
        "k": RRF_K, "fuse_n": DENSE_CANDIDATES + LEXICAL_CANDIDATES,
    }
    with conn() as c, c.cursor() as cur:
        cur.execute(sql, params)
        return [_row_to_chunk(r) for r in cur.fetchall()]


# ─── rerank step ──────────────────────────────────────────────────

def _rerank(candidates: list[RetrievedChunk], question: str) -> list[RetrievedChunk]:
    if not RERANK_ENABLED or not candidates or len(candidates) <= RERANK_TOPK:
        return candidates[:RERANK_TOPK]
    docs = [c.content for c in candidates]
    ranked = voyage.rerank(question, docs, top_n=RERANK_TOPK)
    out: list[RetrievedChunk] = []
    for idx, score in ranked:
        ch = candidates[idx]
        ch.score = score
        out.append(ch)
    return out


# ─── public API ───────────────────────────────────────────────────

def retrieve_per_kitab(kitab_id: str, question: str) -> RetrievalResult:
    query_vec = voyage.embed_query(question)
    fused = _hybrid_per_kitab(kitab_id, query_vec, question)
    if not fused:
        return RetrievalResult(
            strategy="hybrid_empty",
            reasoning="no chunks matched dense or lexical within this kitab",
        )
    top = _rerank(fused, question)
    strategy = "hybrid_rrf_plus_rerank" if RERANK_ENABLED else "hybrid_rrf"
    return RetrievalResult(
        strategy=strategy,
        reasoning=f"{len(fused)} fused candidates → top-{len(top)}",
        chunks=top,
    )


def retrieve_general(question: str) -> RetrievalResult:
    query_vec = voyage.embed_query(question)
    fused = _hybrid_general(query_vec, question)
    if not fused:
        return RetrievalResult(
            strategy="hybrid_empty",
            reasoning="no chunks matched across published kitab",
        )
    top = _rerank(fused, question)
    strategy = "hybrid_rrf_plus_rerank" if RERANK_ENABLED else "hybrid_rrf"
    return RetrievalResult(
        strategy=strategy,
        reasoning=f"{len(fused)} fused candidates → top-{len(top)}",
        chunks=top,
    )
