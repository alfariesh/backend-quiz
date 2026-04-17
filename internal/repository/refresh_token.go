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

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, t *domain.RefreshToken) error {
	const q = `
INSERT INTO refresh_tokens (user_id, token_hash, parent_id, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at`
	var parentID pgtype.UUID
	if t.ParentID != nil {
		parentID = pgtype.UUID{Bytes: *t.ParentID, Valid: true}
	}
	return conn(ctx, r.pool).QueryRow(ctx, q,
		t.UserID, t.TokenHash, parentID, t.UserAgent, t.IPAddress, t.ExpiresAt,
	).Scan(&t.ID, &t.CreatedAt)
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	const q = `
SELECT id, user_id, token_hash, parent_id, user_agent, ip_address, revoked_at, expires_at, created_at
FROM refresh_tokens
WHERE token_hash = $1`
	var t domain.RefreshToken
	var parentID pgtype.UUID
	var revokedAt pgtype.Timestamptz
	err := conn(ctx, r.pool).QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &parentID, &t.UserAgent, &t.IPAddress, &revokedAt, &t.ExpiresAt, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if parentID.Valid {
		id := uuid.UUID(parentID.Bytes)
		t.ParentID = &id
	}
	if revokedAt.Valid {
		t.RevokedAt = &revokedAt.Time
	}
	return &t, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	const q = `UPDATE refresh_tokens SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`
	_, err := conn(ctx, r.pool).Exec(ctx, q, id, at)
	return err
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	const q = `UPDATE refresh_tokens SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := conn(ctx, r.pool).Exec(ctx, q, userID, at)
	return err
}

func (r *RefreshTokenRepository) ListActiveByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]*domain.RefreshToken, error) {
	const q = `
SELECT id, user_id, token_hash, parent_id, user_agent, ip_address, revoked_at, expires_at, created_at
FROM refresh_tokens
WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > $2
ORDER BY created_at DESC`
	rows, err := conn(ctx, r.pool).Query(ctx, q, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.RefreshToken
	for rows.Next() {
		var t domain.RefreshToken
		var parentID pgtype.UUID
		var revokedAt pgtype.Timestamptz
		if err := rows.Scan(&t.ID, &t.UserID, &t.TokenHash, &parentID, &t.UserAgent, &t.IPAddress, &revokedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			id := uuid.UUID(parentID.Bytes)
			t.ParentID = &id
		}
		if revokedAt.Valid {
			t.RevokedAt = &revokedAt.Time
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *RefreshTokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	const q = `
SELECT id, user_id, token_hash, parent_id, user_agent, ip_address, revoked_at, expires_at, created_at
FROM refresh_tokens
WHERE id = $1`
	var t domain.RefreshToken
	var parentID pgtype.UUID
	var revokedAt pgtype.Timestamptz
	err := conn(ctx, r.pool).QueryRow(ctx, q, id).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &parentID, &t.UserAgent, &t.IPAddress, &revokedAt, &t.ExpiresAt, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if parentID.Valid {
		pid := uuid.UUID(parentID.Bytes)
		t.ParentID = &pid
	}
	if revokedAt.Valid {
		t.RevokedAt = &revokedAt.Time
	}
	return &t, nil
}

// DeviceSeen returns true if any prior refresh token exists for this user matching
// the given user_agent AND ip_address. Used for new-device detection.
func (r *RefreshTokenRepository) DeviceSeen(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (bool, error) {
	const q = `
SELECT EXISTS (
    SELECT 1 FROM refresh_tokens
    WHERE user_id = $1 AND user_agent = $2 AND ip_address = $3
)`
	var exists bool
	err := conn(ctx, r.pool).QueryRow(ctx, q, userID, userAgent, ipAddress).Scan(&exists)
	return exists, err
}

func (r *RefreshTokenRepository) RevokeChildren(ctx context.Context, parentID uuid.UUID, at time.Time) error {
	const q = `
WITH RECURSIVE chain AS (
    SELECT id FROM refresh_tokens WHERE parent_id = $1
    UNION
    SELECT rt.id FROM refresh_tokens rt JOIN chain c ON rt.parent_id = c.id
)
UPDATE refresh_tokens SET revoked_at = $2
WHERE id IN (SELECT id FROM chain) AND revoked_at IS NULL`
	_, err := conn(ctx, r.pool).Exec(ctx, q, parentID, at)
	return err
}
