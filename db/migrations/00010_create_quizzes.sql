-- +goose Up
-- +goose StatementBegin
CREATE TABLE quizzes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id             UUID REFERENCES decks(id) ON DELETE SET NULL,
    title               VARCHAR(500) NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    quiz_type           VARCHAR(20) NOT NULL DEFAULT 'mixed',
    time_limit_seconds  INT,
    shuffle_questions   BOOLEAN NOT NULL DEFAULT true,
    is_published        BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_quizzes_user_id ON quizzes(user_id);
CREATE INDEX idx_quizzes_deck_id ON quizzes(deck_id);

CREATE TABLE quiz_questions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id         UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    card_id         UUID REFERENCES cards(id) ON DELETE SET NULL,
    question_type   VARCHAR(20) NOT NULL,
    question_text   TEXT NOT NULL,
    options         JSONB,
    correct_answer  TEXT NOT NULL,
    explanation     TEXT NOT NULL DEFAULT '',
    position        INT NOT NULL DEFAULT 0,
    points          INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_quiz_questions_quiz_id ON quiz_questions(quiz_id);
CREATE INDEX idx_quiz_questions_card_id ON quiz_questions(card_id);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS quiz_questions;
DROP TABLE IF EXISTS quizzes;
