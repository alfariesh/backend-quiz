package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name, timezone, desired_retention, daily_new_limit, daily_review_limit)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		user.Email, user.PasswordHash, user.DisplayName, user.Timezone,
		user.DesiredRetention, user.DailyNewLimit, user.DailyReviewLimit,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, timezone, desired_retention, daily_new_limit, daily_review_limit, fsrs_weights, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Timezone,
		&u.DesiredRetention, &u.DailyNewLimit, &u.DailyReviewLimit, &u.FSRSWeights,
		&u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, timezone, desired_retention, daily_new_limit, daily_review_limit, fsrs_weights, created_at, updated_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Timezone,
		&u.DesiredRetention, &u.DailyNewLimit, &u.DailyReviewLimit, &u.FSRSWeights,
		&u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET display_name=$2, timezone=$3, desired_retention=$4, daily_new_limit=$5, daily_review_limit=$6, fsrs_weights=$7, updated_at=now()
		WHERE id = $1`,
		user.ID, user.DisplayName, user.Timezone, user.DesiredRetention,
		user.DailyNewLimit, user.DailyReviewLimit, user.FSRSWeights,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *UserRepository) CreateOAuthAccount(ctx context.Context, account *domain.OAuthAccount) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_id, email, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		account.UserID, account.Provider, account.ProviderID, account.Email, account.AvatarURL,
	).Scan(&account.ID, &account.CreatedAt)
}

func (r *UserRepository) GetOAuthAccount(ctx context.Context, provider, providerID string) (*domain.OAuthAccount, error) {
	var a domain.OAuthAccount
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, provider, provider_id, email, avatar_url, created_at
		FROM oauth_accounts WHERE provider = $1 AND provider_id = $2`,
		provider, providerID,
	).Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderID, &a.Email, &a.AvatarURL, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &a, err
}
