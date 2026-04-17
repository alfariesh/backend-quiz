import os
from typing import Iterator

from openai import OpenAI

_client = OpenAI(
    api_key=os.environ["LLM_API_KEY"],
    base_url=os.environ["LLM_BASE_URL"],
    default_headers={
        "HTTP-Referer": "https://surau.app",
        "X-Title": "Surau RAG",
    },
)

DEFAULT_MODEL = os.getenv("LLM_MODEL", "x-ai/grok-4.1-fast")


def chat(
    messages: list[dict],
    temperature: float = 0.2,
    response_format: dict | None = None,
    model: str | None = None,
) -> tuple[str, dict]:
    kwargs: dict = {
        "model": model or DEFAULT_MODEL,
        "messages": messages,
        "temperature": temperature,
    }
    if response_format is not None:
        kwargs["response_format"] = response_format
    resp = _client.chat.completions.create(**kwargs)
    usage = {
        "prompt_tokens": resp.usage.prompt_tokens if resp.usage else 0,
        "completion_tokens": resp.usage.completion_tokens if resp.usage else 0,
        "total_tokens": resp.usage.total_tokens if resp.usage else 0,
    }
    return resp.choices[0].message.content or "", usage


def chat_stream(
    messages: list[dict], temperature: float = 0.0, model: str | None = None,
) -> Iterator[str]:
    stream = _client.chat.completions.create(
        model=model or DEFAULT_MODEL,
        messages=messages,
        temperature=temperature,
        stream=True,
    )
    for event in stream:
        delta = event.choices[0].delta.content if event.choices else None
        if delta:
            yield delta
