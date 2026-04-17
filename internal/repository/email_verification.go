package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

type EmailVerificationRepository struct {
	pool *pgxpool.Pool
}

func NewEmailVerificationRepository(pool *pgxpool.Pool) *EmailVerificationRepository {
	return &EmailVerificationRepository{pool: pool}
}

func (r *EmailVerificationRepository) Create(ctx context.Context, ev *domain.EmailVerification) error {
	const q = `
INSERT INTO email_verifications (user_id, code_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, attempts, created_at`
	row := conn(ctx, r.pool).QueryRow(ctx, q, ev.UserID, ev.CodeHash, ev.ExpiresAt)
	return row.Scan(&ev.ID, &ev.Attempts, &ev.CreatedAt)
}

func (r *EmailVerificationRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailVerification, error) {
	const q = `
SELECT id, user_id, code_hash, attempts, expires_at, consumed_at, created_at
FROM email_verifications
WHERE user_id = $1 AND consumed_at IS NULL
ORDER BY created_at DESC
LIMIT 1`
	var ev domain.EmailVerification
	var consumedAt pgtype.Timestamptz
	err := conn(ctx, r.pool).QueryRow(ctx, q, userID).Scan(
		&ev.ID, &ev.UserID, &ev.CodeHash, &ev.Attempts, &ev.ExpiresAt, &consumedAt, &ev.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if consumedAt.Valid {
		ev.ConsumedAt = &consumedAt.Time
	}
	return &ev, nil
}

func (r *EmailVerificationRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE email_verifications SET attempts = attempts + 1 WHERE id = $1`
	_, err := conn(ctx, r.pool).Exec(ctx, q, id)
	return err
}

func (r *EmailVerificationRepository) Consume(ctx context.Context, id uuid.UUID, at time.Time) error {
	const q = `UPDATE email_verifications SET consumed_at = $2 WHERE id = $1`
	_, err := conn(ctx, r.pool).Exec(ctx, q, id, at)
	return err
}

func (r *EmailVerificationRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	const q = `DELETE FROM email_verifications WHERE user_id = $1`
	_, err := conn(ctx, r.pool).Exec(ctx, q, userID)
	return err
}
