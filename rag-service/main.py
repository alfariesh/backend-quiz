import json
import os
import time
from concurrent.futures import ThreadPoolExecutor
from contextlib import asynccontextmanager

from dotenv import load_dotenv

load_dotenv()

from fastapi import FastAPI, HTTPException, Request  # noqa: E402
from pydantic import BaseModel, Field  # noqa: E402
from sse_starlette.sse import EventSourceResponse  # noqa: E402
from starlette.middleware.base import BaseHTTPMiddleware  # noqa: E402
from starlette.responses import JSONResponse  # noqa: E402

from db import (  # noqa: E402
    close_pool, create_conversation, init_pool,
    load_history, save_assistant_message, save_user_message,
    update_conversation_title,
)
from llm import chat  # noqa: E402
from retrieval import RetrievalResult, retrieve_general, retrieve_per_kitab  # noqa: E402


CONVERSATION_HISTORY_TURNS = int(os.getenv("CONVERSATION_HISTORY_TURNS", "6"))
INTERNAL_AUTH_TOKEN = os.getenv("INTERNAL_AUTH_TOKEN", "")
TITLE_PLACEHOLDER = "New conversation"
# Small/cheap model for the background title-generation job.
# Falls back to LLM_MODEL (the answer model) if unset.
LLM_TITLE_MODEL = os.getenv("LLM_TITLE_MODEL", "mistralai/mistral-nemo")

# Bounded executor so title regeneration cannot fork unbounded threads under load.
_title_executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="title-regen")

STYLE_INSTRUCTIONS = {
    "ringkas": "Answer very briefly (1-3 sentences). Get to the point.",
    "standard": "Answer clearly and structured (1-3 paragraphs).",
    "syarah": "Answer deeply in syarah style: quote the kitab text verbatim, explain meaning, provide scholarly context.",
}


@asynccontextmanager
async def lifespan(_app: FastAPI):
    init_pool()
    yield
    _title_executor.shutdown(wait=False, cancel_futures=True)
    close_pool()


app = FastAPI(title="Surau RAG service", lifespan=lifespan)


class InternalTokenMiddleware(BaseHTTPMiddleware):
    """
    Reject requests missing a shared internal token, except /health.
    When INTERNAL_AUTH_TOKEN is unset (dev default), check is disabled.
    This service is meant to sit behind the Go backend — never exposed publicly.
    """
    async def dispatch(self, request: Request, call_next):
        if not INTERNAL_AUTH_TOKEN or request.url.path == "/health":
            return await call_next(request)
        if request.headers.get("x-internal-token") != INTERNAL_AUTH_TOKEN:
            return JSONResponse(status_code=401, content={"detail": "internal token required"})
        return await call_next(request)


app.add_middleware(InternalTokenMiddleware)


class QueryRequest(BaseModel):
    mode: str = Field(..., description="per_kitab | general")
    kitab_id: str | None = None
    question: str
    style: str = "standard"
    user_id: str | None = None
    conversation_id: str | None = None
    model_override: str | None = None


class Citation(BaseModel):
    chunk_id: int
    kitab_id: str
    kitab_title: str
    tree_node_id: str
    tree_title: str
    start_page: int
    end_page: int
    score: float | None = None


class ExtractedQuote(BaseModel):
    kutipan_n: int
    text_arabic: str


class QueryResponse(BaseModel):
    conversation_id: str | None = None
    answer: str
    extracted_quotes: list[ExtractedQuote]
    suggested_questions: list[str]
    citations: list[Citation]
    retrieval_strategy: str
    retrieval_reasoning: str
    picked_tree_node_ids: list[str]
    token_usage: dict
    latency_ms: int
    model_used: str


# ─── shared pipeline pieces ───────────────────────────────────────

def _run_retrieval(req: QueryRequest) -> RetrievalResult:
    if req.mode == "per_kitab":
        if not req.kitab_id:
            raise HTTPException(400, "kitab_id required for per_kitab mode")
        return retrieve_per_kitab(req.kitab_id, req.question)
    if req.mode == "general":
        return retrieve_general(req.question)
    raise HTTPException(400, f"invalid mode: {req.mode}")


def _build_context(result: RetrievalResult) -> str:
    blocks = []
    for i, ch in enumerate(result.chunks, 1):
        header = f"[Kutipan {i}] {ch.kitab_title} — {ch.tree_title} (pp. {ch.start_page}-{ch.end_page})"
        blocks.append(f"{header}\n{ch.content}")
    return "\n\n---\n\n".join(blocks)


