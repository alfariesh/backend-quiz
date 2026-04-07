package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type QuizRepository struct {
	db *pgxpool.Pool
}

func NewQuizRepository(db *pgxpool.Pool) *QuizRepository {
	return &QuizRepository{db: db}
}

func (r *QuizRepository) Create(ctx context.Context, quiz *domain.Quiz) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO quizzes (user_id, deck_id, title, description, quiz_type, time_limit_seconds, shuffle_questions)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, is_published, created_at, updated_at`,
		quiz.UserID, quiz.DeckID, quiz.Title, quiz.Description, quiz.QuizType,
		quiz.TimeLimitSeconds, quiz.ShuffleQuestions,
	).Scan(&quiz.ID, &quiz.IsPublished, &quiz.CreatedAt, &quiz.UpdatedAt)
}

func (r *QuizRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Quiz, error) {
	var q domain.Quiz
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, deck_id, title, description, quiz_type, time_limit_seconds,
			shuffle_questions, is_published, created_at, updated_at
		FROM quizzes WHERE id = $1`, id,
	).Scan(&q.ID, &q.UserID, &q.DeckID, &q.Title, &q.Description, &q.QuizType,
		&q.TimeLimitSeconds, &q.ShuffleQuestions, &q.IsPublished, &q.CreatedAt, &q.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &q, err
}

func (r *QuizRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizWithCounts, int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT q.id, q.user_id, q.deck_id, q.title, q.description, q.quiz_type, q.time_limit_seconds,
			q.shuffle_questions, q.is_published, q.created_at, q.updated_at,
			COALESCE(qc.cnt, 0)::int AS question_count,
			COALESCE(ac.cnt, 0)::int AS attempt_count
		FROM quizzes q
		LEFT JOIN LATERAL (SELECT COUNT(*) AS cnt FROM quiz_questions WHERE quiz_id = q.id) qc ON true
		LEFT JOIN LATERAL (SELECT COUNT(*) AS cnt FROM quiz_attempts WHERE quiz_id = q.id) ac ON true
		WHERE q.user_id = $1
		ORDER BY q.created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var quizzes []domain.QuizWithCounts
	for rows.Next() {
		var qc domain.QuizWithCounts
		if err := rows.Scan(&qc.ID, &qc.UserID, &qc.DeckID, &qc.Title, &qc.Description, &qc.QuizType,
			&qc.TimeLimitSeconds, &qc.ShuffleQuestions, &qc.IsPublished, &qc.CreatedAt, &qc.UpdatedAt,
			&qc.QuestionCount, &qc.AttemptCount); err != nil {
			return nil, 0, err
		}
		quizzes = append(quizzes, qc)
	}

	var total int
	err = r.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM quizzes WHERE user_id = $1`, userID).Scan(&total)
	return quizzes, total, err
}

func (r *QuizRepository) Update(ctx context.Context, quiz *domain.Quiz) error {
	_, err := r.db.Exec(ctx,
		`UPDATE quizzes SET title=$2, description=$3, quiz_type=$4, time_limit_seconds=$5,
			shuffle_questions=$6, is_published=$7, updated_at=now()
		WHERE id = $1`,
		quiz.ID, quiz.Title, quiz.Description, quiz.QuizType,
		quiz.TimeLimitSeconds, quiz.ShuffleQuestions, quiz.IsPublished,
	)
	return err
}

func (r *QuizRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM quizzes WHERE id = $1`, id)
	return err
}

// Question operations

func (r *QuizRepository) CreateQuestion(ctx context.Context, q *domain.QuizQuestion) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO quiz_questions (quiz_id, card_id, question_type, question_text, options, correct_answer, explanation, position, points)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`,
		q.QuizID, q.CardID, q.QuestionType, q.QuestionText, q.Options,
		q.CorrectAnswer, q.Explanation, q.Position, q.Points,
	).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
}

func (r *QuizRepository) BulkCreateQuestions(ctx context.Context, questions []*domain.QuizQuestion) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, q := range questions {
		err := tx.QueryRow(ctx,
			`INSERT INTO quiz_questions (quiz_id, card_id, question_type, question_text, options, correct_answer, explanation, position, points)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_at, updated_at`,
			q.QuizID, q.CardID, q.QuestionType, q.QuestionText, q.Options,
			q.CorrectAnswer, q.Explanation, q.Position, q.Points,
		).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *QuizRepository) GetQuestionByID(ctx context.Context, id uuid.UUID) (*domain.QuizQuestion, error) {
	var q domain.QuizQuestion
	err := r.db.QueryRow(ctx,
		`SELECT id, quiz_id, card_id, question_type, question_text, options, correct_answer, explanation, position, points, created_at, updated_at
		FROM quiz_questions WHERE id = $1`, id,
	).Scan(&q.ID, &q.QuizID, &q.CardID, &q.QuestionType, &q.QuestionText, &q.Options,
		&q.CorrectAnswer, &q.Explanation, &q.Position, &q.Points, &q.CreatedAt, &q.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &q, err
}

func (r *QuizRepository) ListQuestionsByQuizID(ctx context.Context, quizID uuid.UUID) ([]domain.QuizQuestion, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, quiz_id, card_id, question_type, question_text, options, correct_answer, explanation, position, points, created_at, updated_at
		FROM quiz_questions WHERE quiz_id = $1
		ORDER BY position, created_at`, quizID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []domain.QuizQuestion
	for rows.Next() {
		var q domain.QuizQuestion
		if err := rows.Scan(&q.ID, &q.QuizID, &q.CardID, &q.QuestionType, &q.QuestionText, &q.Options,
			&q.CorrectAnswer, &q.Explanation, &q.Position, &q.Points, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}
	return questions, nil
}

func (r *QuizRepository) UpdateQuestion(ctx context.Context, q *domain.QuizQuestion) error {
	_, err := r.db.Exec(ctx,
		`UPDATE quiz_questions SET question_type=$2, question_text=$3, options=$4, correct_answer=$5, explanation=$6, position=$7, points=$8, updated_at=now()
		WHERE id = $1`,
		q.ID, q.QuestionType, q.QuestionText, q.Options, q.CorrectAnswer, q.Explanation, q.Position, q.Points,
	)
	return err
}

func (r *QuizRepository) DeleteQuestion(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM quiz_questions WHERE id = $1`, id)
	return err
}

func (r *QuizRepository) CountQuestionsByQuizID(ctx context.Context, quizID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM quiz_questions WHERE quiz_id = $1`, quizID).Scan(&count)
	return count, err
}
