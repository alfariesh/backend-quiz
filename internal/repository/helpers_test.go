package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

func createTestDeck(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, name string) *domain.Deck {
	t.Helper()
	repo := NewDeckRepository(pool)
	deck := &domain.Deck{UserID: userID, Name: name}
	require.NoError(t, repo.Create(context.Background(), deck))
	return deck
}

func createTestCard(t *testing.T, pool *pgxpool.Pool, deckID uuid.UUID, front, back string) *domain.Card {
	t.Helper()
	repo := NewCardRepository(pool)
	card := &domain.Card{DeckID: deckID, Front: front, Back: back, Tags: []string{}}
	require.NoError(t, repo.Create(context.Background(), card))
	return card
}

func createTestQuiz(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, title, quizType string) *domain.Quiz {
	t.Helper()
	repo := NewQuizRepository(pool)
	quiz := &domain.Quiz{UserID: userID, Title: title, QuizType: quizType}
	require.NoError(t, repo.Create(context.Background(), quiz))
	return quiz
}

func createTestQuestion(t *testing.T, pool *pgxpool.Pool, quizID uuid.UUID, text, answer string, pos int) *domain.QuizQuestion {
	t.Helper()
	repo := NewQuizRepository(pool)
	q := &domain.QuizQuestion{
		QuizID:        quizID,
		QuestionType:  domain.QuestionTypeMCQ,
		QuestionText:  text,
		Options:       json.RawMessage(`[{"text":"A","is_correct":true},{"text":"B","is_correct":false}]`),
		CorrectAnswer: answer,
		Position:      pos,
		Points:        10,
	}
	require.NoError(t, repo.CreateQuestion(context.Background(), q))
	return q
}

func createTestAttempt(t *testing.T, pool *pgxpool.Pool, quizID, userID uuid.UUID) *domain.QuizAttempt {
	t.Helper()
	repo := NewQuizAttemptRepository(pool)
	attempt := &domain.QuizAttempt{
		QuizID:         quizID,
		UserID:         userID,
		TotalPoints:    100,
		TotalQuestions: 10,
	}
	require.NoError(t, repo.Create(context.Background(), attempt))
	return attempt
}

func createTestReview(t *testing.T, pool *pgxpool.Pool, cardID, userID uuid.UUID) *domain.ReviewLog {
	t.Helper()
	repo := NewReviewRepository(pool)
	log := &domain.ReviewLog{
		CardID:     cardID,
		UserID:     userID,
		Rating:     domain.RatingGood,
		State:      domain.CardStateNew,
		Stability:  1.0,
		Difficulty: 5.0,
		DurationMS: 3000,
		Source:     domain.ReviewSourceFlashcard,
		ReviewedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(context.Background(), log))
	return log
}
