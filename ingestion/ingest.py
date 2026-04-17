"""
Surau RAG ingestion — kitab (PageIndex tree JSON + PDF) → postgres + pgvector.

Usage:
    python ingest.py list                           # list detected kitab pairs
    python ingest.py one --basename "AFDHALUSH SHALAWAT GROK"
    python ingest.py all

Env: see .env.example
"""
from __future__ import annotations

import argparse
import json
import os
import re
import sys
import time
import unicodedata
from dataclasses import dataclass
from pathlib import Path
from typing import Iterator

import fitz  # PyMuPDF
import psycopg
import tiktoken
from dotenv import load_dotenv
from openai import OpenAI
from pgvector.psycopg import register_vector
from psycopg.types.json import Jsonb

load_dotenv()

DATABASE_URL = os.environ["DATABASE_URL"]
PDF_DIR = Path(os.environ["PDF_DIR"])
TREE_DIR = Path(os.environ["TREE_DIR"])
EMBED_MODEL = os.getenv("EMBED_MODEL", "openai/text-embedding-3-large")
EMBED_DIMENSIONS = int(os.getenv("EMBED_DIMENSIONS", "1536"))
CHUNK_TOKENS = int(os.getenv("CHUNK_TOKENS", "800"))
CHUNK_OVERLAP = int(os.getenv("CHUNK_OVERLAP", "100"))

OPENROUTER_KEY = os.getenv("OPENROUTER_API_KEY", "").strip()
OPENAI_KEY = os.getenv("OPENAI_API_KEY", "").strip()
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL", "").strip()

if OPENROUTER_KEY:
    oai = OpenAI(api_key=OPENROUTER_KEY, base_url="https://openrouter.ai/api/v1")
elif OPENAI_KEY:
    oai = OpenAI(api_key=OPENAI_KEY, base_url=OPENAI_BASE_URL or None)
else:
    sys.exit("Set OPENROUTER_API_KEY or OPENAI_API_KEY in .env")

ENC = tiktoken.get_encoding("cl100k_base")

PLACEHOLDER_AUTHOR_SLUG = "unknown-author"
PLACEHOLDER_GENRE_SLUG = "uncategorized"


# ─── helpers ──────────────────────────────────────────────────────

def slugify(text: str) -> str:
    text = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode("ascii")
    text = re.sub(r"[^a-zA-Z0-9]+", "-", text).strip("-").lower()
    return text or "kitab"


def count_tokens(text: str) -> int:
    return len(ENC.encode(text, disallowed_special=()))


def chunk_text(text: str, max_tokens: int, overlap: int) -> list[str]:
    tokens = ENC.encode(text, disallowed_special=())
    if len(tokens) <= max_tokens:
        return [text] if text.strip() else []
    chunks: list[str] = []
    step = max_tokens - overlap
    for i in range(0, len(tokens), step):
        window = tokens[i : i + max_tokens]
        chunks.append(ENC.decode(window))
        if i + max_tokens >= len(tokens):
            break
    return [c for c in chunks if c.strip()]


def extract_pdf_pages(pdf_doc, start_1idx: int, end_1idx: int) -> str:
    if end_1idx < start_1idx:
        return ""
    total = pdf_doc.page_count
    start = max(0, start_1idx - 1)
    end = min(total, end_1idx)
    parts: list[str] = []
    for pno in range(start, end):
        parts.append(pdf_doc.load_page(pno).get_text())
    return "\n".join(parts).strip()


def embed_batch(texts: list[str]) -> list[list[float]]:
    resp = oai.embeddings.create(
        model=EMBED_MODEL,
        input=texts,
        dimensions=EMBED_DIMENSIONS,
    )
    return [d.embedding for d in resp.data]


# ─── tree traversal ───────────────────────────────────────────────

@dataclass
class TreeNode:
    node_id: str
    title: str
    summary: str
    start: int
    end: int
    depth: int
    sort_order: int
    parent_node_id: str | None


