package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	mockdomain "github.com/rekanesiads/backend-quiz/internal/mocks/domain"
)


// --- validateFSRSState ---

func TestValidateFSRSState_Valid(t *testing.T) {
	state := FSRSCardState{
		Due:           time.Now().Add(24 * time.Hour),
		Stability:     5.0,
		Difficulty:    3.5,
		ElapsedDays:   1,
		ScheduledDays: 3,
		Reps:          2,
		Lapses:        0,
		State:         1,
		LastReview:    time.Now(),
	}
	assert.NoError(t, validateFSRSState(state))
}

func TestValidateFSRSState_InvalidState(t *testing.T) {
	tests := []struct {
		name  string
		state int
	}{
		{"negative state", -1},
		{"state too high", 4},
		{"state way too high", 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := FSRSCardState{
				Due:        time.Now().Add(time.Hour),
				Stability:  1.0,
				Difficulty: 5.0,
				State:      tt.state,
				LastReview: time.Now(),
			}
			err := validateFSRSState(state)
			require.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrInvalidInput)
			assert.Contains(t, err.Error(), "invalid card state")
		})
	}
}

func TestValidateFSRSState_NegativeStability(t *testing.T) {
	state := FSRSCardState{
		Due:        time.Now().Add(time.Hour),
		Stability:  -0.1,
		Difficulty: 5.0,
		State:      0,
		LastReview: time.Now(),
	}
	err := validateFSRSState(state)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Contains(t, err.Error(), "stability")
}

func TestValidateFSRSState_InvalidDifficulty(t *testing.T) {
	tests := []struct {
		name       string
		difficulty float64
	}{
		{"negative", -1.0},
		{"too high", 10.1},
		{"way too high", 50.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := FSRSCardState{
				Due:        time.Now().Add(time.Hour),
				Stability:  1.0,
				Difficulty: tt.difficulty,
				State:      0,
				LastReview: time.Now(),
			}
			err := validateFSRSState(state)
			require.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrInvalidInput)
			assert.Contains(t, err.Error(), "difficulty")
		})
	}
}

func TestValidateFSRSState_DueTooFarInFuture(t *testing.T) {
	state := FSRSCardState{
		Due:        time.Now().AddDate(0, 0, 36501),
		Stability:  1.0,
		Difficulty: 5.0,
		State:      0,
		LastReview: time.Now(),
	}
	err := validateFSRSState(state)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Contains(t, err.Error(), "due date too far")
}

func TestValidateFSRSState_BoundaryValues(t *testing.T) {
	tests := []struct {
		name  string
		state FSRSCardState
	}{
		{
			"state 0 (new)",
			FSRSCardState{Due: time.Now(), Stability: 0, Difficulty: 0, State: 0, LastReview: time.Now()},
		},
		{
			"state 3 (relearning)",
			FSRSCardState{Due: time.Now(), Stability: 0, Difficulty: 10, State: 3, LastReview: time.Now()},
		},
		{
			"max difficulty 10",
			FSRSCardState{Due: time.Now(), Stability: 100, Difficulty: 10.0, State: 2, LastReview: time.Now()},
		},
		{
			"zero stability",
			FSRSCardState{Due: time.Now(), Stability: 0, Difficulty: 5.0, State: 0, LastReview: time.Now()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, validateFSRSState(tt.state))
		})
	}
}

// --- applyFSRSCardState ---

