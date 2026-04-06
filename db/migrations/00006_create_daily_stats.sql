-- +goose Up
-- +goose StatementBegin
CREATE TABLE daily_stats (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date              DATE NOT NULL,
    new_cards         INT NOT NULL DEFAULT 0,
    reviews           INT NOT NULL DEFAULT 0,
    relearns          INT NOT NULL DEFAULT 0,
    total_duration_ms INT NOT NULL DEFAULT 0,
    retention_rate    REAL,
    UNIQUE(user_id, date)
);

CREATE INDEX idx_daily_stats_user_date ON daily_stats(user_id, date);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS daily_stats;
