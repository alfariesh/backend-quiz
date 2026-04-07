package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type MediaRepository struct {
	db *pgxpool.Pool
}

func NewMediaRepository(db *pgxpool.Pool) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) Create(ctx context.Context, media *domain.Media) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO media (user_id, card_id, file_name, file_size, mime_type, r2_key, url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		media.UserID, media.CardID, media.FileName, media.FileSize, media.MimeType, media.R2Key, media.URL,
	).Scan(&media.ID, &media.CreatedAt)
}

func (r *MediaRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Media, error) {
	var m domain.Media
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, card_id, file_name, file_size, mime_type, r2_key, url, created_at
		FROM media WHERE id = $1`, id,
	).Scan(&m.ID, &m.UserID, &m.CardID, &m.FileName, &m.FileSize, &m.MimeType, &m.R2Key, &m.URL, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &m, err
}

func (r *MediaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM media WHERE id = $1`, id)
	return err
}

func (r *MediaRepository) ListByCardID(ctx context.Context, cardID uuid.UUID) ([]domain.Media, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, card_id, file_name, file_size, mime_type, r2_key, url, created_at
		FROM media WHERE card_id = $1 ORDER BY created_at`, cardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []domain.Media
	for rows.Next() {
		var m domain.Media
		if err := rows.Scan(&m.ID, &m.UserID, &m.CardID, &m.FileName, &m.FileSize,
			&m.MimeType, &m.R2Key, &m.URL, &m.CreatedAt); err != nil {
			return nil, err
		}
		media = append(media, m)
	}
	return media, nil
}
