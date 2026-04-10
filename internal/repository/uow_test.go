package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alfariesh/backend-quiz/internal/domain"
)

func TestUoW_Commit(t *testing.T) {
	pool := testPool(t)
	uow := NewUnitOfWork(pool)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	var userID = ""
	err := uow.Do(ctx, func(ctx context.Context) error {
		user := &domain.User{Email: "commit@example.com", PasswordHash: "h", DisplayName: "Commit", Timezone: "UTC"}
		if err := repo.Create(ctx, user); err != nil {
			return err
		}
		userID = user.ID.String()
		return nil
	})
	require.NoError(t, err)

	// User should exist after commit
	got, err := repo.GetByEmail(ctx, "commit@example.com")
	require.NoError(t, err)
	assert.Equal(t, userID, got.ID.String())
}

func TestUoW_Rollback(t *testing.T) {
	pool := testPool(t)
	uow := NewUnitOfWork(pool)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	testErr := errors.New("intentional error")
	err := uow.Do(ctx, func(ctx context.Context) error {
		user := &domain.User{Email: "rollback@example.com", PasswordHash: "h", DisplayName: "Rollback", Timezone: "UTC"}
		if err := repo.Create(ctx, user); err != nil {
			return err
		}
		return testErr // trigger rollback
	})
	assert.ErrorIs(t, err, testErr)

	// User should NOT exist after rollback
	_, err = repo.GetByEmail(ctx, "rollback@example.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUoW_NestedJoinsExistingTx(t *testing.T) {
	pool := testPool(t)
	uow := NewUnitOfWork(pool)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	err := uow.Do(ctx, func(ctx context.Context) error {
		// First write in outer tx
		user1 := &domain.User{Email: "outer@example.com", PasswordHash: "h", DisplayName: "Outer", Timezone: "UTC"}
		if err := repo.Create(ctx, user1); err != nil {
			return err
		}

		// Nested Do should join the same tx, not start a new one
		return uow.Do(ctx, func(ctx context.Context) error {
			user2 := &domain.User{Email: "inner@example.com", PasswordHash: "h", DisplayName: "Inner", Timezone: "UTC"}
			return repo.Create(ctx, user2)
		})
	})
	require.NoError(t, err)

	// Both users should exist
	_, err = repo.GetByEmail(ctx, "outer@example.com")
	require.NoError(t, err)
	_, err = repo.GetByEmail(ctx, "inner@example.com")
	require.NoError(t, err)
}