func TestApplyFSRSCardState(t *testing.T) {
	card := &domain.Card{
		ID:     uuid.New(),
		DeckID: uuid.New(),
		Front:  "test front",
		Back:   "test back",
	}

	now := time.Now()
	lastReview := now.Add(-24 * time.Hour)
	dueDate := now.Add(72 * time.Hour)

	state := FSRSCardState{
		Due:           dueDate,
		Stability:     12.5,
		Difficulty:    4.2,
		ElapsedDays:   7,
		ScheduledDays: 14,
		Reps:          5,
		Lapses:        1,
		State:         2,
		LastReview:    lastReview,
	}

	applyFSRSCardState(card, state)

	assert.Equal(t, dueDate, card.Due)
	assert.Equal(t, 12.5, card.Stability)
	assert.Equal(t, 4.2, card.Difficulty)
	assert.Equal(t, 7, card.ElapsedDays)
	assert.Equal(t, 14, card.ScheduledDays)
	assert.Equal(t, 5, card.Reps)
	assert.Equal(t, 1, card.Lapses)
	assert.Equal(t, domain.CardStateReview, card.State)
	require.NotNil(t, card.LastReview)
	assert.Equal(t, lastReview, *card.LastReview)
}

func TestApplyFSRSCardState_PreservesNonFSRSFields(t *testing.T) {
	deckID := uuid.New()
	cardID := uuid.New()
	card := &domain.Card{
		ID:          cardID,
		DeckID:      deckID,
		Front:       "original front",
		Back:        "original back",
		ContentType: "markdown",
		Tags:        []string{"tag1"},
		IsSuspended: false,
		Position:    3,
	}

	state := FSRSCardState{
		Due:        time.Now(),
		Stability:  1.0,
		Difficulty: 5.0,
		State:      1,
		LastReview: time.Now(),
	}

	applyFSRSCardState(card, state)

	assert.Equal(t, cardID, card.ID)
	assert.Equal(t, deckID, card.DeckID)
	assert.Equal(t, "original front", card.Front)
	assert.Equal(t, "original back", card.Back)
	assert.Equal(t, "markdown", card.ContentType)
	assert.Equal(t, []string{"tag1"}, card.Tags)
	assert.False(t, card.IsSuspended)
	assert.Equal(t, 3, card.Position)
}

// --- countCards ---

func TestCountCards_Empty(t *testing.T) {
	svc := &StudyService{}
	counts := svc.countCards(nil)

	assert.Equal(t, 0, counts.New)
	assert.Equal(t, 0, counts.Learning)
	assert.Equal(t, 0, counts.Review)
	assert.Equal(t, 0, counts.Total)
}

func TestCountCards_Mixed(t *testing.T) {
	svc := &StudyService{}
	cards := []domain.Card{
		{State: domain.CardStateNew},
		{State: domain.CardStateNew},
		{State: domain.CardStateLearning},
		{State: domain.CardStateReview},
		{State: domain.CardStateReview},
		{State: domain.CardStateReview},
		{State: domain.CardStateRelearning},
	}

	counts := svc.countCards(cards)

	assert.Equal(t, 2, counts.New)
	assert.Equal(t, 2, counts.Learning) // learning + relearning
	assert.Equal(t, 3, counts.Review)
	assert.Equal(t, 7, counts.Total)
}

func TestCountCards_AllNew(t *testing.T) {
	svc := &StudyService{}
	cards := []domain.Card{
		{State: domain.CardStateNew},
		{State: domain.CardStateNew},
		{State: domain.CardStateNew},
	}

	counts := svc.countCards(cards)

	assert.Equal(t, 3, counts.New)
	assert.Equal(t, 0, counts.Learning)
	assert.Equal(t, 0, counts.Review)
	assert.Equal(t, 3, counts.Total)
}

func TestCountCards_AllReview(t *testing.T) {
	svc := &StudyService{}
	cards := []domain.Card{
		{State: domain.CardStateReview},
		{State: domain.CardStateReview},
	}

	counts := svc.countCards(cards)

	assert.Equal(t, 0, counts.New)
	assert.Equal(t, 0, counts.Learning)
	assert.Equal(t, 2, counts.Review)
	assert.Equal(t, 2, counts.Total)
}

// =============================================================================
// StudyService method tests (with mocks)
// =============================================================================

func studyTestUser() *domain.User {
	return &domain.User{
		ID:               uuid.New(),
		Email:            "study@test.com",
		DailyNewLimit:    20,
		DailyReviewLimit: 200,
	}
}

