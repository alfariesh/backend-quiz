package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	QuizTypeMCQ       = "mcq"
	QuizTypeTrueFalse = "true_false"
	QuizTypeFillBlank = "fill_blank"
	QuizTypeMixed     = "mixed"
	QuizTypeMixedAyat = "mixed_ayat"
	QuizTypePretest   = "pretest"
	QuizTypePosttest  = "posttest"
)

const (
	QuestionTypeMCQ       = "mcq"
	QuestionTypeTrueFalse = "true_false"
	QuestionTypeFillBlank = "fill_blank"

	// Ayat/Hadits question types
	QuestionTypeAyatCloze        = "ayat_cloze"
	QuestionTypeAyatContinuation = "ayat_continuation"
	QuestionTypeSurahID          = "surah_identification"
	QuestionTypeOrdering         = "ordering"
)

type Quiz struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	DeckID           *uuid.UUID `json:"deck_id,omitempty"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	QuizType         string     `json:"quiz_type"`
	TimeLimitSeconds *int       `json:"time_limit_seconds,omitempty"`
	ShuffleQuestions bool       `json:"shuffle_questions"`
	IsPublished      bool       `json:"is_published"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type QuizWithCounts struct {
	Quiz
	QuestionCount int `json:"question_count"`
	AttemptCount  int `json:"attempt_count"`
}

type MCQOption struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

type QuizQuestion struct {
	ID            uuid.UUID       `json:"id"`
	QuizID        uuid.UUID       `json:"quiz_id"`
	CardID        *uuid.UUID      `json:"card_id,omitempty"`
	QuestionType  string          `json:"question_type"`
	QuestionText  string          `json:"question_text"`
	Options       json.RawMessage `json:"options,omitempty"`
	CorrectAnswer string          `json:"correct_answer"`
	Explanation   string          `json:"explanation"`
	Position      int             `json:"position"`
	Points        int             `json:"points"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type QuizAttempt struct {
	ID             uuid.UUID  `json:"id"`
	QuizID         uuid.UUID  `json:"quiz_id"`
	UserID         uuid.UUID  `json:"user_id"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Score          int        `json:"score"`
	TotalPoints    int        `json:"total_points"`
	TotalQuestions int        `json:"total_questions"`
	CorrectCount   int        `json:"correct_count"`
	DurationMS     int        `json:"duration_ms"`
	CreatedAt      time.Time  `json:"created_at"`
}

type QuizAnswer struct {
	ID           uuid.UUID `json:"id"`
	AttemptID    uuid.UUID `json:"attempt_id"`
	QuestionID   uuid.UUID `json:"question_id"`
	UserAnswer   string    `json:"user_answer"`
	IsCorrect    bool      `json:"is_correct"`
	PointsEarned int       `json:"points_earned"`
	DurationMS   int       `json:"duration_ms"`
	AnsweredAt   time.Time `json:"answered_at"`
}

type QuizRepository interface {
	Create(ctx context.Context, quiz *Quiz) error
	GetByID(ctx context.Context, id uuid.UUID) (*Quiz, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]QuizWithCounts, int, error)
	Update(ctx context.Context, quiz *Quiz) error
	Delete(ctx context.Context, id uuid.UUID) error

	CreateQuestion(ctx context.Context, q *QuizQuestion) error
	BulkCreateQuestions(ctx context.Context, questions []*QuizQuestion) error
	GetQuestionByID(ctx context.Context, id uuid.UUID) (*QuizQuestion, error)
	ListQuestionsByQuizID(ctx context.Context, quizID uuid.UUID) ([]QuizQuestion, error)
	UpdateQuestion(ctx context.Context, q *QuizQuestion) error
	DeleteQuestion(ctx context.Context, id uuid.UUID) error
	CountQuestionsByQuizID(ctx context.Context, quizID uuid.UUID) (int, error)
	ListByDeckAndType(ctx context.Context, deckID uuid.UUID, quizType string) ([]Quiz, error)
}

type QuizAttemptRepository interface {
	Create(ctx context.Context, attempt *QuizAttempt) error
	GetByID(ctx context.Context, id uuid.UUID) (*QuizAttempt, error)
	Update(ctx context.Context, attempt *QuizAttempt) error
	ListByQuizID(ctx context.Context, quizID uuid.UUID, limit, offset int) ([]QuizAttempt, int, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]QuizAttempt, int, error)

	CreateAnswer(ctx context.Context, answer *QuizAnswer) error
	ListAnswersByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]QuizAnswer, error)
	GetAnswerByAttemptAndQuestion(ctx context.Context, attemptID, questionID uuid.UUID) (*QuizAnswer, error)
	GetBestAttemptByQuizID(ctx context.Context, quizID uuid.UUID) (*QuizAttempt, error)
}
