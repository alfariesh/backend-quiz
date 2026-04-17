-- +goose Up
-- +goose StatementBegin
CREATE TABLE login_attempts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT NOT NULL,
    ip_address   TEXT NOT NULL DEFAULT '',
    successful   BOOLEAN NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_login_attempts_email_created_at ON login_attempts(email, created_at DESC);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS login_attempts;
