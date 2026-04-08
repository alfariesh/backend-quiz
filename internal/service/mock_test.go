package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

// --- UserRepository ---

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	// Simulate DB assigning ID
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	return m.Called(ctx, user).Error(0)
}
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockUserRepo) CreateOAuthAccount(ctx context.Context, account *domain.OAuthAccount) error {
	return m.Called(ctx, account).Error(0)
}
func (m *mockUserRepo) GetOAuthAccount(ctx context.Context, provider, providerID string) (*domain.OAuthAccount, error) {
	args := m.Called(ctx, provider, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OAuthAccount), args.Error(1)
}

// --- DeckRepository ---

type mockDeckRepo struct{ mock.Mock }

func (m *mockDeckRepo) Create(ctx context.Context, deck *domain.Deck) error {
	args := m.Called(ctx, deck)
	if deck.ID == uuid.Nil {
		deck.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockDeckRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deck, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deck), args.Error(1)
}
func (m *mockDeckRepo) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.DeckWithCounts, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]domain.DeckWithCounts), args.Int(1), args.Error(2)
}
func (m *mockDeckRepo) Update(ctx context.Context, deck *domain.Deck) error {
	return m.Called(ctx, deck).Error(0)
}
func (m *mockDeckRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockDeckRepo) CreateShare(ctx context.Context, share *domain.DeckShare) error {
	args := m.Called(ctx, share)
	if share.ID == uuid.Nil {
		share.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockDeckRepo) GetShareByDeckID(ctx context.Context, deckID uuid.UUID) (*domain.DeckShare, error) {
	args := m.Called(ctx, deckID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DeckShare), args.Error(1)
}
func (m *mockDeckRepo) GetShareByCode(ctx context.Context, code string) (*domain.DeckShare, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DeckShare), args.Error(1)
}
func (m *mockDeckRepo) DeleteShare(ctx context.Context, deckID uuid.UUID) error {
	return m.Called(ctx, deckID).Error(0)
}
func (m *mockDeckRepo) ListPublicDecks(ctx context.Context, search string, limit, offset int) ([]domain.DeckWithCounts, int, error) {
	args := m.Called(ctx, search, limit, offset)
	return args.Get(0).([]domain.DeckWithCounts), args.Int(1), args.Error(2)
}
func (m *mockDeckRepo) CloneDeck(ctx context.Context, sourceDeckID, targetUserID uuid.UUID, name string) (*domain.Deck, error) {
	args := m.Called(ctx, sourceDeckID, targetUserID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deck), args.Error(1)
}

// --- CardRepository ---

type mockCardRepo struct{ mock.Mock }

