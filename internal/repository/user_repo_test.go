package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

func TestUserRepo_CreateAndGetByID(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{
		Email:            "test@example.com",
		PasswordHash:     "hashed",
		DisplayName:      "Test User",
		Timezone:         "Asia/Jakarta",
		DesiredRetention: 0.9,
		DailyNewLimit:    20,
		DailyReviewLimit: 200,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.False(t, user.CreatedAt.IsZero())

	// GetByID
	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, got.Email)
	assert.Equal(t, user.DisplayName, got.DisplayName)
	assert.Equal(t, "Asia/Jakarta", got.Timezone)
}

func TestUserRepo_GetByEmail(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{
		Email:        "find@example.com",
		PasswordHash: "hashed",
		DisplayName:  "Find Me",
		Timezone:     "UTC",
	}
	require.NoError(t, repo.Create(ctx, user))

	got, err := repo.GetByEmail(ctx, "find@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nobody@example.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepo_Update(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{
		Email:            "update@example.com",
		PasswordHash:     "hashed",
		DisplayName:      "Before",
		Timezone:         "UTC",
		DesiredRetention: 0.9,
		DailyNewLimit:    20,
		DailyReviewLimit: 200,
	}
	require.NoError(t, repo.Create(ctx, user))

	user.DisplayName = "After"
	user.Timezone = "Asia/Jakarta"
	user.DesiredRetention = 0.85
	user.FSRSWeights = []float64{0.1, 0.2, 0.3}

	err := repo.Update(ctx, user)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "After", got.DisplayName)
	assert.Equal(t, "Asia/Jakarta", got.Timezone)
	assert.InDelta(t, 0.85, got.DesiredRetention, 0.01)
	assert.Len(t, got.FSRSWeights, 3)
}

func TestUserRepo_Delete(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{
		Email:        "delete@example.com",
		PasswordHash: "hashed",
		DisplayName:  "Delete Me",
		Timezone:     "UTC",
	}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.Delete(ctx, user.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, user.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepo_DuplicateEmail(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user1 := &domain.User{Email: "dup@example.com", PasswordHash: "h", DisplayName: "A", Timezone: "UTC"}
	require.NoError(t, repo.Create(ctx, user1))

	user2 := &domain.User{Email: "dup@example.com", PasswordHash: "h", DisplayName: "B", Timezone: "UTC"}
	err := repo.Create(ctx, user2)
	assert.Error(t, err) // unique constraint violation
}
