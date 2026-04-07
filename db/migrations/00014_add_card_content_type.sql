-- +goose Up
-- +goose StatementBegin
ALTER TABLE cards ADD COLUMN content_type VARCHAR(20) NOT NULL DEFAULT 'plain';
-- +goose StatementEnd

-- +goose Down
ALTER TABLE cards DROP COLUMN content_type;
