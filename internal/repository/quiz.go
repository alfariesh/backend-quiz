package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/repository/sqlc"
)

type QuizRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewQuizRepository(pool *pgxpool.Pool) *QuizRepository {
	return &QuizRepository{q: sqlc.New(pool), pool: pool}
}

func (r *QuizRepository) Create(ctx context.Context, quiz *domain.Quiz) error {
	result, err := r.q.CreateQuiz(ctx, sqlc.CreateQuizParams{
		UserID:           quiz.UserID,
		DeckID:           uuidToNullable(quiz.DeckID),
		Title:            quiz.Title,
		Description:      quiz.Description,
		QuizType:         quiz.QuizType,
		TimeLimitSeconds: intToNullable(quiz.TimeLimitSeconds),
		ShuffleQuestions: quiz.ShuffleQuestions,
	})
	if err != nil {
		return err
	}
	*quiz = quizFromSqlc(result)
	return nil
}

func (r *QuizRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Quiz, error) {
	result, err := r.q.GetQuizByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	q := quizFromSqlc(result)
	return &q, nil
}

func (r *QuizRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizWithCounts, int, error) {
	rows, err := r.q.ListQuizzesByUserID(ctx, sqlc.ListQuizzesByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.q.CountQuizzesByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	quizzes := make([]domain.QuizWithCounts, len(rows))
	for i, row := range rows {
		quizzes[i] = domain.QuizWithCounts{
			Quiz: domain.Quiz{
				ID:               row.ID,
				UserID:           row.UserID,
				DeckID:           nullableToUUID(row.DeckID),
				Title:            row.Title,
				Description:      row.Description,
				QuizType:         row.QuizType,
				TimeLimitSeconds: nullableToInt(row.TimeLimitSeconds),
				ShuffleQuestions: row.ShuffleQuestions,
				IsPublished:      row.IsPublished,
				CreatedAt:        row.CreatedAt,
				UpdatedAt:        row.UpdatedAt,
			},
			QuestionCount: int(row.QuestionCount),
			AttemptCount:  int(row.AttemptCount),
		}
	}
	return quizzes, int(total), nil
}

func (r *QuizRepository) Update(ctx context.Context, quiz *domain.Quiz) error {
	return r.q.UpdateQuiz(ctx, sqlc.UpdateQuizParams{
		ID:               quiz.ID,
		Title:            quiz.Title,
		Description:      quiz.Description,
		QuizType:         quiz.QuizType,
		TimeLimitSeconds: intToNullable(quiz.TimeLimitSeconds),
		ShuffleQuestions: quiz.ShuffleQuestions,
		IsPublished:      quiz.IsPublished,
	})
}

func (r *QuizRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteQuiz(ctx, id)
}

// Question operations

func (r *QuizRepository) CreateQuestion(ctx context.Context, q *domain.QuizQuestion) error {
	result, err := r.q.CreateQuizQuestion(ctx, sqlc.CreateQuizQuestionParams{
		QuizID:        q.QuizID,
		CardID:        uuidToNullable(q.CardID),
		QuestionType:  q.QuestionType,
		QuestionText:  q.QuestionText,
		Options:       []byte(q.Options),
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   q.Explanation,
		Position:      int32(q.Position),
		Points:        int32(q.Points),
	})
	if err != nil {
		return err
	}
	*q = questionFromSqlc(result)
	return nil
}

func (r *QuizRepository) BulkCreateQuestions(ctx context.Context, questions []*domain.QuizQuestion) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)
	for _, q := range questions {
		result, err := qtx.CreateQuizQuestion(ctx, sqlc.CreateQuizQuestionParams{
			QuizID:        q.QuizID,
			CardID:        uuidToNullable(q.CardID),
			QuestionType:  q.QuestionType,
			QuestionText:  q.QuestionText,
			Options:       []byte(q.Options),
			CorrectAnswer: q.CorrectAnswer,
			Explanation:   q.Explanation,
			Position:      int32(q.Position),
			Points:        int32(q.Points),
		})
		if err != nil {
			return err
		}
		*q = questionFromSqlc(result)
	}

	return tx.Commit(ctx)
}

func (r *QuizRepository) GetQuestionByID(ctx context.Context, id uuid.UUID) (*domain.QuizQuestion, error) {
	result, err := r.q.GetQuizQuestionByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	q := questionFromSqlc(result)
	return &q, nil
}

func (r *QuizRepository) ListQuestionsByQuizID(ctx context.Context, quizID uuid.UUID) ([]domain.QuizQuestion, error) {
	rows, err := r.q.ListQuizQuestionsByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	questions := make([]domain.QuizQuestion, len(rows))
	for i, row := range rows {
		questions[i] = questionFromSqlc(row)
	}
	return questions, nil
}

func (r *QuizRepository) UpdateQuestion(ctx context.Context, q *domain.QuizQuestion) error {
	return r.q.UpdateQuizQuestion(ctx, sqlc.UpdateQuizQuestionParams{
		ID:            q.ID,
		QuestionType:  q.QuestionType,
		QuestionText:  q.QuestionText,
		Options:       []byte(q.Options),
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   q.Explanation,
		Position:      int32(q.Position),
		Points:        int32(q.Points),
	})
}

func (r *QuizRepository) DeleteQuestion(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteQuizQuestion(ctx, id)
}

func (r *QuizRepository) CountQuestionsByQuizID(ctx context.Context, quizID uuid.UUID) (int, error) {
	count, err := r.q.CountQuizQuestionsByQuizID(ctx, quizID)
	return int(count), err
}

func (r *QuizRepository) ListByDeckAndType(ctx context.Context, deckID uuid.UUID, quizType string) ([]domain.Quiz, error) {
	rows, err := r.q.ListQuizzesByDeckAndType(ctx, sqlc.ListQuizzesByDeckAndTypeParams{
		DeckID:   uuidToNullable(&deckID),
		QuizType: quizType,
	})
	if err != nil {
		return nil, err
	}
	quizzes := make([]domain.Quiz, len(rows))
	for i, row := range rows {
		quizzes[i] = quizFromSqlc(row)
	}
	return quizzes, nil
}

