-- +goose Up
ALTER TABLE users
    ADD COLUMN reminder_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN reminder_time TIME NOT NULL DEFAULT '08:00';

-- +goose Down
ALTER TABLE users
    DROP COLUMN reminder_enabled,
    DROP COLUMN reminder_time;
