package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type StatsRepository struct {
	db *pgxpool.Pool
}

func NewStatsRepository(db *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{db: db}
}

func (r *StatsRepository) UpsertDailyStats(ctx context.Context, stats *domain.DailyStats) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO daily_stats (user_id, date, new_cards, reviews, relearns, total_duration_ms, retention_rate)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, date) DO UPDATE SET
			new_cards = EXCLUDED.new_cards,
			reviews = EXCLUDED.reviews,
			relearns = EXCLUDED.relearns,
			total_duration_ms = EXCLUDED.total_duration_ms,
			retention_rate = EXCLUDED.retention_rate
		RETURNING id`,
		stats.UserID, stats.Date, stats.NewCards, stats.Reviews, stats.Relearns,
		stats.TotalDurationMS, stats.RetentionRate,
	).Scan(&stats.ID)
}

func (r *StatsRepository) GetDailyStats(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.DailyStats, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, date, new_cards, reviews, relearns, total_duration_ms, retention_rate
		FROM daily_stats WHERE user_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date`, userID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.DailyStats
	for rows.Next() {
		var s domain.DailyStats
		if err := rows.Scan(&s.ID, &s.UserID, &s.Date, &s.NewCards, &s.Reviews, &s.Relearns,
			&s.TotalDurationMS, &s.RetentionRate); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *StatsRepository) GetStreak(ctx context.Context, userID uuid.UUID) (int, error) {
	var streak int
	err := r.db.QueryRow(ctx,
		`WITH streak AS (
			SELECT date,
				date - (ROW_NUMBER() OVER (ORDER BY date DESC))::int * INTERVAL '1 day' AS grp
			FROM daily_stats
			WHERE user_id = $1 AND reviews > 0
		)
		SELECT COALESCE(COUNT(*)::int, 0) AS streak
		FROM streak
		WHERE grp = (SELECT grp FROM streak LIMIT 1)`, userID,
	).Scan(&streak)
	return streak, err
}
