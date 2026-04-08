package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type StudyService struct {
	cardRepo    domain.CardRepository
	reviewRepo  domain.ReviewRepository
	sessionRepo domain.StudySessionRepository
	userRepo    domain.UserRepository
}

func NewStudyService(
	cardRepo domain.CardRepository,
	reviewRepo domain.ReviewRepository,
	sessionRepo domain.StudySessionRepository,
	userRepo domain.UserRepository,
) *StudyService {
	return &StudyService{
		cardRepo:    cardRepo,
		reviewRepo:  reviewRepo,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

type StartSessionRequest struct {
	DeckID uuid.UUID `json:"deck_id" validate:"required"`
}

type StartSessionResponse struct {
	Session *domain.StudySession `json:"session"`
	Cards   []domain.Card       `json:"cards"`
	Counts  DueCounts           `json:"counts"`
}

type DueCounts struct {
	New      int `json:"new"`
	Learning int `json:"learning"`
	Review   int `json:"review"`
	Total    int `json:"total"`
}

// FSRSCardState represents the pre-computed FSRS card state from the client (ts-fsrs).
type FSRSCardState struct {
	Due           time.Time `json:"due" validate:"required"`
	Stability     float64   `json:"stability" validate:"min=0"`
	Difficulty    float64   `json:"difficulty" validate:"min=0,max=10"`
	ElapsedDays   int       `json:"elapsed_days" validate:"min=0"`
	ScheduledDays int       `json:"scheduled_days" validate:"min=0"`
	Reps          int       `json:"reps" validate:"min=0"`
	Lapses        int       `json:"lapses" validate:"min=0"`
	State         int       `json:"state" validate:"min=0,max=3"`
	LastReview    time.Time `json:"last_review" validate:"required"`
}

// FSRSLogState represents the pre-computed FSRS review log state from the client.
type FSRSLogState struct {
	ScheduledDays int     `json:"scheduled_days"`
	ElapsedDays   int     `json:"elapsed_days"`
	Stability     float64 `json:"stability"`
	Difficulty    float64 `json:"difficulty"`
}

type SubmitReviewRequest struct {
	CardID     uuid.UUID     `json:"card_id" validate:"required"`
	Rating     int           `json:"rating" validate:"required,min=1,max=4"`
	DurationMS int           `json:"duration_ms" validate:"min=0"`
	Card       FSRSCardState `json:"card" validate:"required"`
	Log        FSRSLogState  `json:"log"`
}

type ReviewResult struct {
	Card      domain.Card      `json:"card"`
	ReviewLog domain.ReviewLog `json:"review_log"`
	NextDue   time.Time        `json:"next_due"`
}

type BatchReviewRequest struct {
	Reviews []BatchReviewItem `json:"reviews" validate:"required,min=1,max=500,dive"`
}

type BatchReviewItem struct {
	CardID     uuid.UUID     `json:"card_id" validate:"required"`
	Rating     int           `json:"rating" validate:"required,min=1,max=4"`
	DurationMS int           `json:"duration_ms" validate:"min=0"`
	ReviewedAt time.Time     `json:"reviewed_at" validate:"required"`
	Card       FSRSCardState `json:"card" validate:"required"`
	Log        FSRSLogState  `json:"log"`
}

type BatchReviewResult struct {
	Processed int `json:"processed"`
	Errors    int `json:"errors"`
}

func (s *StudyService) StartSession(ctx context.Context, userID uuid.UUID, req StartSessionRequest) (*StartSessionResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	cards, err := s.cardRepo.GetDueCards(ctx, req.DeckID, now, user.DailyNewLimit, user.DailyReviewLimit)
	if err != nil {
		return nil, err
	}

	session := &domain.StudySession{
		UserID: userID,
		DeckID: &req.DeckID,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	counts := s.countCards(cards)

	return &StartSessionResponse{
		Session: session,
		Cards:   cards,
		Counts:  counts,
	}, nil
}

func (s *StudyService) GetSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*domain.StudySession, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return session, nil
}

func (s *StudyService) SubmitReview(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, req SubmitReviewRequest) (*ReviewResult, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if session.EndedAt != nil {
		return nil, domain.ErrSessionEnded
	}

	card, err := s.cardRepo.GetByID(ctx, req.CardID)
	if err != nil {
		return nil, err
	}
	if card.IsSuspended {
		return nil, domain.ErrCardSuspended
	}

	if err := validateFSRSState(req.Card); err != nil {
		return nil, err
	}

	// Store state before review for the log
	stateBefore := card.State

	// Apply client-computed FSRS state
	applyFSRSCardState(card, req.Card)
	if err := s.cardRepo.UpdateFSRS(ctx, card); err != nil {
		return nil, err
	}

	now := time.Now()

	// Create review log
	reviewLog := &domain.ReviewLog{
		CardID:        card.ID,
		UserID:        userID,
		Rating:        domain.Rating(req.Rating),
		State:         stateBefore,
		ScheduledDays: req.Log.ScheduledDays,
		ElapsedDays:   req.Log.ElapsedDays,
		Stability:     req.Log.Stability,
		Difficulty:    req.Log.Difficulty,
		DurationMS:    req.DurationMS,
		Source:        domain.ReviewSourceFlashcard,
		ReviewedAt:    now,
	}
	if err := s.reviewRepo.Create(ctx, reviewLog); err != nil {
		return nil, err
	}

	// Update session counters
	newInc, reviewInc, relearnInc := 0, 0, 0
	switch stateBefore {
	case domain.CardStateNew:
		newInc = 1
	case domain.CardStateReview:
		reviewInc = 1
	case domain.CardStateLearning:
		reviewInc = 1
	case domain.CardStateRelearning:
		relearnInc = 1
	}
	session.NewCount += newInc
	session.ReviewCount += reviewInc
	session.RelearnCount += relearnInc
	session.TotalDurationMS += req.DurationMS
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	return &ReviewResult{
		Card:      *card,
		ReviewLog: *reviewLog,
		NextDue:   card.Due,
	}, nil
}

func (s *StudyService) BatchReview(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, req BatchReviewRequest) (*BatchReviewResult, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if session.EndedAt != nil {
		return nil, domain.ErrSessionEnded
	}

	processed, errCount := 0, 0
	newInc, reviewInc, relearnInc, totalDuration := 0, 0, 0, 0

	for _, item := range req.Reviews {
		if err := validateFSRSState(item.Card); err != nil {
			errCount++
			continue
		}

		card, err := s.cardRepo.GetByID(ctx, item.CardID)
		if err != nil || card.IsSuspended {
			errCount++
			continue
		}

		stateBefore := card.State

		applyFSRSCardState(card, item.Card)
		if err := s.cardRepo.UpdateFSRS(ctx, card); err != nil {
			errCount++
			continue
		}

		reviewLog := &domain.ReviewLog{
			CardID:        card.ID,
			UserID:        userID,
			Rating:        domain.Rating(item.Rating),
			State:         stateBefore,
			ScheduledDays: item.Log.ScheduledDays,
			ElapsedDays:   item.Log.ElapsedDays,
			Stability:     item.Log.Stability,
			Difficulty:    item.Log.Difficulty,
			DurationMS:    item.DurationMS,
			Source:        domain.ReviewSourceFlashcard,
			ReviewedAt:    item.ReviewedAt,
		}
		if err := s.reviewRepo.Create(ctx, reviewLog); err != nil {
			errCount++
			continue
		}

		switch stateBefore {
		case domain.CardStateNew:
			newInc++
		case domain.CardStateReview, domain.CardStateLearning:
			reviewInc++
		case domain.CardStateRelearning:
			relearnInc++
		}
		totalDuration += item.DurationMS
		processed++
	}

	session.NewCount += newInc
	session.ReviewCount += reviewInc
	session.RelearnCount += relearnInc
	session.TotalDurationMS += totalDuration
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	return &BatchReviewResult{
		Processed: processed,
		Errors:    errCount,
	}, nil
}

func (s *StudyService) EndSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*domain.StudySession, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if session.EndedAt != nil {
		return nil, domain.ErrSessionEnded
	}

	now := time.Now()
	session.EndedAt = &now
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *StudyService) countCards(cards []domain.Card) DueCounts {
	counts := DueCounts{}
	for _, c := range cards {
		switch c.State {
		case domain.CardStateNew:
			counts.New++
		case domain.CardStateLearning, domain.CardStateRelearning:
			counts.Learning++
		case domain.CardStateReview:
			counts.Review++
		}
	}
	counts.Total = len(cards)
	return counts
}

// applyFSRSCardState writes the client-computed FSRS state onto a domain card.
func applyFSRSCardState(card *domain.Card, state FSRSCardState) {
	card.Due = state.Due
	card.Stability = state.Stability
	card.Difficulty = state.Difficulty
	card.ElapsedDays = state.ElapsedDays
	card.ScheduledDays = state.ScheduledDays
	card.Reps = state.Reps
	card.Lapses = state.Lapses
	card.State = domain.CardState(state.State)
	card.LastReview = &state.LastReview
}

// validateFSRSState performs basic sanity checks on client-provided FSRS state.
func validateFSRSState(card FSRSCardState) error {
	if card.State < 0 || card.State > 3 {
		return fmt.Errorf("%w: invalid card state", domain.ErrInvalidInput)
	}
	if card.Stability < 0 {
		return fmt.Errorf("%w: stability must be >= 0", domain.ErrInvalidInput)
	}
	if card.Difficulty < 0 || card.Difficulty > 10 {
		return fmt.Errorf("%w: difficulty must be 0-10", domain.ErrInvalidInput)
	}
	maxDue := time.Now().AddDate(0, 0, 36500)
	if card.Due.After(maxDue) {
		return fmt.Errorf("%w: due date too far in future", domain.ErrInvalidInput)
	}
	return nil
}
