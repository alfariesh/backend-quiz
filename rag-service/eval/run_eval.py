"""
Surau RAG eval harness.

Run:
    cd rag-service
    python eval/run_eval.py \\
        --endpoint http://localhost:8001 \\
        --golden eval/golden_set.yaml \\
        --label baseline-2026-04-18

Produces:
    eval/results/<label>_<timestamp>.json      — per-question detail + summary
    stdout                                     — summary table

Metrics (per question):
    citation_hit         (0/1)   — ≥1 returned citation overlaps expected_pages
    citation_precision   (0-1)   — fraction of returned citations that overlap
    verbatim_fidelity    (0/1)   — every expected_arabic_contains string appears
    answer_correct       (0-1)   — LLM-as-judge vs expected_answer_gist

Requires env: LLM_API_KEY, LLM_BASE_URL (same vars as rag-service).
"""
from __future__ import annotations

import argparse
import json
import os
import statistics
import sys
import time
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import httpx
import yaml
from dotenv import load_dotenv

load_dotenv()

# Allow `import db` when running from rag-service/ root: add parent to path.
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

JUDGE_MODEL = os.getenv("EVAL_JUDGE_MODEL", "anthropic/claude-haiku-4.5")
JUDGE_PROMPT = """You are grading a RAG answer against a ground-truth gist.

Question: {question}
Expected gist (what a correct answer should convey): {gist}
Model answer: {answer}

Rate the model answer on correctness vs the expected gist:
  1.0  — fully correct; conveys the expected point with matching facts
  0.5  — partially correct; direction right but misses or distorts facts
  0.0  — wrong, missing, or contradicts

Output JSON only: {{"score": <0.0|0.5|1.0>, "reason": "<1 sentence>"}}"""


# ─── data types ───────────────────────────────────────────────────

@dataclass
class Question:
    id: str
    question: str
    mode: str
    kitab_slug: str | None = None
    kitab_id: str | None = None        # UUID; resolved from kitab_slug if empty
    style: str = "standard"
    expected_pages: list[int] = field(default_factory=list)
    expected_arabic_contains: list[str] = field(default_factory=list)
    expected_answer_gist: str = ""
    notes: str = ""


@dataclass
class QuestionResult:
    id: str
    question: str
    latency_ms: int
    http_status: int
    answer: str
    citations: list[dict]
    extracted_quotes: list[dict]
    retrieval_strategy: str
    # metrics (None if not measurable for this Q)
    citation_hit: int | None
    citation_precision: float | None
    verbatim_fidelity: int | None
    answer_correct: float | None
    judge_reason: str = ""
    error: str = ""


# ─── I/O ──────────────────────────────────────────────────────────

def load_golden(path: Path) -> tuple[dict, list[Question]]:
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    defaults = data.get("defaults", {}) or {}
    items: list[Question] = []
    for raw in data.get("questions", []) or []:
        merged = {**defaults, **raw}
        items.append(Question(
            id=str(merged["id"]),
            question=merged["question"],
            mode=merged.get("mode", "per_kitab"),
            kitab_slug=merged.get("kitab_slug") or None,
            kitab_id=merged.get("kitab_id") or None,
            style=merged.get("style", "standard"),
            expected_pages=list(merged.get("expected_pages") or []),
            expected_arabic_contains=list(merged.get("expected_arabic_contains") or []),
            expected_answer_gist=merged.get("expected_answer_gist") or "",
            notes=merged.get("notes") or "",
        ))
    return defaults, items


_slug_cache: dict[str, str] = {}


def resolve_kitab_id(slug: str) -> str | None:
    """Look up kitab.id by slug via shared rag-service DB pool."""
    if not slug:
        return None
    if slug in _slug_cache:
        return _slug_cache[slug]
    try:
        from db import conn, init_pool
        init_pool()
        with conn() as c, c.cursor() as cur:
            cur.execute("SELECT id FROM kitab WHERE slug = %s", (slug,))
            row = cur.fetchone()
            if not row:
                return None
            kid = str(row[0])
            _slug_cache[slug] = kid
            return kid
    except Exception as e:
        print(f"    [warn] kitab lookup failed for {slug!r}: {e}", file=sys.stderr)
        return None


