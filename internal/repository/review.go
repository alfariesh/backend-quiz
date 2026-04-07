package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, log *domain.ReviewLog) error {
	if log.Source == "" {
		log.Source = domain.ReviewSourceFlashcard
	}
	return r.db.QueryRow(ctx,
		`INSERT INTO review_logs (card_id, user_id, rating, state, scheduled_days, elapsed_days, stability, difficulty, duration_ms, source, reviewed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		log.CardID, log.UserID, int16(log.Rating), int16(log.State), log.ScheduledDays, log.ElapsedDays,
		log.Stability, log.Difficulty, log.DurationMS, log.Source, log.ReviewedAt,
	).Scan(&log.ID)
}

func (r *ReviewRepository) ListByCardID(ctx context.Context, cardID uuid.UUID) ([]domain.ReviewLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, card_id, user_id, rating, state, scheduled_days, elapsed_days, stability, difficulty, duration_ms, source, reviewed_at
		FROM review_logs WHERE card_id = $1 ORDER BY reviewed_at DESC`, cardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.ReviewLog
	for rows.Next() {
		var l domain.ReviewLog
		if err := rows.Scan(&l.ID, &l.CardID, &l.UserID, &l.Rating, &l.State, &l.ScheduledDays,
			&l.ElapsedDays, &l.Stability, &l.Difficulty, &l.DurationMS, &l.Source, &l.ReviewedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (r *ReviewRepository) ListByUserID(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int) ([]domain.ReviewLog, int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, card_id, user_id, rating, state, scheduled_days, elapsed_days, stability, difficulty, duration_ms, source, reviewed_at
		FROM review_logs
		WHERE user_id = $1 AND reviewed_at >= $2 AND reviewed_at < $3
		ORDER BY reviewed_at DESC
		LIMIT $4 OFFSET $5`,
		userID, from, to, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []domain.ReviewLog
	for rows.Next() {
		var l domain.ReviewLog
		if err := rows.Scan(&l.ID, &l.CardID, &l.UserID, &l.Rating, &l.State, &l.ScheduledDays,
			&l.ElapsedDays, &l.Stability, &l.Difficulty, &l.DurationMS, &l.Source, &l.ReviewedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	var total int
	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM review_logs WHERE user_id = $1 AND reviewed_at >= $2 AND reviewed_at < $3`,
		userID, from, to,
	).Scan(&total)
	return logs, total, err
}

func (r *ReviewRepository) CountByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM review_logs WHERE user_id = $1 AND reviewed_at::date = $2::date`,
		userID, date,
	).Scan(&count)
	return count, err
}

func (r *ReviewRepository) GetReviewCountsPerDay(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.DailyReviewCount, error) {
	rows, err := r.db.Query(ctx,
		`SELECT reviewed_at::date AS date, COUNT(*)::int AS count, COUNT(*) FILTER (WHERE rating >= 3)::int AS correct
		FROM review_logs
		WHERE user_id = $1 AND reviewed_at >= $2 AND reviewed_at < $3
		GROUP BY reviewed_at::date
		ORDER BY date`, userID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []domain.DailyReviewCount
	for rows.Next() {
		var c domain.DailyReviewCount
		if err := rows.Scan(&c.Date, &c.Count, &c.Correct); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, nil
}