func (m *mockCardRepo) Create(ctx context.Context, card *domain.Card) error {
	args := m.Called(ctx, card)
	if card.ID == uuid.Nil {
		card.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockCardRepo) BulkCreate(ctx context.Context, cards []*domain.Card) error {
	args := m.Called(ctx, cards)
	for _, c := range cards {
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
	}
	return args.Error(0)
}
func (m *mockCardRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Card), args.Error(1)
}
func (m *mockCardRepo) ListByDeckID(ctx context.Context, deckID uuid.UUID, filter domain.CardFilter, limit, offset int) ([]domain.Card, int, error) {
	args := m.Called(ctx, deckID, filter, limit, offset)
	return args.Get(0).([]domain.Card), args.Int(1), args.Error(2)
}
func (m *mockCardRepo) Update(ctx context.Context, card *domain.Card) error {
	return m.Called(ctx, card).Error(0)
}
func (m *mockCardRepo) UpdateFSRS(ctx context.Context, card *domain.Card) error {
	return m.Called(ctx, card).Error(0)
}
func (m *mockCardRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockCardRepo) SetSuspended(ctx context.Context, id uuid.UUID, suspended bool) error {
	return m.Called(ctx, id, suspended).Error(0)
}
func (m *mockCardRepo) ResetFSRS(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockCardRepo) GetDueCards(ctx context.Context, deckID uuid.UUID, now time.Time, newLimit, reviewLimit int) ([]domain.Card, error) {
	args := m.Called(ctx, deckID, now, newLimit, reviewLimit)
	return args.Get(0).([]domain.Card), args.Error(1)
}
func (m *mockCardRepo) CountByState(ctx context.Context, deckID uuid.UUID) (map[domain.CardState]int, error) {
	args := m.Called(ctx, deckID)
	return args.Get(0).(map[domain.CardState]int), args.Error(1)
}
func (m *mockCardRepo) CountDue(ctx context.Context, deckID uuid.UUID, now time.Time) (int, error) {
	args := m.Called(ctx, deckID, now)
	return args.Int(0), args.Error(1)
}
func (m *mockCardRepo) GetDeckMasteryStats(ctx context.Context, deckID uuid.UUID) (int, int, float64, error) {
	args := m.Called(ctx, deckID)
	return args.Int(0), args.Int(1), args.Get(2).(float64), args.Error(3)
}
func (m *mockCardRepo) GetWeakCards(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Card, error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]domain.Card), args.Error(1)
}
func (m *mockCardRepo) GetUpcomingDueSummary(ctx context.Context, userID uuid.UUID, now time.Time, horizon time.Time) ([]domain.DeckDueSummary, error) {
	args := m.Called(ctx, userID, now, horizon)
	return args.Get(0).([]domain.DeckDueSummary), args.Error(1)
}
func (m *mockCardRepo) GetNextDueAt(ctx context.Context, userID uuid.UUID, now time.Time) (*time.Time, error) {
	args := m.Called(ctx, userID, now)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

// --- ReviewRepository ---

type mockReviewRepo struct{ mock.Mock }

func (m *mockReviewRepo) Create(ctx context.Context, log *domain.ReviewLog) error {
	args := m.Called(ctx, log)
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockReviewRepo) ListByCardID(ctx context.Context, cardID uuid.UUID) ([]domain.ReviewLog, error) {
	args := m.Called(ctx, cardID)
	return args.Get(0).([]domain.ReviewLog), args.Error(1)
}
func (m *mockReviewRepo) ListByUserID(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int) ([]domain.ReviewLog, int, error) {
	args := m.Called(ctx, userID, from, to, limit, offset)
	return args.Get(0).([]domain.ReviewLog), args.Int(1), args.Error(2)
}
func (m *mockReviewRepo) CountByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}
func (m *mockReviewRepo) GetReviewCountsPerDay(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.DailyReviewCount, error) {
	args := m.Called(ctx, userID, from, to)
	return args.Get(0).([]domain.DailyReviewCount), args.Error(1)
}

// --- StudySessionRepository ---

type mockSessionRepo struct{ mock.Mock }

func (m *mockSessionRepo) Create(ctx context.Context, session *domain.StudySession) error {
	args := m.Called(ctx, session)
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.StudySession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StudySession), args.Error(1)
}
func (m *mockSessionRepo) Update(ctx context.Context, session *domain.StudySession) error {
	return m.Called(ctx, session).Error(0)
}
func (m *mockSessionRepo) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.StudySession, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]domain.StudySession), args.Int(1), args.Error(2)
}

// --- StatsRepository ---

type mockStatsRepo struct{ mock.Mock }

func (m *mockStatsRepo) UpsertDailyStats(ctx context.Context, stats *domain.DailyStats) error {
	return m.Called(ctx, stats).Error(0)
}
func (m *mockStatsRepo) GetDailyStats(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.DailyStats, error) {
	args := m.Called(ctx, userID, from, to)
	return args.Get(0).([]domain.DailyStats), args.Error(1)
}
func (m *mockStatsRepo) GetStreak(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}
func (m *mockStatsRepo) GetGlobalLeaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]domain.LeaderboardEntry), args.Error(1)
}

// --- QuizRepository ---

type mockQuizRepo struct{ mock.Mock }

