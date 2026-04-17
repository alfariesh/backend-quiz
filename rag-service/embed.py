import os

from openai import OpenAI

_client = OpenAI(
    api_key=os.environ["EMBED_API_KEY"],
    base_url=os.environ["EMBED_BASE_URL"],
)

EMBED_MODEL = os.getenv("EMBED_MODEL", "text-embedding-3-large")
EMBED_DIMENSIONS = int(os.getenv("EMBED_DIMENSIONS", "1536"))


def embed_query(text: str) -> list[float]:
    resp = _client.embeddings.create(
        model=EMBED_MODEL,
        input=[text],
        dimensions=EMBED_DIMENSIONS,
    )
    return resp.data[0].embedding