def _quote_then_answer_prompt(req: QueryRequest, context: str) -> str:
    style = STYLE_INSTRUCTIONS[req.style]
    return f"""You are an assistant helping a student study classical Islamic books (kitab).

You must follow a STRICT two-stage protocol inside a single JSON response:

STAGE 1 — EXTRACT: Scan the provided [Kutipan N] blocks. Find sentences (in ARABIC, verbatim, character-for-character copy-paste from the blocks) that directly answer the user's question. Do NOT paraphrase, do NOT add diacritics not in the source, do NOT translate, do NOT invent any Arabic text. If a sentence contains a number (e.g., 80, 100, 1000 times; years; rakaat), copy it exactly.

STAGE 2 — ANSWER: Compose the answer in {"English" if req.style != "syarah" else "English with inline Arabic quotations"}, based STRICTLY on the Stage 1 extracts. Never introduce facts not present in Stage 1. Cite quotes using [Kutipan N].

Also produce 3 suggested follow-up questions a student might ask next.

HARD RULES:
- If a claim (especially a number, name, or date) is not present in Stage 1 extracts, do NOT state it.
- If the extracts are insufficient to answer, say so explicitly.
- Style: {style}

Output JSON only, matching this schema exactly:
{{
  "extracted_quotes": [
    {{"kutipan_n": 1, "text_arabic": "verbatim Arabic sentence copied from [Kutipan 1]"}},
    ...
  ],
  "answer": "English answer citing [Kutipan N] for each claim",
  "suggested_questions": ["q1", "q2", "q3"]
}}

User question: {req.question}

Retrieved passages:
{context}"""


def _to_citations(result: RetrievalResult) -> list[Citation]:
    return [
        Citation(
            chunk_id=c.chunk_id, kitab_id=c.kitab_id, kitab_title=c.kitab_title,
            tree_node_id=c.tree_node_id, tree_title=c.tree_title,
            start_page=c.start_page, end_page=c.end_page, score=c.score,
        )
        for c in result.chunks
    ]


def _ensure_conversation(req: QueryRequest) -> tuple[str | None, bool]:
    """Return (conversation_id, is_new). is_new=True means a fresh row was just created."""
    if not req.user_id:
        return None, False
    if req.conversation_id:
        return req.conversation_id, False
    conv_id = create_conversation(
        user_id=req.user_id, mode=req.mode, style=req.style,
        scope_kitab_id=req.kitab_id if req.mode == "per_kitab" else None,
        scope_fatwa_id=None,
        title=TITLE_PLACEHOLDER,
    )
    return conv_id, True


def _generate_title(question: str, answer: str) -> str:
    """Ask the LLM for a short descriptive title. Returns "" on any failure."""
    try:
        prompt = (
            "Produce a concise conversation title (max 8 words, no quotes, no trailing period) "
            "that summarizes what the user is asking about. Reply with the title only.\n\n"
            f"User question: {question}\n\nAssistant answer: {answer}"
        )
        content, _ = chat(
            messages=[{"role": "user", "content": prompt}],
            temperature=0.2,
            model=LLM_TITLE_MODEL or None,
        )
        title = (content or "").strip().strip('"').strip("'")
        return title[:120]
    except Exception:
        return ""


def _regen_title_job(conv_id: str, question: str, answer: str) -> None:
    title = _generate_title(question, answer)
    if title:
        try:
            update_conversation_title(conv_id, title)
        except Exception:
            pass


def _schedule_title_regen(conv_id: str, question: str, answer: str) -> None:
    try:
        _title_executor.submit(_regen_title_job, conv_id, question, answer)
    except RuntimeError:
        # executor already shut down during app teardown; nothing to do.
        pass


def _parse_structured(content: str) -> tuple[list[dict], str, list[str]]:
    try:
        data = json.loads(content)
        return (
            data.get("extracted_quotes", []),
            data.get("answer", ""),
            data.get("suggested_questions", [])[:3],
        )
    except (json.JSONDecodeError, AttributeError):
        return [], content, []


# ─── endpoints ────────────────────────────────────────────────────

