-- +goose Up
-- +goose StatementBegin
CREATE TABLE auth_audit_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    event      TEXT NOT NULL,
    ip_address TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    metadata   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_audit_logs_user_id_created_at ON auth_audit_logs(user_id, created_at DESC);
CREATE INDEX idx_auth_audit_logs_event_created_at ON auth_audit_logs(event, created_at DESC);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS auth_audit_logs;
