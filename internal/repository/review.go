package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/repository/sqlc"
)

type ReviewRepository struct {
	q *sqlc.Queries
}

func NewReviewRepository(pool *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{q: sqlc.New(pool)}
}

func (r *ReviewRepository) Create(ctx context.Context, log *domain.ReviewLog) error {
	if log.Source == "" {
		log.Source = domain.ReviewSourceFlashcard
	}
	result, err := querier(r.q, ctx).CreateReviewLog(ctx, sqlc.CreateReviewLogParams{
		CardID:        log.CardID,
		UserID:        log.UserID,
		Rating:        int16(log.Rating),
		State:         int16(log.State),
		ScheduledDays: int32(log.ScheduledDays),
		ElapsedDays:   int32(log.ElapsedDays),
		Stability:     float32(log.Stability),
		Difficulty:    float32(log.Difficulty),
		DurationMs:    int32(log.DurationMS),
		Source:        log.Source,
		ReviewedAt:    log.ReviewedAt,
	})
	if err != nil {
		return err
	}
	log.ID = result.ID
	return nil
}

func (r *ReviewRepository) ListByCardID(ctx context.Context, cardID uuid.UUID) ([]domain.ReviewLog, error) {
	rows, err := querier(r.q, ctx).ListReviewLogsByCardID(ctx, cardID)
	if err != nil {
		return nil, err
	}
	logs := make([]domain.ReviewLog, len(rows))
	for i, row := range rows {
		logs[i] = reviewLogFromSqlc(row)
	}
	return logs, nil
}

func (r *ReviewRepository) ListByUserID(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int) ([]domain.ReviewLog, int, error) {
	rows, err := querier(r.q, ctx).ListReviewLogsByUserID(ctx, sqlc.ListReviewLogsByUserIDParams{
		UserID:     userID,
		ReviewedAt: from,
		ReviewedAt_2: to,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := querier(r.q, ctx).CountReviewLogsByUserID(ctx, sqlc.CountReviewLogsByUserIDParams{
		UserID:     userID,
		ReviewedAt: from,
		ReviewedAt_2: to,
	})
	if err != nil {
		return nil, 0, err
	}

	logs := make([]domain.ReviewLog, len(rows))
	for i, row := range rows {
		logs[i] = reviewLogFromSqlc(row)
	}
	return logs, int(total), nil
}

func (r *ReviewRepository) CountByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	count, err := querier(r.q, ctx).CountReviewsByUserAndDate(ctx, sqlc.CountReviewsByUserAndDateParams{
		UserID: userID,
		Date:   pgtype.Date{Time: date, Valid: true},
	})
	return int(count), err
}

func (r *ReviewRepository) GetReviewCountsPerDay(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.DailyReviewCount, error) {
	rows, err := querier(r.q, ctx).GetReviewCountsPerDay(ctx, sqlc.GetReviewCountsPerDayParams{
		UserID:       userID,
		ReviewedAt:   from,
		ReviewedAt_2: to,
	})
	if err != nil {
		return nil, err
	}

	counts := make([]domain.DailyReviewCount, len(rows))
	for i, row := range rows {
		counts[i] = domain.DailyReviewCount{
			Date:    row.Date.Time,
			Count:   int(row.Count),
			Correct: int(row.Correct),
		}
	}
	return counts, nil
}
