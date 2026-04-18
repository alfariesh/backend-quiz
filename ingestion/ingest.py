"""
Surau RAG ingestion — kitab PDF → postgres + pgvector (voyage-context-3).

No PageIndex tree generation. Chunks are sentence-aware over the raw PDF text.
Citation works via per-chunk start_page/end_page. Existing books with kitab_tree
records remain valid (kitab_tree is now UI-navigation-only).

Usage:
    python ingest.py list                         # list detected PDFs
    python ingest.py one --pdf <path>             # single PDF
    python ingest.py all                          # every PDF in PDF_DIR

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
from pathlib import Path

import fitz  # PyMuPDF
import psycopg
from dotenv import load_dotenv
from pgvector.psycopg import register_vector

load_dotenv()

# Reuse rag-service/voyage.py — single source of truth for the API client.
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "rag-service"))
import voyage  # noqa: E402

DATABASE_URL = os.environ["DATABASE_URL"]
PDF_DIR = Path(os.environ["PDF_DIR"])
CHUNK_TARGET_CHARS = int(os.getenv("CHUNK_TARGET_CHARS", "1200"))
EMBED_BATCH = int(os.getenv("EMBED_BATCH", "25"))  # ≤ ~25K tokens for Arabic, fits 32K context window

PLACEHOLDER_AUTHOR_SLUG = "unknown-author"
PLACEHOLDER_GENRE_SLUG = "uncategorized"

# Sentence boundaries: Arabic + Latin punctuation, plus blank-line breaks.
SENT_SPLIT = re.compile(r"(?<=[\.\?\!۔؟؛])\s+|\n{2,}")


# ─── helpers ──────────────────────────────────────────────────────

def slugify(text: str) -> str:
    text = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode("ascii")
    text = re.sub(r"[^a-zA-Z0-9]+", "-", text).strip("-").lower()
    return text or "kitab"


def extract_pages(pdf_path: Path) -> list[tuple[int, str]]:
    """Returns [(page_no, text)] for non-empty pages, page_no is 1-indexed."""
    doc = fitz.open(pdf_path)
    out = []
    for i, page in enumerate(doc, start=1):
        text = page.get_text("text").strip()
        if text:
            out.append((i, text))
    doc.close()
    return out


def chunk_pages(pages: list[tuple[int, str]]) -> list[dict]:
    """Sentence-aware chunks; each chunk records start/end page span."""
    chunks: list[dict] = []
    buf: list[str] = []
    buf_len = 0
    start_pg: int | None = None
    last_pg = pages[0][0] if pages else 0

    def flush(end_pg: int):
        nonlocal buf, buf_len, start_pg
        if not buf:
            return
        chunks.append({
            "content": " ".join(buf).strip(),
            "start_page": start_pg,
            "end_page": end_pg,
        })
        # 1-sentence overlap for context continuity at chunk boundaries.
        overlap = buf[-1:]
        buf = list(overlap)
        buf_len = sum(len(s) for s in buf)
        start_pg = end_pg if overlap else None

    for page_no, text in pages:
        last_pg = page_no
        for s in (s.strip() for s in SENT_SPLIT.split(text) if s.strip()):
            if start_pg is None:
                start_pg = page_no
            buf.append(s)
            buf_len += len(s)
            if buf_len >= CHUNK_TARGET_CHARS:
                flush(page_no)
    flush(last_pg)
    return chunks


def embed_with_retry(texts: list[str]) -> list[list[float]]:
    for attempt in range(5):
        try:
            return voyage.embed_document(texts)
        except Exception as e:
            if attempt == 4:
                raise
            wait = 2 ** (attempt + 2)  # 4, 8, 16, 32
            print(f"    [retry {attempt+1}/5 in {wait}s] {e}", file=sys.stderr)
            time.sleep(wait)


# ─── DB ────────────────────────────────────────────────────────────

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
    conn, *, slug: str, author_id: str, genre_id: str,
    title_ar: str, pages: int, pdf_path: str,
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
                slug, author_id, genre_id,
                json.dumps({"ar": title_ar}),
                title_ar,
                json.dumps({}),
                pages, pdf_path,
            ),
        )
        kitab_id = cur.fetchone()[0]
    conn.commit()
    return kitab_id, False


def insert_chunks(conn, kitab_id: str, chunks: list[dict], embeddings: list[list[float]]):
    with conn.cursor() as cur:
        cur.executemany(
            """
            INSERT INTO kitab_chunks (
                kitab_id, content, embedding_voyage,
                start_page, end_page, token_count
            )
            VALUES (%s, %s, %s, %s, %s, %s)
            """,
            [
                (kitab_id, c["content"], emb, c["start_page"], c["end_page"],
                 len(c["content"]) // 4)  # rough token estimate; not load-bearing
                for c, emb in zip(chunks, embeddings)
            ],
        )
    conn.commit()


def mark_processed(conn, kitab_id: str):
    with conn.cursor() as cur:
        cur.execute(
            "UPDATE kitab SET embeddings_processed=true, updated_at=now() WHERE id=%s",
            (kitab_id,),
        )
    conn.commit()


# ─── orchestration ────────────────────────────────────────────────

def find_pdfs() -> list[Path]:
    return sorted(PDF_DIR.glob("*.pdf"))


def ingest_one(conn, pdf_path: Path):
    print(f"\n═══ {pdf_path.name}")
    pages = extract_pages(pdf_path)
    print(f"  pages with text: {len(pages)}")

    chunks = chunk_pages(pages)
    print(f"  chunks: {len(chunks)}")
    if not chunks:
        print("  [skip] no text extracted")
        return

    author_id, genre_id = ensure_placeholders(conn)
    slug = slugify(pdf_path.stem)
    title_ar = pdf_path.stem
    kitab_id, already = upsert_kitab(
        conn,
        slug=slug, author_id=author_id, genre_id=genre_id,
        title_ar=title_ar, pages=len(pages), pdf_path=str(pdf_path),
    )
    if already:
        print(f"  [skip] already embedded: {kitab_id}")
        return

    embeddings: list[list[float]] = []
    for i in range(0, len(chunks), EMBED_BATCH):
        window = chunks[i : i + EMBED_BATCH]
        texts = [c["content"] for c in window]
        vecs = embed_with_retry(texts)
        if len(vecs) != len(texts):
            raise RuntimeError(f"voyage returned {len(vecs)} vectors for {len(texts)} chunks")
        embeddings.extend(vecs)
        print(f"  · batch {i // EMBED_BATCH + 1}: {len(vecs)} vectors "
              f"({len(embeddings)}/{len(chunks)})")

    insert_chunks(conn, kitab_id, chunks, embeddings)
    mark_processed(conn, kitab_id)
    print(f"  done — {len(chunks)} chunks embedded")


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("list")
    one = sub.add_parser("one")
    one.add_argument("--pdf", required=True, help="path to PDF")
    sub.add_parser("all")
    args = ap.parse_args()

    if args.cmd == "list":
        for p in find_pdfs():
            print(p)
        return

    with psycopg.connect(DATABASE_URL) as conn:
        register_vector(conn)
        if args.cmd == "one":
            pdf = Path(args.pdf)
            if not pdf.exists():
                sys.exit(f"PDF not found: {pdf}")
            ingest_one(conn, pdf)
        elif args.cmd == "all":
            for pdf in find_pdfs():
                try:
                    ingest_one(conn, pdf)
                except Exception as e:
                    print(f"  [error] {pdf.name}: {e}", file=sys.stderr)


if __name__ == "__main__":
    main()
