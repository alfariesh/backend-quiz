-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE language_code AS ENUM ('ar', 'en', 'id', 'ur', 'ms');

-- ── AUTHORS ──────────────────────────────────────────────────
CREATE TABLE authors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,
    name            JSONB NOT NULL DEFAULT '{}'::jsonb,
    full_name       JSONB NOT NULL DEFAULT '{}'::jsonb,
    title           JSONB NOT NULL DEFAULT '{}'::jsonb,
    bio             JSONB NOT NULL DEFAULT '{}'::jsonb,
    quote           JSONB NOT NULL DEFAULT '{}'::jsonb,
    birth_year      INT,
    death_year      INT,
    birth_place     TEXT NOT NULL DEFAULT '',
    nationality     TEXT NOT NULL DEFAULT '',
    era             TEXT NOT NULL DEFAULT '',
    madhhab         TEXT NOT NULL DEFAULT '',
    expertise       TEXT[] NOT NULL DEFAULT '{}',
    notable_works   TEXT[] NOT NULL DEFAULT '{}',
    teachers        TEXT[] NOT NULL DEFAULT '{}',
    image_url       TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_authors_madhhab ON authors(madhhab);
CREATE INDEX idx_authors_era ON authors(era);
CREATE INDEX idx_authors_name ON authors USING GIN (name);
CREATE INDEX idx_authors_metadata ON authors USING GIN (metadata);

