package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
)

type AuthServicer interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.TokenPair, *domain.User, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenPair, *domain.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenPair, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*domain.User, error)
	FindOrCreateOAuthUser(ctx context.Context, provider, providerID, email, displayName string, avatarURL *string) (*dto.TokenPair, *domain.User, error)
}

type CardServicer interface {
	Create(ctx context.Context, userID, deckID uuid.UUID, req dto.CreateCardRequest) (*domain.Card, error)
	BatchCreate(ctx context.Context, userID, deckID uuid.UUID, req dto.BatchCreateRequest) ([]*domain.Card, error)
	Get(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)
	List(ctx context.Context, userID, deckID uuid.UUID, filter domain.CardFilter, limit, offset int) ([]domain.Card, int, error)
	Update(ctx context.Context, userID, cardID uuid.UUID, req dto.UpdateCardRequest) (*domain.Card, error)
	Delete(ctx context.Context, userID, cardID uuid.UUID) error
	ResetFSRS(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error)
	Suspend(ctx context.Context, userID, cardID uuid.UUID, suspended bool) error
}

type DeckServicer interface {
	Create(ctx context.Context, userID uuid.UUID, req dto.CreateDeckRequest) (*domain.Deck, error)
	Get(ctx context.Context, userID, deckID uuid.UUID) (*domain.Deck, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.DeckWithCounts, int, error)
	Update(ctx context.Context, userID, deckID uuid.UUID, req dto.UpdateDeckRequest) (*domain.Deck, error)
	Delete(ctx context.Context, userID, deckID uuid.UUID) error
	Share(ctx context.Context, userID, deckID uuid.UUID, req dto.ShareDeckRequest) (*domain.DeckShare, error)
	Unshare(ctx context.Context, userID, deckID uuid.UUID) error
	Clone(ctx context.Context, userID uuid.UUID, shareCode string) (*domain.Deck, error)
	ListPublic(ctx context.Context, search string, limit, offset int) ([]domain.DeckWithCounts, int, error)
	Export(ctx context.Context, userID, deckID uuid.UUID) ([]byte, error)
	Import(ctx context.Context, userID uuid.UUID, req dto.ImportDeckRequest) (*domain.Deck, error)
}

type StudyServicer interface {
	StartSession(ctx context.Context, userID uuid.UUID, req dto.StartSessionRequest) (*dto.StartSessionResponse, error)
	GetSession(ctx context.Context, userID, sessionID uuid.UUID) (*domain.StudySession, error)
	SubmitReview(ctx context.Context, userID, sessionID uuid.UUID, req dto.SubmitReviewRequest) (*dto.ReviewResult, error)
	BatchReview(ctx context.Context, userID, sessionID uuid.UUID, req dto.BatchReviewRequest) (*dto.BatchReviewResult, error)
	EndSession(ctx context.Context, userID, sessionID uuid.UUID) (*domain.StudySession, error)
	GetReminders(ctx context.Context, userID uuid.UUID, hoursAhead int) (*dto.ReminderResponse, error)
}

type StatsServicer interface {
	Overview(ctx context.Context, userID uuid.UUID) (*dto.OverviewStats, error)
	Heatmap(ctx context.Context, userID uuid.UUID) ([]dto.HeatmapEntry, error)
	Forecast(ctx context.Context, userID uuid.UUID) ([]dto.ForecastDay, error)
	Leaderboard(ctx context.Context, limit int) ([]domain.LeaderboardEntry, error)
	DeckStats(ctx context.Context, userID, deckID uuid.UUID) (map[string]any, error)
	Mastery(ctx context.Context, userID uuid.UUID) ([]dto.DeckMastery, error)
	WeakAreas(ctx context.Context, userID uuid.UUID) (*dto.WeakAreasResponse, error)
	TestComparison(ctx context.Context, userID, deckID uuid.UUID) (*dto.TestComparison, error)
}

type QuizServicer interface {
	CreateQuiz(ctx context.Context, userID uuid.UUID, req dto.CreateQuizRequest) (*domain.Quiz, error)
	GetQuiz(ctx context.Context, userID, quizID uuid.UUID) (*dto.QuizDetail, error)
	ListQuizzes(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizWithCounts, int, error)
	UpdateQuiz(ctx context.Context, userID, quizID uuid.UUID, req dto.UpdateQuizRequest) (*domain.Quiz, error)
	DeleteQuiz(ctx context.Context, userID, quizID uuid.UUID) error
	AddQuestion(ctx context.Context, userID, quizID uuid.UUID, req dto.AddQuestionRequest) (*domain.QuizQuestion, error)
	BatchAddQuestions(ctx context.Context, userID, quizID uuid.UUID, req dto.BatchAddQuestionsRequest) ([]*domain.QuizQuestion, error)
	UpdateQuestion(ctx context.Context, userID, quizID, questionID uuid.UUID, req dto.UpdateQuestionRequest) (*domain.QuizQuestion, error)
	DeleteQuestion(ctx context.Context, userID, quizID, questionID uuid.UUID) error
	GenerateFromDeck(ctx context.Context, userID, quizID uuid.UUID, req dto.GenerateFromDeckRequest) ([]*domain.QuizQuestion, error)
	GenerateAyatQuiz(ctx context.Context, userID, quizID uuid.UUID, req dto.GenerateAyatQuizRequest) ([]*domain.QuizQuestion, error)
	StartAttempt(ctx context.Context, userID, quizID uuid.UUID) (*dto.StartAttemptResponse, error)
	SubmitAnswer(ctx context.Context, userID, attemptID uuid.UUID, req dto.SubmitAnswerRequest) (*dto.AnswerResult, error)
	CompleteAttempt(ctx context.Context, userID, attemptID uuid.UUID) (*domain.QuizAttempt, error)
	GetAttempt(ctx context.Context, userID, attemptID uuid.UUID) (*dto.AttemptDetail, error)
	ListAttempts(ctx context.Context, userID, quizID uuid.UUID, limit, offset int) ([]domain.QuizAttempt, int, error)
}

type MediaServicer interface {
	Upload(ctx context.Context, userID, cardID uuid.UUID, req dto.UploadMediaRequest) (*domain.Media, error)
	Delete(ctx context.Context, userID, mediaID uuid.UUID) error
	ListByCard(ctx context.Context, userID, cardID uuid.UUID) ([]domain.Media, error)
}

type GoalServicer interface {
	SetGoal(ctx context.Context, userID uuid.UUID, req dto.SetGoalRequest) (*domain.StudyGoal, error)
	ListWithProgress(ctx context.Context, userID uuid.UUID) ([]dto.GoalProgress, error)
	DeleteGoal(ctx context.Context, userID, goalID uuid.UUID) error
}
