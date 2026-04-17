"""
Retrieval strategies:
- per_kitab: tree-reasoning → (optional vector) within selected nodes
- general:   vector across all kitab_chunks → (optional Jina rerank)

Caching: kitab tree payload cached in-memory (TTL) — avoids DB + rebuild per request.
Progressive tree reasoning: for trees > threshold, pass depth-0 + depth-1 only in first pass.
"""
from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from typing import Any

from cachetools import TTLCache

import jina
from db import conn
from embed import embed_query
from llm import chat

TREE_REASONING_MAX_NODES = int(os.getenv("TREE_REASONING_MAX_NODES", "10"))
TREE_PROGRESSIVE_THRESHOLD = int(os.getenv("TREE_PROGRESSIVE_THRESHOLD", "60"))
VECTOR_TOPK = int(os.getenv("VECTOR_TOPK", "8"))
SKIP_VECTOR_CHUNK_THRESHOLD = int(os.getenv("SKIP_VECTOR_CHUNK_THRESHOLD", "10"))
GENERAL_CANDIDATES = int(os.getenv("GENERAL_CANDIDATES", "25"))
GENERAL_TOPK = int(os.getenv("GENERAL_TOPK", "8"))
TREE_CACHE_TTL = int(os.getenv("TREE_CACHE_TTL_SECONDS", "300"))


_tree_cache: TTLCache = TTLCache(maxsize=128, ttl=TREE_CACHE_TTL)


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


# ─── tree helpers ─────────────────────────────────────────────────

def _pick_text(j: Any) -> str:
    if not j:
        return ""
    if isinstance(j, str):
        return j
    for key in ("id", "en", "ar"):
        if j.get(key):
            return j[key]
    return next(iter(j.values()), "") if j else ""


def _load_kitab_tree_raw(kitab_id: str) -> tuple[list[dict], dict[str, str]]:
    with conn() as c, c.cursor() as cur:
        cur.execute(
            """
            SELECT t.id, t.node_id, t.title, t.summary, t.depth, t.sort_order,
                   parent.node_id AS parent_node_id
            FROM kitab_tree t
            LEFT JOIN kitab_tree parent ON parent.id = t.parent_id
            WHERE t.kitab_id = %s
            ORDER BY t.sort_order
            """,
            (kitab_id,),
        )
        rows = cur.fetchall()
    flat, id_map = [], {}
    for db_id, node_id, title, summary, depth, _sort, parent_node_id in rows:
        id_map[node_id] = str(db_id)
        flat.append({
            "node_id": node_id, "parent": parent_node_id,
            "title": _pick_text(title), "summary": _pick_text(summary),
            "depth": depth,
        })
    return flat, id_map


def _cached_tree(kitab_id: str) -> tuple[list[dict], dict[str, str]]:
    hit = _tree_cache.get(kitab_id)
    if hit is not None:
        return hit
    val = _load_kitab_tree_raw(kitab_id)
    _tree_cache[kitab_id] = val
    return val


def _compact(nodes: list[dict]) -> list[dict]:
    out = []
    for n in nodes:
        summary = (n["summary"] or "")[:350]
        out.append({
            "node_id": n["node_id"], "parent": n["parent"],
            "depth": n["depth"], "title": n["title"], "summary": summary,
        })
    return out


def _tree_reason(question: str, nodes_payload: list[dict], hint: str = "") -> tuple[list[str], str]:
    prompt = f"""You help a theology student navigate a classical Islamic book.

Given a question and the book's tree-of-contents (flat, each node has node_id, parent, depth, title, summary), pick node_ids most likely to contain the answer.

Rules:
- Pick at most {TREE_REASONING_MAX_NODES} nodes.
- Prefer leaf nodes (specific bab/fashl) over top-level sections.
- If multiple bab cover the topic, include them all (within limit).
- Return JSON ONLY: {{"thinking": "brief reasoning", "node_ids": ["0003", ...]}}
{hint}

Question: {question}

Tree:
{json.dumps(nodes_payload, ensure_ascii=False)}"""
    content, _ = chat(
        messages=[{"role": "user", "content": prompt}],
        response_format={"type": "json_object"},
        temperature=0.0,
    )
    try:
        data = json.loads(content)
        return data.get("node_ids", [])[:TREE_REASONING_MAX_NODES], data.get("thinking", "")
    except (json.JSONDecodeError, AttributeError):
        return [], f"failed to parse: {content[:160]}"


