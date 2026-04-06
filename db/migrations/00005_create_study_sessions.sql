-- +goose Up
-- +goose StatementBegin
CREATE TABLE study_sessions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id           UUID REFERENCES decks(id) ON DELETE SET NULL,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at          TIMESTAMPTZ,
    new_count         INT NOT NULL DEFAULT 0,
    review_count      INT NOT NULL DEFAULT 0,
    relearn_count     INT NOT NULL DEFAULT 0,
    total_duration_ms INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_study_sessions_user_id ON study_sessions(user_id, started_at);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS study_sessions;
