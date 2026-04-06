-- +goose Up
-- +goose StatementBegin
CREATE TABLE deck_shares (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deck_id     UUID NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    share_code  TEXT NOT NULL UNIQUE,
    is_public   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deck_shares_deck_id ON deck_shares(deck_id);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS deck_shares;
