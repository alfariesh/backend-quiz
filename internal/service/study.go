package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/rekanesiads/backend-quiz/config"
	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type StudyService struct {
	cardRepo    domain.CardRepository
	reviewRepo  domain.ReviewRepository
	sessionRepo domain.StudySessionRepository
	userRepo    domain.UserRepository
	fsrsConfig  config.FSRSConfig
}

func NewStudyService(
	cardRepo domain.CardRepository,
	reviewRepo domain.ReviewRepository,
	sessionRepo domain.StudySessionRepository,
	userRepo domain.UserRepository,
	fsrsConfig config.FSRSConfig,
) *StudyService {
	return &StudyService{
		cardRepo:    cardRepo,
		reviewRepo:  reviewRepo,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
		fsrsConfig:  fsrsConfig,
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

type SubmitReviewRequest struct {
	CardID     uuid.UUID `json:"card_id" validate:"required"`
	Rating     int       `json:"rating" validate:"required,min=1,max=4"`
	DurationMS int       `json:"duration_ms" validate:"min=0"`
}

type ReviewResult struct {
	Card          domain.Card `json:"card"`
	ReviewLog     domain.ReviewLog `json:"review_log"`
	Retrievability float64    `json:"retrievability"`
	NextDue       time.Time   `json:"next_due"`
}

type PreviewResult struct {
	Again PreviewInfo `json:"again"`
	Hard  PreviewInfo `json:"hard"`
	Good  PreviewInfo `json:"good"`
	Easy  PreviewInfo `json:"easy"`
}

type PreviewInfo struct {
	Due           time.Time `json:"due"`
	Stability     float64   `json:"stability"`
	Difficulty    float64   `json:"difficulty"`
	ScheduledDays int       `json:"scheduled_days"`
	State         string    `json:"state"`
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

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build FSRS instance with user parameters
	f := s.buildFSRS(user)
	now := time.Now()

	// Convert domain card to FSRS card
	fsrsCard := toFSRSCard(card)
	rating := fsrs.Rating(req.Rating)

	// Get scheduling result
	schedulingInfo := f.Next(fsrsCard, now, rating)

	// Store state before review for the log
	stateBefore := card.State

	// Update card with new FSRS state
	fromFSRSCard(&schedulingInfo.Card, card)
	if err := s.cardRepo.UpdateFSRS(ctx, card); err != nil {
		return nil, err
	}

	// Create review log
	reviewLog := &domain.ReviewLog{
		CardID:        card.ID,
		UserID:        userID,
		Rating:        domain.Rating(req.Rating),
		State:         stateBefore,
		ScheduledDays: int(schedulingInfo.ReviewLog.ScheduledDays),
		ElapsedDays:   int(schedulingInfo.ReviewLog.ElapsedDays),
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		DurationMS:    req.DurationMS,
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

	retrievability := f.GetRetrievability(schedulingInfo.Card, now)

	return &ReviewResult{
		Card:           *card,
		ReviewLog:      *reviewLog,
		Retrievability: retrievability,
		NextDue:        card.Due,
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

func (s *StudyService) Preview(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) (*PreviewResult, *DueCounts, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	cards, err := s.cardRepo.GetDueCards(ctx, deckID, now, user.DailyNewLimit, user.DailyReviewLimit)
	if err != nil {
		return nil, nil, err
	}

	counts := s.countCards(cards)

	if len(cards) == 0 {
		return nil, &counts, nil
	}

	// Preview the first due card
	f := s.buildFSRS(user)
	fsrsCard := toFSRSCard(&cards[0])
	recordLog := f.Repeat(fsrsCard, now)

	preview := &PreviewResult{
		Again: toPreviewInfo(recordLog[fsrs.Again]),
		Hard:  toPreviewInfo(recordLog[fsrs.Hard]),
		Good:  toPreviewInfo(recordLog[fsrs.Good]),
		Easy:  toPreviewInfo(recordLog[fsrs.Easy]),
	}

	return preview, &counts, nil
}

func (s *StudyService) buildFSRS(user *domain.User) *fsrs.FSRS {
	params := fsrs.DefaultParam()
	params.RequestRetention = user.DesiredRetention
	params.MaximumInterval = s.fsrsConfig.MaxInterval
	params.EnableFuzz = s.fsrsConfig.EnableFuzz

	if len(user.FSRSWeights) == len(params.W) {
		copy(params.W[:], user.FSRSWeights)
	}

	return fsrs.NewFSRS(params)
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

func toFSRSCard(c *domain.Card) fsrs.Card {
	fc := fsrs.Card{
		Due:           c.Due,
		Stability:     c.Stability,
		Difficulty:    c.Difficulty,
		ElapsedDays:   uint64(c.ElapsedDays),
		ScheduledDays: uint64(c.ScheduledDays),
		Reps:          uint64(c.Reps),
		Lapses:        uint64(c.Lapses),
		State:         fsrs.State(c.State),
	}
	if c.LastReview != nil {
		fc.LastReview = *c.LastReview
	}
	return fc
}

func fromFSRSCard(fc *fsrs.Card, c *domain.Card) {
	c.Due = fc.Due
	c.Stability = fc.Stability
	c.Difficulty = fc.Difficulty
	c.ElapsedDays = int(fc.ElapsedDays)
	c.ScheduledDays = int(fc.ScheduledDays)
	c.Reps = int(fc.Reps)
	c.Lapses = int(fc.Lapses)
	c.State = domain.CardState(fc.State)
	if !fc.LastReview.IsZero() {
		c.LastReview = &fc.LastReview
	}
}

func toPreviewInfo(info fsrs.SchedulingInfo) PreviewInfo {
	return PreviewInfo{
		Due:           info.Card.Due,
		Stability:     info.Card.Stability,
		Difficulty:    info.Card.Difficulty,
		ScheduledDays: int(info.Card.ScheduledDays),
		State:         domain.CardState(info.Card.State).String(),
	}
}