-- ── GENRES ───────────────────────────────────────────────────
CREATE TABLE genres (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,
    name            JSONB NOT NULL DEFAULT '{}'::jsonb,
    description     JSONB NOT NULL DEFAULT '{}'::jsonb,
    icon            TEXT NOT NULL DEFAULT '',
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── KITAB ────────────────────────────────────────────────────
CREATE TABLE kitab (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                    TEXT NOT NULL UNIQUE,
    author_id               UUID NOT NULL REFERENCES authors(id) ON DELETE RESTRICT,
    genre_id                UUID NOT NULL REFERENCES genres(id) ON DELETE RESTRICT,
    matan_kitab_id          UUID REFERENCES kitab(id) ON DELETE SET NULL,
    title                   JSONB NOT NULL DEFAULT '{}'::jsonb,
    original_title          TEXT NOT NULL DEFAULT '',
    synopsis                JSONB NOT NULL DEFAULT '{}'::jsonb,
    target_audience         JSONB NOT NULL DEFAULT '{}'::jsonb,
    main_topics             TEXT[] NOT NULL DEFAULT '{}',
    tags                    TEXT[] NOT NULL DEFAULT '{}',
    kitab_type              TEXT NOT NULL DEFAULT 'standalone',
    difficulty_level        TEXT NOT NULL DEFAULT '',
    era                     TEXT NOT NULL DEFAULT '',
    madhhab                 TEXT NOT NULL DEFAULT '',
    source_language         language_code NOT NULL DEFAULT 'ar',
    pages                   INT NOT NULL DEFAULT 0,
    publisher               TEXT NOT NULL DEFAULT '',
    published_year          INT,
    pdf_storage_path        TEXT NOT NULL DEFAULT '',
    cover_url               TEXT NOT NULL DEFAULT '',
    accent_color            TEXT NOT NULL DEFAULT '',
    seo                     JSONB NOT NULL DEFAULT '{}'::jsonb,
    rating                  REAL,
    total_readers           INT NOT NULL DEFAULT 0,
    is_published            BOOLEAN NOT NULL DEFAULT false,
    tree_processed          BOOLEAN NOT NULL DEFAULT false,
    embeddings_processed    BOOLEAN NOT NULL DEFAULT false,
    metadata                JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_kitab_author_id ON kitab(author_id);
CREATE INDEX idx_kitab_genre_id ON kitab(genre_id);
CREATE INDEX idx_kitab_matan_kitab_id ON kitab(matan_kitab_id);
CREATE INDEX idx_kitab_madhhab ON kitab(madhhab);
CREATE INDEX idx_kitab_is_published ON kitab(is_published) WHERE is_published;
CREATE INDEX idx_kitab_tags ON kitab USING GIN (tags);
CREATE INDEX idx_kitab_title ON kitab USING GIN (title);
CREATE INDEX idx_kitab_metadata ON kitab USING GIN (metadata);

-- ── KITAB TREE ───────────────────────────────────────────────
CREATE TABLE kitab_tree (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kitab_id        UUID NOT NULL REFERENCES kitab(id) ON DELETE CASCADE,
    parent_id       UUID REFERENCES kitab_tree(id) ON DELETE CASCADE,
    node_id         TEXT NOT NULL,
    title           JSONB NOT NULL DEFAULT '{}'::jsonb,
    summary         JSONB NOT NULL DEFAULT '{}'::jsonb,
    content         TEXT NOT NULL DEFAULT '',
    start_page      INT NOT NULL DEFAULT 0,
    end_page        INT NOT NULL DEFAULT 0,
    depth           INT NOT NULL DEFAULT 0,
    sort_order      INT NOT NULL DEFAULT 0,
    icon_url        TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (kitab_id, node_id)
);

CREATE INDEX idx_kitab_tree_kitab_id ON kitab_tree(kitab_id);
CREATE INDEX idx_kitab_tree_parent_id ON kitab_tree(parent_id);
CREATE INDEX idx_kitab_tree_kitab_depth ON kitab_tree(kitab_id, depth);

-- ── KITAB CHUNKS ─────────────────────────────────────────────
CREATE TABLE kitab_chunks (
    id              BIGSERIAL PRIMARY KEY,
    kitab_id        UUID NOT NULL REFERENCES kitab(id) ON DELETE CASCADE,
    tree_node_id    UUID REFERENCES kitab_tree(id) ON DELETE SET NULL,
    content         TEXT NOT NULL,
    embedding       vector(1536),
    start_page      INT NOT NULL DEFAULT 0,
    end_page        INT NOT NULL DEFAULT 0,
    token_count     INT NOT NULL DEFAULT 0,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_kitab_chunks_kitab_id ON kitab_chunks(kitab_id);
CREATE INDEX idx_kitab_chunks_tree_node_id ON kitab_chunks(tree_node_id);
CREATE INDEX idx_kitab_chunks_embedding ON kitab_chunks
    USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);
CREATE INDEX idx_kitab_chunks_content_trgm ON kitab_chunks USING GIN (content gin_trgm_ops);
CREATE INDEX idx_kitab_chunks_metadata ON kitab_chunks USING GIN (metadata);

-- ── KITAB SYARAH LINKS (matn ↔ syarah per node) ──────────────
CREATE TABLE kitab_syarah_links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matan_node_id   UUID NOT NULL REFERENCES kitab_tree(id) ON DELETE CASCADE,
    syarah_node_id  UUID NOT NULL REFERENCES kitab_tree(id) ON DELETE CASCADE,
    link_type       TEXT NOT NULL DEFAULT 'explains',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (matan_node_id, syarah_node_id)
);

CREATE INDEX idx_kitab_syarah_links_matan ON kitab_syarah_links(matan_node_id);
CREATE INDEX idx_kitab_syarah_links_syarah ON kitab_syarah_links(syarah_node_id);

-- ── FATWA ISSUERS ────────────────────────────────────────────
CREATE TABLE fatwa_issuers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,
    name            JSONB NOT NULL DEFAULT '{}'::jsonb,
    description     JSONB NOT NULL DEFAULT '{}'::jsonb,
    region          TEXT NOT NULL DEFAULT '',
    madhhab_leaning TEXT NOT NULL DEFAULT '',
    website         TEXT NOT NULL DEFAULT '',
    logo_url        TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── FATWA ────────────────────────────────────────────────────
CREATE TABLE fatwa (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                    TEXT NOT NULL UNIQUE,
    issuer_id               UUID NOT NULL REFERENCES fatwa_issuers(id) ON DELETE RESTRICT,
    title                   JSONB NOT NULL DEFAULT '{}'::jsonb,
    question                JSONB NOT NULL DEFAULT '{}'::jsonb,
    tags                    TEXT[] NOT NULL DEFAULT '{}',
    source_language         language_code NOT NULL DEFAULT 'id',
    issued_at               DATE,
    source_url              TEXT NOT NULL DEFAULT '',
    is_published            BOOLEAN NOT NULL DEFAULT false,
    tree_processed          BOOLEAN NOT NULL DEFAULT false,
    embeddings_processed    BOOLEAN NOT NULL DEFAULT false,
    metadata                JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fatwa_issuer_id ON fatwa(issuer_id);
CREATE INDEX idx_fatwa_issued_at ON fatwa(issued_at);
CREATE INDEX idx_fatwa_is_published ON fatwa(is_published) WHERE is_published;
CREATE INDEX idx_fatwa_tags ON fatwa USING GIN (tags);
CREATE INDEX idx_fatwa_title ON fatwa USING GIN (title);
CREATE INDEX idx_fatwa_metadata ON fatwa USING GIN (metadata);

-- ── FATWA TREE ───────────────────────────────────────────────
CREATE TABLE fatwa_tree (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fatwa_id        UUID NOT NULL REFERENCES fatwa(id) ON DELETE CASCADE,
    parent_id       UUID REFERENCES fatwa_tree(id) ON DELETE CASCADE,
    node_id         TEXT NOT NULL,
    title           JSONB NOT NULL DEFAULT '{}'::jsonb,
    summary         JSONB NOT NULL DEFAULT '{}'::jsonb,
    content         TEXT NOT NULL DEFAULT '',
    depth           INT NOT NULL DEFAULT 0,
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (fatwa_id, node_id)
);

CREATE INDEX idx_fatwa_tree_fatwa_id ON fatwa_tree(fatwa_id);
CREATE INDEX idx_fatwa_tree_parent_id ON fatwa_tree(parent_id);

-- ── FATWA CHUNKS ─────────────────────────────────────────────
CREATE TABLE fatwa_chunks (
    id              BIGSERIAL PRIMARY KEY,
    fatwa_id        UUID NOT NULL REFERENCES fatwa(id) ON DELETE CASCADE,
    tree_node_id    UUID REFERENCES fatwa_tree(id) ON DELETE SET NULL,
    content         TEXT NOT NULL,
    embedding       vector(1536),
    token_count     INT NOT NULL DEFAULT 0,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fatwa_chunks_fatwa_id ON fatwa_chunks(fatwa_id);
CREATE INDEX idx_fatwa_chunks_tree_node_id ON fatwa_chunks(tree_node_id);
CREATE INDEX idx_fatwa_chunks_embedding ON fatwa_chunks
    USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);
CREATE INDEX idx_fatwa_chunks_content_trgm ON fatwa_chunks USING GIN (content gin_trgm_ops);
CREATE INDEX idx_fatwa_chunks_metadata ON fatwa_chunks USING GIN (metadata);

-- ── CONVERSATIONS ────────────────────────────────────────────
CREATE TABLE conversations (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode                    TEXT NOT NULL CHECK (mode IN ('per_kitab', 'per_fatwa', 'general')),
    scope_kitab_id          UUID REFERENCES kitab(id) ON DELETE SET NULL,
    scope_fatwa_id          UUID REFERENCES fatwa(id) ON DELETE SET NULL,
    style                   TEXT NOT NULL DEFAULT 'standard'
                              CHECK (style IN ('ringkas', 'standard', 'syarah')),
    flags                   JSONB NOT NULL DEFAULT '{}'::jsonb,
    title                   TEXT NOT NULL DEFAULT '',
    message_count           INT NOT NULL DEFAULT 0,
    last_message_preview    TEXT NOT NULL DEFAULT '',
    is_bookmarked           BOOLEAN NOT NULL DEFAULT false,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_scope_kitab_id ON conversations(scope_kitab_id);
CREATE INDEX idx_conversations_scope_fatwa_id ON conversations(scope_fatwa_id);
CREATE INDEX idx_conversations_user_updated ON conversations(user_id, updated_at DESC);

-- ── MESSAGES ─────────────────────────────────────────────────
CREATE TABLE messages (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id         UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role                    TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content                 TEXT NOT NULL,
    parts                   JSONB,
    reasoning               TEXT NOT NULL DEFAULT '',
    citations               JSONB NOT NULL DEFAULT '[]'::jsonb,
    suggested_questions     TEXT[] NOT NULL DEFAULT '{}',
    retrieval_strategy      TEXT NOT NULL DEFAULT '',
    token_usage             JSONB NOT NULL DEFAULT '{}'::jsonb,
    latency_ms              INT NOT NULL DEFAULT 0,
    feedback                TEXT CHECK (feedback IS NULL OR feedback IN ('up', 'down')),
    feedback_comment        TEXT NOT NULL DEFAULT '',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_conversation_created ON messages(conversation_id, created_at);

-- ── CHUNK BOOKMARKS (bridge ke cards / FSRS Segment 1) ───────
CREATE TABLE chunk_bookmarks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_type     TEXT NOT NULL CHECK (source_type IN ('kitab', 'fatwa')),
    kitab_chunk_id  BIGINT REFERENCES kitab_chunks(id) ON DELETE CASCADE,
    fatwa_chunk_id  BIGINT REFERENCES fatwa_chunks(id) ON DELETE CASCADE,
    note            TEXT NOT NULL DEFAULT '',
    card_id         UUID REFERENCES cards(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chunk_bookmarks_source_consistency CHECK (
        (source_type = 'kitab' AND kitab_chunk_id IS NOT NULL AND fatwa_chunk_id IS NULL)
        OR
        (source_type = 'fatwa' AND fatwa_chunk_id IS NOT NULL AND kitab_chunk_id IS NULL)
    )
);

CREATE UNIQUE INDEX idx_chunk_bookmarks_user_kitab
    ON chunk_bookmarks(user_id, kitab_chunk_id)
    WHERE kitab_chunk_id IS NOT NULL;
CREATE UNIQUE INDEX idx_chunk_bookmarks_user_fatwa
    ON chunk_bookmarks(user_id, fatwa_chunk_id)
    WHERE fatwa_chunk_id IS NOT NULL;
CREATE INDEX idx_chunk_bookmarks_user_id ON chunk_bookmarks(user_id);
CREATE INDEX idx_chunk_bookmarks_card_id ON chunk_bookmarks(card_id);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS chunk_bookmarks;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS fatwa_chunks;
DROP TABLE IF EXISTS fatwa_tree;
DROP TABLE IF EXISTS fatwa;
DROP TABLE IF EXISTS fatwa_issuers;
DROP TABLE IF EXISTS kitab_syarah_links;
DROP TABLE IF EXISTS kitab_chunks;
DROP TABLE IF EXISTS kitab_tree;
DROP TABLE IF EXISTS kitab;
DROP TABLE IF EXISTS genres;
DROP TABLE IF EXISTS authors;
DROP TYPE IF EXISTS language_code;
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS vector;
