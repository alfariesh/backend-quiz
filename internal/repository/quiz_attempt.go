package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type QuizAttemptRepository struct {
	db *pgxpool.Pool
}

func NewQuizAttemptRepository(db *pgxpool.Pool) *QuizAttemptRepository {
	return &QuizAttemptRepository{db: db}
}

func (r *QuizAttemptRepository) Create(ctx context.Context, attempt *domain.QuizAttempt) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO quiz_attempts (quiz_id, user_id, total_points, total_questions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, started_at, score, correct_count, duration_ms, created_at`,
		attempt.QuizID, attempt.UserID, attempt.TotalPoints, attempt.TotalQuestions,
	).Scan(&attempt.ID, &attempt.StartedAt, &attempt.Score, &attempt.CorrectCount,
		&attempt.DurationMS, &attempt.CreatedAt)
}

func (r *QuizAttemptRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizAttempt, error) {
	var a domain.QuizAttempt
	err := r.db.QueryRow(ctx,
		`SELECT id, quiz_id, user_id, started_at, completed_at, score, total_points, total_questions, correct_count, duration_ms, created_at
		FROM quiz_attempts WHERE id = $1`, id,
	).Scan(&a.ID, &a.QuizID, &a.UserID, &a.StartedAt, &a.CompletedAt, &a.Score,
		&a.TotalPoints, &a.TotalQuestions, &a.CorrectCount, &a.DurationMS, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &a, err
}

func (r *QuizAttemptRepository) Update(ctx context.Context, attempt *domain.QuizAttempt) error {
	_, err := r.db.Exec(ctx,
		`UPDATE quiz_attempts SET completed_at=$2, score=$3, correct_count=$4, duration_ms=$5
		WHERE id = $1`,
		attempt.ID, attempt.CompletedAt, attempt.Score, attempt.CorrectCount, attempt.DurationMS,
	)
	return err
}

func (r *QuizAttemptRepository) ListByQuizID(ctx context.Context, quizID uuid.UUID, limit, offset int) ([]domain.QuizAttempt, int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, quiz_id, user_id, started_at, completed_at, score, total_points, total_questions, correct_count, duration_ms, created_at
		FROM quiz_attempts WHERE quiz_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`, quizID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var attempts []domain.QuizAttempt
	for rows.Next() {
		var a domain.QuizAttempt
		if err := rows.Scan(&a.ID, &a.QuizID, &a.UserID, &a.StartedAt, &a.CompletedAt, &a.Score,
			&a.TotalPoints, &a.TotalQuestions, &a.CorrectCount, &a.DurationMS, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		attempts = append(attempts, a)
	}

	var total int
	err = r.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM quiz_attempts WHERE quiz_id = $1`, quizID).Scan(&total)
	return attempts, total, err
}

func (r *QuizAttemptRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizAttempt, int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, quiz_id, user_id, started_at, completed_at, score, total_points, total_questions, correct_count, duration_ms, created_at
		FROM quiz_attempts WHERE user_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var attempts []domain.QuizAttempt
	for rows.Next() {
		var a domain.QuizAttempt
		if err := rows.Scan(&a.ID, &a.QuizID, &a.UserID, &a.StartedAt, &a.CompletedAt, &a.Score,
			&a.TotalPoints, &a.TotalQuestions, &a.CorrectCount, &a.DurationMS, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		attempts = append(attempts, a)
	}

	var total int
	err = r.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM quiz_attempts WHERE user_id = $1`, userID).Scan(&total)
	return attempts, total, err
}

// Answer operations

func (r *QuizAttemptRepository) CreateAnswer(ctx context.Context, answer *domain.QuizAnswer) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO quiz_answers (attempt_id, question_id, user_answer, is_correct, points_earned, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, answered_at`,
		answer.AttemptID, answer.QuestionID, answer.UserAnswer, answer.IsCorrect,
		answer.PointsEarned, answer.DurationMS,
	).Scan(&answer.ID, &answer.AnsweredAt)
}

func (r *QuizAttemptRepository) ListAnswersByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]domain.QuizAnswer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, attempt_id, question_id, user_answer, is_correct, points_earned, duration_ms, answered_at
		FROM quiz_answers WHERE attempt_id = $1
		ORDER BY answered_at`, attemptID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var answers []domain.QuizAnswer
	for rows.Next() {
		var a domain.QuizAnswer
		if err := rows.Scan(&a.ID, &a.AttemptID, &a.QuestionID, &a.UserAnswer, &a.IsCorrect,
			&a.PointsEarned, &a.DurationMS, &a.AnsweredAt); err != nil {
			return nil, err
		}
		answers = append(answers, a)
	}
	return answers, nil
}

func (r *QuizAttemptRepository) GetAnswerByAttemptAndQuestion(ctx context.Context, attemptID, questionID uuid.UUID) (*domain.QuizAnswer, error) {
	var a domain.QuizAnswer
	err := r.db.QueryRow(ctx,
		`SELECT id, attempt_id, question_id, user_answer, is_correct, points_earned, duration_ms, answered_at
		FROM quiz_answers WHERE attempt_id = $1 AND question_id = $2`,
		attemptID, questionID,
	).Scan(&a.ID, &a.AttemptID, &a.QuestionID, &a.UserAnswer, &a.IsCorrect,
		&a.PointsEarned, &a.DurationMS, &a.AnsweredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &a, err
}

func (r *QuizAttemptRepository) GetBestAttemptByQuizID(ctx context.Context, quizID uuid.UUID) (*domain.QuizAttempt, error) {
	var a domain.QuizAttempt
	err := r.db.QueryRow(ctx,
		`SELECT id, quiz_id, user_id, started_at, completed_at, score, total_points, total_questions, correct_count, duration_ms, created_at
		FROM quiz_attempts WHERE quiz_id = $1 AND completed_at IS NOT NULL
		ORDER BY score DESC
		LIMIT 1`, quizID,
	).Scan(&a.ID, &a.QuizID, &a.UserID, &a.StartedAt, &a.CompletedAt, &a.Score,
		&a.TotalPoints, &a.TotalQuestions, &a.CorrectCount, &a.DurationMS, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &a, err
}