@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/query", response_model=QueryResponse)
def query(req: QueryRequest):
    t0 = time.time()
    if req.style not in STYLE_INSTRUCTIONS:
        raise HTTPException(400, f"invalid style: {req.style}")

    conv_id, conv_is_new = _ensure_conversation(req)
    if conv_id:
        save_user_message(conv_id, req.question)

    history = load_history(conv_id, CONVERSATION_HISTORY_TURNS) if conv_id else []
    if history and history[-1]["role"] == "user":
        history = history[:-1]

    result = _run_retrieval(req)
    citations = _to_citations(result)
    model_used = req.model_override or os.getenv("LLM_MODEL", "x-ai/grok-4.1-fast")

    if not result.chunks:
        latency_ms = int((time.time() - t0) * 1000)
        return QueryResponse(
            conversation_id=conv_id,
            answer="No relevant kitab passages found for this question.",
            extracted_quotes=[], suggested_questions=[], citations=[],
            retrieval_strategy=result.strategy,
            retrieval_reasoning=result.reasoning,
            picked_tree_node_ids=result.picked_tree_node_ids,
            token_usage={}, latency_ms=latency_ms, model_used=model_used,
        )

    context = _build_context(result)
    messages: list[dict] = []
    messages.extend(history)
    messages.append({"role": "user", "content": _quote_then_answer_prompt(req, context)})

    content, usage = chat(
        messages=messages,
        temperature=0.0,
        response_format={"type": "json_object"},
        model=req.model_override,
    )
    extracted_raw, answer, suggestions = _parse_structured(content)
    extracted = [
        ExtractedQuote(kutipan_n=q.get("kutipan_n", 0), text_arabic=q.get("text_arabic", ""))
        for q in extracted_raw if isinstance(q, dict)
    ]

    latency_ms = int((time.time() - t0) * 1000)

    if conv_id:
        save_assistant_message(
            conv_id, answer,
            citations=[c.model_dump() for c in citations],
            suggested_questions=suggestions,
            retrieval_strategy=result.strategy,
            token_usage=usage,
            latency_ms=latency_ms,
        )
        if conv_is_new and answer:
            _schedule_title_regen(conv_id, req.question, answer)

    return QueryResponse(
        conversation_id=conv_id,
        answer=answer,
        extracted_quotes=extracted,
        suggested_questions=suggestions,
        citations=citations,
        retrieval_strategy=result.strategy,
        retrieval_reasoning=result.reasoning,
        picked_tree_node_ids=result.picked_tree_node_ids,
        token_usage=usage,
        latency_ms=latency_ms,
        model_used=model_used,
    )


@app.post("/query/stream")
def query_stream(req: QueryRequest):
    """Stream: 2-stage — extract (fast, non-stream) then stream answer."""
    if req.style not in STYLE_INSTRUCTIONS:
        raise HTTPException(400, f"invalid style: {req.style}")

    async def event_gen():
        t0 = time.time()
        conv_id, conv_is_new = _ensure_conversation(req)
        if conv_id:
            save_user_message(conv_id, req.question)
            yield {"event": "conversation", "data": json.dumps({"id": conv_id})}

        result = _run_retrieval(req)
        citations = _to_citations(result)
        model_used = req.model_override or os.getenv("LLM_MODEL", "x-ai/grok-4.1-fast")

        yield {
            "event": "meta",
            "data": json.dumps({
                "retrieval_strategy": result.strategy,
                "retrieval_reasoning": result.reasoning,
                "picked_tree_node_ids": result.picked_tree_node_ids,
                "model_used": model_used,
            }),
        }
        yield {"event": "citations", "data": json.dumps([c.model_dump() for c in citations])}

        if not result.chunks:
            yield {"event": "answer_delta", "data": json.dumps({"text": "No relevant kitab passages found."})}
            yield {"event": "done", "data": json.dumps({"latency_ms": int((time.time() - t0) * 1000)})}
            return

        context = _build_context(result)
        content, _ = chat(
            messages=[{"role": "user", "content": _quote_then_answer_prompt(req, context)}],
            temperature=0.0,
            response_format={"type": "json_object"},
            model=req.model_override,
        )
        extracted_raw, answer, suggestions = _parse_structured(content)

        yield {
            "event": "extracted_quotes",
            "data": json.dumps([
                {"kutipan_n": q.get("kutipan_n", 0), "text_arabic": q.get("text_arabic", "")}
                for q in extracted_raw if isinstance(q, dict)
            ]),
        }
        # emit answer in one chunk (JSON mode can't stream partial JSON reliably)
        yield {"event": "answer_delta", "data": json.dumps({"text": answer})}
        yield {"event": "suggestions", "data": json.dumps({"questions": suggestions})}

        latency_ms = int((time.time() - t0) * 1000)
        if conv_id:
            save_assistant_message(
                conv_id, answer,
                citations=[c.model_dump() for c in citations],
                suggested_questions=suggestions,
                retrieval_strategy=result.strategy,
                token_usage={}, latency_ms=latency_ms,
            )
            if conv_is_new and answer:
                _schedule_title_regen(conv_id, req.question, answer)
        yield {"event": "done", "data": json.dumps({"latency_ms": latency_ms})}

    return EventSourceResponse(event_gen())


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host=os.getenv("HOST", "0.0.0.0"),
        port=int(os.getenv("PORT", "8001")),
        reload=False,
    )
