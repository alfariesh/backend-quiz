package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/repository/sqlc"
)

type StatsRepository struct {
	q *sqlc.Queries
}

func NewStatsRepository(pool *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{q: sqlc.New(pool)}
}

func (r *StatsRepository) UpsertDailyStats(ctx context.Context, stats *domain.DailyStats) error {
	var retentionRate pgtype.Float4
	if stats.RetentionRate != nil {
		retentionRate = pgtype.Float4{Float32: float32(*stats.RetentionRate), Valid: true}
	}

	result, err := querier(r.q, ctx).UpsertDailyStats(ctx, sqlc.UpsertDailyStatsParams{
		UserID:          stats.UserID,
		Date:            pgtype.Date{Time: stats.Date, Valid: true},
		NewCards:        int32(stats.NewCards),
		Reviews:         int32(stats.Reviews),
		Relearns:        int32(stats.Relearns),
		TotalDurationMs: int32(stats.TotalDurationMS),
		RetentionRate:   retentionRate,
	})
	if err != nil {
		return err
	}
	stats.ID = result.ID
	return nil
}

func (r *StatsRepository) GetDailyStats(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.DailyStats, error) {
	rows, err := querier(r.q, ctx).GetDailyStats(ctx, sqlc.GetDailyStatsParams{
		UserID: userID,
		Date:   pgtype.Date{Time: from, Valid: true},
		Date_2: pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	stats := make([]domain.DailyStats, len(rows))
	for i, row := range rows {
		var retention *float64
		if row.RetentionRate.Valid {
			v := float64(row.RetentionRate.Float32)
			retention = &v
		}
		stats[i] = domain.DailyStats{
			ID:              row.ID,
			UserID:          row.UserID,
			Date:            row.Date.Time,
			NewCards:        int(row.NewCards),
			Reviews:         int(row.Reviews),
			Relearns:        int(row.Relearns),
			TotalDurationMS: int(row.TotalDurationMs),
			RetentionRate:   retention,
		}
	}
	return stats, nil
}

func (r *StatsRepository) GetStreak(ctx context.Context, userID uuid.UUID) (int, error) {
	result, err := querier(r.q, ctx).GetStreak(ctx, userID)
	if err != nil {
		return 0, err
	}
	// GetStreak returns interface{} due to COALESCE
	switch v := result.(type) {
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	default:
		return 0, nil
	}
}

func (r *StatsRepository) GetGlobalLeaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error) {
	rows, err := querier(r.q, ctx).GetGlobalLeaderboard(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	entries := make([]domain.LeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = domain.LeaderboardEntry{
			UserID:      row.UserID,
			DisplayName: row.DisplayName,
			Streak:      int(row.Streak),
			Rank:        i + 1,
		}
	}
	return entries, nil
}
