-- +goose Up
-- +goose StatementBegin
CREATE TABLE cards (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deck_id         UUID NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    front           TEXT NOT NULL,
    back            TEXT NOT NULL,
    tags            TEXT[] NOT NULL DEFAULT '{}',
    due             TIMESTAMPTZ NOT NULL DEFAULT now(),
    stability       REAL NOT NULL DEFAULT 0,
    difficulty      REAL NOT NULL DEFAULT 0,
    elapsed_days    INT NOT NULL DEFAULT 0,
    scheduled_days  INT NOT NULL DEFAULT 0,
    reps            INT NOT NULL DEFAULT 0,
    lapses          INT NOT NULL DEFAULT 0,
    state           SMALLINT NOT NULL DEFAULT 0,
    last_review     TIMESTAMPTZ,
    is_suspended    BOOLEAN NOT NULL DEFAULT false,
    position        INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cards_deck_id ON cards(deck_id);
CREATE INDEX idx_cards_due_state ON cards(deck_id, state, due) WHERE NOT is_suspended;
CREATE INDEX idx_cards_tags ON cards USING GIN(tags);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS cards;
