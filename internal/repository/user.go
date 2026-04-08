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

type UserRepository struct {
	q *sqlc.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{q: sqlc.New(pool)}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	result, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:            user.Email,
		PasswordHash:     user.PasswordHash,
		DisplayName:      user.DisplayName,
		Timezone:         user.Timezone,
		DesiredRetention: float32(user.DesiredRetention),
		DailyNewLimit:    int32(user.DailyNewLimit),
		DailyReviewLimit: int32(user.DailyReviewLimit),
	})
	if err != nil {
		return err
	}
	*user = userFromSqlc(result)
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	result, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	u := userFromSqlc(result)
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	result, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	u := userFromSqlc(result)
	return &u, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	var weights []float32
	if len(user.FSRSWeights) > 0 {
		weights = make([]float32, len(user.FSRSWeights))
		for i, w := range user.FSRSWeights {
			weights[i] = float32(w)
		}
	}

	result, err := r.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:               user.ID,
		DisplayName:      pgtype.Text{String: user.DisplayName, Valid: true},
		Timezone:         pgtype.Text{String: user.Timezone, Valid: true},
		DesiredRetention: pgtype.Float4{Float32: float32(user.DesiredRetention), Valid: true},
		DailyNewLimit:    pgtype.Int4{Int32: int32(user.DailyNewLimit), Valid: true},
		DailyReviewLimit: pgtype.Int4{Int32: int32(user.DailyReviewLimit), Valid: true},
		FsrsWeights:      weights,
	})
	if err != nil {
		return err
	}
	*user = userFromSqlc(result)
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteUser(ctx, id)
}

func (r *UserRepository) CreateOAuthAccount(ctx context.Context, account *domain.OAuthAccount) error {
	var avatarURL pgtype.Text
	if account.AvatarURL != nil {
		avatarURL = pgtype.Text{String: *account.AvatarURL, Valid: true}
	}
	result, err := r.q.CreateOAuthAccount(ctx, sqlc.CreateOAuthAccountParams{
		UserID:     account.UserID,
		Provider:   account.Provider,
		ProviderID: account.ProviderID,
		Email:      account.Email,
		AvatarUrl:  avatarURL,
	})
	if err != nil {
		return err
	}
	account.ID = result.ID
	account.CreatedAt = result.CreatedAt
	return nil
}

func (r *UserRepository) GetOAuthAccount(ctx context.Context, provider, providerID string) (*domain.OAuthAccount, error) {
	result, err := r.q.GetOAuthAccount(ctx, sqlc.GetOAuthAccountParams{
		Provider:   provider,
		ProviderID: providerID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	var avatarURL *string
	if result.AvatarUrl.Valid {
		avatarURL = &result.AvatarUrl.String
	}
	return &domain.OAuthAccount{
		ID:         result.ID,
		UserID:     result.UserID,
		Provider:   result.Provider,
		ProviderID: result.ProviderID,
		Email:      result.Email,
		AvatarURL:  avatarURL,
		CreatedAt:  result.CreatedAt,
	}, nil
}