def _progressive_tree_reason(question: str, flat: list[dict]) -> tuple[list[str], str]:
    """For large trees: pass depth<=1 first, then drill into selected subtrees."""
    top = [n for n in flat if n["depth"] <= 1]
    picked_top, reasoning_top = _tree_reason(
        question, _compact(top),
        hint="(This is a top-level overview only — pick sections to explore deeper.)",
    )
    if not picked_top:
        return [], reasoning_top
    selected_subtree = [n for n in flat if n["node_id"] in picked_top or n["parent"] in picked_top]
    # descend: also include grandchildren of picked_top
    ancestors = set(picked_top)
    changed = True
    while changed:
        changed = False
        for n in flat:
            if n["parent"] in ancestors and n["node_id"] not in ancestors:
                ancestors.add(n["node_id"])
                changed = True
    subtree = [n for n in flat if n["node_id"] in ancestors]
    if len(subtree) <= TREE_PROGRESSIVE_THRESHOLD:
        picked, reasoning = _tree_reason(question, _compact(subtree))
        return picked, f"[progressive] top: {reasoning_top} | detail: {reasoning}"
    return picked_top, f"[progressive, top-only] {reasoning_top}"


# ─── chunk fetching ───────────────────────────────────────────────

def _fetch_chunks_by_nodes(kitab_id: str, node_db_ids: list[str]) -> list[RetrievedChunk]:
    if not node_db_ids:
        return []
    with conn() as c, c.cursor() as cur:
        cur.execute(
            """
            SELECT kc.id, kc.kitab_id, k.title, kc.content, kc.start_page, kc.end_page,
                   t.id, t.node_id, t.title, t.sort_order
            FROM kitab_chunks kc
            JOIN kitab_tree t ON t.id = kc.tree_node_id
            JOIN kitab k ON k.id = kc.kitab_id
            WHERE kc.kitab_id = %s AND kc.tree_node_id = ANY(%s::uuid[])
            ORDER BY t.sort_order, kc.id
            """,
            (kitab_id, node_db_ids),
        )
        return [_row_to_chunk(r) for r in cur.fetchall()]


def _row_to_chunk(r, score: float | None = None) -> RetrievedChunk:
    return RetrievedChunk(
        chunk_id=r[0], kitab_id=str(r[1]), kitab_title=_pick_text(r[2]),
        content=r[3], start_page=r[4], end_page=r[5],
        tree_node_db_id=str(r[6]), tree_node_id=r[7], tree_title=_pick_text(r[8]),
        score=score,
    )


def _rank_by_vector(
    kitab_id: str, node_db_ids: list[str], query_vec: list[float], top_k: int,
) -> list[RetrievedChunk]:
    with conn() as c, c.cursor() as cur:
        cur.execute(
            """
            SELECT kc.id, kc.kitab_id, k.title, kc.content, kc.start_page, kc.end_page,
                   t.id, t.node_id, t.title, 0,
                   1 - (kc.embedding <=> %s::vector) AS score
            FROM kitab_chunks kc
            JOIN kitab_tree t ON t.id = kc.tree_node_id
            JOIN kitab k ON k.id = kc.kitab_id
            WHERE kc.kitab_id = %s AND kc.tree_node_id = ANY(%s::uuid[])
            ORDER BY kc.embedding <=> %s::vector
            LIMIT %s
            """,
            (query_vec, kitab_id, node_db_ids, query_vec, top_k),
        )
        return [
            RetrievedChunk(
                chunk_id=r[0], kitab_id=str(r[1]), kitab_title=_pick_text(r[2]),
                content=r[3], start_page=r[4], end_page=r[5],
                tree_node_db_id=str(r[6]), tree_node_id=r[7], tree_title=_pick_text(r[8]),
                score=float(r[10]),
            )
            for r in cur.fetchall()
        ]


