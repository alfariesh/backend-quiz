package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type DeckRepository struct {
	db *pgxpool.Pool
}

func NewDeckRepository(db *pgxpool.Pool) *DeckRepository {
	return &DeckRepository{db: db}
}

func (r *DeckRepository) Create(ctx context.Context, deck *domain.Deck) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO decks (user_id, name, description, new_cards_per_day, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		deck.UserID, deck.Name, deck.Description, deck.NewCardsPerDay, deck.Position,
	).Scan(&deck.ID, &deck.CreatedAt, &deck.UpdatedAt)
}

func (r *DeckRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deck, error) {
	var d domain.Deck
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, name, description, is_archived, new_cards_per_day, position, created_at, updated_at
		FROM decks WHERE id = $1`, id,
	).Scan(&d.ID, &d.UserID, &d.Name, &d.Description, &d.IsArchived,
		&d.NewCardsPerDay, &d.Position, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &d, err
}

func (r *DeckRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.DeckWithCounts, int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT d.id, d.user_id, d.name, d.description, d.is_archived, d.new_cards_per_day, d.position, d.created_at, d.updated_at,
			COALESCE(cc.total, 0)::int,
			COALESCE(cc.new_count, 0)::int,
			COALESCE(cc.due_count, 0)::int,
			COALESCE(cc.learn_count, 0)::int,
			COALESCE(cc.relearn_count, 0)::int
		FROM decks d
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*)::int AS total,
				COUNT(*) FILTER (WHERE c.state = 0 AND NOT c.is_suspended)::int AS new_count,
				COUNT(*) FILTER (WHERE c.state = 2 AND c.due <= now() AND NOT c.is_suspended)::int AS due_count,
				COUNT(*) FILTER (WHERE c.state = 1 AND NOT c.is_suspended)::int AS learn_count,
				COUNT(*) FILTER (WHERE c.state = 3 AND NOT c.is_suspended)::int AS relearn_count
			FROM cards c WHERE c.deck_id = d.id
		) cc ON true
		WHERE d.user_id = $1
		ORDER BY d.position, d.created_at
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var decks []domain.DeckWithCounts
	for rows.Next() {
		var d domain.DeckWithCounts
		if err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.Description, &d.IsArchived,
			&d.NewCardsPerDay, &d.Position, &d.CreatedAt, &d.UpdatedAt,
			&d.CardCount, &d.NewCount, &d.DueCount, &d.LearnCount, &d.RelearnCount); err != nil {
			return nil, 0, err
		}
		decks = append(decks, d)
	}

	var total int
	err = r.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM decks WHERE user_id = $1`, userID).Scan(&total)
	return decks, total, err
}

func (r *DeckRepository) Update(ctx context.Context, deck *domain.Deck) error {
	_, err := r.db.Exec(ctx,
		`UPDATE decks SET name=$2, description=$3, is_archived=$4, new_cards_per_day=$5, position=$6, updated_at=now()
		WHERE id = $1`,
		deck.ID, deck.Name, deck.Description, deck.IsArchived, deck.NewCardsPerDay, deck.Position,
	)
	return err
}

func (r *DeckRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM decks WHERE id = $1`, id)
	return err
}

func (r *DeckRepository) CreateShare(ctx context.Context, share *domain.DeckShare) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO deck_shares (deck_id, share_code, is_public) VALUES ($1, $2, $3) RETURNING id, created_at`,
		share.DeckID, share.ShareCode, share.IsPublic,
	).Scan(&share.ID, &share.CreatedAt)
}

func (r *DeckRepository) GetShareByDeckID(ctx context.Context, deckID uuid.UUID) (*domain.DeckShare, error) {
	var s domain.DeckShare
	err := r.db.QueryRow(ctx,
		`SELECT id, deck_id, share_code, is_public, created_at FROM deck_shares WHERE deck_id = $1`, deckID,
	).Scan(&s.ID, &s.DeckID, &s.ShareCode, &s.IsPublic, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &s, err
}

func (r *DeckRepository) GetShareByCode(ctx context.Context, code string) (*domain.DeckShare, error) {
	var s domain.DeckShare
	err := r.db.QueryRow(ctx,
		`SELECT id, deck_id, share_code, is_public, created_at FROM deck_shares WHERE share_code = $1`, code,
	).Scan(&s.ID, &s.DeckID, &s.ShareCode, &s.IsPublic, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &s, err
}

func (r *DeckRepository) DeleteShare(ctx context.Context, deckID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM deck_shares WHERE deck_id = $1`, deckID)
	return err
}

func (r *DeckRepository) ListPublicDecks(ctx context.Context, search string, limit, offset int) ([]domain.DeckWithCounts, int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT d.id, d.user_id, d.name, d.description, d.is_archived, d.new_cards_per_day, d.position, d.created_at, d.updated_at,
			COALESCE(cc.total, 0)::int, 0::int, 0::int, 0::int, 0::int
		FROM decks d
		INNER JOIN deck_shares ds ON ds.deck_id = d.id AND ds.is_public = true
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS total FROM cards c WHERE c.deck_id = d.id
		) cc ON true
		WHERE ($1::text = '' OR d.name ILIKE '%' || $1 || '%')
		ORDER BY d.created_at DESC
		LIMIT $2 OFFSET $3`, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var decks []domain.DeckWithCounts
	for rows.Next() {
		var d domain.DeckWithCounts
		if err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.Description, &d.IsArchived,
			&d.NewCardsPerDay, &d.Position, &d.CreatedAt, &d.UpdatedAt,
			&d.CardCount, &d.NewCount, &d.DueCount, &d.LearnCount, &d.RelearnCount); err != nil {
			return nil, 0, err
		}
		decks = append(decks, d)
	}

	var total int
	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM decks d INNER JOIN deck_shares ds ON ds.deck_id = d.id AND ds.is_public = true
		WHERE ($1::text = '' OR d.name ILIKE '%' || $1 || '%')`, search,
	).Scan(&total)
	return decks, total, err
}

func (r *DeckRepository) CloneDeck(ctx context.Context, sourceDeckID, targetUserID uuid.UUID, name string) (*domain.Deck, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var deck domain.Deck
	err = tx.QueryRow(ctx,
		`INSERT INTO decks (user_id, name, description)
		SELECT $1, $2, description FROM decks WHERE id = $3
		RETURNING id, user_id, name, description, is_archived, new_cards_per_day, position, created_at, updated_at`,
		targetUserID, name, sourceDeckID,
	).Scan(&deck.ID, &deck.UserID, &deck.Name, &deck.Description, &deck.IsArchived,
		&deck.NewCardsPerDay, &deck.Position, &deck.CreatedAt, &deck.UpdatedAt)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO cards (deck_id, front, back, tags, position)
		SELECT $1, front, back, tags, position FROM cards WHERE deck_id = $2 ORDER BY position`,
		deck.ID, sourceDeckID,
	)
	if err != nil {
		return nil, err
	}

	return &deck, tx.Commit(ctx)
}
