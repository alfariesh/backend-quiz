-- +goose Up
-- +goose StatementBegin
CREATE TABLE media (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id     UUID REFERENCES cards(id) ON DELETE SET NULL,
    file_name   TEXT NOT NULL,
    file_size   INT NOT NULL,
    mime_type   TEXT NOT NULL,
    r2_key      TEXT NOT NULL,
    url         TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_media_user_id ON media(user_id);
CREATE INDEX idx_media_card_id ON media(card_id);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS media;
