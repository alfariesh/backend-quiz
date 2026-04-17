package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

type AuthAuditLogRepository struct {
	pool *pgxpool.Pool
}

func NewAuthAuditLogRepository(pool *pgxpool.Pool) *AuthAuditLogRepository {
	return &AuthAuditLogRepository{pool: pool}
}

func (r *AuthAuditLogRepository) Create(ctx context.Context, log *domain.AuthAuditLog) error {
	meta := log.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	payload, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	const q = `
INSERT INTO auth_audit_logs (user_id, event, ip_address, user_agent, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at`
	return conn(ctx, r.pool).QueryRow(ctx, q,
		log.UserID, string(log.Event), log.IPAddress, log.UserAgent, payload,
	).Scan(&log.ID, &log.CreatedAt)
}
