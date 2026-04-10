package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

type CreateQuizRequest struct {
	DeckID           *uuid.UUID `json:"deck_id,omitempty"`
	Title            string     `json:"title" validate:"required,min=1,max=500"`
	Description      string     `json:"description" validate:"max=2000"`
	QuizType         string     `json:"quiz_type" validate:"required,oneof=mcq true_false fill_blank mixed mixed_ayat pretest posttest"`
	TimeLimitSeconds *int       `json:"time_limit_seconds,omitempty" validate:"omitempty,min=30,max=7200"`
	ShuffleQuestions *bool      `json:"shuffle_questions,omitempty"`
}

type UpdateQuizRequest struct {
	Title            *string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description      *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	QuizType         *string `json:"quiz_type,omitempty" validate:"omitempty,oneof=mcq true_false fill_blank mixed mixed_ayat pretest posttest"`
	TimeLimitSeconds *int    `json:"time_limit_seconds,omitempty" validate:"omitempty,min=30,max=7200"`
	ShuffleQuestions *bool   `json:"shuffle_questions,omitempty"`
	IsPublished      *bool   `json:"is_published,omitempty"`
}

type AddQuestionRequest struct {
	CardID        *uuid.UUID      `json:"card_id,omitempty"`
	QuestionType  string          `json:"question_type" validate:"required,oneof=mcq true_false fill_blank ayat_cloze ayat_continuation surah_identification ordering"`
	QuestionText  string          `json:"question_text" validate:"required,min=1"`
	Options       json.RawMessage `json:"options,omitempty"`
	CorrectAnswer string          `json:"correct_answer" validate:"required,min=1"`
	Explanation   string          `json:"explanation" validate:"max=2000"`
	Points        *int            `json:"points,omitempty" validate:"omitempty,min=1,max=100"`
}

type UpdateQuestionRequest struct {
	QuestionType  *string          `json:"question_type,omitempty" validate:"omitempty,oneof=mcq true_false fill_blank ayat_cloze ayat_continuation surah_identification ordering"`
	QuestionText  *string          `json:"question_text,omitempty" validate:"omitempty,min=1"`
	Options       *json.RawMessage `json:"options,omitempty"`
	CorrectAnswer *string          `json:"correct_answer,omitempty" validate:"omitempty,min=1"`
	Explanation   *string          `json:"explanation,omitempty" validate:"omitempty,max=2000"`
	Points        *int             `json:"points,omitempty" validate:"omitempty,min=1,max=100"`
}

type BatchAddQuestionsRequest struct {
	Questions []AddQuestionRequest `json:"questions" validate:"required,min=1,max=200,dive"`
}

type GenerateFromDeckRequest struct {
	DeckID       uuid.UUID `json:"deck_id" validate:"required"`
	QuestionType string    `json:"question_type" validate:"required,oneof=mcq true_false fill_blank mixed"`
	Count        int       `json:"count" validate:"required,min=1,max=100"`
}

type GenerateAyatQuizRequest struct {
	DeckID       uuid.UUID `json:"deck_id" validate:"required"`
	QuestionType string    `json:"question_type" validate:"required,oneof=ayat_cloze ayat_continuation surah_identification ordering mixed_ayat"`
	Count        int       `json:"count" validate:"required,min=1,max=100"`
}

type SubmitAnswerRequest struct {
	QuestionID uuid.UUID      `json:"question_id" validate:"required"`
	Answer     string         `json:"answer" validate:"required"`
	DurationMS int            `json:"duration_ms" validate:"min=0"`
	FSRSCard   *FSRSCardState `json:"fsrs_card,omitempty"`
	FSRSLog    *FSRSLogState  `json:"fsrs_log,omitempty"`
	Rating     *int           `json:"rating,omitempty" validate:"omitempty,min=1,max=4"`
}

type AnswerResult struct {
	Answer        domain.QuizAnswer `json:"answer"`
	IsCorrect     bool              `json:"is_correct"`
	Explanation   string            `json:"explanation"`
	CorrectAnswer string            `json:"correct_answer"`
	CardUpdated   bool              `json:"card_updated"`
	NextDue       *time.Time        `json:"next_due,omitempty"`
}

type QuizDetail struct {
	Quiz      domain.Quiz           `json:"quiz"`
	Questions []domain.QuizQuestion `json:"questions"`
}

type StartAttemptResponse struct {
	Attempt   domain.QuizAttempt  `json:"attempt"`
	Questions []QuestionForAttempt `json:"questions"`
}

type QuestionForAttempt struct {
	ID           uuid.UUID       `json:"id"`
	QuizID       uuid.UUID       `json:"quiz_id"`
	QuestionType string          `json:"question_type"`
	QuestionText string          `json:"question_text"`
	Options      json.RawMessage `json:"options,omitempty"`
	Position     int             `json:"position"`
	Points       int             `json:"points"`
}

type AttemptDetail struct {
	Attempt domain.QuizAttempt `json:"attempt"`
	Answers []QuizAnswerDetail `json:"answers"`
}

type QuizAnswerDetail struct {
	domain.QuizAnswer
	Question domain.QuizQuestion `json:"question"`
}
