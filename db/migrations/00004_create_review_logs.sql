-- +goose Up
-- +goose StatementBegin
CREATE TABLE review_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id         UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating          SMALLINT NOT NULL,
    state           SMALLINT NOT NULL,
    scheduled_days  INT NOT NULL,
    elapsed_days    INT NOT NULL,
    stability       REAL NOT NULL,
    difficulty      REAL NOT NULL,
    duration_ms     INT NOT NULL DEFAULT 0,
    reviewed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_review_logs_card_id ON review_logs(card_id);
CREATE INDEX idx_review_logs_user_id_date ON review_logs(user_id, reviewed_at);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS review_logs;
