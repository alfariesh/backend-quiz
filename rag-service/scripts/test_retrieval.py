"""
Test the real hybrid retrieval pipeline against the Postgres DB.

Calls retrieve_per_kitab and retrieve_general — the same code paths used by
/query — without going through HTTP or the LLM.

Usage:
    cd rag-service
    DATABASE_URL=postgres://... VOYAGE_API_KEY=... \\
        python scripts/test_retrieval.py <kitab_slug> "q1" "q2" ...
"""
from __future__ import annotations

import sys
from pathlib import Path

from dotenv import load_dotenv

load_dotenv()

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from db import conn, init_pool  # noqa: E402
from retrieval import retrieve_general, retrieve_per_kitab  # noqa: E402


def resolve_kitab_id(slug: str) -> str | None:
    with conn() as c, c.cursor() as cur:
        cur.execute("SELECT id FROM kitab WHERE slug = %s", (slug,))
        row = cur.fetchone()
        return str(row[0]) if row else None


def print_result(label: str, result):
    print(f"\n--- {label}  strategy={result.strategy}")
    print(f"    {result.reasoning}")
    for i, c in enumerate(result.chunks, 1):
        snippet = c.content.replace("\n", " ")[:140]
        score = f"{c.score:.4f}" if c.score is not None else "—"
        print(f"  {i}. pp.{c.start_page}-{c.end_page}  score={score}\n     {snippet}…")


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)
    slug = sys.argv[1]
    queries = sys.argv[2:]

    init_pool()
    kid = resolve_kitab_id(slug)
    if not kid:
        sys.exit(f"no kitab with slug={slug}")
    print(f"kitab: {slug} ({kid})\n")

    for q in queries:
        print(f"\n{'='*70}\nQUERY: {q!r}\n{'='*70}")
        print_result("PER_KITAB", retrieve_per_kitab(kid, q))
        print_result("GENERAL  ", retrieve_general(q))


if __name__ == "__main__":
    main()
