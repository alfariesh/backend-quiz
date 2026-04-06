package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type CardRepository struct {
	db *pgxpool.Pool
}

func NewCardRepository(db *pgxpool.Pool) *CardRepository {
	return &CardRepository{db: db}
}

func (r *CardRepository) Create(ctx context.Context, card *domain.Card) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO cards (deck_id, front, back, tags, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, state, last_review, is_suspended, created_at, updated_at`,
		card.DeckID, card.Front, card.Back, card.Tags, card.Position,
	).Scan(&card.ID, &card.Due, &card.Stability, &card.Difficulty, &card.ElapsedDays,
		&card.ScheduledDays, &card.Reps, &card.Lapses, &card.State, &card.LastReview,
		&card.IsSuspended, &card.CreatedAt, &card.UpdatedAt)
}

func (r *CardRepository) BulkCreate(ctx context.Context, cards []*domain.Card) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, card := range cards {
		err := tx.QueryRow(ctx,
			`INSERT INTO cards (deck_id, front, back, tags, position)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, due, created_at, updated_at`,
			card.DeckID, card.Front, card.Back, card.Tags, card.Position,
		).Scan(&card.ID, &card.Due, &card.CreatedAt, &card.UpdatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *CardRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	var c domain.Card
	err := r.db.QueryRow(ctx,
		`SELECT id, deck_id, front, back, tags, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, state, last_review, is_suspended, position, created_at, updated_at
		FROM cards WHERE id = $1`, id,
	).Scan(&c.ID, &c.DeckID, &c.Front, &c.Back, &c.Tags, &c.Due, &c.Stability, &c.Difficulty,
		&c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &c.State, &c.LastReview,
		&c.IsSuspended, &c.Position, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &c, err
}

func (r *CardRepository) ListByDeckID(ctx context.Context, deckID uuid.UUID, filter domain.CardFilter, limit, offset int) ([]domain.Card, int, error) {
	stateParam := (*int16)(nil)
	if filter.State != nil {
		v := int16(*filter.State)
		stateParam = &v
	}
	tag := ""
	if filter.Tag != "" {
		tag = filter.Tag
	}
	query := ""
	if filter.Query != "" {
		query = filter.Query
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, deck_id, front, back, tags, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, state, last_review, is_suspended, position, created_at, updated_at
		FROM cards
		WHERE deck_id = $1
			AND ($2::smallint IS NULL OR state = $2)
			AND ($3::text = '' OR $3 = ANY(tags))
			AND ($4::text = '' OR front ILIKE '%' || $4 || '%' OR back ILIKE '%' || $4 || '%')
		ORDER BY position, created_at
		LIMIT $5 OFFSET $6`,
		deckID, stateParam, tag, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var cards []domain.Card
	for rows.Next() {
		var c domain.Card
		if err := rows.Scan(&c.ID, &c.DeckID, &c.Front, &c.Back, &c.Tags, &c.Due, &c.Stability, &c.Difficulty,
			&c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &c.State, &c.LastReview,
			&c.IsSuspended, &c.Position, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		cards = append(cards, c)
	}

	var total int
	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM cards
		WHERE deck_id = $1
			AND ($2::smallint IS NULL OR state = $2)
			AND ($3::text = '' OR $3 = ANY(tags))
			AND ($4::text = '' OR front ILIKE '%' || $4 || '%' OR back ILIKE '%' || $4 || '%')`,
		deckID, stateParam, tag, query).Scan(&total)
	return cards, total, err
}

func (r *CardRepository) Update(ctx context.Context, card *domain.Card) error {
	_, err := r.db.Exec(ctx,
		`UPDATE cards SET front=$2, back=$3, tags=$4, updated_at=now() WHERE id = $1`,
		card.ID, card.Front, card.Back, card.Tags,
	)
	return err
}

func (r *CardRepository) UpdateFSRS(ctx context.Context, card *domain.Card) error {
	_, err := r.db.Exec(ctx,
		`UPDATE cards SET due=$2, stability=$3, difficulty=$4, elapsed_days=$5, scheduled_days=$6, reps=$7, lapses=$8, state=$9, last_review=$10, updated_at=now()
		WHERE id = $1`,
		card.ID, card.Due, card.Stability, card.Difficulty, card.ElapsedDays,
		card.ScheduledDays, card.Reps, card.Lapses, int16(card.State), card.LastReview,
	)
	return err
}

func (r *CardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM cards WHERE id = $1`, id)
	return err
}

func (r *CardRepository) SetSuspended(ctx context.Context, id uuid.UUID, suspended bool) error {
	_, err := r.db.Exec(ctx, `UPDATE cards SET is_suspended = $2, updated_at = now() WHERE id = $1`, id, suspended)
	return err
}

func (r *CardRepository) ResetFSRS(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE cards SET due=now(), stability=0, difficulty=0, elapsed_days=0, scheduled_days=0, reps=0, lapses=0, state=0, last_review=NULL, updated_at=now()
		WHERE id = $1`, id,
	)
	return err
}

func (r *CardRepository) GetDueCards(ctx context.Context, deckID uuid.UUID, now time.Time, newLimit, reviewLimit int) ([]domain.Card, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, deck_id, front, back, tags, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, state, last_review, is_suspended, position, created_at, updated_at
		FROM cards
		WHERE deck_id = $1 AND NOT is_suspended
			AND (
				(state = 0)
				OR (state IN (1, 3) AND due <= $2)
				OR (state = 2 AND due <= $2)
			)
		ORDER BY
			CASE WHEN state IN (1, 3) THEN 0
				WHEN state = 2 THEN 1
				WHEN state = 0 THEN 2
			END,
			due ASC
		LIMIT $3`,
		deckID, now, newLimit+reviewLimit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []domain.Card
	newCount := 0
	for rows.Next() {
		var c domain.Card
		if err := rows.Scan(&c.ID, &c.DeckID, &c.Front, &c.Back, &c.Tags, &c.Due, &c.Stability, &c.Difficulty,
			&c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &c.State, &c.LastReview,
			&c.IsSuspended, &c.Position, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if c.State == domain.CardStateNew {
			if newCount >= newLimit {
				continue
			}
			newCount++
		}
		cards = append(cards, c)
	}

	return cards, nil
}

func (r *CardRepository) CountByState(ctx context.Context, deckID uuid.UUID) (map[domain.CardState]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT state, COUNT(*)::int FROM cards WHERE deck_id = $1 AND NOT is_suspended GROUP BY state`, deckID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[domain.CardState]int)
	for rows.Next() {
		var state int16
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			return nil, err
		}
		counts[domain.CardState(state)] = count
	}
	return counts, nil
}

func (r *CardRepository) CountDue(ctx context.Context, deckID uuid.UUID, now time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM cards
		WHERE deck_id = $1 AND NOT is_suspended AND state IN (1, 2, 3) AND due <= $2`,
		deckID, now,
	).Scan(&count)
	return count, err
}
