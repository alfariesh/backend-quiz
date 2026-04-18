from __future__ import annotations

import json
import os
from contextlib import contextmanager

from pgvector.psycopg import register_vector
from psycopg_pool import ConnectionPool
from psycopg.types.json import Jsonb

DATABASE_URL = os.environ["DATABASE_URL"]

_pool: ConnectionPool | None = None


def init_pool():
    global _pool
    if _pool is None:
        _pool = ConnectionPool(
            DATABASE_URL,
            min_size=1,
            max_size=5,
            configure=lambda c: register_vector(c),
            open=True,
        )
    return _pool


def close_pool():
    global _pool
    if _pool is not None:
        _pool.close()
        _pool = None


@contextmanager
def conn():
    if _pool is None:
        init_pool()
    with _pool.connection() as c:
        yield c


# ─── conversation persistence ─────────────────────────────────────

def create_conversation(
    user_id: str, mode: str, style: str,
    scope_kitab_id: str | None, scope_fatwa_id: str | None,
    title: str = "",
) -> str:
    with conn() as c, c.cursor() as cur:
        cur.execute(
            """
            INSERT INTO conversations (user_id, mode, scope_kitab_id, scope_fatwa_id, style, title)
            VALUES (%s, %s, %s, %s, %s, %s)
            RETURNING id
            """,
            (user_id, mode, scope_kitab_id, scope_fatwa_id, style, title),
        )
        return str(cur.fetchone()[0])


def update_conversation_title(conversation_id: str, title: str) -> None:
    with conn() as c, c.cursor() as cur:
        cur.execute(
            "UPDATE conversations SET title = %s WHERE id = %s",
            (title[:120], conversation_id),
        )


def load_history(conversation_id: str, turns: int) -> list[dict]:
    """Return messages oldest-first, capped at last `turns*2` rows."""
    with conn() as c, c.cursor() as cur:
        cur.execute(
            """
            SELECT role, content FROM (
                SELECT role, content, created_at
                FROM messages
                WHERE conversation_id = %s
                ORDER BY created_at DESC
                LIMIT %s
            ) t
            ORDER BY created_at ASC
            """,
            (conversation_id, turns * 2),
        )
        return [{"role": r, "content": c_} for r, c_ in cur.fetchall()]


def save_user_message(conversation_id: str, content: str) -> str:
    with conn() as c, c.cursor() as cur:
        cur.execute(
            """
            INSERT INTO messages (conversation_id, role, content)
            VALUES (%s, 'user', %s)
            RETURNING id
            """,
            (conversation_id, content),
        )
        msg_id = str(cur.fetchone()[0])
        cur.execute(
            """
            UPDATE conversations
            SET message_count = message_count + 1,
                last_message_preview = LEFT(%s, 120),
                updated_at = now()
            WHERE id = %s
            """,
            (content, conversation_id),
        )
    return msg_id


def save_assistant_message(
    conversation_id: str, content: str, *,
    citations: list[dict], suggested_questions: list[str],
    retrieval_strategy: str, token_usage: dict, latency_ms: int,
) -> str:
    with conn() as c, c.cursor() as cur:
        cur.execute(
            """
            INSERT INTO messages (
                conversation_id, role, content,
                citations, suggested_questions, retrieval_strategy,
                token_usage, latency_ms
            )
            VALUES (%s, 'assistant', %s, %s, %s, %s, %s, %s)
            RETURNING id
            """,
            (
                conversation_id,
                content,
                Jsonb(citations),
                suggested_questions,
                retrieval_strategy,
                Jsonb(token_usage),
                latency_ms,
            ),
        )
        msg_id = str(cur.fetchone()[0])
        cur.execute(
            """
            UPDATE conversations
            SET message_count = message_count + 1,
                last_message_preview = LEFT(%s, 120),
                updated_at = now()
            WHERE id = %s
            """,
            (content, conversation_id),
        )
    return msg_id
