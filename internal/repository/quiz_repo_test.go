package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

func TestQuizRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quiz@example.com")

	timeLimit := 300
	quiz := &domain.Quiz{
		UserID:           user.ID,
		Title:            "Fiqh Quiz",
		Description:      "Test your knowledge",
		QuizType:         domain.QuizTypeMCQ,
		TimeLimitSeconds: &timeLimit,
		ShuffleQuestions: true,
	}
	err := quizRepo.Create(ctx, quiz)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, quiz.ID)

	got, err := quizRepo.GetByID(ctx, quiz.ID)
	require.NoError(t, err)
	assert.Equal(t, "Fiqh Quiz", got.Title)
	assert.Equal(t, domain.QuizTypeMCQ, got.QuizType)
	assert.Equal(t, 300, *got.TimeLimitSeconds)
	assert.True(t, got.ShuffleQuestions)
}

func TestQuizRepo_GetByID_NotFound(t *testing.T) {
	pool := testPool(t)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	_, err := quizRepo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestQuizRepo_ListByUserID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizlist@example.com")

	for _, title := range []string{"Quiz A", "Quiz B"} {
		createTestQuiz(t, pool, user.ID, title, domain.QuizTypeMCQ)
	}

	quizzes, total, err := quizRepo.ListByUserID(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, quizzes, 2)
}

func TestQuizRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizupd@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "Old Title", domain.QuizTypeMCQ)

	quiz.Title = "New Title"
	quiz.IsPublished = true
	err := quizRepo.Update(ctx, quiz)
	require.NoError(t, err)

	got, err := quizRepo.GetByID(ctx, quiz.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Title", got.Title)
	assert.True(t, got.IsPublished)
}

func TestQuizRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizdel@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "Delete Me", domain.QuizTypeMCQ)

	err := quizRepo.Delete(ctx, quiz.ID)
	require.NoError(t, err)

	_, err = quizRepo.GetByID(ctx, quiz.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Question tests

func TestQuizRepo_CreateQuestion(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizq@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "Question Quiz", domain.QuizTypeMCQ)

	q := &domain.QuizQuestion{
		QuizID:        quiz.ID,
		QuestionType:  domain.QuestionTypeMCQ,
		QuestionText:  "What is wudu?",
		Options:       json.RawMessage(`[{"text":"Ablution","is_correct":true},{"text":"Fasting","is_correct":false}]`),
		CorrectAnswer: "Ablution",
		Explanation:   "Wudu is ritual ablution",
		Position:      1,
		Points:        10,
	}
	err := quizRepo.CreateQuestion(ctx, q)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, q.ID)

	got, err := quizRepo.GetQuestionByID(ctx, q.ID)
	require.NoError(t, err)
	assert.Equal(t, "What is wudu?", got.QuestionText)
	assert.Equal(t, "Ablution", got.CorrectAnswer)
	assert.Equal(t, 10, got.Points)
}

func TestQuizRepo_GetQuestionByID_NotFound(t *testing.T) {
	pool := testPool(t)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	_, err := quizRepo.GetQuestionByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestQuizRepo_BulkCreateQuestions(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizbulkq@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "Bulk Q Quiz", domain.QuizTypeMCQ)

	questions := []*domain.QuizQuestion{
		{QuizID: quiz.ID, QuestionType: domain.QuestionTypeMCQ, QuestionText: "Q1", Options: json.RawMessage(`[]`), CorrectAnswer: "A1", Position: 1, Points: 10},
		{QuizID: quiz.ID, QuestionType: domain.QuestionTypeMCQ, QuestionText: "Q2", Options: json.RawMessage(`[]`), CorrectAnswer: "A2", Position: 2, Points: 10},
	}
	err := quizRepo.BulkCreateQuestions(ctx, questions)
	require.NoError(t, err)

	for _, q := range questions {
		assert.NotEqual(t, uuid.Nil, q.ID)
	}

	count, err := quizRepo.CountQuestionsByQuizID(ctx, quiz.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestQuizRepo_ListQuestionsByQuizID(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizlistq@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "List Q Quiz", domain.QuizTypeMCQ)

	createTestQuestion(t, pool, quiz.ID, "Q1", "A1", 1)
	createTestQuestion(t, pool, quiz.ID, "Q2", "A2", 2)

	questions, err := quizRepo.ListQuestionsByQuizID(ctx, quiz.ID)
	require.NoError(t, err)
	assert.Len(t, questions, 2)
}

func TestQuizRepo_UpdateQuestion(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizupdq@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "UpdQ Quiz", domain.QuizTypeMCQ)
	q := createTestQuestion(t, pool, quiz.ID, "Old Q", "Old A", 1)

	q.QuestionText = "New Q"
	q.CorrectAnswer = "New A"
	q.Points = 20
	err := quizRepo.UpdateQuestion(ctx, q)
	require.NoError(t, err)

	got, err := quizRepo.GetQuestionByID(ctx, q.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Q", got.QuestionText)
	assert.Equal(t, "New A", got.CorrectAnswer)
	assert.Equal(t, 20, got.Points)
}

func TestQuizRepo_DeleteQuestion(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizdelq@example.com")
	quiz := createTestQuiz(t, pool, user.ID, "DelQ Quiz", domain.QuizTypeMCQ)
	q := createTestQuestion(t, pool, quiz.ID, "Delete Me", "A", 1)

	err := quizRepo.DeleteQuestion(ctx, q.ID)
	require.NoError(t, err)

	_, err = quizRepo.GetQuestionByID(ctx, q.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestQuizRepo_ListByDeckAndType(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepository(pool)
	quizRepo := NewQuizRepository(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, "quizdecktype@example.com")
	deck := createTestDeck(t, pool, user.ID, "Quiz Deck")

	// Create quiz linked to deck
	quiz := &domain.Quiz{
		UserID:   user.ID,
		DeckID:   &deck.ID,
		Title:    "Deck Quiz",
		QuizType: domain.QuizTypeMCQ,
	}
	require.NoError(t, quizRepo.Create(ctx, quiz))

	quizzes, err := quizRepo.ListByDeckAndType(ctx, deck.ID, domain.QuizTypeMCQ)
	require.NoError(t, err)
	assert.Len(t, quizzes, 1)
	assert.Equal(t, "Deck Quiz", quizzes[0].Title)
}