func (m *mockQuizRepo) Create(ctx context.Context, quiz *domain.Quiz) error {
	args := m.Called(ctx, quiz)
	if quiz.ID == uuid.Nil {
		quiz.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockQuizRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Quiz, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Quiz), args.Error(1)
}
func (m *mockQuizRepo) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizWithCounts, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]domain.QuizWithCounts), args.Int(1), args.Error(2)
}
func (m *mockQuizRepo) Update(ctx context.Context, quiz *domain.Quiz) error {
	return m.Called(ctx, quiz).Error(0)
}
func (m *mockQuizRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockQuizRepo) CreateQuestion(ctx context.Context, q *domain.QuizQuestion) error {
	args := m.Called(ctx, q)
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockQuizRepo) BulkCreateQuestions(ctx context.Context, questions []*domain.QuizQuestion) error {
	args := m.Called(ctx, questions)
	for _, q := range questions {
		if q.ID == uuid.Nil {
			q.ID = uuid.New()
		}
	}
	return args.Error(0)
}
func (m *mockQuizRepo) GetQuestionByID(ctx context.Context, id uuid.UUID) (*domain.QuizQuestion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.QuizQuestion), args.Error(1)
}
func (m *mockQuizRepo) ListQuestionsByQuizID(ctx context.Context, quizID uuid.UUID) ([]domain.QuizQuestion, error) {
	args := m.Called(ctx, quizID)
	return args.Get(0).([]domain.QuizQuestion), args.Error(1)
}
func (m *mockQuizRepo) UpdateQuestion(ctx context.Context, q *domain.QuizQuestion) error {
	return m.Called(ctx, q).Error(0)
}
func (m *mockQuizRepo) DeleteQuestion(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockQuizRepo) CountQuestionsByQuizID(ctx context.Context, quizID uuid.UUID) (int, error) {
	args := m.Called(ctx, quizID)
	return args.Int(0), args.Error(1)
}
func (m *mockQuizRepo) ListByDeckAndType(ctx context.Context, deckID uuid.UUID, quizType string) ([]domain.Quiz, error) {
	args := m.Called(ctx, deckID, quizType)
	return args.Get(0).([]domain.Quiz), args.Error(1)
}

// --- QuizAttemptRepository ---

type mockQuizAttemptRepo struct{ mock.Mock }

func (m *mockQuizAttemptRepo) Create(ctx context.Context, attempt *domain.QuizAttempt) error {
	args := m.Called(ctx, attempt)
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockQuizAttemptRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizAttempt, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.QuizAttempt), args.Error(1)
}
func (m *mockQuizAttemptRepo) Update(ctx context.Context, attempt *domain.QuizAttempt) error {
	return m.Called(ctx, attempt).Error(0)
}
func (m *mockQuizAttemptRepo) ListByQuizID(ctx context.Context, quizID uuid.UUID, limit, offset int) ([]domain.QuizAttempt, int, error) {
	args := m.Called(ctx, quizID, limit, offset)
	return args.Get(0).([]domain.QuizAttempt), args.Int(1), args.Error(2)
}
func (m *mockQuizAttemptRepo) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizAttempt, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]domain.QuizAttempt), args.Int(1), args.Error(2)
}
func (m *mockQuizAttemptRepo) CreateAnswer(ctx context.Context, answer *domain.QuizAnswer) error {
	args := m.Called(ctx, answer)
	if answer.ID == uuid.Nil {
		answer.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockQuizAttemptRepo) ListAnswersByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]domain.QuizAnswer, error) {
	args := m.Called(ctx, attemptID)
	return args.Get(0).([]domain.QuizAnswer), args.Error(1)
}
func (m *mockQuizAttemptRepo) GetAnswerByAttemptAndQuestion(ctx context.Context, attemptID, questionID uuid.UUID) (*domain.QuizAnswer, error) {
	args := m.Called(ctx, attemptID, questionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.QuizAnswer), args.Error(1)
}
func (m *mockQuizAttemptRepo) GetBestAttemptByQuizID(ctx context.Context, quizID uuid.UUID) (*domain.QuizAttempt, error) {
	args := m.Called(ctx, quizID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.QuizAttempt), args.Error(1)
}

// --- GoalRepository ---

type mockGoalRepo struct{ mock.Mock }

func (m *mockGoalRepo) Upsert(ctx context.Context, goal *domain.StudyGoal) error {
	args := m.Called(ctx, goal)
	if goal.ID == uuid.Nil {
		goal.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockGoalRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.StudyGoal, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StudyGoal), args.Error(1)
}
func (m *mockGoalRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.StudyGoal, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.StudyGoal), args.Error(1)
}
func (m *mockGoalRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- MediaRepository ---

type mockMediaRepo struct{ mock.Mock }

func (m *mockMediaRepo) Create(ctx context.Context, media *domain.Media) error {
	args := m.Called(ctx, media)
	if media.ID == uuid.Nil {
		media.ID = uuid.New()
	}
	return args.Error(0)
}
func (m *mockMediaRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Media, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}
func (m *mockMediaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockMediaRepo) ListByCardID(ctx context.Context, cardID uuid.UUID) ([]domain.Media, error) {
	args := m.Called(ctx, cardID)
	return args.Get(0).([]domain.Media), args.Error(1)
}
