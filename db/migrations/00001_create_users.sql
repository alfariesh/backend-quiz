-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email             TEXT NOT NULL UNIQUE,
    password_hash     TEXT NOT NULL DEFAULT '',
    display_name      TEXT NOT NULL DEFAULT '',
    timezone          TEXT NOT NULL DEFAULT 'UTC',
    desired_retention REAL NOT NULL DEFAULT 0.9,
    daily_new_limit   INT NOT NULL DEFAULT 20,
    daily_review_limit INT NOT NULL DEFAULT 200,
    fsrs_weights      REAL[] DEFAULT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS users;
