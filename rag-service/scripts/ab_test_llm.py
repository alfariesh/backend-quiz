"""
A/B test multiple LLMs against the same retrieved context.

Retrieves chunks once via the real production retrieval pipeline, builds the
exact prompt /query uses, then calls each LLM with identical input. Prints
side-by-side: latency, token usage, extracted quotes count, and the answer.

Usage:
    cd rag-service
    python scripts/ab_test_llm.py <kitab_slug> "q1" "q2" ...
"""
from __future__ import annotations

import json
import os
import sys
import time
from pathlib import Path

from dotenv import load_dotenv
from openai import OpenAI

load_dotenv()

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from db import conn, init_pool  # noqa: E402
from retrieval import retrieve_per_kitab  # noqa: E402

MODELS = [
    "gemini/gemini-3.1-flash-lite-preview",
    "mimo-v2-omni",
    "seed-2-0-mini",
]

API_KEY = os.environ["LLM_API_KEY"]
BASE_URL = os.environ["LLM_BASE_URL"]

client = OpenAI(api_key=API_KEY, base_url=BASE_URL)


# ─── prompt (mirrors main.py _build_context + _quote_then_answer_prompt) ─

def build_context(chunks) -> str:
    blocks = []
    for i, ch in enumerate(chunks, 1):
        header = f"[Kutipan {i}] {ch.kitab_title} — {ch.tree_title} (pp. {ch.start_page}-{ch.end_page})"
        blocks.append(f"{header}\n{ch.content}")
    return "\n\n---\n\n".join(blocks)


def build_prompt(question: str, context: str) -> str:
    return f"""You are an assistant helping a student study classical Islamic books (kitab).

You must follow a STRICT two-stage protocol inside a single JSON response:

STAGE 1 — EXTRACT: Scan the provided [Kutipan N] blocks. Find sentences (in ARABIC, verbatim, character-for-character copy-paste from the blocks) that directly answer the user's question. Do NOT paraphrase, do NOT translate, do NOT invent any Arabic text.

STAGE 2 — ANSWER: Detect the language of the user's question. Compose the answer in THAT SAME LANGUAGE (Indonesian → Indonesian; English → English). Keep Arabic quotations inline in Arabic regardless. Base the answer STRICTLY on the Stage 1 extracts. Cite quotes using [Kutipan N].

Also produce 3 suggested follow-up questions in the user's question language.

HARD RULES:
- If a claim is not present in Stage 1 extracts, do NOT state it.
- If extracts are insufficient, say so explicitly.

Output JSON only, matching this schema exactly:
{{
  "extracted_quotes": [
    {{"kutipan_n": 1, "text_arabic": "verbatim Arabic sentence copied from [Kutipan 1]"}}
  ],
  "answer": "Answer in user's language, citing [Kutipan N]",
  "suggested_questions": ["q1", "q2", "q3"]
}}

User question: {question}

Retrieved passages:
{context}"""


def call_llm(model: str, prompt: str) -> dict:
    t0 = time.time()
    try:
        resp = client.chat.completions.create(
            model=model,
            messages=[{"role": "user", "content": prompt}],
            temperature=0.2,
            response_format={"type": "json_object"},
        )
        content = resp.choices[0].message.content or ""
        usage = resp.usage
        latency = int((time.time() - t0) * 1000)
        try:
            parsed = json.loads(content)
        except json.JSONDecodeError:
            parsed = {"answer": content, "extracted_quotes": [], "suggested_questions": []}
        return {
            "model": model,
            "latency_ms": latency,
            "prompt_tokens": usage.prompt_tokens if usage else 0,
            "completion_tokens": usage.completion_tokens if usage else 0,
            "quotes_count": len(parsed.get("extracted_quotes", [])),
            "answer": parsed.get("answer", "")[:600],
            "suggestions": parsed.get("suggested_questions", []),
            "raw_quotes": parsed.get("extracted_quotes", [])[:3],
        }
    except Exception as e:
        return {"model": model, "error": str(e)[:300], "latency_ms": int((time.time() - t0) * 1000)}


def resolve_kitab_id(slug: str) -> str | None:
    with conn() as c, c.cursor() as cur:
        cur.execute("SELECT id FROM kitab WHERE slug = %s", (slug,))
        row = cur.fetchone()
        return str(row[0]) if row else None


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

    for q in queries:
        print(f"\n{'='*70}\nQUERY: {q!r}\n{'='*70}")
        res = retrieve_per_kitab(kid, q)
        if not res.chunks:
            print("  (no chunks retrieved)")
            continue
        print(f"  Retrieval: {res.strategy}, {len(res.chunks)} chunks, "
              f"top score={res.chunks[0].score:.3f}")
        context = build_context(res.chunks)
        prompt = build_prompt(q, context)

        for model in MODELS:
            print(f"\n--- {model}")
            r = call_llm(model, prompt)
            if "error" in r:
                print(f"  ERROR ({r['latency_ms']}ms): {r['error']}")
                continue
            print(f"  latency: {r['latency_ms']}ms  "
                  f"tokens: {r['prompt_tokens']}/{r['completion_tokens']}  "
                  f"quotes: {r['quotes_count']}")
            print(f"  Answer: {r['answer']}")
            if r["suggestions"]:
                print(f"  Suggestions:")
                for s in r["suggestions"]:
                    print(f"    - {s}")


if __name__ == "__main__":
    main()