func studyTestSession(userID uuid.UUID, deckID *uuid.UUID) *domain.StudySession {
	return &domain.StudySession{
		ID:     uuid.New(),
		UserID: userID,
		DeckID: deckID,
	}
}

// --- StartSession ---

func TestStudyService_StartSession_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	user := studyTestUser()
	deckID := uuid.New()

	userRepo.On("GetByID", ctx, user.ID).Return(user, nil)
	cardRepo.On("GetDueCards", ctx, deckID, mock.AnythingOfType("time.Time"), 20, 200).Return([]domain.Card{
		{ID: uuid.New(), State: domain.CardStateNew},
		{ID: uuid.New(), State: domain.CardStateReview},
	}, nil)
	sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	resp, err := svc.StartSession(ctx, user.ID, StartSessionRequest{DeckID: deckID})

	require.NoError(t, err)
	assert.NotNil(t, resp.Session)
	assert.Len(t, resp.Cards, 2)
	assert.Equal(t, 1, resp.Counts.New)
	assert.Equal(t, 1, resp.Counts.Review)
	assert.Equal(t, 2, resp.Counts.Total)

}

func TestStudyService_StartSession_UserNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	userRepo.On("GetByID", ctx, userID).Return(nil, domain.ErrNotFound)

	_, err := svc.StartSession(ctx, userID, StartSessionRequest{DeckID: uuid.New()})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- GetSession ---

func TestStudyService_GetSession_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)

	result, err := svc.GetSession(ctx, userID, session.ID)

	require.NoError(t, err)
	assert.Equal(t, session.ID, result.ID)
}

func TestStudyService_GetSession_NotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	sessionID := uuid.New()
	sessionRepo.On("GetByID", ctx, sessionID).Return(nil, domain.ErrNotFound)

	_, err := svc.GetSession(ctx, uuid.New(), sessionID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestStudyService_GetSession_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	deckID := uuid.New()
	session := studyTestSession(uuid.New(), &deckID)

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)

	_, err := svc.GetSession(ctx, uuid.New(), session.ID) // different user
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- SubmitReview ---

func TestStudyService_SubmitReview_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()
	card := &domain.Card{ID: cardID, DeckID: deckID, State: domain.CardStateNew}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(card, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	result, err := svc.SubmitReview(ctx, userID, session.ID, SubmitReviewRequest{
		CardID:     cardID,
		Rating:     3,
		DurationMS: 5000,
		Card: FSRSCardState{
			Due: now.Add(24 * time.Hour), Stability: 2.5, Difficulty: 5.0,
			State: 1, LastReview: now,
		},
		Log: FSRSLogState{ScheduledDays: 1, ElapsedDays: 0, Stability: 2.5, Difficulty: 5.0},
	})

	require.NoError(t, err)
	assert.Equal(t, cardID, result.Card.ID)
	assert.Equal(t, domain.ReviewSourceFlashcard, result.ReviewLog.Source)
	assert.Equal(t, domain.CardStateNew, result.ReviewLog.State) // state before
	assert.Equal(t, 1, session.NewCount)                         // incremented

}

