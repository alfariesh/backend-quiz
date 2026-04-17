package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/repository/sqlc"
)

func stringToTime(s string) pgtype.Time {
	if s == "" {
		return pgtype.Time{}
	}
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return pgtype.Time{}
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return pgtype.Time{}
	}
	micros := int64(h)*3_600_000_000 + int64(m)*60_000_000
	return pgtype.Time{Microseconds: micros, Valid: true}
}

type UserRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool, q: sqlc.New(pool)}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	result, err := querier(r.q, ctx).CreateUser(ctx, sqlc.CreateUserParams{
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
	u := userFromSqlc(result)
	*user = u
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.getUser(ctx, "id = $1", id)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.getUser(ctx, "email = $1", email)
}

func (r *UserRepository) getUser(ctx context.Context, where string, arg any) (*domain.User, error) {
	q := `
SELECT id, email, password_hash, display_name, timezone, desired_retention,
       daily_new_limit, daily_review_limit, fsrs_weights, reminder_enabled,
       reminder_time, email_verified_at, deletion_requested_at, created_at, updated_at
FROM users WHERE ` + where
	var u domain.User
	var weights []float32
	var reminderTime pgtype.Time
	var emailVerifiedAt, deletionRequestedAt pgtype.Timestamptz
	var desiredRetention float32
	var dailyNew, dailyReview int32
	err := conn(ctx, r.pool).QueryRow(ctx, q, arg).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Timezone, &desiredRetention,
		&dailyNew, &dailyReview, &weights, &u.ReminderEnabled,
		&reminderTime, &emailVerifiedAt, &deletionRequestedAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	u.DesiredRetention = float64(desiredRetention)
	u.DailyNewLimit = int(dailyNew)
	u.DailyReviewLimit = int(dailyReview)
	if len(weights) > 0 {
		u.FSRSWeights = make([]float64, len(weights))
		for i, w := range weights {
			u.FSRSWeights[i] = float64(w)
		}
	}
	u.ReminderTime = pgTimeToString(reminderTime)
	if emailVerifiedAt.Valid {
		t := emailVerifiedAt.Time
		u.EmailVerifiedAt = &t
	}
	if deletionRequestedAt.Valid {
		t := deletionRequestedAt.Time
		u.DeletionRequestedAt = &t
	}
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

	_, err := querier(r.q, ctx).UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:               user.ID,
		DisplayName:      pgtype.Text{String: user.DisplayName, Valid: true},
		Timezone:         pgtype.Text{String: user.Timezone, Valid: true},
		DesiredRetention: pgtype.Float4{Float32: float32(user.DesiredRetention), Valid: true},
		DailyNewLimit:    pgtype.Int4{Int32: int32(user.DailyNewLimit), Valid: true},
		DailyReviewLimit: pgtype.Int4{Int32: int32(user.DailyReviewLimit), Valid: true},
		FsrsWeights:      weights,
		ReminderEnabled:  pgtype.Bool{Bool: user.ReminderEnabled, Valid: true},
		ReminderTime:     stringToTime(user.ReminderTime),
	})
	if err != nil {
		return err
	}
	refreshed, err := r.getUser(ctx, "id = $1", user.ID)
	if err != nil {
		return err
	}
	*user = *refreshed
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return querier(r.q, ctx).DeleteUser(ctx, id)
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	const q = `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`
	_, err := conn(ctx, r.pool).Exec(ctx, q, userID, passwordHash)
	return err
}

func (r *UserRepository) MarkEmailVerified(ctx context.Context, userID uuid.UUID, at time.Time) error {
	const q = `UPDATE users SET email_verified_at = $2, updated_at = now() WHERE id = $1 AND email_verified_at IS NULL`
	_, err := conn(ctx, r.pool).Exec(ctx, q, userID, at)
	return err
}

func (r *UserRepository) RequestDeletion(ctx context.Context, userID uuid.UUID, at time.Time) error {
	const q = `UPDATE users SET deletion_requested_at = $2, updated_at = now() WHERE id = $1 AND deletion_requested_at IS NULL`
	tag, err := conn(ctx, r.pool).Exec(ctx, q, userID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Either user not found or deletion already pending.
		if _, err := r.GetByID(ctx, userID); err != nil {
			return err
		}
		return domain.ErrDeletionPending
	}
	return nil
}

func (r *UserRepository) CancelDeletion(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE users SET deletion_requested_at = NULL, updated_at = now() WHERE id = $1 AND deletion_requested_at IS NOT NULL`
	tag, err := conn(ctx, r.pool).Exec(ctx, q, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if _, err := r.GetByID(ctx, userID); err != nil {
			return err
		}
		return domain.ErrDeletionNotPending
	}
	return nil
}

func (r *UserRepository) ListExpiredDeletions(ctx context.Context, before time.Time, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `SELECT id FROM users WHERE deletion_requested_at IS NOT NULL AND deletion_requested_at <= $1 ORDER BY deletion_requested_at ASC LIMIT $2`
	rows, err := conn(ctx, r.pool).Query(ctx, q, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *UserRepository) CreateOAuthAccount(ctx context.Context, account *domain.OAuthAccount) error {
	var avatarURL pgtype.Text
	if account.AvatarURL != nil {
		avatarURL = pgtype.Text{String: *account.AvatarURL, Valid: true}
	}
	result, err := querier(r.q, ctx).CreateOAuthAccount(ctx, sqlc.CreateOAuthAccountParams{
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

func (r *UserRepository) ListOAuthAccountsByUser(ctx context.Context, userID uuid.UUID) ([]domain.OAuthAccount, error) {
	const q = `SELECT id, user_id, provider, provider_id, email, avatar_url, created_at FROM oauth_accounts WHERE user_id = $1 ORDER BY created_at ASC`
	rows, err := conn(ctx, r.pool).Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OAuthAccount
	for rows.Next() {
		var a domain.OAuthAccount
		var avatarURL pgtype.Text
		if err := rows.Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderID, &a.Email, &avatarURL, &a.CreatedAt); err != nil {
			return nil, err
		}
		if avatarURL.Valid {
			s := avatarURL.String
			a.AvatarURL = &s
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *UserRepository) GetOAuthAccount(ctx context.Context, provider, providerID string) (*domain.OAuthAccount, error) {
	result, err := querier(r.q, ctx).GetOAuthAccount(ctx, sqlc.GetOAuthAccountParams{
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
