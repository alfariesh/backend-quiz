-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN deletion_requested_at TIMESTAMPTZ;
CREATE INDEX idx_users_deletion_requested_at
    ON users(deletion_requested_at)
    WHERE deletion_requested_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_deletion_requested_at;
ALTER TABLE users DROP COLUMN IF EXISTS deletion_requested_at;
-- +goose StatementEnd
