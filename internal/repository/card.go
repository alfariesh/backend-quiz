package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/repository/sqlc"
)

type CardRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewCardRepository(pool *pgxpool.Pool) *CardRepository {
	return &CardRepository{q: sqlc.New(pool), pool: pool}
}

func (r *CardRepository) Create(ctx context.Context, card *domain.Card) error {
	if card.ContentType == "" {
		card.ContentType = domain.ContentTypePlain
	}
	result, err := querier(r.q, ctx).CreateCard(ctx, sqlc.CreateCardParams{
		DeckID:      card.DeckID,
		Front:       card.Front,
		Back:        card.Back,
		ContentType: card.ContentType,
		Tags:        card.Tags,
		Position:    int32(card.Position),
	})
	if err != nil {
		return err
	}
	*card = cardFromSqlc(result)
	return nil
}

func (r *CardRepository) BulkCreate(ctx context.Context, cards []*domain.Card) error {
	ctx, tx, isOwner, err := beginOrJoin(ctx, r.pool)
	if err != nil {
		return err
	}
	if isOwner {
		defer tx.Rollback(ctx)
	}

	q := querier(r.q, ctx)
	for _, card := range cards {
		if card.ContentType == "" {
			card.ContentType = domain.ContentTypePlain
		}
		result, err := q.CreateCard(ctx, sqlc.CreateCardParams{
			DeckID:      card.DeckID,
			Front:       card.Front,
			Back:        card.Back,
			ContentType: card.ContentType,
			Tags:        card.Tags,
			Position:    int32(card.Position),
		})
		if err != nil {
			return err
		}
		*card = cardFromSqlc(result)
	}

	if isOwner {
		return tx.Commit(ctx)
	}
	return nil
}

func (r *CardRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	result, err := querier(r.q, ctx).GetCardByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	c := cardFromSqlc(result)
	return &c, nil
}

