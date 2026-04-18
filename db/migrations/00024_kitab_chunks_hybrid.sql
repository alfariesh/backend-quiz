-- +goose Up
-- +goose StatementBegin

-- Hybrid retrieval for kitab_chunks:
--   - content_tsv: tsvector for lexical (BM25-like) ranking via ts_rank.
--     Using 'simple' config because Arabic/Indonesian do not have Postgres
--     stemmer support as stable as English; 'simple' preserves exact tokens.
--   - embedding_voyage: new column for voyage-context-3 output (1024-dim via
--     Matryoshka truncation from 2048). Coexists with the old 1536-dim
--     `embedding` column so we can A/B test and rollback.
--
-- The old `embedding vector(1536)` and its HNSW index stay untouched.
-- Drop them once voyage-context-3 is validated by eval harness.

ALTER TABLE kitab_chunks
  ADD COLUMN content_tsv tsvector
    GENERATED ALWAYS AS (to_tsvector('simple', content)) STORED;

CREATE INDEX idx_kitab_chunks_content_tsv
  ON kitab_chunks USING GIN (content_tsv);

ALTER TABLE kitab_chunks
  ADD COLUMN embedding_voyage vector(1024);

CREATE INDEX idx_kitab_chunks_embedding_voyage ON kitab_chunks
  USING hnsw (embedding_voyage vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_kitab_chunks_embedding_voyage;
ALTER TABLE kitab_chunks DROP COLUMN IF EXISTS embedding_voyage;
DROP INDEX IF EXISTS idx_kitab_chunks_content_tsv;
ALTER TABLE kitab_chunks DROP COLUMN IF EXISTS content_tsv;
-- +goose StatementEnd