# ─── HTTP call ────────────────────────────────────────────────────

def call_query(endpoint: str, token: str | None, q: Question) -> tuple[int, dict, int]:
    payload: dict[str, Any] = {
        "mode": q.mode,
        "question": q.question,
        "style": q.style,
    }
    if q.mode == "per_kitab":
        kid = q.kitab_id or resolve_kitab_id(q.kitab_slug or "")
        if not kid:
            return 0, {"error": f"no kitab_id resolved for slug={q.kitab_slug!r}"}, 0
        payload["kitab_id"] = kid
    headers = {"Content-Type": "application/json"}
    if token:
        headers["x-internal-token"] = token

    t0 = time.time()
    try:
        resp = httpx.post(
            f"{endpoint.rstrip('/')}/query",
            json=payload, headers=headers, timeout=120.0,
        )
        latency = int((time.time() - t0) * 1000)
        return resp.status_code, resp.json() if resp.content else {}, latency
    except httpx.HTTPError as e:
        return 0, {"error": str(e)}, int((time.time() - t0) * 1000)


# ─── metrics ──────────────────────────────────────────────────────

def _page_overlap(c: dict, expected_pages: list[int]) -> bool:
    if not expected_pages:
        return False
    lo, hi = int(c.get("start_page", 0)), int(c.get("end_page", 0))
    if lo == 0 and hi == 0:
        return False
    return any(lo <= p <= hi for p in expected_pages)


def citation_metrics(citations: list[dict], expected_pages: list[int]) -> tuple[int | None, float | None]:
    if not expected_pages:
        return None, None
    if not citations:
        return 0, 0.0
    overlaps = [_page_overlap(c, expected_pages) for c in citations]
    hit = 1 if any(overlaps) else 0
    precision = sum(overlaps) / len(citations)
    return hit, round(precision, 3)


def verbatim_fidelity(
    answer: str, extracted: list[dict], expected_arabic: list[str],
) -> int | None:
    if not expected_arabic:
        return None
    haystack = answer + "\n" + "\n".join(
        q.get("text_arabic", "") for q in extracted if isinstance(q, dict)
    )
    return int(all(needle in haystack for needle in expected_arabic))


def llm_judge(question: str, answer: str, gist: str) -> tuple[float | None, str]:
    if not gist:
        return None, ""
    api_key = os.environ.get("LLM_API_KEY")
    base_url = os.environ.get("LLM_BASE_URL")
    if not api_key or not base_url:
        return None, "LLM_API_KEY/LLM_BASE_URL unset"
    try:
        resp = httpx.post(
            f"{base_url.rstrip('/')}/chat/completions",
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
            },
            json={
                "model": JUDGE_MODEL,
                "messages": [{
                    "role": "user",
                    "content": JUDGE_PROMPT.format(
                        question=question, gist=gist, answer=answer or "(empty)",
                    ),
                }],
                "temperature": 0.0,
                "response_format": {"type": "json_object"},
            },
            timeout=60.0,
        )
        resp.raise_for_status()
        content = resp.json()["choices"][0]["message"]["content"]
        data = json.loads(content)
        score = float(data.get("score", 0.0))
        score = max(0.0, min(1.0, score))
        return score, str(data.get("reason", ""))[:240]
    except Exception as e:
        return None, f"judge error: {e}"


# ─── runner ───────────────────────────────────────────────────────

