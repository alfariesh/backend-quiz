-- +goose Up
-- +goose StatementBegin
CREATE TABLE oauth_accounts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider     TEXT NOT NULL,
    provider_id  TEXT NOT NULL,
    email        TEXT NOT NULL,
    avatar_url   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_id)
);

CREATE INDEX idx_oauth_accounts_user_id ON oauth_accounts(user_id);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS oauth_accounts;
