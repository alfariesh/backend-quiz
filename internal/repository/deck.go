package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/repository/sqlc"
)

type DeckRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewDeckRepository(pool *pgxpool.Pool) *DeckRepository {
	return &DeckRepository{q: sqlc.New(pool), pool: pool}
}

func (r *DeckRepository) Create(ctx context.Context, deck *domain.Deck) error {
	result, err := r.q.CreateDeck(ctx, sqlc.CreateDeckParams{
		UserID:         deck.UserID,
		Name:           deck.Name,
		Description:    deck.Description,
		NewCardsPerDay: intToNullable(deck.NewCardsPerDay),
		Position:       int32(deck.Position),
	})
	if err != nil {
		return err
	}
	*deck = deckFromSqlc(result)
	return nil
}

func (r *DeckRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deck, error) {
	result, err := r.q.GetDeckByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	d := deckFromSqlc(result)
	return &d, nil
}

func (r *DeckRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.DeckWithCounts, int, error) {
	rows, err := r.q.ListDecksByUserID(ctx, sqlc.ListDecksByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.q.CountDecksByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	decks := make([]domain.DeckWithCounts, len(rows))
	for i, row := range rows {
		decks[i] = domain.DeckWithCounts{
			Deck: domain.Deck{
				ID:             row.ID,
				UserID:         row.UserID,
				Name:           row.Name,
				Description:    row.Description,
				IsArchived:     row.IsArchived,
				NewCardsPerDay: nullableToInt(row.NewCardsPerDay),
				Position:       int(row.Position),
				CreatedAt:      row.CreatedAt,
				UpdatedAt:      row.UpdatedAt,
			},
			CardCount:    int(row.CardCount),
			NewCount:     int(row.NewCount),
			DueCount:     int(row.DueCount),
			LearnCount:   int(row.LearnCount),
			RelearnCount: int(row.RelearnCount),
		}
	}
	return decks, int(total), nil
}

func (r *DeckRepository) Update(ctx context.Context, deck *domain.Deck) error {
	result, err := r.q.UpdateDeck(ctx, sqlc.UpdateDeckParams{
		ID:             deck.ID,
		Name:           pgtype.Text{String: deck.Name, Valid: true},
		Description:    pgtype.Text{String: deck.Description, Valid: true},
		IsArchived:     pgtype.Bool{Bool: deck.IsArchived, Valid: true},
		NewCardsPerDay: intToNullable(deck.NewCardsPerDay),
		Position:       pgtype.Int4{Int32: int32(deck.Position), Valid: true},
	})
	if err != nil {
		return err
	}
	*deck = deckFromSqlc(result)
	return nil
}

func (r *DeckRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteDeck(ctx, id)
}

func (r *DeckRepository) CreateShare(ctx context.Context, share *domain.DeckShare) error {
	result, err := r.q.CreateDeckShare(ctx, sqlc.CreateDeckShareParams{
		DeckID:    share.DeckID,
		ShareCode: share.ShareCode,
		IsPublic:  share.IsPublic,
	})
	if err != nil {
		return err
	}
	*share = deckShareFromSqlc(result)
	return nil
}

func (r *DeckRepository) GetShareByDeckID(ctx context.Context, deckID uuid.UUID) (*domain.DeckShare, error) {
	result, err := r.q.GetDeckShareByDeckID(ctx, deckID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s := deckShareFromSqlc(result)
	return &s, nil
}

func (r *DeckRepository) GetShareByCode(ctx context.Context, code string) (*domain.DeckShare, error) {
	result, err := r.q.GetDeckShareByCode(ctx, code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s := deckShareFromSqlc(result)
	return &s, nil
}

func (r *DeckRepository) DeleteShare(ctx context.Context, deckID uuid.UUID) error {
	return r.q.DeleteDeckShare(ctx, deckID)
}

func (r *DeckRepository) ListPublicDecks(ctx context.Context, search string, limit, offset int) ([]domain.DeckWithCounts, int, error) {
	rows, err := r.q.ListPublicDecks(ctx, sqlc.ListPublicDecksParams{
		Column1: search,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.q.CountPublicDecks(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	decks := make([]domain.DeckWithCounts, len(rows))
	for i, row := range rows {
		decks[i] = domain.DeckWithCounts{
			Deck: domain.Deck{
				ID:             row.ID,
				UserID:         row.UserID,
				Name:           row.Name,
				Description:    row.Description,
				IsArchived:     row.IsArchived,
				NewCardsPerDay: nullableToInt(row.NewCardsPerDay),
				Position:       int(row.Position),
				CreatedAt:      row.CreatedAt,
				UpdatedAt:      row.UpdatedAt,
			},
			CardCount: int(row.CardCount),
		}
	}
	return decks, int(total), nil
}

func (r *DeckRepository) CloneDeck(ctx context.Context, sourceDeckID, targetUserID uuid.UUID, name string) (*domain.Deck, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// CloneDeck is a complex transaction not covered by sqlc — use raw pgx
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
		`INSERT INTO cards (deck_id, front, back, content_type, tags, position)
		SELECT $1, front, back, content_type, tags, position FROM cards WHERE deck_id = $2 ORDER BY position`,
		deck.ID, sourceDeckID,
	)
	if err != nil {
		return nil, err
	}

	return &deck, tx.Commit(ctx)
}