def flatten_tree(structure: list[dict]) -> Iterator[TreeNode]:
    order_counter = 0

    def walk(nodes: list[dict], depth: int, parent_id: str | None):
        nonlocal order_counter
        for node in nodes:
            order_counter += 1
            yield TreeNode(
                node_id=node["node_id"],
                title=node.get("title", ""),
                summary=node.get("summary", ""),
                start=int(node.get("start_index", 0) or 0),
                end=int(node.get("end_index", 0) or 0),
                depth=depth,
                sort_order=order_counter,
                parent_node_id=parent_id,
            )
            yield from walk(node.get("nodes", []) or [], depth + 1, node["node_id"])

    yield from walk(structure, 0, None)


# ─── DB operations ────────────────────────────────────────────────

def ensure_placeholders(conn) -> tuple[str, str]:
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO authors (slug, name, full_name)
            VALUES (%s, %s::jsonb, %s::jsonb)
            ON CONFLICT (slug) DO NOTHING
            """,
            (
                PLACEHOLDER_AUTHOR_SLUG,
                json.dumps({"en": "Unknown Author", "id": "Penulis Belum Diisi"}),
                json.dumps({"en": "Unknown Author"}),
            ),
        )
        cur.execute(
            """
            INSERT INTO genres (slug, name, description)
            VALUES (%s, %s::jsonb, %s::jsonb)
            ON CONFLICT (slug) DO NOTHING
            """,
            (
                PLACEHOLDER_GENRE_SLUG,
                json.dumps({"en": "Uncategorized", "id": "Belum Dikategorikan"}),
                json.dumps({}),
            ),
        )
        cur.execute("SELECT id FROM authors WHERE slug = %s", (PLACEHOLDER_AUTHOR_SLUG,))
        author_id = cur.fetchone()[0]
        cur.execute("SELECT id FROM genres WHERE slug = %s", (PLACEHOLDER_GENRE_SLUG,))
        genre_id = cur.fetchone()[0]
    conn.commit()
    return author_id, genre_id


def upsert_kitab(
    conn,
    *,
    slug: str,
    author_id: str,
    genre_id: str,
    title_ar: str,
    synopsis_en: str,
    pages: int,
    pdf_path: str,
) -> tuple[str, bool]:
    """Returns (kitab_id, already_embedded)."""
    with conn.cursor() as cur:
        cur.execute(
            "SELECT id, embeddings_processed FROM kitab WHERE slug = %s", (slug,)
        )
        row = cur.fetchone()
        if row:
            return row[0], bool(row[1])
        cur.execute(
            """
            INSERT INTO kitab (
                slug, author_id, genre_id, title, original_title,
                synopsis, source_language, pages, pdf_storage_path, is_published
            )
            VALUES (%s, %s, %s, %s::jsonb, %s, %s::jsonb, 'ar', %s, %s, false)
            RETURNING id
            """,
            (
                slug,
                author_id,
                genre_id,
                json.dumps({"ar": title_ar}),
                title_ar,
                json.dumps({"en": synopsis_en}),
                pages,
                pdf_path,
            ),
        )
        kitab_id = cur.fetchone()[0]
    conn.commit()
    return kitab_id, False


def insert_tree(conn, kitab_id: str, nodes: list[TreeNode]) -> dict[str, str]:
    node_id_to_uuid: dict[str, str] = {}
    with conn.cursor() as cur:
        for n in nodes:
            parent_uuid = node_id_to_uuid.get(n.parent_node_id) if n.parent_node_id else None
            cur.execute(
                """
                INSERT INTO kitab_tree (
                    kitab_id, parent_id, node_id, title, summary,
                    content, start_page, end_page, depth, sort_order
                )
                VALUES (%s, %s, %s, %s::jsonb, %s::jsonb, '', %s, %s, %s, %s)
                RETURNING id
                """,
                (
                    kitab_id,
                    parent_uuid,
                    n.node_id,
                    json.dumps({"ar": n.title}),
                    json.dumps({"en": n.summary}),
                    n.start,
                    n.end,
                    n.depth,
                    n.sort_order,
                ),
            )
            node_id_to_uuid[n.node_id] = cur.fetchone()[0]
    conn.commit()
    return node_id_to_uuid


def insert_chunks(
    conn,
    kitab_id: str,
    tree_node_uuid: str,
    chunks: list[str],
    embeddings: list[list[float]],
    start_page: int,
    end_page: int,
):
    with conn.cursor() as cur:
        for content, emb in zip(chunks, embeddings):
            cur.execute(
                """
                INSERT INTO kitab_chunks (
                    kitab_id, tree_node_id, content, embedding,
                    start_page, end_page, token_count
                )
                VALUES (%s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    kitab_id,
                    tree_node_uuid,
                    content,
                    emb,
                    start_page,
                    end_page,
                    count_tokens(content),
                ),
            )