# ─── public: per_kitab ────────────────────────────────────────────

def retrieve_per_kitab(kitab_id: str, question: str) -> RetrievalResult:
    flat, id_map = _cached_tree(kitab_id)
    if not flat:
        return RetrievalResult(strategy="empty", reasoning="kitab has no tree")

    if len(flat) > TREE_PROGRESSIVE_THRESHOLD:
        picked, reasoning = _progressive_tree_reason(question, flat)
    else:
        picked, reasoning = _tree_reason(question, _compact(flat))

    node_db_ids = [id_map[n] for n in picked if n in id_map]
    if not node_db_ids:
        return RetrievalResult(
            strategy="tree_reasoning_miss",
            reasoning=reasoning, picked_tree_node_ids=picked,
        )

    all_chunks = _fetch_chunks_by_nodes(kitab_id, node_db_ids)
    if len(all_chunks) <= SKIP_VECTOR_CHUNK_THRESHOLD:
        return RetrievalResult(
            strategy="tree_reasoning_only",
            reasoning=reasoning, picked_tree_node_ids=picked, chunks=all_chunks,
        )

    query_vec = embed_query(question)
    top = _rank_by_vector(kitab_id, node_db_ids, query_vec, VECTOR_TOPK)
    return RetrievalResult(
        strategy="tree_reasoning_plus_vector",
        reasoning=reasoning, picked_tree_node_ids=picked, chunks=top,
    )


# ─── public: general (across all kitab) ───────────────────────────

def _vector_candidates_all_kitab(query_vec: list[float], limit: int) -> list[RetrievedChunk]:
    with conn() as c, c.cursor() as cur:
        cur.execute(
            """
            SELECT kc.id, kc.kitab_id, k.title, kc.content, kc.start_page, kc.end_page,
                   t.id, t.node_id, t.title,
                   1 - (kc.embedding <=> %s::vector) AS score
            FROM kitab_chunks kc
            JOIN kitab_tree t ON t.id = kc.tree_node_id
            JOIN kitab k ON k.id = kc.kitab_id
            WHERE k.is_published = true AND k.embeddings_processed = true
            ORDER BY kc.embedding <=> %s::vector
            LIMIT %s
            """,
            (query_vec, query_vec, limit),
        )
        return [
            RetrievedChunk(
                chunk_id=r[0], kitab_id=str(r[1]), kitab_title=_pick_text(r[2]),
                content=r[3], start_page=r[4], end_page=r[5],
                tree_node_db_id=str(r[6]), tree_node_id=r[7], tree_title=_pick_text(r[8]),
                score=float(r[9]),
            )
            for r in cur.fetchall()
        ]


def retrieve_general(question: str) -> RetrievalResult:
    query_vec = embed_query(question)
    candidates = _vector_candidates_all_kitab(query_vec, GENERAL_CANDIDATES)
    if not candidates:
        return RetrievalResult(strategy="general_empty", reasoning="no chunks available")

    if jina.enabled() and len(candidates) > GENERAL_TOPK:
        docs = [c.content[:4000] for c in candidates]
        ranked = jina.rerank(question, docs, top_n=GENERAL_TOPK)
        top = []
        for idx, score in ranked:
            ch = candidates[idx]
            ch.score = score
            top.append(ch)
        return RetrievalResult(
            strategy="vector_plus_jina_rerank",
            reasoning=f"{len(candidates)} vector candidates → Jina top-{GENERAL_TOPK}",
            chunks=top,
        )

    return RetrievalResult(
        strategy="vector_only",
        reasoning=f"vector top-{GENERAL_TOPK} (Jina disabled)",
        chunks=candidates[:GENERAL_TOPK],
    )
