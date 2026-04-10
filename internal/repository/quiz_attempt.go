package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/repository/sqlc"
)

type QuizAttemptRepository struct {
	q *sqlc.Queries
}

func NewQuizAttemptRepository(pool *pgxpool.Pool) *QuizAttemptRepository {
	return &QuizAttemptRepository{q: sqlc.New(pool)}
}

func (r *QuizAttemptRepository) Create(ctx context.Context, attempt *domain.QuizAttempt) error {
	result, err := querier(r.q, ctx).CreateQuizAttempt(ctx, sqlc.CreateQuizAttemptParams{
		QuizID:         attempt.QuizID,
		UserID:         attempt.UserID,
		TotalPoints:    int32(attempt.TotalPoints),
		TotalQuestions: int32(attempt.TotalQuestions),
	})
	if err != nil {
		return err
	}
	*attempt = attemptFromSqlc(result)
	return nil
}

func (r *QuizAttemptRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizAttempt, error) {
	result, err := querier(r.q, ctx).GetQuizAttemptByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	a := attemptFromSqlc(result)
	return &a, nil
}

func (r *QuizAttemptRepository) Update(ctx context.Context, attempt *domain.QuizAttempt) error {
	return querier(r.q, ctx).UpdateQuizAttempt(ctx, sqlc.UpdateQuizAttemptParams{
		ID:          attempt.ID,
		CompletedAt: timeToNullable(attempt.CompletedAt),
		Score:       int32(attempt.Score),
		CorrectCount: int32(attempt.CorrectCount),
		DurationMs:  int32(attempt.DurationMS),
	})
}

func (r *QuizAttemptRepository) ListByQuizID(ctx context.Context, quizID uuid.UUID, limit, offset int) ([]domain.QuizAttempt, int, error) {
	rows, err := querier(r.q, ctx).ListQuizAttemptsByQuizID(ctx, sqlc.ListQuizAttemptsByQuizIDParams{
		QuizID: quizID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := querier(r.q, ctx).CountQuizAttemptsByQuizID(ctx, quizID)
	if err != nil {
		return nil, 0, err
	}

	attempts := make([]domain.QuizAttempt, len(rows))
	for i, row := range rows {
		attempts[i] = attemptFromSqlc(row)
	}
	return attempts, int(total), nil
}

func (r *QuizAttemptRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizAttempt, int, error) {
	rows, err := querier(r.q, ctx).ListQuizAttemptsByUserID(ctx, sqlc.ListQuizAttemptsByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := querier(r.q, ctx).CountQuizAttemptsByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	attempts := make([]domain.QuizAttempt, len(rows))
	for i, row := range rows {
		attempts[i] = attemptFromSqlc(row)
	}
	return attempts, int(total), nil
}

// Answer operations

func (r *QuizAttemptRepository) CreateAnswer(ctx context.Context, answer *domain.QuizAnswer) error {
	result, err := querier(r.q, ctx).CreateQuizAnswer(ctx, sqlc.CreateQuizAnswerParams{
		AttemptID:    answer.AttemptID,
		QuestionID:   answer.QuestionID,
		UserAnswer:   answer.UserAnswer,
		IsCorrect:    answer.IsCorrect,
		PointsEarned: int32(answer.PointsEarned),
		DurationMs:   int32(answer.DurationMS),
	})
	if err != nil {
		return err
	}
	*answer = answerFromSqlc(result)
	return nil
}

func (r *QuizAttemptRepository) ListAnswersByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]domain.QuizAnswer, error) {
	rows, err := querier(r.q, ctx).ListQuizAnswersByAttemptID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	answers := make([]domain.QuizAnswer, len(rows))
	for i, row := range rows {
		answers[i] = answerFromSqlc(row)
	}
	return answers, nil
}

func (r *QuizAttemptRepository) GetAnswerByAttemptAndQuestion(ctx context.Context, attemptID, questionID uuid.UUID) (*domain.QuizAnswer, error) {
	result, err := querier(r.q, ctx).GetQuizAnswerByAttemptAndQuestion(ctx, sqlc.GetQuizAnswerByAttemptAndQuestionParams{
		AttemptID:  attemptID,
		QuestionID: questionID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	a := answerFromSqlc(result)
	return &a, nil
}

func (r *QuizAttemptRepository) GetBestAttemptByQuizID(ctx context.Context, quizID uuid.UUID) (*domain.QuizAttempt, error) {
	result, err := querier(r.q, ctx).GetBestAttemptByQuizID(ctx, quizID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	a := attemptFromSqlc(result)
	return &a, nil
}