func (r *CardRepository) ListByDeckID(ctx context.Context, deckID uuid.UUID, filter domain.CardFilter, limit, offset int) ([]domain.Card, int, error) {
	var stateParam pgtype.Int2
	if filter.State != nil {
		stateParam = pgtype.Int2{Int16: int16(*filter.State), Valid: true}
	}
	var tagParam pgtype.Text
	if filter.Tag != "" {
		tagParam = pgtype.Text{String: filter.Tag, Valid: true}
	}
	var queryParam pgtype.Text
	if filter.Query != "" {
		queryParam = pgtype.Text{String: filter.Query, Valid: true}
	}

	rows, err := querier(r.q, ctx).ListCardsByDeckID(ctx, sqlc.ListCardsByDeckIDParams{
		DeckID:    deckID,
		State:     stateParam,
		Tag:       tagParam,
		Query:     queryParam,
		RowLimit:  int32(limit),
		RowOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := querier(r.q, ctx).CountCardsByDeckID(ctx, sqlc.CountCardsByDeckIDParams{
		DeckID: deckID,
		State:  stateParam,
		Tag:    tagParam,
		Query:  queryParam,
	})
	if err != nil {
		return nil, 0, err
	}

	return cardsFromSqlc(rows), int(total), nil
}

func (r *CardRepository) Update(ctx context.Context, card *domain.Card) error {
	return querier(r.q, ctx).UpdateCard(ctx, sqlc.UpdateCardParams{
		ID:          card.ID,
		Front:       card.Front,
		Back:        card.Back,
		ContentType: card.ContentType,
		Tags:        card.Tags,
	})
}

func (r *CardRepository) UpdateFSRS(ctx context.Context, card *domain.Card) error {
	return querier(r.q, ctx).UpdateCardFSRS(ctx, sqlc.UpdateCardFSRSParams{
		ID:            card.ID,
		Due:           card.Due,
		Stability:     float32(card.Stability),
		Difficulty:    float32(card.Difficulty),
		ElapsedDays:   int32(card.ElapsedDays),
		ScheduledDays: int32(card.ScheduledDays),
		Reps:          int32(card.Reps),
		Lapses:        int32(card.Lapses),
		State:         int16(card.State),
		LastReview:    timeToNullable(card.LastReview),
	})
}

func (r *CardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return querier(r.q, ctx).DeleteCard(ctx, id)
}

func (r *CardRepository) SetSuspended(ctx context.Context, id uuid.UUID, suspended bool) error {
	return querier(r.q, ctx).SetCardSuspended(ctx, sqlc.SetCardSuspendedParams{
		ID:          id,
		IsSuspended: suspended,
	})
}

func (r *CardRepository) ResetFSRS(ctx context.Context, id uuid.UUID) error {
	return querier(r.q, ctx).ResetCardFSRS(ctx, id)
}

func (r *CardRepository) GetDueCards(ctx context.Context, deckID uuid.UUID, now time.Time, newLimit, reviewLimit int) ([]domain.Card, error) {
	rows, err := querier(r.q, ctx).GetDueCards(ctx, sqlc.GetDueCardsParams{
		DeckID:  deckID,
		Due:     now,
		Limit:   int32(newLimit + reviewLimit),
	})
	if err != nil {
		return nil, err
	}

	cards := cardsFromSqlc(rows)

	// Apply new card limit
	var result []domain.Card
	newCount := 0
	for _, c := range cards {
		if c.State == domain.CardStateNew {
			if newCount >= newLimit {
				continue
			}
			newCount++
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *CardRepository) CountByState(ctx context.Context, deckID uuid.UUID) (map[domain.CardState]int, error) {
	rows, err := querier(r.q, ctx).CountCardsByState(ctx, deckID)
	if err != nil {
		return nil, err
	}

	counts := make(map[domain.CardState]int)
	for _, row := range rows {
		counts[domain.CardState(row.State)] = int(row.Count)
	}
	return counts, nil
}

func (r *CardRepository) CountDue(ctx context.Context, deckID uuid.UUID, now time.Time) (int, error) {
	count, err := querier(r.q, ctx).CountDueCards(ctx, sqlc.CountDueCardsParams{
		DeckID: deckID,
		Due:    now,
	})
	return int(count), err
}

func (r *CardRepository) GetDeckMasteryStats(ctx context.Context, deckID uuid.UUID) (totalCards, matureCards int, avgStability float64, err error) {
	row, err := querier(r.q, ctx).GetDeckMasteryStats(ctx, deckID)
	if err != nil {
		return 0, 0, 0, err
	}
	return int(row.TotalCards), int(row.MatureCards), float64(row.AvgStability), nil
}

func (r *CardRepository) GetNextDueAt(ctx context.Context, userID uuid.UUID, now time.Time) (*time.Time, error) {
	result, err := querier(r.q, ctx).GetNextDueAt(ctx, sqlc.GetNextDueAtParams{
		UserID: userID,
		Due:    now,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	if t, ok := result.(time.Time); ok {
		return &t, nil
	}
	return nil, nil
}

func (r *CardRepository) GetUpcomingDueSummary(ctx context.Context, userID uuid.UUID, now time.Time, horizon time.Time) ([]domain.DeckDueSummary, error) {
	rows, err := querier(r.q, ctx).GetUpcomingDueSummary(ctx, sqlc.GetUpcomingDueSummaryParams{
		UserID: userID,
		Due:    now,
		Due_2:  horizon,
	})
	if err != nil {
		return nil, err
	}

	summaries := make([]domain.DeckDueSummary, len(rows))
	for i, row := range rows {
		summaries[i] = domain.DeckDueSummary{
			DeckID:   row.DeckID,
			DeckName: row.DeckName,
			NewCount: int(row.NewCount),
			DueNow:   int(row.DueNow),
			DueSoon:  int(row.DueSoon),
		}
	}
	return summaries, nil
}

func (r *CardRepository) GetWeakCards(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Card, error) {
	rows, err := querier(r.q, ctx).GetWeakCards(ctx, sqlc.GetWeakCardsParams{
		UserID: userID,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return cardsFromSqlc(rows), nil
}