def run(endpoint: str, golden: Path, label: str, output_dir: Path) -> dict:
    token = os.environ.get("INTERNAL_AUTH_TOKEN") or None
    _, questions = load_golden(golden)
    if not questions:
        print("[error] golden_set has no questions", file=sys.stderr)
        sys.exit(1)

    print(f"Running {len(questions)} questions against {endpoint}")
    results: list[QuestionResult] = []

    for i, q in enumerate(questions, 1):
        print(f"  [{i}/{len(questions)}] {q.id} … ", end="", flush=True)
        status, body, latency = call_query(endpoint, token, q)

        if status != 200:
            results.append(QuestionResult(
                id=q.id, question=q.question, latency_ms=latency,
                http_status=status, answer="", citations=[], extracted_quotes=[],
                retrieval_strategy="", citation_hit=None, citation_precision=None,
                verbatim_fidelity=None, answer_correct=None,
                error=str(body)[:500],
            ))
            print(f"HTTP {status} ({latency}ms)")
            continue

        answer = body.get("answer", "")
        citations = body.get("citations", []) or []
        extracted = body.get("extracted_quotes", []) or []

        hit, prec = citation_metrics(citations, q.expected_pages)
        fidelity = verbatim_fidelity(answer, extracted, q.expected_arabic_contains)
        correct, judge_reason = llm_judge(q.question, answer, q.expected_answer_gist)

        results.append(QuestionResult(
            id=q.id, question=q.question, latency_ms=latency, http_status=200,
            answer=answer, citations=citations, extracted_quotes=extracted,
            retrieval_strategy=body.get("retrieval_strategy", ""),
            citation_hit=hit, citation_precision=prec,
            verbatim_fidelity=fidelity, answer_correct=correct,
            judge_reason=judge_reason,
        ))
        marks = []
        if hit is not None: marks.append(f"cit={hit}")
        if fidelity is not None: marks.append(f"verb={fidelity}")
        if correct is not None: marks.append(f"ans={correct}")
        print(f"OK ({latency}ms) {' '.join(marks)}")

    summary = summarize(results)
    output_dir.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    out_path = output_dir / f"{label}_{stamp}.json"
    out_path.write_text(json.dumps({
        "label": label, "endpoint": endpoint, "timestamp": stamp,
        "summary": summary,
        "results": [asdict(r) for r in results],
    }, ensure_ascii=False, indent=2), encoding="utf-8")

    print_summary(label, summary)
    print(f"\nDetails written to: {out_path}")
    return summary


def summarize(results: list[QuestionResult]) -> dict:
    def mean(xs: list[float]) -> float | None:
        return round(statistics.mean(xs), 3) if xs else None

    def collect(attr: str) -> list[float]:
        return [float(getattr(r, attr)) for r in results if getattr(r, attr) is not None]

    latencies = [r.latency_ms for r in results if r.http_status == 200]
    return {
        "n_total": len(results),
        "n_ok": sum(1 for r in results if r.http_status == 200),
        "n_error": sum(1 for r in results if r.http_status != 200),
        "citation_hit_rate": mean(collect("citation_hit")),
        "citation_precision_mean": mean(collect("citation_precision")),
        "verbatim_fidelity_rate": mean(collect("verbatim_fidelity")),
        "answer_correct_mean": mean(collect("answer_correct")),
        "latency_p50_ms": round(statistics.median(latencies)) if latencies else None,
        "latency_p95_ms": (
            round(statistics.quantiles(latencies, n=20)[-1]) if len(latencies) >= 20 else None
        ),
    }


def print_summary(label: str, s: dict) -> None:
    print(f"\n── Summary [{label}] ──")
    rows = [
        ("questions",              f"{s['n_ok']}/{s['n_total']} ok"),
        ("citation_hit_rate",      s["citation_hit_rate"]),
        ("citation_precision_avg", s["citation_precision_mean"]),
        ("verbatim_fidelity",      s["verbatim_fidelity_rate"]),
        ("answer_correct_avg",     s["answer_correct_mean"]),
        ("latency p50 / p95 ms",   f"{s['latency_p50_ms']} / {s['latency_p95_ms']}"),
    ]
    for k, v in rows:
        print(f"  {k:<26} {v}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--endpoint", default=os.getenv("RAG_ENDPOINT", "http://localhost:8001"))
    ap.add_argument("--golden",   default="eval/golden_set.yaml", type=Path)
    ap.add_argument("--label",    default="run", help="label for output filename")
    ap.add_argument("--output",   default="eval/results", type=Path)
    args = ap.parse_args()
    run(args.endpoint, args.golden, args.label, args.output)


if __name__ == "__main__":
    main()
