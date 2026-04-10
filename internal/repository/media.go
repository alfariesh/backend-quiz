package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/repository/sqlc"
)

type MediaRepository struct {
	q *sqlc.Queries
}

func NewMediaRepository(pool *pgxpool.Pool) *MediaRepository {
	return &MediaRepository{q: sqlc.New(pool)}
}

func (r *MediaRepository) Create(ctx context.Context, media *domain.Media) error {
	result, err := querier(r.q, ctx).CreateMedia(ctx, sqlc.CreateMediaParams{
		UserID:   media.UserID,
		CardID:   uuidToNullable(media.CardID),
		FileName: media.FileName,
		FileSize: int32(media.FileSize),
		MimeType: media.MimeType,
		R2Key:    media.R2Key,
		Url:      media.URL,
	})
	if err != nil {
		return err
	}
	*media = mediaFromSqlc(result)
	return nil
}

func (r *MediaRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Media, error) {
	result, err := querier(r.q, ctx).GetMediaByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	m := mediaFromSqlc(result)
	return &m, nil
}

func (r *MediaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return querier(r.q, ctx).DeleteMedia(ctx, id)
}

func (r *MediaRepository) ListByCardID(ctx context.Context, cardID uuid.UUID) ([]domain.Media, error) {
	rows, err := querier(r.q, ctx).ListMediaByCardID(ctx, uuidToNullable(&cardID))
	if err != nil {
		return nil, err
	}
	media := make([]domain.Media, len(rows))
	for i, row := range rows {
		media[i] = mediaFromSqlc(row)
	}
	return media, nil
}
