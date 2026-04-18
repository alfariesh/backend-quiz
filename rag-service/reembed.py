"""
Populate `kitab_chunks.embedding_voyage` using voyage-context-3.

Runs after migration 00024. Does NOT touch the legacy `embedding` column,
so you can roll back by reading from the old column if needed.

Usage (from rag-service/):
    python reembed.py                      # all kitab with chunks
    python reembed.py --kitab-slug <slug>  # single kitab
    python reembed.py --force              # re-embed even if already populated
    python reembed.py --batch 50           # chunks per API call (default 50)

voyage-context-3 processes all chunks in an API call with awareness of
siblings in the same request. We batch per kitab in groups of N chunks
to stay under Voyage request limits while preserving as much local context
as possible. Chunks within a batch are contextually aware of each other;
chunks across batches of the same kitab are independent (acceptable
trade-off for large kitab).
"""
from __future__ import annotations

import argparse
import sys
import time
from pathlib import Path

from dotenv import load_dotenv

load_dotenv()

# Make sibling modules importable when run as a script.
sys.path.insert(0, str(Path(__file__).resolve().parent))

import voyage  # noqa: E402
from db import conn, init_pool  # noqa: E402


def _list_kitab(slug: str | None, only_missing: bool) -> list[tuple[str, str]]:
    """Return [(kitab_id, slug)] of kitab that have chunks to embed."""
    sql = """
    SELECT k.id, k.slug
    FROM kitab k
    WHERE EXISTS (
        SELECT 1 FROM kitab_chunks kc
        WHERE kc.kitab_id = k.id
        {missing_filter}
    )
    {slug_filter}
    ORDER BY k.slug
    """
    missing = "AND kc.embedding_voyage IS NULL" if only_missing else ""
    slug_f = "AND k.slug = %s" if slug else ""
    params = (slug,) if slug else ()
    with conn() as c, c.cursor() as cur:
        cur.execute(sql.format(missing_filter=missing, slug_filter=slug_f), params)
        return [(str(r[0]), r[1]) for r in cur.fetchall()]


def _fetch_chunks(kitab_id: str, only_missing: bool) -> list[tuple[int, str]]:
    """Return [(chunk_id, content)] for a kitab, ordered by tree sort_order then chunk id."""
    sql = """
    SELECT kc.id, kc.content
    FROM kitab_chunks kc
    LEFT JOIN kitab_tree t ON t.id = kc.tree_node_id
    WHERE kc.kitab_id = %s
    {missing}
    ORDER BY COALESCE(t.sort_order, 0), kc.id
    """
    missing = "AND kc.embedding_voyage IS NULL" if only_missing else ""
    with conn() as c, c.cursor() as cur:
        cur.execute(sql.format(missing=missing), (kitab_id,))
        return [(r[0], r[1]) for r in cur.fetchall()]


def _write_embeddings(rows: list[tuple[int, list[float]]]) -> None:
    with conn() as c, c.cursor() as cur:
        cur.executemany(
            "UPDATE kitab_chunks SET embedding_voyage = %s WHERE id = %s",
            [(vec, cid) for cid, vec in rows],
        )


def reembed_kitab(kitab_id: str, slug: str, *, batch: int, force: bool) -> int:
    chunks = _fetch_chunks(kitab_id, only_missing=not force)
    if not chunks:
        print(f"  [skip] {slug}: nothing to embed")
        return 0

    total = 0
    for i in range(0, len(chunks), batch):
        window = chunks[i : i + batch]
        texts = [c for _, c in window]
        ids = [cid for cid, _ in window]

        for attempt in range(3):
            try:
                vecs = voyage.embed_document(texts)
                break
            except Exception as e:
                if attempt == 2:
                    raise
                wait = 2 ** attempt
                print(f"    [retry {attempt+1} in {wait}s] {e}", file=sys.stderr)
                time.sleep(wait)

        if len(vecs) != len(ids):
            raise RuntimeError(
                f"voyage returned {len(vecs)} vectors for {len(ids)} inputs"
            )
        _write_embeddings(list(zip(ids, vecs)))
        total += len(ids)
        print(f"    · batch {i // batch + 1}: {len(ids)} chunks ({total}/{len(chunks)})")

    return total


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--kitab-slug", default=None, help="restrict to one kitab")
    ap.add_argument("--force", action="store_true",
                    help="re-embed chunks that already have embedding_voyage")
    ap.add_argument("--batch", type=int, default=50,
                    help="chunks per contextualized-embedding request")
    args = ap.parse_args()

    init_pool()
    kitabs = _list_kitab(args.kitab_slug, only_missing=not args.force)
    if not kitabs:
        print("No kitab need embedding. Use --force to rebuild.")
        return

    print(f"Re-embedding {len(kitabs)} kitab with {voyage.VOYAGE_EMBED_MODEL} "
          f"(dim={voyage.VOYAGE_EMBED_DIM}, batch={args.batch})")
    grand = 0
    for kid, slug in kitabs:
        print(f"\n═══ {slug}")
        try:
            grand += reembed_kitab(kid, slug, batch=args.batch, force=args.force)
        except Exception as e:
            print(f"  [error] {slug}: {e}", file=sys.stderr)
    print(f"\nDone. {grand} chunks embedded.")


if __name__ == "__main__":
    main()
