-- +goose Up
-- +goose StatementBegin
CREATE TABLE decks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    is_archived     BOOLEAN NOT NULL DEFAULT false,
    new_cards_per_day INT,
    position        INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_decks_user_id ON decks(user_id);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS decks;
