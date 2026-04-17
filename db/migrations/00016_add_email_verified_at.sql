-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