func TestStudyService_SubmitReview_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	deckID := uuid.New()
	session := studyTestSession(uuid.New(), &deckID)

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)

	_, err := svc.SubmitReview(ctx, uuid.New(), session.ID, SubmitReviewRequest{CardID: uuid.New()})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestStudyService_SubmitReview_SessionNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	sessionID := uuid.New()
	sessionRepo.On("GetByID", ctx, sessionID).Return(nil, domain.ErrNotFound)

	_, err := svc.SubmitReview(ctx, uuid.New(), sessionID, SubmitReviewRequest{CardID: uuid.New()})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestStudyService_SubmitReview_ReviewState(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()
	card := &domain.Card{ID: cardID, DeckID: deckID, State: domain.CardStateReview}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(card, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	result, err := svc.SubmitReview(ctx, userID, session.ID, SubmitReviewRequest{
		CardID: cardID, Rating: 3,
		Card: FSRSCardState{
			Due: now.Add(24 * time.Hour), Stability: 5.0, Difficulty: 4.0,
			State: 2, LastReview: now,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, domain.CardStateReview, result.ReviewLog.State)
	assert.Equal(t, 1, session.ReviewCount)
}

func TestStudyService_SubmitReview_RelearningState(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()
	card := &domain.Card{ID: cardID, DeckID: deckID, State: domain.CardStateRelearning}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(card, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	result, err := svc.SubmitReview(ctx, userID, session.ID, SubmitReviewRequest{
		CardID: cardID, Rating: 3,
		Card: FSRSCardState{
			Due: now.Add(24 * time.Hour), Stability: 5.0, Difficulty: 4.0,
			State: 2, LastReview: now,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, domain.CardStateRelearning, result.ReviewLog.State)
	assert.Equal(t, 1, session.RelearnCount)
}

func TestStudyService_SubmitReview_SessionEnded(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	endedAt := time.Now()
	session := &domain.StudySession{ID: uuid.New(), UserID: userID, DeckID: &deckID, EndedAt: &endedAt}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)

	_, err := svc.SubmitReview(ctx, userID, session.ID, SubmitReviewRequest{CardID: uuid.New()})
	assert.ErrorIs(t, err, domain.ErrSessionEnded)
}

func TestStudyService_SubmitReview_CardSuspended(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()
	card := &domain.Card{ID: cardID, IsSuspended: true}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(card, nil)

	_, err := svc.SubmitReview(ctx, userID, session.ID, SubmitReviewRequest{CardID: cardID})
	assert.ErrorIs(t, err, domain.ErrCardSuspended)
}

func TestStudyService_SubmitReview_InvalidFSRS(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()
	card := &domain.Card{ID: cardID}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(card, nil)

	_, err := svc.SubmitReview(ctx, userID, session.ID, SubmitReviewRequest{
		CardID: cardID, Rating: 3,
		Card: FSRSCardState{Stability: -1}, // invalid
	})
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// --- BatchReview ---

func TestStudyService_BatchReview_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	card1ID, card2ID := uuid.New(), uuid.New()

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, card1ID).Return(&domain.Card{ID: card1ID, State: domain.CardStateNew}, nil)
	cardRepo.On("GetByID", ctx, card2ID).Return(&domain.Card{ID: card2ID, State: domain.CardStateReview}, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	validCard := FSRSCardState{
		Due: now.Add(24 * time.Hour), Stability: 2.0, Difficulty: 5.0,
		State: 1, LastReview: now,
	}

	result, err := svc.BatchReview(ctx, userID, session.ID, BatchReviewRequest{
		Reviews: []BatchReviewItem{
			{CardID: card1ID, Rating: 3, DurationMS: 3000, ReviewedAt: now, Card: validCard},
			{CardID: card2ID, Rating: 4, DurationMS: 2000, ReviewedAt: now, Card: validCard},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Processed)
	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 1, session.NewCount)
	assert.Equal(t, 1, session.ReviewCount)
	assert.Equal(t, 5000, session.TotalDurationMS)
}

func TestStudyService_BatchReview_PartialErrors(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	goodCardID := uuid.New()
	badCardID := uuid.New()

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, goodCardID).Return(&domain.Card{ID: goodCardID, State: domain.CardStateNew}, nil)
	cardRepo.On("GetByID", ctx, badCardID).Return(nil, domain.ErrNotFound) // card not found
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	validCard := FSRSCardState{
		Due: now.Add(24 * time.Hour), Stability: 2.0, Difficulty: 5.0,
		State: 1, LastReview: now,
	}

	result, err := svc.BatchReview(ctx, userID, session.ID, BatchReviewRequest{
		Reviews: []BatchReviewItem{
			{CardID: goodCardID, Rating: 3, DurationMS: 3000, ReviewedAt: now, Card: validCard},
			{CardID: badCardID, Rating: 3, DurationMS: 2000, ReviewedAt: now, Card: validCard},
			{CardID: uuid.New(), Rating: 3, ReviewedAt: now, Card: FSRSCardState{Stability: -1}}, // invalid FSRS
		},
	})

	require.NoError(t, err) // BatchReview itself doesn't fail
	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 2, result.Errors)
}

func TestStudyService_BatchReview_SessionEnded(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	endedAt := time.Now()
	session := &domain.StudySession{ID: uuid.New(), UserID: userID, DeckID: &deckID, EndedAt: &endedAt}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)

	_, err := svc.BatchReview(ctx, userID, session.ID, BatchReviewRequest{
		Reviews: []BatchReviewItem{{}},
	})
	assert.ErrorIs(t, err, domain.ErrSessionEnded)
}

func TestStudyService_BatchReview_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	deckID := uuid.New()
	session := studyTestSession(uuid.New(), &deckID) // owned by different user

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)

	_, err := svc.BatchReview(ctx, uuid.New(), session.ID, BatchReviewRequest{
		Reviews: []BatchReviewItem{{}},
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestStudyService_BatchReview_SkipsSuspendedCards(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	suspendedID := uuid.New()

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, suspendedID).Return(&domain.Card{ID: suspendedID, IsSuspended: true}, nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	validCard := FSRSCardState{
		Due: now.Add(24 * time.Hour), Stability: 2.0, Difficulty: 5.0,
		State: 1, LastReview: now,
	}

	result, err := svc.BatchReview(ctx, userID, session.ID, BatchReviewRequest{
		Reviews: []BatchReviewItem{
			{CardID: suspendedID, Rating: 3, ReviewedAt: now, Card: validCard},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
	assert.Equal(t, 1, result.Errors)
}

// --- EndSession ---

func TestStudyService_EndSession_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	result, err := svc.EndSession(ctx, userID, session.ID)

	require.NoError(t, err)
	assert.NotNil(t, result.EndedAt)
}

func TestStudyService_EndSession_AlreadyEnded(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	endedAt := time.Now()
	session := &domain.StudySession{ID: uuid.New(), UserID: userID, DeckID: &deckID, EndedAt: &endedAt}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)

	_, err := svc.EndSession(ctx, userID, session.ID)
	assert.ErrorIs(t, err, domain.ErrSessionEnded)
}

func TestStudyService_EndSession_Forbidden(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	deckID := uuid.New()
	session := studyTestSession(uuid.New(), &deckID)

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)

	_, err := svc.EndSession(ctx, uuid.New(), session.ID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- GetReminders ---

func TestStudyService_GetReminders_Success(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()
	userID := uuid.New()

	deckID1, deckID2 := uuid.New(), uuid.New()
	summaries := []domain.DeckDueSummary{
		{DeckID: deckID1, DeckName: "Fiqh", NewCount: 5, DueNow: 10, DueSoon: 3},
		{DeckID: deckID2, DeckName: "Aqidah", NewCount: 2, DueNow: 8, DueSoon: 1},
	}

	nextDue := time.Now().Add(2 * time.Hour)

	cardRepo.On("GetUpcomingDueSummary", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(summaries, nil)
	cardRepo.On("GetNextDueAt", ctx, userID, mock.AnythingOfType("time.Time")).Return(&nextDue, nil)
	reviewRepo.On("CountByUserAndDate", ctx, userID, mock.AnythingOfType("time.Time")).Return(5, nil)

	statsRepo.On("GetStreak", ctx, userID).Return(7, nil)

	result, err := svc.GetReminders(ctx, userID, 24)

	require.NoError(t, err)
	assert.Len(t, result.Decks, 2)
	assert.Equal(t, 18, result.TotalDue)  // 10+8
	assert.Equal(t, 4, result.DueSoon)    // 3+1
	assert.Equal(t, 7, result.Streak)
	assert.True(t, result.StudiedToday)   // todayCount=5 > 0
	assert.NotNil(t, result.NextDueAt)
}

func TestStudyService_GetReminders_NoStudyToday(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()
	userID := uuid.New()

	cardRepo.On("GetUpcomingDueSummary", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]domain.DeckDueSummary{}, nil)
	cardRepo.On("GetNextDueAt", ctx, userID, mock.AnythingOfType("time.Time")).Return((*time.Time)(nil), nil)
	reviewRepo.On("CountByUserAndDate", ctx, userID, mock.AnythingOfType("time.Time")).Return(0, nil)

	statsRepo.On("GetStreak", ctx, userID).Return(0, nil)

	result, err := svc.GetReminders(ctx, userID, 12)

	require.NoError(t, err)
	assert.Empty(t, result.Decks)
	assert.Equal(t, 0, result.TotalDue)
	assert.Equal(t, 0, result.Streak)
	assert.False(t, result.StudiedToday)
	assert.Nil(t, result.NextDueAt)
}

func TestStudyService_GetReminders_DueSummaryError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()
	userID := uuid.New()

	cardRepo.On("GetUpcomingDueSummary", ctx, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]domain.DeckDueSummary(nil), assert.AnError)

	result, err := svc.GetReminders(ctx, userID, 24)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// --- StartSession: card fetch error ---

func TestStudyService_StartSession_CardError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	user := studyTestUser()
	deckID := uuid.New()

	userRepo.On("GetByID", ctx, user.ID).Return(user, nil)
	cardRepo.On("GetDueCards", ctx, deckID, mock.AnythingOfType("time.Time"), 20, 200).Return(nil, assert.AnError)

	_, err := svc.StartSession(ctx, user.ID, StartSessionRequest{DeckID: deckID})
	assert.Error(t, err)
}

func TestStudyService_StartSession_SessionCreateError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	user := studyTestUser()
	deckID := uuid.New()

	userRepo.On("GetByID", ctx, user.ID).Return(user, nil)
	cardRepo.On("GetDueCards", ctx, deckID, mock.AnythingOfType("time.Time"), 20, 200).Return([]domain.Card{}, nil)
	sessionRepo.On("Create", ctx, mock.AnythingOfType("*domain.StudySession")).Return(assert.AnError)

	_, err := svc.StartSession(ctx, user.ID, StartSessionRequest{DeckID: deckID})
	assert.Error(t, err)
}

// --- EndSession: not found ---

func TestStudyService_EndSession_NotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	sessionID := uuid.New()
	sessionRepo.On("GetByID", ctx, sessionID).Return(nil, domain.ErrNotFound)

	_, err := svc.EndSession(ctx, uuid.New(), sessionID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- BatchReview: session not found ---

func TestStudyService_BatchReview_SessionNotFound(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	sessionID := uuid.New()
	sessionRepo.On("GetByID", ctx, sessionID).Return(nil, domain.ErrNotFound)

	_, err := svc.BatchReview(ctx, uuid.New(), sessionID, BatchReviewRequest{})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- SubmitReview: Learning state ---

func TestStudyService_SubmitReview_LearningState(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()
	card := &domain.Card{ID: cardID, DeckID: deckID, State: domain.CardStateLearning}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(card, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	result, err := svc.SubmitReview(ctx, userID, session.ID, SubmitReviewRequest{
		CardID: cardID, Rating: 3, DurationMS: 3000,
		Card: FSRSCardState{Due: now.Add(24 * time.Hour), Stability: 2.0, Difficulty: 5.0, State: 1, LastReview: now},
		Log:  FSRSLogState{ScheduledDays: 1, ElapsedDays: 0, Stability: 2.0, Difficulty: 5.0},
	})

	require.NoError(t, err)
	assert.Equal(t, domain.CardStateLearning, result.ReviewLog.State)
	assert.Equal(t, 1, session.ReviewCount) // learning increments reviewInc
}

// --- SubmitReview: UoW error ---

func TestStudyService_SubmitReview_UpdateFSRSError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()
	card := &domain.Card{ID: cardID, DeckID: deckID, State: domain.CardStateNew}

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(card, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(assert.AnError)

	now := time.Now()
	_, err := svc.SubmitReview(ctx, userID, session.ID, SubmitReviewRequest{
		CardID: cardID, Rating: 3, DurationMS: 3000,
		Card: FSRSCardState{Due: now.Add(24 * time.Hour), Stability: 2.0, Difficulty: 5.0, State: 1, LastReview: now},
		Log:  FSRSLogState{ScheduledDays: 1, ElapsedDays: 0, Stability: 2.0, Difficulty: 5.0},
	})

	assert.ErrorIs(t, err, assert.AnError)
}

// --- BatchReview: UpdateFSRS error ---

func TestStudyService_BatchReview_UpdateFSRSError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, State: domain.CardStateNew}, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(assert.AnError)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	validCard := FSRSCardState{Due: now.Add(24 * time.Hour), Stability: 2.0, Difficulty: 5.0, State: 1, LastReview: now}

	result, err := svc.BatchReview(ctx, userID, session.ID, BatchReviewRequest{
		Reviews: []BatchReviewItem{
			{CardID: cardID, Rating: 3, DurationMS: 3000, ReviewedAt: now, Card: validCard,
				Log: FSRSLogState{ScheduledDays: 1, ElapsedDays: 0, Stability: 2.0, Difficulty: 5.0}},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
	assert.Equal(t, 1, result.Errors)
}

// --- BatchReview: reviewRepo.Create error ---

func TestStudyService_BatchReview_ReviewCreateError(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, State: domain.CardStateRelearning}, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(assert.AnError)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	validCard := FSRSCardState{Due: now.Add(24 * time.Hour), Stability: 2.0, Difficulty: 5.0, State: 1, LastReview: now}

	result, err := svc.BatchReview(ctx, userID, session.ID, BatchReviewRequest{
		Reviews: []BatchReviewItem{
			{CardID: cardID, Rating: 3, DurationMS: 3000, ReviewedAt: now, Card: validCard,
				Log: FSRSLogState{ScheduledDays: 1, ElapsedDays: 0, Stability: 2.0, Difficulty: 5.0}},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
	assert.Equal(t, 1, result.Errors)
}

// --- BatchReview: Relearning state counter ---

func TestStudyService_BatchReview_RelearningState(t *testing.T) {
	cardRepo := mockdomain.NewMockCardRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	sessionRepo := mockdomain.NewMockStudySessionRepository(t)
	userRepo := mockdomain.NewMockUserRepository(t)
	statsRepo := mockdomain.NewMockStatsRepository(t)
	svc := NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, noopUoW{})
	ctx := context.Background()

	userID := uuid.New()
	deckID := uuid.New()
	session := studyTestSession(userID, &deckID)
	cardID := uuid.New()

	sessionRepo.On("GetByID", ctx, session.ID).Return(session, nil)
	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, State: domain.CardStateRelearning}, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)
	sessionRepo.On("Update", ctx, mock.AnythingOfType("*domain.StudySession")).Return(nil)

	now := time.Now()
	validCard := FSRSCardState{Due: now.Add(24 * time.Hour), Stability: 2.0, Difficulty: 5.0, State: 1, LastReview: now}

	result, err := svc.BatchReview(ctx, userID, session.ID, BatchReviewRequest{
		Reviews: []BatchReviewItem{
			{CardID: cardID, Rating: 3, DurationMS: 3000, ReviewedAt: now, Card: validCard,
				Log: FSRSLogState{ScheduledDays: 1, ElapsedDays: 0, Stability: 2.0, Difficulty: 5.0}},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 1, session.RelearnCount)
}
