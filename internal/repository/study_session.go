package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type StudySessionRepository struct {
	db *pgxpool.Pool
}

func NewStudySessionRepository(db *pgxpool.Pool) *StudySessionRepository {
	return &StudySessionRepository{db: db}
}

func (r *StudySessionRepository) Create(ctx context.Context, session *domain.StudySession) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO study_sessions (user_id, deck_id) VALUES ($1, $2)
		RETURNING id, started_at, new_count, review_count, relearn_count, total_duration_ms`,
		session.UserID, session.DeckID,
	).Scan(&session.ID, &session.StartedAt, &session.NewCount, &session.ReviewCount,
		&session.RelearnCount, &session.TotalDurationMS)
}

func (r *StudySessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.StudySession, error) {
	var s domain.StudySession
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, deck_id, started_at, ended_at, new_count, review_count, relearn_count, total_duration_ms
		FROM study_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.UserID, &s.DeckID, &s.StartedAt, &s.EndedAt,
		&s.NewCount, &s.ReviewCount, &s.RelearnCount, &s.TotalDurationMS)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &s, err
}

func (r *StudySessionRepository) Update(ctx context.Context, session *domain.StudySession) error {
	_, err := r.db.Exec(ctx,
		`UPDATE study_sessions SET ended_at=$2, new_count=$3, review_count=$4, relearn_count=$5, total_duration_ms=$6
		WHERE id = $1`,
		session.ID, session.EndedAt, session.NewCount, session.ReviewCount,
		session.RelearnCount, session.TotalDurationMS,
	)
	return err
}

func (r *StudySessionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.StudySession, int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, deck_id, started_at, ended_at, new_count, review_count, relearn_count, total_duration_ms
		FROM study_sessions WHERE user_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []domain.StudySession
	for rows.Next() {
		var s domain.StudySession
		if err := rows.Scan(&s.ID, &s.UserID, &s.DeckID, &s.StartedAt, &s.EndedAt,
			&s.NewCount, &s.ReviewCount, &s.RelearnCount, &s.TotalDurationMS); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, s)
	}

	var total int
	err = r.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM study_sessions WHERE user_id = $1`, userID).Scan(&total)
	return sessions, total, err
}
