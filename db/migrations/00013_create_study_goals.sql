-- +goose Up
-- +goose StatementBegin
CREATE TABLE study_goals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal_type       VARCHAR(20) NOT NULL,
    target_value    INT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_study_goals_user_type ON study_goals(user_id, goal_type);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS study_goals;
