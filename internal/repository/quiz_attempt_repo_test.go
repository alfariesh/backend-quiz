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

func TestAttemptRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "attempt@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "Attempt Quiz", domain.QuizTypeMCQ)

	attempt := &domain.QuizAttempt{
		QuizID:         quiz.ID,
		UserID:         user.ID,
		TotalPoints:    100,
		TotalQuestions: 10,
	}
	err := attemptRepo.Create(ctx, attempt)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, attempt.ID)
	assert.False(t, attempt.StartedAt.IsZero())

	got, err := attemptRepo.GetByID(ctx, attempt.ID)
	require.NoError(t, err)
	assert.Equal(t, quiz.ID, got.QuizID)
	assert.Equal(t, 100, got.TotalPoints)
	assert.Equal(t, 10, got.TotalQuestions)
	assert.Nil(t, got.CompletedAt)
}

func TestAttemptRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	_, err := attemptRepo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAttemptRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "attemptupd@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "UpdAttempt Quiz", domain.QuizTypeMCQ)
	attempt := createTestAttempt(t, pool, quiz.ID, user.ID)

	now := time.Now().UTC()
	attempt.CompletedAt = &now
	attempt.Score = 80
	attempt.CorrectCount = 8
	attempt.DurationMS = 120000

	err := attemptRepo.Update(ctx, attempt)
	require.NoError(t, err)

	got, err := attemptRepo.GetByID(ctx, attempt.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.CompletedAt)
	assert.Equal(t, 80, got.Score)
	assert.Equal(t, 8, got.CorrectCount)
	assert.Equal(t, 120000, got.DurationMS)
}

func TestAttemptRepo_ListByQuizID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "attemptlistq@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "ListQ Quiz", domain.QuizTypeMCQ)

	for i := 0; i < 3; i++ {
		createTestAttempt(t, pool, quiz.ID, user.ID)
	}

	attempts, total, err := attemptRepo.ListByQuizID(ctx, quiz.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, attempts, 3)
}

func TestAttemptRepo_ListByUserID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "attemptlistu@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "ListU Quiz", domain.QuizTypeMCQ)

	for i := 0; i < 2; i++ {
		createTestAttempt(t, pool, quiz.ID, user.ID)
	}

	attempts, total, err := attemptRepo.ListByUserID(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, attempts, 2)
}

// Answer tests

func TestAttemptRepo_CreateAnswer(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "answer@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "Answer Quiz", domain.QuizTypeMCQ)
	question := createTestQuestion(t, pool, quiz.ID, "Q1", "A1", 1)
	attempt := createTestAttempt(t, pool, quiz.ID, user.ID)

	answer := &domain.QuizAnswer{
		AttemptID:    attempt.ID,
		QuestionID:   question.ID,
		UserAnswer:   "A1",
		IsCorrect:    true,
		PointsEarned: 10,
		DurationMS:   5000,
	}
	err := attemptRepo.CreateAnswer(ctx, answer)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, answer.ID)
}

func TestAttemptRepo_ListAnswersByAttemptID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "answerlist@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "AnswerList Quiz", domain.QuizTypeMCQ)
	q1 := createTestQuestion(t, pool, quiz.ID, "Q1", "A1", 1)
	q2 := createTestQuestion(t, pool, quiz.ID, "Q2", "A2", 2)
	attempt := createTestAttempt(t, pool, quiz.ID, user.ID)

	for _, q := range []*domain.QuizQuestion{q1, q2} {
		require.NoError(t, attemptRepo.CreateAnswer(ctx, &domain.QuizAnswer{
			AttemptID:    attempt.ID,
			QuestionID:   q.ID,
			UserAnswer:   q.CorrectAnswer,
			IsCorrect:    true,
			PointsEarned: 10,
			DurationMS:   3000,
		}))
	}

	answers, err := attemptRepo.ListAnswersByAttemptID(ctx, attempt.ID)
	require.NoError(t, err)
	assert.Len(t, answers, 2)
}

func TestAttemptRepo_GetAnswerByAttemptAndQuestion(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "answerget@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "AnswerGet Quiz", domain.QuizTypeMCQ)
	question := createTestQuestion(t, pool, quiz.ID, "Q1", "A1", 1)
	attempt := createTestAttempt(t, pool, quiz.ID, user.ID)

	require.NoError(t, attemptRepo.CreateAnswer(ctx, &domain.QuizAnswer{
		AttemptID:    attempt.ID,
		QuestionID:   question.ID,
		UserAnswer:   "A1",
		IsCorrect:    true,
		PointsEarned: 10,
		DurationMS:   3000,
	}))

	got, err := attemptRepo.GetAnswerByAttemptAndQuestion(ctx, attempt.ID, question.ID)
	require.NoError(t, err)
	assert.Equal(t, "A1", got.UserAnswer)
	assert.True(t, got.IsCorrect)
}

func TestAttemptRepo_GetAnswerByAttemptAndQuestion_NotFound(t *testing.T) {
	pool := testPool(t)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	_, err := attemptRepo.GetAnswerByAttemptAndQuestion(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAttemptRepo_GetBestAttemptByQuizID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "bestatt@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "Best Quiz", domain.QuizTypeMCQ)

	// Create two attempts with different scores
	a1 := createTestAttempt(t, pool, quiz.ID, user.ID)
	now := time.Now().UTC()
	a1.CompletedAt = &now
	a1.Score = 60
	require.NoError(t, attemptRepo.Update(ctx, a1))

	a2 := createTestAttempt(t, pool, quiz.ID, user.ID)
	a2.CompletedAt = &now
	a2.Score = 90
	require.NoError(t, attemptRepo.Update(ctx, a2))

	best, err := attemptRepo.GetBestAttemptByQuizID(ctx, quiz.ID)
	require.NoError(t, err)
	assert.Equal(t, 90, best.Score)
}

func TestAttemptRepo_GetBestAttemptByQuizID_NotFound(t *testing.T) {
	pool := testPool(t)
	attemptRepo := NewQuizAttemptRepository(pool)
	ctx := context.Background()

	_, err := attemptRepo.GetBestAttemptByQuizID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
