package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

type LoginAttemptRepository struct {
	pool *pgxpool.Pool
}

func NewLoginAttemptRepository(pool *pgxpool.Pool) *LoginAttemptRepository {
	return &LoginAttemptRepository{pool: pool}
}

func (r *LoginAttemptRepository) Create(ctx context.Context, a *domain.LoginAttempt) error {
	const q = `
INSERT INTO login_attempts (email, ip_address, successful)
VALUES ($1, $2, $3)
RETURNING id, created_at`
	return conn(ctx, r.pool).QueryRow(ctx, q, a.Email, a.IPAddress, a.Successful).Scan(&a.ID, &a.CreatedAt)
}

func (r *LoginAttemptRepository) CountRecentFailures(ctx context.Context, email string, since time.Time) (int, error) {
	const q = `
SELECT COUNT(*) FROM login_attempts
WHERE email = $1 AND successful = false AND created_at >= $2`
	var count int
	err := conn(ctx, r.pool).QueryRow(ctx, q, email, since).Scan(&count)
	return count, err
}
