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

type PasswordResetTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPasswordResetTokenRepository(pool *pgxpool.Pool) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{pool: pool}
}

func (r *PasswordResetTokenRepository) Create(ctx context.Context, t *domain.PasswordResetToken) error {
	const q = `
INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, created_at`
	return conn(ctx, r.pool).QueryRow(ctx, q, t.UserID, t.TokenHash, t.ExpiresAt).Scan(&t.ID, &t.CreatedAt)
}

func (r *PasswordResetTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	const q = `
SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
FROM password_reset_tokens
WHERE token_hash = $1`
	var t domain.PasswordResetToken
	var consumedAt pgtype.Timestamptz
	err := conn(ctx, r.pool).QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &consumedAt, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if consumedAt.Valid {
		t.ConsumedAt = &consumedAt.Time
	}
	return &t, nil
}

func (r *PasswordResetTokenRepository) Consume(ctx context.Context, id uuid.UUID, at time.Time) error {
	const q = `UPDATE password_reset_tokens SET consumed_at = $2 WHERE id = $1`
	_, err := conn(ctx, r.pool).Exec(ctx, q, id, at)
	return err
}

func (r *PasswordResetTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	const q = `DELETE FROM password_reset_tokens WHERE user_id = $1`
	_, err := conn(ctx, r.pool).Exec(ctx, q, userID)
	return err
}
