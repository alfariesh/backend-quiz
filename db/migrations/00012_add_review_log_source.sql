-- +goose Up
-- +goose StatementBegin
ALTER TABLE review_logs ADD COLUMN source VARCHAR(20) NOT NULL DEFAULT 'flashcard';
-- +goose StatementEnd

-- +goose Down
ALTER TABLE review_logs DROP COLUMN source;