def mark_processed(conn, kitab_id: str):
    with conn.cursor() as cur:
        cur.execute(
            "UPDATE kitab SET tree_processed=true, embeddings_processed=true, updated_at=now() WHERE id=%s",
            (kitab_id,),
        )
    conn.commit()


# ─── main orchestration ───────────────────────────────────────────

def find_pairs() -> list[tuple[str, Path, Path]]:
    pairs = []
    for tree_file in sorted(TREE_DIR.glob("*_structure.json")):
        basename = tree_file.name.removesuffix("_structure.json")
        pdf_path = PDF_DIR / f"{basename}.pdf"
        if pdf_path.exists():
            pairs.append((basename, pdf_path, tree_file))
        else:
            print(f"[skip] no PDF for {basename}", file=sys.stderr)
    return pairs


def ingest_one(conn, basename: str, pdf_path: Path, tree_path: Path):
    print(f"\n═══ {basename}")
    tree = json.loads(tree_path.read_text())
    doc_name = tree.get("doc_name", basename)
    doc_description = tree.get("doc_description", "")
    nodes = list(flatten_tree(tree.get("structure", [])))
    print(f"  tree nodes: {len(nodes)}")

    pdf = fitz.open(pdf_path)
    total_pages = pdf.page_count
    print(f"  pdf pages: {total_pages}")

    author_id, genre_id = ensure_placeholders(conn)
    slug = slugify(basename)
    kitab_id, already = upsert_kitab(
        conn,
        slug=slug,
        author_id=author_id,
        genre_id=genre_id,
        title_ar=doc_name.replace(".pdf", ""),
        synopsis_en=doc_description,
        pages=total_pages,
        pdf_path=str(pdf_path),
    )
    if already:
        print(f"  [skip] already embedded: {kitab_id}")
        pdf.close()
        return

    node_uuids = insert_tree(conn, kitab_id, nodes)
    print(f"  tree inserted: {len(node_uuids)} nodes")

    total_chunks = 0
    for n in nodes:
        raw = extract_pdf_pages(pdf, n.start, n.end)
        if not raw:
            continue
        chunks = chunk_text(raw, CHUNK_TOKENS, CHUNK_OVERLAP)
        if not chunks:
            continue
        # batch embed in groups of 32 to stay under API limits
        embeddings: list[list[float]] = []
        for i in range(0, len(chunks), 32):
            batch = chunks[i : i + 32]
            for attempt in range(3):
                try:
                    embeddings.extend(embed_batch(batch))
                    break
                except Exception as e:
                    if attempt == 2:
                        raise
                    print(f"    [retry {attempt+1}] {e}", file=sys.stderr)
                    time.sleep(2 ** attempt)
        insert_chunks(
            conn,
            kitab_id,
            node_uuids[n.node_id],
            chunks,
            embeddings,
            n.start,
            n.end,
        )
        total_chunks += len(chunks)
        print(f"  · {n.node_id} {n.title[:40]} → {len(chunks)} chunks")
    conn.commit()
    mark_processed(conn, kitab_id)
    pdf.close()
    print(f"  done — {total_chunks} chunks total")


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("list")
    one = sub.add_parser("one")
    one.add_argument("--basename", required=True)
    sub.add_parser("all")
    args = ap.parse_args()

    pairs = find_pairs()
    if args.cmd == "list":
        for b, pdf, tree in pairs:
            print(f"{b}\n  pdf:  {pdf}\n  tree: {tree}")
        return

    with psycopg.connect(DATABASE_URL) as conn:
        register_vector(conn)
        if args.cmd == "one":
            match = [p for p in pairs if p[0] == args.basename]
            if not match:
                sys.exit(f"no pair for basename: {args.basename}")
            ingest_one(conn, *match[0])
        elif args.cmd == "all":
            for pair in pairs:
                try:
                    ingest_one(conn, *pair)
                except Exception as e:
                    print(f"  [error] {pair[0]}: {e}", file=sys.stderr)


if __name__ == "__main__":
    main()
