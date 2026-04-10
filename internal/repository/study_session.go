package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/repository/sqlc"
)

type StudySessionRepository struct {
	q *sqlc.Queries
}

func NewStudySessionRepository(pool *pgxpool.Pool) *StudySessionRepository {
	return &StudySessionRepository{q: sqlc.New(pool)}
}

func (r *StudySessionRepository) Create(ctx context.Context, session *domain.StudySession) error {
	result, err := querier(r.q, ctx).CreateStudySession(ctx, sqlc.CreateStudySessionParams{
		UserID: session.UserID,
		DeckID: uuidToNullable(session.DeckID),
	})
	if err != nil {
		return err
	}
	*session = sessionFromSqlc(result)
	return nil
}

func (r *StudySessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.StudySession, error) {
	result, err := querier(r.q, ctx).GetStudySessionByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s := sessionFromSqlc(result)
	return &s, nil
}

func (r *StudySessionRepository) Update(ctx context.Context, session *domain.StudySession) error {
	_, err := querier(r.q, ctx).UpdateStudySession(ctx, sqlc.UpdateStudySessionParams{
		ID:              session.ID,
		EndedAt:         timeToNullable(session.EndedAt),
		NewCount:        pgtype.Int4{Int32: int32(session.NewCount), Valid: true},
		ReviewCount:     pgtype.Int4{Int32: int32(session.ReviewCount), Valid: true},
		RelearnCount:    pgtype.Int4{Int32: int32(session.RelearnCount), Valid: true},
		TotalDurationMs: pgtype.Int4{Int32: int32(session.TotalDurationMS), Valid: true},
	})
	return err
}

func (r *StudySessionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.StudySession, int, error) {
	rows, err := querier(r.q, ctx).ListStudySessionsByUserID(ctx, sqlc.ListStudySessionsByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := querier(r.q, ctx).CountStudySessionsByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	sessions := make([]domain.StudySession, len(rows))
	for i, row := range rows {
		sessions[i] = sessionFromSqlc(row)
	}
	return sessions, int(total), nil
}
