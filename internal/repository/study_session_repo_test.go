package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

func TestSessionRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	sessionRepo := NewStudySessionRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "session@example.com")
	deck := createTestDeck(t, pool, user.ID, "Session Deck")

	session := &domain.StudySession{
		UserID: user.ID,
		DeckID: &deck.ID,
	}
	err := sessionRepo.Create(ctx, session)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, session.ID)
	assert.False(t, session.StartedAt.IsZero())

	got, err := sessionRepo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.UserID)
	assert.Equal(t, &deck.ID, got.DeckID)
	assert.Nil(t, got.EndedAt)
}

func TestSessionRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	sessionRepo := NewStudySessionRepository(pool)
	ctx := context.Background()

	_, err := sessionRepo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSessionRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	sessionRepo := NewStudySessionRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "sessionupd@example.com")

	session := &domain.StudySession{UserID: user.ID}
	require.NoError(t, sessionRepo.Create(ctx, session))

	now := time.Now().UTC()
	session.EndedAt = &now
	session.NewCount = 5
	session.ReviewCount = 10
	session.RelearnCount = 2
	session.TotalDurationMS = 60000

	err := sessionRepo.Update(ctx, session)
	require.NoError(t, err)

	got, err := sessionRepo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.EndedAt)
	assert.Equal(t, 5, got.NewCount)
	assert.Equal(t, 10, got.ReviewCount)
	assert.Equal(t, 2, got.RelearnCount)
	assert.Equal(t, 60000, got.TotalDurationMS)
}

func TestSessionRepo_ListByUserID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	sessionRepo := NewStudySessionRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "sessionlist@example.com")

	for i := 0; i < 3; i++ {
		require.NoError(t, sessionRepo.Create(ctx, &domain.StudySession{UserID: user.ID}))
	}

	sessions, total, err := sessionRepo.ListByUserID(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, sessions, 3)
}
