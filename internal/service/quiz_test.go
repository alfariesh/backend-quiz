package service

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	mockdomain "github.com/rekanesiads/backend-quiz/internal/mocks/domain"
)


// --- gradeAnswer ---

func TestGradeAnswer_MCQ(t *testing.T) {
	svc := &QuizService{}
	q := &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeMCQ,
		CorrectAnswer: "Photosynthesis",
	}

	tests := []struct {
		name    string
		answer  string
		correct bool
	}{
		{"exact match", "Photosynthesis", true},
		{"case insensitive", "photosynthesis", true},
		{"with spaces", "  Photosynthesis  ", true},
		{"wrong answer", "Respiration", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.correct, svc.gradeAnswer(q, tt.answer))
		})
	}
}

func TestGradeAnswer_TrueFalse(t *testing.T) {
	svc := &QuizService{}

	tests := []struct {
		name    string
		correct string
		answer  string
		result  bool
	}{
		{"true correct", "true", "true", true},
		{"true case insensitive", "true", "True", true},
		{"true wrong", "true", "false", false},
		{"false correct", "false", "false", true},
		{"false case insensitive", "false", "FALSE", true},
		{"false wrong", "false", "true", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &domain.QuizQuestion{
				QuestionType:  domain.QuestionTypeTrueFalse,
				CorrectAnswer: tt.correct,
			}
			assert.Equal(t, tt.result, svc.gradeAnswer(q, tt.answer))
		})
	}
}

func TestGradeAnswer_FillBlank(t *testing.T) {
	svc := &QuizService{}
	q := &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeFillBlank,
		CorrectAnswer: "Jakarta",
	}

	tests := []struct {
		name    string
		answer  string
		correct bool
	}{
		{"exact", "Jakarta", true},
		{"case insensitive", "jakarta", true},
		{"with spaces", "  Jakarta  ", true},
		{"wrong", "Bandung", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.correct, svc.gradeAnswer(q, tt.answer))
		})
	}
}

func TestGradeAnswer_FillBlank_WithAlternatives(t *testing.T) {
	svc := &QuizService{}
	alternatives, _ := json.Marshal([]string{"Al-Fatihah", "Al Fatihah", "Alfatihah"})

	q := &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeFillBlank,
		CorrectAnswer: "Al-Faatihah",
		Options:       alternatives,
	}

	tests := []struct {
		name    string
		answer  string
		correct bool
	}{
		{"primary answer", "Al-Faatihah", true},
		{"alternative 1", "Al-Fatihah", true},
		{"alternative 2", "Al Fatihah", true},
		{"alternative 3", "Alfatihah", true},
		{"alternative case insensitive", "al-fatihah", true},
		{"wrong", "Al-Baqarah", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.correct, svc.gradeAnswer(q, tt.answer))
		})
	}
}

func TestGradeAnswer_AyatCloze(t *testing.T) {
	svc := &QuizService{}
	q := &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeAyatCloze,
		CorrectAnswer: "الرحمن",
	}

	assert.True(t, svc.gradeAnswer(q, "الرحمن"))
	assert.False(t, svc.gradeAnswer(q, "الرحيم"))
}

func TestGradeAnswer_AyatContinuation(t *testing.T) {
	svc := &QuizService{}
	q := &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeAyatContinuation,
		CorrectAnswer: "مالك يوم الدين",
	}

	assert.True(t, svc.gradeAnswer(q, "مالك يوم الدين"))
	assert.True(t, svc.gradeAnswer(q, "  مالك يوم الدين  "))
	assert.False(t, svc.gradeAnswer(q, "إياك نعبد"))
}

func TestGradeAnswer_SurahID(t *testing.T) {
	svc := &QuizService{}
	q := &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeSurahID,
		CorrectAnswer: "Al-Baqarah",
	}

	assert.True(t, svc.gradeAnswer(q, "Al-Baqarah"))
	assert.True(t, svc.gradeAnswer(q, "al-baqarah"))
	assert.False(t, svc.gradeAnswer(q, "Al-Imran"))
}

func TestGradeAnswer_Ordering(t *testing.T) {
	svc := &QuizService{}
	q := &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeOrdering,
		CorrectAnswer: "id1,id2,id3",
	}

	assert.True(t, svc.gradeAnswer(q, "id1,id2,id3"))
	assert.False(t, svc.gradeAnswer(q, "id2,id1,id3"))
	assert.False(t, svc.gradeAnswer(q, "ID1,ID2,ID3")) // ordering is exact match
	assert.False(t, svc.gradeAnswer(q, "id1, id2, id3"))
}

func TestGradeAnswer_UnknownType(t *testing.T) {
	svc := &QuizService{}
	q := &domain.QuizQuestion{
		QuestionType:  "unknown_type",
		CorrectAnswer: "answer",
	}

	assert.False(t, svc.gradeAnswer(q, "answer"))
}

// --- validateQuestionOptions ---

func TestValidateQuestionOptions_MCQ_Valid(t *testing.T) {
	svc := &QuizService{}
	opts, _ := json.Marshal([]domain.MCQOption{
		{Text: "A", IsCorrect: false},
		{Text: "B", IsCorrect: true},
		{Text: "C", IsCorrect: false},
		{Text: "D", IsCorrect: false},
	})

	err := svc.validateQuestionOptions(domain.QuestionTypeMCQ, opts, "B")
	assert.NoError(t, err)
}

func TestValidateQuestionOptions_MCQ_NoOptions(t *testing.T) {
	svc := &QuizService{}
	err := svc.validateQuestionOptions(domain.QuestionTypeMCQ, nil, "answer")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Contains(t, err.Error(), "MCQ requires options")
}

func TestValidateQuestionOptions_MCQ_InvalidJSON(t *testing.T) {
	svc := &QuizService{}
	err := svc.validateQuestionOptions(domain.QuestionTypeMCQ, []byte(`not json`), "answer")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestValidateQuestionOptions_MCQ_TooFewOptions(t *testing.T) {
	svc := &QuizService{}
	opts, _ := json.Marshal([]domain.MCQOption{
		{Text: "A", IsCorrect: true},
	})

	err := svc.validateQuestionOptions(domain.QuestionTypeMCQ, opts, "A")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2-6 options")
}

func TestValidateQuestionOptions_MCQ_TooManyOptions(t *testing.T) {
	svc := &QuizService{}
	opts, _ := json.Marshal([]domain.MCQOption{
		{Text: "A", IsCorrect: true},
		{Text: "B", IsCorrect: false},
		{Text: "C", IsCorrect: false},
		{Text: "D", IsCorrect: false},
		{Text: "E", IsCorrect: false},
		{Text: "F", IsCorrect: false},
		{Text: "G", IsCorrect: false},
	})

	err := svc.validateQuestionOptions(domain.QuestionTypeMCQ, opts, "A")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2-6 options")
}

func TestValidateQuestionOptions_MCQ_NoCorrectOption(t *testing.T) {
	svc := &QuizService{}
	opts, _ := json.Marshal([]domain.MCQOption{
		{Text: "A", IsCorrect: false},
		{Text: "B", IsCorrect: false},
	})

	err := svc.validateQuestionOptions(domain.QuestionTypeMCQ, opts, "A")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one correct")
}

func TestValidateQuestionOptions_MCQ_MultipleCorrect(t *testing.T) {
	svc := &QuizService{}
	opts, _ := json.Marshal([]domain.MCQOption{
		{Text: "A", IsCorrect: true},
		{Text: "B", IsCorrect: true},
	})

	err := svc.validateQuestionOptions(domain.QuestionTypeMCQ, opts, "A")
	require.Error(t, err)
}

func TestValidateQuestionOptions_MCQ_CorrectTextMismatch(t *testing.T) {
	svc := &QuizService{}
	opts, _ := json.Marshal([]domain.MCQOption{
		{Text: "Option A", IsCorrect: true},
		{Text: "Option B", IsCorrect: false},
	})

	err := svc.validateQuestionOptions(domain.QuestionTypeMCQ, opts, "different answer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "correct option text must match")
}

func TestValidateQuestionOptions_TrueFalse(t *testing.T) {
	svc := &QuizService{}

	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeTrueFalse, nil, "true"))
	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeTrueFalse, nil, "false"))
	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeTrueFalse, nil, "TRUE"))
	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeTrueFalse, nil, " True "))

	err := svc.validateQuestionOptions(domain.QuestionTypeTrueFalse, nil, "yes")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'true' or 'false'")
}

func TestValidateQuestionOptions_FillBlank_NoAlternatives(t *testing.T) {
	svc := &QuizService{}
	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeFillBlank, nil, "answer"))
}

func TestValidateQuestionOptions_FillBlank_ValidAlternatives(t *testing.T) {
	svc := &QuizService{}
	alts, _ := json.Marshal([]string{"alt1", "alt2"})
	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeFillBlank, alts, "answer"))
}

func TestValidateQuestionOptions_FillBlank_InvalidAlternatives(t *testing.T) {
	svc := &QuizService{}
	err := svc.validateQuestionOptions(domain.QuestionTypeFillBlank, []byte(`{"not":"array"}`), "answer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "array of strings")
}

func TestValidateQuestionOptions_AyatCloze(t *testing.T) {
	svc := &QuizService{}
	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeAyatCloze, nil, "answer"))
}

func TestValidateQuestionOptions_SurahID_Valid(t *testing.T) {
	svc := &QuizService{}
	opts, _ := json.Marshal([]domain.MCQOption{
		{Text: "Al-Fatihah", IsCorrect: true},
		{Text: "Al-Baqarah", IsCorrect: false},
		{Text: "Al-Imran", IsCorrect: false},
	})
	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeSurahID, opts, "Al-Fatihah"))
}

func TestValidateQuestionOptions_SurahID_NoOptions(t *testing.T) {
	svc := &QuizService{}
	err := svc.validateQuestionOptions(domain.QuestionTypeSurahID, nil, "Al-Fatihah")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires options")
}

func TestValidateQuestionOptions_Ordering_Valid(t *testing.T) {
	svc := &QuizService{}
	opts, _ := json.Marshal([]orderingOption{
		{ID: "1", Text: "ayat 1"},
		{ID: "2", Text: "ayat 2"},
	})
	assert.NoError(t, svc.validateQuestionOptions(domain.QuestionTypeOrdering, opts, "1,2"))
}

func TestValidateQuestionOptions_Ordering_NoOptions(t *testing.T) {
	svc := &QuizService{}
	err := svc.validateQuestionOptions(domain.QuestionTypeOrdering, nil, "1,2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ordering requires options")
}

// --- parseCardTags ---

func TestParseCardTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		surat    string
		ayatNum  int
		ok       bool
	}{
		{
			"valid ayat tag",
			[]string{"surat:Al-Fatihah", "ayat:3", "topic:iman"},
			"Al-Fatihah", 3, true,
		},
		{
			"missing surat",
			[]string{"ayat:5"},
			"", 5, false,
		},
		{
			"missing ayat",
			[]string{"surat:Al-Baqarah"},
			"Al-Baqarah", 0, false,
		},
		{
			"empty tags",
			[]string{},
			"", 0, false,
		},
		{
			"nil tags",
			nil,
			"", 0, false,
		},
		{
			"ayat zero",
			[]string{"surat:Al-Fatihah", "ayat:0"},
			"Al-Fatihah", 0, false,
		},
		{
			"non-numeric ayat",
			[]string{"surat:Al-Fatihah", "ayat:abc"},
			"Al-Fatihah", 0, false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			surat, ayatNum, ok := parseCardTags(tt.tags)
			assert.Equal(t, tt.surat, surat)
			assert.Equal(t, tt.ayatNum, ayatNum)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

// --- filterAyatCards ---

func TestFilterAyatCards(t *testing.T) {
	cards := []domain.Card{
		{Front: "ayat 1", Tags: []string{"surat:Al-Fatihah", "ayat:1"}},
		{Front: "regular card", Tags: []string{"topic:fiqh"}},
		{Front: "ayat 2", Tags: []string{"surat:Al-Fatihah", "ayat:2"}},
		{Front: "no tags"},
		{Front: "ayat 255", Tags: []string{"surat:Al-Baqarah", "ayat:255"}},
	}

	result := filterAyatCards(cards)

	assert.Len(t, result, 3)
	assert.Equal(t, "Al-Fatihah", result[0].Surat)
	assert.Equal(t, 1, result[0].AyatNum)
	assert.Equal(t, "Al-Fatihah", result[1].Surat)
	assert.Equal(t, 2, result[1].AyatNum)
	assert.Equal(t, "Al-Baqarah", result[2].Surat)
	assert.Equal(t, 255, result[2].AyatNum)
}

func TestFilterAyatCards_Empty(t *testing.T) {
	assert.Nil(t, filterAyatCards(nil))
	assert.Nil(t, filterAyatCards([]domain.Card{}))
}

func TestFilterAyatCards_NoneMatch(t *testing.T) {
	cards := []domain.Card{
		{Front: "card 1", Tags: []string{"topic:iman"}},
		{Front: "card 2", Tags: []string{"topic:fiqh"}},
	}
	assert.Nil(t, filterAyatCards(cards))
}

// --- groupBySurat ---

func TestGroupBySurat(t *testing.T) {
	cards := []ayatCard{
		{Surat: "Al-Fatihah", AyatNum: 3},
		{Surat: "Al-Fatihah", AyatNum: 1},
		{Surat: "Al-Baqarah", AyatNum: 255},
		{Surat: "Al-Fatihah", AyatNum: 2},
		{Surat: "Al-Baqarah", AyatNum: 1},
	}

	groups := groupBySurat(cards)

	require.Len(t, groups, 2)

	fatihah := groups["Al-Fatihah"]
	require.Len(t, fatihah, 3)
	assert.Equal(t, 1, fatihah[0].AyatNum)
	assert.Equal(t, 2, fatihah[1].AyatNum)
	assert.Equal(t, 3, fatihah[2].AyatNum)

	baqarah := groups["Al-Baqarah"]
	require.Len(t, baqarah, 2)
	assert.Equal(t, 1, baqarah[0].AyatNum)
	assert.Equal(t, 255, baqarah[1].AyatNum)
}

func TestGroupBySurat_Empty(t *testing.T) {
	groups := groupBySurat(nil)
	assert.Empty(t, groups)
}

func TestGroupBySurat_SingleSurat(t *testing.T) {
	cards := []ayatCard{
		{Surat: "Al-Ikhlas", AyatNum: 2},
		{Surat: "Al-Ikhlas", AyatNum: 1},
		{Surat: "Al-Ikhlas", AyatNum: 4},
		{Surat: "Al-Ikhlas", AyatNum: 3},
	}

	groups := groupBySurat(cards)

	require.Len(t, groups, 1)
	ikhlas := groups["Al-Ikhlas"]
	require.Len(t, ikhlas, 4)
	for i := 0; i < len(ikhlas)-1; i++ {
		assert.Less(t, ikhlas[i].AyatNum, ikhlas[i+1].AyatNum, "should be sorted by ayat number")
	}
}

// ============================================================
// QuizService Method Tests
// ============================================================

// --- CreateQuiz ---

func TestQuizService_CreateQuiz_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	quizRepo.On("Create", ctx, mock.AnythingOfType("*domain.Quiz")).Return(nil)

	quiz, err := svc.CreateQuiz(ctx, userID, CreateQuizRequest{
		Title:    "Fiqh Quiz",
		QuizType: "mcq",
	})

	require.NoError(t, err)
	assert.Equal(t, "Fiqh Quiz", quiz.Title)
	assert.Equal(t, userID, quiz.UserID)
	assert.True(t, quiz.ShuffleQuestions) // default true
}

func TestQuizService_CreateQuiz_WithDeck(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()
	deckID := uuid.New()

	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	quizRepo.On("Create", ctx, mock.AnythingOfType("*domain.Quiz")).Return(nil)

	shuffle := false
	quiz, err := svc.CreateQuiz(ctx, userID, CreateQuizRequest{
		DeckID:           &deckID,
		Title:            "Deck Quiz",
		QuizType:         "mixed",
		ShuffleQuestions: &shuffle,
	})

	require.NoError(t, err)
	assert.Equal(t, &deckID, quiz.DeckID)
	assert.False(t, quiz.ShuffleQuestions)
}

func TestQuizService_CreateQuiz_DeckForbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	deckID := uuid.New()
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: uuid.New()}, nil)

	_, err := svc.CreateQuiz(ctx, uuid.New(), CreateQuizRequest{
		DeckID: &deckID, Title: "X", QuizType: "mcq",
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- GetQuiz ---

func TestQuizService_GetQuiz_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	quiz := &domain.Quiz{ID: quizID, UserID: userID, Title: "Test"}
	questions := []domain.QuizQuestion{{ID: uuid.New(), QuizID: quizID}}

	quizRepo.On("GetByID", ctx, quizID).Return(quiz, nil)
	quizRepo.On("ListQuestionsByQuizID", ctx, quizID).Return(questions, nil)

	detail, err := svc.GetQuiz(ctx, userID, quizID)

	require.NoError(t, err)
	assert.Equal(t, "Test", detail.Quiz.Title)
	assert.Len(t, detail.Questions, 1)
}

func TestQuizService_GetQuiz_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	_, err := svc.GetQuiz(ctx, uuid.New(), quizID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- ListQuizzes ---

func TestQuizService_ListQuizzes_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()
	userID := uuid.New()

	quizzes := []domain.QuizWithCounts{{Quiz: domain.Quiz{Title: "Q1"}}}
	quizRepo.On("ListByUserID", ctx, userID, 20, 0).Return(quizzes, 1, nil)

	result, total, err := svc.ListQuizzes(ctx, userID, 20, 0)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
}

// --- UpdateQuiz ---

func TestQuizService_UpdateQuiz_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	quiz := &domain.Quiz{ID: quizID, UserID: userID, Title: "Old", QuizType: "mcq"}

	quizRepo.On("GetByID", ctx, quizID).Return(quiz, nil)
	quizRepo.On("Update", ctx, mock.AnythingOfType("*domain.Quiz")).Return(nil)

	newTitle := "New Title"
	published := true
	timeLimit := 600
	result, err := svc.UpdateQuiz(ctx, userID, quizID, UpdateQuizRequest{
		Title:            &newTitle,
		IsPublished:      &published,
		TimeLimitSeconds: &timeLimit,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Title", result.Title)
	assert.True(t, result.IsPublished)
	assert.Equal(t, 600, *result.TimeLimitSeconds)
}

func TestQuizService_UpdateQuiz_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	title := "hack"
	_, err := svc.UpdateQuiz(ctx, uuid.New(), quizID, UpdateQuizRequest{Title: &title})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- DeleteQuiz ---

func TestQuizService_DeleteQuiz_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("Delete", ctx, quizID).Return(nil)

	err := svc.DeleteQuiz(ctx, userID, quizID)
	assert.NoError(t, err)
}

func TestQuizService_DeleteQuiz_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	err := svc.DeleteQuiz(ctx, uuid.New(), quizID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- AddQuestion ---

func TestQuizService_AddQuestion_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(3, nil)
	quizRepo.On("CreateQuestion", ctx, mock.AnythingOfType("*domain.QuizQuestion")).Return(nil)

	points := 5
	q, err := svc.AddQuestion(ctx, userID, quizID, AddQuestionRequest{
		QuestionType:  domain.QuestionTypeTrueFalse,
		QuestionText:  "Is this true?",
		CorrectAnswer: "true",
		Points:        &points,
	})

	require.NoError(t, err)
	assert.Equal(t, quizID, q.QuizID)
	assert.Equal(t, 3, q.Position)
	assert.Equal(t, 5, q.Points)
}

func TestQuizService_AddQuestion_DefaultPoints(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("CreateQuestion", ctx, mock.AnythingOfType("*domain.QuizQuestion")).Return(nil)

	q, err := svc.AddQuestion(ctx, userID, quizID, AddQuestionRequest{
		QuestionType:  domain.QuestionTypeFillBlank,
		QuestionText:  "Capital?",
		CorrectAnswer: "Jakarta",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, q.Points) // default
}

func TestQuizService_AddQuestion_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	_, err := svc.AddQuestion(ctx, uuid.New(), quizID, AddQuestionRequest{
		QuestionType: domain.QuestionTypeFillBlank, QuestionText: "X", CorrectAnswer: "Y",
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestQuizService_AddQuestion_InvalidOptions(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)

	// MCQ without options
	_, err := svc.AddQuestion(ctx, userID, quizID, AddQuestionRequest{
		QuestionType:  domain.QuestionTypeMCQ,
		QuestionText:  "Q?",
		CorrectAnswer: "A",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// --- BatchAddQuestions ---

func TestQuizService_BatchAddQuestions_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(2, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	questions, err := svc.BatchAddQuestions(ctx, userID, quizID, BatchAddQuestionsRequest{
		Questions: []AddQuestionRequest{
			{QuestionType: domain.QuestionTypeTrueFalse, QuestionText: "Q1", CorrectAnswer: "true"},
			{QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Q2", CorrectAnswer: "answer"},
		},
	})

	require.NoError(t, err)
	assert.Len(t, questions, 2)
	assert.Equal(t, 2, questions[0].Position)
	assert.Equal(t, 3, questions[1].Position)
}

func TestQuizService_BatchAddQuestions_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	_, err := svc.BatchAddQuestions(ctx, uuid.New(), quizID, BatchAddQuestionsRequest{
		Questions: []AddQuestionRequest{{QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Q", CorrectAnswer: "A"}},
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestQuizService_BatchAddQuestions_ValidationError(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)

	_, err := svc.BatchAddQuestions(ctx, userID, quizID, BatchAddQuestionsRequest{
		Questions: []AddQuestionRequest{
			{QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Good", CorrectAnswer: "A"},
			{QuestionType: domain.QuestionTypeMCQ, QuestionText: "Bad MCQ", CorrectAnswer: "A"}, // no options
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "question 2")
}

// --- UpdateQuestion ---

func TestQuizService_UpdateQuestion_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: quizID, QuestionType: domain.QuestionTypeFillBlank,
		QuestionText: "Old", CorrectAnswer: "Old Answer", Points: 1,
	}, nil)
	quizRepo.On("UpdateQuestion", ctx, mock.AnythingOfType("*domain.QuizQuestion")).Return(nil)

	newText := "New Question"
	newAnswer := "New Answer"
	newPoints := 3
	q, err := svc.UpdateQuestion(ctx, userID, quizID, questionID, UpdateQuestionRequest{
		QuestionText:  &newText,
		CorrectAnswer: &newAnswer,
		Points:        &newPoints,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Question", q.QuestionText)
	assert.Equal(t, "New Answer", q.CorrectAnswer)
	assert.Equal(t, 3, q.Points)
}

func TestQuizService_UpdateQuestion_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	text := "hack"
	_, err := svc.UpdateQuestion(ctx, uuid.New(), quizID, uuid.New(), UpdateQuestionRequest{QuestionText: &text})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestQuizService_UpdateQuestion_NotInQuiz(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: uuid.New(), // different quiz
	}, nil)

	text := "X"
	_, err := svc.UpdateQuestion(ctx, userID, quizID, questionID, UpdateQuestionRequest{QuestionText: &text})
	assert.ErrorIs(t, err, domain.ErrQuestionNotInQuiz)
}

// --- DeleteQuestion ---

func TestQuizService_DeleteQuestion_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{ID: questionID, QuizID: quizID}, nil)
	quizRepo.On("DeleteQuestion", ctx, questionID).Return(nil)

	err := svc.DeleteQuestion(ctx, userID, quizID, questionID)
	assert.NoError(t, err)
}

func TestQuizService_DeleteQuestion_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	err := svc.DeleteQuestion(ctx, uuid.New(), quizID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestQuizService_DeleteQuestion_NotInQuiz(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{ID: questionID, QuizID: uuid.New()}, nil)

	err := svc.DeleteQuestion(ctx, userID, quizID, questionID)
	assert.ErrorIs(t, err, domain.ErrQuestionNotInQuiz)
}

// --- GenerateFromDeck ---

func TestQuizService_GenerateFromDeck_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	cards := []domain.Card{
		{ID: uuid.New(), Front: "Q1", Back: "A1"},
		{ID: uuid.New(), Front: "Q2", Back: "A2"},
		{ID: uuid.New(), Front: "Q3", Back: "A3"},
		{ID: uuid.New(), Front: "Q4", Back: "A4"},
	}
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, 4, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	questions, err := svc.GenerateFromDeck(ctx, userID, quizID, GenerateFromDeckRequest{
		DeckID:       deckID,
		QuestionType: domain.QuestionTypeFillBlank,
		Count:        2,
	})

	require.NoError(t, err)
	assert.Len(t, questions, 2)
	for _, q := range questions {
		assert.Equal(t, domain.QuestionTypeFillBlank, q.QuestionType)
		assert.Equal(t, quizID, q.QuizID)
	}
}

func TestQuizService_GenerateFromDeck_InsufficientCards(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return([]domain.Card{
		{ID: uuid.New(), Front: "Q1", Back: "A1"},
	}, 1, nil)

	_, err := svc.GenerateFromDeck(ctx, userID, quizID, GenerateFromDeckRequest{
		DeckID: deckID, QuestionType: domain.QuestionTypeFillBlank, Count: 5,
	})
	assert.ErrorIs(t, err, domain.ErrInsufficientCards)
}

func TestQuizService_GenerateFromDeck_MCQNeed4Cards(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return([]domain.Card{
		{ID: uuid.New(), Front: "Q1", Back: "A1"},
		{ID: uuid.New(), Front: "Q2", Back: "A2"},
		{ID: uuid.New(), Front: "Q3", Back: "A3"},
	}, 3, nil)

	_, err := svc.GenerateFromDeck(ctx, userID, quizID, GenerateFromDeckRequest{
		DeckID: deckID, QuestionType: domain.QuestionTypeMCQ, Count: 2,
	})
	assert.ErrorIs(t, err, domain.ErrInsufficientCards)
}

func TestQuizService_GenerateFromDeck_DeckForbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: uuid.New()}, nil)

	_, err := svc.GenerateFromDeck(ctx, userID, quizID, GenerateFromDeckRequest{
		DeckID: deckID, QuestionType: domain.QuestionTypeFillBlank, Count: 1,
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- GenerateAyatQuiz ---

func TestQuizService_GenerateAyatQuiz_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	cards := []domain.Card{
		{ID: uuid.New(), Front: "بسم الله الرحمن الرحيم", Tags: []string{"surat:Al-Fatihah", "ayat:1"}},
		{ID: uuid.New(), Front: "الحمد لله رب العالمين", Tags: []string{"surat:Al-Fatihah", "ayat:2"}},
		{ID: uuid.New(), Front: "الرحمن الرحيم", Tags: []string{"surat:Al-Fatihah", "ayat:3"}},
	}
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, 3, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	questions, err := svc.GenerateAyatQuiz(ctx, userID, quizID, GenerateAyatQuizRequest{
		DeckID:       deckID,
		QuestionType: domain.QuestionTypeAyatCloze,
		Count:        2,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, questions)
	for _, q := range questions {
		assert.Equal(t, domain.QuestionTypeAyatCloze, q.QuestionType)
	}
}

func TestQuizService_GenerateAyatQuiz_InsufficientAyatCards(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)
	// Cards without ayat tags
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return([]domain.Card{
		{ID: uuid.New(), Front: "Q1", Back: "A1", Tags: []string{"topic:fiqh"}},
	}, 1, nil)

	_, err := svc.GenerateAyatQuiz(ctx, userID, quizID, GenerateAyatQuizRequest{
		DeckID: deckID, QuestionType: domain.QuestionTypeAyatCloze, Count: 1,
	})
	assert.ErrorIs(t, err, domain.ErrInsufficientCards)
}

// --- StartAttempt ---

func TestQuizService_StartAttempt_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{
		ID: quizID, UserID: userID, ShuffleQuestions: false,
	}, nil)
	quizRepo.On("ListQuestionsByQuizID", ctx, quizID).Return([]domain.QuizQuestion{
		{ID: uuid.New(), QuizID: quizID, QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Q1", CorrectAnswer: "A1", Points: 2},
		{ID: uuid.New(), QuizID: quizID, QuestionType: domain.QuestionTypeTrueFalse, QuestionText: "Q2", CorrectAnswer: "true", Points: 1},
	}, nil)
	attemptRepo.On("Create", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	resp, err := svc.StartAttempt(ctx, userID, quizID)

	require.NoError(t, err)
	assert.Equal(t, quizID, resp.Attempt.QuizID)
	assert.Equal(t, 3, resp.Attempt.TotalPoints) // 2+1
	assert.Equal(t, 2, resp.Attempt.TotalQuestions)
	assert.Len(t, resp.Questions, 2)
	// Correct answers stripped
	assert.Empty(t, resp.Questions[0].Options)
}

func TestQuizService_StartAttempt_PublishedQuiz(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	otherUserID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{
		ID: quizID, UserID: uuid.New(), IsPublished: true, ShuffleQuestions: false,
	}, nil)
	quizRepo.On("ListQuestionsByQuizID", ctx, quizID).Return([]domain.QuizQuestion{}, nil)
	attemptRepo.On("Create", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	resp, err := svc.StartAttempt(ctx, otherUserID, quizID)

	require.NoError(t, err)
	assert.Equal(t, otherUserID, resp.Attempt.UserID)
}

func TestQuizService_StartAttempt_NotPublished(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{
		ID: quizID, UserID: uuid.New(), IsPublished: false,
	}, nil)

	_, err := svc.StartAttempt(ctx, uuid.New(), quizID)
	assert.ErrorIs(t, err, domain.ErrQuizNotPublished)
}

func TestQuizService_StartAttempt_MCQStripsIsCorrect(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	opts, _ := json.Marshal([]domain.MCQOption{
		{Text: "A", IsCorrect: true},
		{Text: "B", IsCorrect: false},
	})

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID, ShuffleQuestions: false}, nil)
	quizRepo.On("ListQuestionsByQuizID", ctx, quizID).Return([]domain.QuizQuestion{
		{ID: uuid.New(), QuizID: quizID, QuestionType: domain.QuestionTypeMCQ, Options: opts, Points: 1},
	}, nil)
	attemptRepo.On("Create", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	resp, err := svc.StartAttempt(ctx, userID, quizID)

	require.NoError(t, err)

	// Options should not contain is_correct
	var stripped []map[string]any
	require.NoError(t, json.Unmarshal(resp.Questions[0].Options, &stripped))
	for _, opt := range stripped {
		_, hasIsCorrect := opt["is_correct"]
		assert.False(t, hasIsCorrect, "is_correct should be stripped from MCQ options")
	}
}

// --- SubmitAnswer ---

func TestQuizService_SubmitAnswer_Correct(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	attempt := &domain.QuizAttempt{ID: attemptID, QuizID: quizID, UserID: userID, StartedAt: time.Now()}

	attemptRepo.On("GetByID", ctx, attemptID).Return(attempt, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: quizID, QuestionType: domain.QuestionTypeFillBlank,
		CorrectAnswer: "Jakarta", Explanation: "Capital of Indonesia", Points: 2,
	}, nil)
	attemptRepo.On("GetAnswerByAttemptAndQuestion", ctx, attemptID, questionID).Return(nil, domain.ErrNotFound)
	attemptRepo.On("CreateAnswer", ctx, mock.AnythingOfType("*domain.QuizAnswer")).Return(nil)
	attemptRepo.On("Update", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	result, err := svc.SubmitAnswer(ctx, userID, attemptID, SubmitAnswerRequest{
		QuestionID: questionID,
		Answer:     "Jakarta",
		DurationMS: 3000,
	})

	require.NoError(t, err)
	assert.True(t, result.IsCorrect)
	assert.Equal(t, "Capital of Indonesia", result.Explanation)
	assert.Equal(t, "Jakarta", result.CorrectAnswer)
	assert.Equal(t, 2, result.Answer.PointsEarned)
	assert.Equal(t, 2, attempt.Score)
	assert.Equal(t, 1, attempt.CorrectCount)
}

func TestQuizService_SubmitAnswer_Wrong(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	attempt := &domain.QuizAttempt{ID: attemptID, QuizID: quizID, UserID: userID, StartedAt: time.Now()}

	attemptRepo.On("GetByID", ctx, attemptID).Return(attempt, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: quizID, QuestionType: domain.QuestionTypeFillBlank,
		CorrectAnswer: "Jakarta", Points: 1,
	}, nil)
	attemptRepo.On("GetAnswerByAttemptAndQuestion", ctx, attemptID, questionID).Return(nil, domain.ErrNotFound)
	attemptRepo.On("CreateAnswer", ctx, mock.AnythingOfType("*domain.QuizAnswer")).Return(nil)
	attemptRepo.On("Update", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	result, err := svc.SubmitAnswer(ctx, userID, attemptID, SubmitAnswerRequest{
		QuestionID: questionID, Answer: "Bandung",
	})

	require.NoError(t, err)
	assert.False(t, result.IsCorrect)
	assert.Equal(t, 0, result.Answer.PointsEarned)
	assert.Equal(t, 0, attempt.CorrectCount)
}

func TestQuizService_SubmitAnswer_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	attemptID := uuid.New()
	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, UserID: uuid.New(), StartedAt: time.Now(),
	}, nil)

	_, err := svc.SubmitAnswer(ctx, uuid.New(), attemptID, SubmitAnswerRequest{QuestionID: uuid.New(), Answer: "X"})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestQuizService_SubmitAnswer_AttemptCompleted(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	completedAt := time.Now()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, UserID: userID, CompletedAt: &completedAt, StartedAt: time.Now(),
	}, nil)

	_, err := svc.SubmitAnswer(ctx, userID, attemptID, SubmitAnswerRequest{QuestionID: uuid.New(), Answer: "X"})
	assert.ErrorIs(t, err, domain.ErrAttemptCompleted)
}

func TestQuizService_SubmitAnswer_QuestionNotInQuiz(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, QuizID: quizID, UserID: userID, StartedAt: time.Now(),
	}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: uuid.New(), // different quiz
	}, nil)

	_, err := svc.SubmitAnswer(ctx, userID, attemptID, SubmitAnswerRequest{QuestionID: questionID, Answer: "X"})
	assert.ErrorIs(t, err, domain.ErrQuestionNotInQuiz)
}

func TestQuizService_SubmitAnswer_AlreadyAnswered(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, QuizID: quizID, UserID: userID, StartedAt: time.Now(),
	}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: quizID,
	}, nil)
	attemptRepo.On("GetAnswerByAttemptAndQuestion", ctx, attemptID, questionID).Return(&domain.QuizAnswer{}, nil)

	_, err := svc.SubmitAnswer(ctx, userID, attemptID, SubmitAnswerRequest{QuestionID: questionID, Answer: "X"})
	assert.ErrorIs(t, err, domain.ErrAlreadyAnswered)
}

func TestQuizService_SubmitAnswer_WithFSRS(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()
	cardID := uuid.New()
	now := time.Now()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, QuizID: quizID, UserID: userID, StartedAt: now,
	}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: quizID, CardID: &cardID,
		QuestionType: domain.QuestionTypeFillBlank, CorrectAnswer: "Jakarta", Points: 1,
	}, nil)
	attemptRepo.On("GetAnswerByAttemptAndQuestion", ctx, attemptID, questionID).Return(nil, domain.ErrNotFound)
	attemptRepo.On("CreateAnswer", ctx, mock.AnythingOfType("*domain.QuizAnswer")).Return(nil)
	attemptRepo.On("Update", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	// FSRS card lookup and update
	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, State: domain.CardStateNew}, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)

	rating := 3
	result, err := svc.SubmitAnswer(ctx, userID, attemptID, SubmitAnswerRequest{
		QuestionID: questionID,
		Answer:     "Jakarta",
		DurationMS: 5000,
		FSRSCard: &FSRSCardState{
			Due: now.Add(24 * time.Hour), Stability: 2.5, Difficulty: 5.0,
			State: 1, LastReview: now,
		},
		FSRSLog: &FSRSLogState{ScheduledDays: 1, ElapsedDays: 0, Stability: 2.5, Difficulty: 5.0},
		Rating:  &rating,
	})

	require.NoError(t, err)
	assert.True(t, result.IsCorrect)
	assert.True(t, result.CardUpdated)
	assert.NotNil(t, result.NextDue)
}

// --- CompleteAttempt ---

func TestQuizService_CompleteAttempt_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, UserID: userID, StartedAt: time.Now().Add(-5 * time.Minute),
	}, nil)
	attemptRepo.On("Update", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	attempt, err := svc.CompleteAttempt(ctx, userID, attemptID)

	require.NoError(t, err)
	assert.NotNil(t, attempt.CompletedAt)
	assert.Greater(t, attempt.DurationMS, 0)
}

func TestQuizService_CompleteAttempt_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	attemptID := uuid.New()
	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, UserID: uuid.New(), StartedAt: time.Now(),
	}, nil)

	_, err := svc.CompleteAttempt(ctx, uuid.New(), attemptID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestQuizService_CompleteAttempt_AlreadyCompleted(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	completedAt := time.Now()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, UserID: userID, CompletedAt: &completedAt, StartedAt: time.Now(),
	}, nil)

	_, err := svc.CompleteAttempt(ctx, userID, attemptID)
	assert.ErrorIs(t, err, domain.ErrAttemptCompleted)
}

// --- GetAttempt ---

func TestQuizService_GetAttempt_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	quizID := uuid.New()
	q1ID := uuid.New()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, QuizID: quizID, UserID: userID, StartedAt: time.Now(),
	}, nil)
	attemptRepo.On("ListAnswersByAttemptID", ctx, attemptID).Return([]domain.QuizAnswer{
		{ID: uuid.New(), AttemptID: attemptID, QuestionID: q1ID, IsCorrect: true},
	}, nil)
	quizRepo.On("GetQuestionByID", ctx, q1ID).Return(&domain.QuizQuestion{
		ID: q1ID, QuizID: quizID, QuestionText: "Q1",
	}, nil)

	detail, err := svc.GetAttempt(ctx, userID, attemptID)

	require.NoError(t, err)
	assert.Len(t, detail.Answers, 1)
	assert.Equal(t, "Q1", detail.Answers[0].Question.QuestionText)
}

func TestQuizService_GetAttempt_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	attemptID := uuid.New()
	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, UserID: uuid.New(), StartedAt: time.Now(),
	}, nil)

	_, err := svc.GetAttempt(ctx, uuid.New(), attemptID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- ListAttempts ---

func TestQuizService_ListAttempts_Success(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	attemptRepo.On("ListByQuizID", ctx, quizID, 20, 0).Return([]domain.QuizAttempt{
		{ID: uuid.New(), QuizID: quizID},
	}, 1, nil)

	attempts, total, err := svc.ListAttempts(ctx, userID, quizID, 20, 0)

	require.NoError(t, err)
	assert.Len(t, attempts, 1)
	assert.Equal(t, 1, total)
}

func TestQuizService_ListAttempts_Forbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	_, _, err := svc.ListAttempts(ctx, uuid.New(), quizID, 20, 0)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// ============================================================
// Generator Function Tests
// ============================================================

func fixedRng() *rand.Rand {
	return rand.New(rand.NewSource(42))
}

// --- generateMCQ ---

func TestGenerateMCQ(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	card := domain.Card{ID: uuid.New(), Front: "What is 1+1?", Back: "2"}
	allCards := []domain.Card{
		card,
		{ID: uuid.New(), Front: "What is 2+2?", Back: "4"},
		{ID: uuid.New(), Front: "What is 3+3?", Back: "6"},
		{ID: uuid.New(), Front: "What is 4+4?", Back: "8"},
		{ID: uuid.New(), Front: "What is 5+5?", Back: "10"},
	}

	q := svc.generateMCQ(rng, card, allCards, 0)

	assert.Equal(t, domain.QuestionTypeMCQ, q.QuestionType)
	assert.Equal(t, "What is 1+1?", q.QuestionText)
	assert.Equal(t, "2", q.CorrectAnswer)
	assert.Equal(t, 1, q.Points)

	var opts []domain.MCQOption
	require.NoError(t, json.Unmarshal(q.Options, &opts))
	assert.Len(t, opts, 4) // 1 correct + 3 distractors

	correctCount := 0
	for _, o := range opts {
		if o.IsCorrect {
			correctCount++
			assert.Equal(t, "2", o.Text)
		}
	}
	assert.Equal(t, 1, correctCount)
}

func TestGenerateMCQ_FewDistractors(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	card := domain.Card{ID: uuid.New(), Front: "Q", Back: "A"}
	allCards := []domain.Card{
		card,
		{ID: uuid.New(), Front: "Q2", Back: "B"},
	}

	q := svc.generateMCQ(rng, card, allCards, 0)

	var opts []domain.MCQOption
	require.NoError(t, json.Unmarshal(q.Options, &opts))
	assert.Len(t, opts, 2) // 1 correct + 1 distractor
}

// --- generateTrueFalse ---

func TestGenerateTrueFalse_TrueCase(t *testing.T) {
	svc := &QuizService{}
	// Use seed that generates isTrue=true (rng.Intn(2)==0)
	rng := rand.New(rand.NewSource(0))

	card := domain.Card{Front: "Capital of Indonesia", Back: "Jakarta"}
	allCards := []domain.Card{card, {Front: "Capital of Japan", Back: "Tokyo"}}

	q := svc.generateTrueFalse(rng, card, allCards)

	assert.Equal(t, domain.QuestionTypeTrueFalse, q.QuestionType)
	assert.Equal(t, 1, q.Points)
	// Either true (showing correct pair) or false (showing wrong pair)
	assert.Contains(t, []string{"true", "false"}, q.CorrectAnswer)
}

func TestGenerateTrueFalse_FalseCase(t *testing.T) {
	svc := &QuizService{}
	// Find a seed where rng.Intn(2) != 0 so we get the false path
	for seed := int64(0); seed < 100; seed++ {
		rng := rand.New(rand.NewSource(seed))
		if rng.Intn(2) != 0 {
			// This seed gives us the false path
			rng = rand.New(rand.NewSource(seed))

			card := domain.Card{Front: "Capital of Indonesia", Back: "Jakarta"}
			allCards := []domain.Card{card, {Front: "Capital of Japan", Back: "Tokyo"}}

			q := svc.generateTrueFalse(rng, card, allCards)

			assert.Equal(t, domain.QuestionTypeTrueFalse, q.QuestionType)
			assert.Equal(t, "false", q.CorrectAnswer)
			assert.Contains(t, q.QuestionText, "Capital of Indonesia: Tokyo")
			return
		}
	}
	t.Fatal("could not find seed for false path")
}

func TestGenerateTrueFalse_AllSameBack(t *testing.T) {
	svc := &QuizService{}
	// Seed where rng.Intn(2)!=0 (false path), but no wrong answers available
	rng := rand.New(rand.NewSource(1))

	card := domain.Card{Front: "Q1", Back: "Same"}
	allCards := []domain.Card{
		card,
		{Front: "Q2", Back: "Same"},
		{Front: "Q3", Back: "Same"},
	}

	q := svc.generateTrueFalse(rng, card, allCards)

	// Fallback: shows correct pair, answer = "true"
	assert.Equal(t, "true", q.CorrectAnswer)
	assert.Contains(t, q.QuestionText, "Q1: Same")
}

// --- generateAyatContinuation ---

func TestGenerateAyatContinuation_Success(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	card1 := domain.Card{ID: uuid.New(), Front: "بسم الله الرحمن الرحيم"}
	card2 := domain.Card{ID: uuid.New(), Front: "الحمد لله رب العالمين"}
	card3 := domain.Card{ID: uuid.New(), Front: "الرحمن الرحيم"}

	groups := map[string][]ayatCard{
		"Al-Fatihah": {
			{Card: card1, Surat: "Al-Fatihah", AyatNum: 1},
			{Card: card2, Surat: "Al-Fatihah", AyatNum: 2},
			{Card: card3, Surat: "Al-Fatihah", AyatNum: 3},
		},
	}

	q := svc.generateAyatContinuation(rng, groups)

	require.NotNil(t, q)
	assert.Equal(t, domain.QuestionTypeAyatContinuation, q.QuestionType)
	assert.Contains(t, q.QuestionText, "Lanjutkan ayat setelah:")
	assert.NotNil(t, q.CardID)
	assert.Equal(t, 1, q.Points)
}

func TestGenerateAyatContinuation_NoPairs(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	// Non-consecutive ayat → no pairs
	groups := map[string][]ayatCard{
		"Al-Fatihah": {
			{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Fatihah", AyatNum: 1},
			{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Fatihah", AyatNum: 5},
		},
	}

	q := svc.generateAyatContinuation(rng, groups)
	assert.Nil(t, q)
}

// --- generateSurahIdentification ---

func TestGenerateSurahIdentification_Success(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	ac1 := ayatCard{Card: domain.Card{ID: uuid.New(), Front: "ayat 1"}, Surat: "Al-Fatihah", AyatNum: 1}
	ac2 := ayatCard{Card: domain.Card{ID: uuid.New(), Front: "ayat 255"}, Surat: "Al-Baqarah", AyatNum: 255}
	ac3 := ayatCard{Card: domain.Card{ID: uuid.New(), Front: "ayat 1 imran"}, Surat: "Al-Imran", AyatNum: 1}

	ayatCards := []ayatCard{ac1, ac2, ac3}
	groups := map[string][]ayatCard{
		"Al-Fatihah": {ac1},
		"Al-Baqarah": {ac2},
		"Al-Imran":   {ac3},
	}

	q := svc.generateSurahIdentification(rng, ayatCards, groups)

	require.NotNil(t, q)
	assert.Equal(t, domain.QuestionTypeSurahID, q.QuestionType)
	assert.NotNil(t, q.CardID)
	assert.Contains(t, q.QuestionText, "Ayat berikut berasal dari surat apa?")
	assert.Equal(t, 1, q.Points)

	var opts []domain.MCQOption
	require.NoError(t, json.Unmarshal(q.Options, &opts))
	assert.GreaterOrEqual(t, len(opts), 2) // correct + at least 1 distractor

	// Correct answer must be one of the surat names
	assert.Contains(t, []string{"Al-Fatihah", "Al-Baqarah", "Al-Imran"}, q.CorrectAnswer)
}

func TestGenerateSurahIdentification_OnlyOneSurat(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	ac := ayatCard{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Fatihah", AyatNum: 1}
	groups := map[string][]ayatCard{"Al-Fatihah": {ac}}

	q := svc.generateSurahIdentification(rng, []ayatCard{ac}, groups)
	assert.Nil(t, q) // needs >= 2 surats
}

// --- generateOrdering ---

func TestGenerateOrdering_Success(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	cards := make([]ayatCard, 5)
	for i := range cards {
		cards[i] = ayatCard{
			Card:    domain.Card{ID: uuid.New(), Front: "ayat " + strings.Repeat("word ", 3)},
			Surat:   "Al-Fatihah",
			AyatNum: i + 1,
		}
	}

	groups := map[string][]ayatCard{"Al-Fatihah": cards}

	q := svc.generateOrdering(rng, groups)

	require.NotNil(t, q)
	assert.Equal(t, domain.QuestionTypeOrdering, q.QuestionType)
	assert.Equal(t, 2, q.Points)
	assert.Contains(t, q.QuestionText, "Urutkan ayat-ayat")
	assert.Contains(t, q.Explanation, "Al-Fatihah")

	// Correct answer is comma-separated UUIDs
	ids := strings.Split(q.CorrectAnswer, ",")
	assert.GreaterOrEqual(t, len(ids), 3)

	var opts []orderingOption
	require.NoError(t, json.Unmarshal(q.Options, &opts))
	assert.Equal(t, len(ids), len(opts))
}

func TestGenerateOrdering_TooFewConsecutive(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	// Only 2 ayat — need >= 3
	groups := map[string][]ayatCard{
		"Al-Fatihah": {
			{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Fatihah", AyatNum: 1},
			{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Fatihah", AyatNum: 2},
		},
	}

	q := svc.generateOrdering(rng, groups)
	assert.Nil(t, q)
}

func TestGenerateOrdering_NonConsecutive(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	// 4 ayat but not consecutive
	groups := map[string][]ayatCard{
		"Al-Baqarah": {
			{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Baqarah", AyatNum: 1},
			{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Baqarah", AyatNum: 3},
			{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Baqarah", AyatNum: 7},
			{Card: domain.Card{ID: uuid.New()}, Surat: "Al-Baqarah", AyatNum: 10},
		},
	}

	q := svc.generateOrdering(rng, groups)
	assert.Nil(t, q) // no run of >= 3 consecutive
}

// ============================================================
// Additional error-path & branch-coverage tests
// ============================================================

// --- UpdateQuiz: all field branches ---

func TestQuizService_UpdateQuiz_AllFields(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	quiz := &domain.Quiz{ID: quizID, UserID: userID, Title: "Old", QuizType: "mcq", Description: "old desc"}

	quizRepo.On("GetByID", ctx, quizID).Return(quiz, nil)
	quizRepo.On("Update", ctx, mock.AnythingOfType("*domain.Quiz")).Return(nil)

	newTitle := "New Title"
	newDesc := "New Desc"
	newType := "mixed"
	newShuffle := false
	newPublished := true
	timeLimit := 300
	result, err := svc.UpdateQuiz(ctx, userID, quizID, UpdateQuizRequest{
		Title:            &newTitle,
		Description:      &newDesc,
		QuizType:         &newType,
		ShuffleQuestions: &newShuffle,
		IsPublished:      &newPublished,
		TimeLimitSeconds: &timeLimit,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Title", result.Title)
	assert.Equal(t, "New Desc", result.Description)
	assert.Equal(t, "mixed", result.QuizType)
	assert.False(t, result.ShuffleQuestions)
	assert.True(t, result.IsPublished)
	assert.Equal(t, 300, *result.TimeLimitSeconds)
}

func TestQuizService_UpdateQuiz_NotFound(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(nil, domain.ErrNotFound)

	title := "X"
	_, err := svc.UpdateQuiz(ctx, uuid.New(), quizID, UpdateQuizRequest{Title: &title})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- UpdateQuestion: all field branches ---

func TestQuizService_UpdateQuestion_AllFields(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: quizID, QuestionType: domain.QuestionTypeFillBlank,
		QuestionText: "Old", CorrectAnswer: "Old", Explanation: "old", Points: 1,
	}, nil)
	quizRepo.On("UpdateQuestion", ctx, mock.AnythingOfType("*domain.QuizQuestion")).Return(nil)

	newType := domain.QuestionTypeTrueFalse
	newText := "Is this true?"
	newAnswer := "true"
	newExplanation := "Because yes"
	newPoints := 5
	opts := json.RawMessage(`null`)

	q, err := svc.UpdateQuestion(ctx, userID, quizID, questionID, UpdateQuestionRequest{
		QuestionType:  &newType,
		QuestionText:  &newText,
		Options:       &opts,
		CorrectAnswer: &newAnswer,
		Explanation:   &newExplanation,
		Points:        &newPoints,
	})

	require.NoError(t, err)
	assert.Equal(t, domain.QuestionTypeTrueFalse, q.QuestionType)
	assert.Equal(t, "Is this true?", q.QuestionText)
	assert.Equal(t, "true", q.CorrectAnswer)
	assert.Equal(t, "Because yes", q.Explanation)
	assert.Equal(t, 5, q.Points)
}

// --- GetQuiz: ListQuestions error ---

func TestQuizService_GetQuiz_ListQuestionsError(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("ListQuestionsByQuizID", ctx, quizID).Return(nil, assert.AnError)

	_, err := svc.GetQuiz(ctx, userID, quizID)
	assert.Error(t, err)
}

func TestQuizService_GetQuiz_NotFound(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(nil, domain.ErrNotFound)

	_, err := svc.GetQuiz(ctx, uuid.New(), quizID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- CreateQuiz: deck error ---

func TestQuizService_CreateQuiz_DeckNotFound(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	deckID := uuid.New()
	deckRepo.On("GetByID", ctx, deckID).Return(nil, domain.ErrNotFound)

	_, err := svc.CreateQuiz(ctx, uuid.New(), CreateQuizRequest{
		DeckID: &deckID, Title: "X", QuizType: "mcq",
	})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- DeleteQuiz: not found ---

func TestQuizService_DeleteQuiz_NotFound(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(nil, domain.ErrNotFound)

	err := svc.DeleteQuiz(ctx, uuid.New(), quizID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- BatchAddQuestions: CountQuestions error ---

func TestQuizService_BatchAddQuestions_CountError(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, assert.AnError)

	_, err := svc.BatchAddQuestions(ctx, userID, quizID, BatchAddQuestionsRequest{
		Questions: []AddQuestionRequest{
			{QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Q", CorrectAnswer: "A"},
		},
	})
	assert.Error(t, err)
}

// --- GenerateFromDeck: Mixed type ---

func TestQuizService_GenerateFromDeck_MixedType(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	cards := []domain.Card{
		{ID: uuid.New(), Front: "Q1", Back: "A1"},
		{ID: uuid.New(), Front: "Q2", Back: "A2"},
		{ID: uuid.New(), Front: "Q3", Back: "A3"},
		{ID: uuid.New(), Front: "Q4", Back: "A4"},
		{ID: uuid.New(), Front: "Q5", Back: "A5"},
	}
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, 5, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	questions, err := svc.GenerateFromDeck(ctx, userID, quizID, GenerateFromDeckRequest{
		DeckID:       deckID,
		QuestionType: domain.QuizTypeMixed,
		Count:        3,
	})

	require.NoError(t, err)
	assert.Len(t, questions, 3)
	for _, q := range questions {
		assert.Contains(t, []string{domain.QuestionTypeMCQ, domain.QuestionTypeTrueFalse, domain.QuestionTypeFillBlank}, q.QuestionType)
	}
}

func TestQuizService_GenerateFromDeck_TrueFalseType(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	cards := []domain.Card{
		{ID: uuid.New(), Front: "Q1", Back: "A1"},
		{ID: uuid.New(), Front: "Q2", Back: "A2"},
		{ID: uuid.New(), Front: "Q3", Back: "A3"},
	}
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, 3, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	questions, err := svc.GenerateFromDeck(ctx, userID, quizID, GenerateFromDeckRequest{
		DeckID:       deckID,
		QuestionType: domain.QuestionTypeTrueFalse,
		Count:        2,
	})

	require.NoError(t, err)
	assert.Len(t, questions, 2)
	for _, q := range questions {
		assert.Equal(t, domain.QuestionTypeTrueFalse, q.QuestionType)
	}
}

func TestQuizService_GenerateFromDeck_MCQType(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	cards := []domain.Card{
		{ID: uuid.New(), Front: "Q1", Back: "A1"},
		{ID: uuid.New(), Front: "Q2", Back: "A2"},
		{ID: uuid.New(), Front: "Q3", Back: "A3"},
		{ID: uuid.New(), Front: "Q4", Back: "A4"},
	}
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, 4, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	questions, err := svc.GenerateFromDeck(ctx, userID, quizID, GenerateFromDeckRequest{
		DeckID:       deckID,
		QuestionType: domain.QuestionTypeMCQ,
		Count:        2,
	})

	require.NoError(t, err)
	assert.Len(t, questions, 2)
	for _, q := range questions {
		assert.Equal(t, domain.QuestionTypeMCQ, q.QuestionType)
	}
}

func TestQuizService_GenerateFromDeck_CountExceedsCards(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	cards := []domain.Card{
		{ID: uuid.New(), Front: "Q1", Back: "A1"},
		{ID: uuid.New(), Front: "Q2", Back: "A2"},
		{ID: uuid.New(), Front: "Q3", Back: "A3"},
	}
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, 3, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	// Request 100 but only 3 cards → should generate 3
	questions, err := svc.GenerateFromDeck(ctx, userID, quizID, GenerateFromDeckRequest{
		DeckID:       deckID,
		QuestionType: domain.QuestionTypeFillBlank,
		Count:        100,
	})

	require.NoError(t, err)
	assert.Len(t, questions, 3)
}

// --- GenerateAyatQuiz: mixed_ayat type ---

func TestQuizService_GenerateAyatQuiz_MixedAyat(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	cards := []domain.Card{
		{ID: uuid.New(), Front: "بسم الله الرحمن الرحيم", Tags: []string{"surat:Al-Fatihah", "ayat:1"}},
		{ID: uuid.New(), Front: "الحمد لله رب العالمين", Tags: []string{"surat:Al-Fatihah", "ayat:2"}},
		{ID: uuid.New(), Front: "الرحمن الرحيم", Tags: []string{"surat:Al-Fatihah", "ayat:3"}},
		{ID: uuid.New(), Front: "مالك يوم الدين", Tags: []string{"surat:Al-Fatihah", "ayat:4"}},
		{ID: uuid.New(), Front: "إياك نعبد وإياك نستعين", Tags: []string{"surat:Al-Fatihah", "ayat:5"}},
		{ID: uuid.New(), Front: "اهدنا الصراط المستقيم", Tags: []string{"surat:Al-Fatihah", "ayat:6"}},
		{ID: uuid.New(), Front: "ألم", Tags: []string{"surat:Al-Baqarah", "ayat:1"}},
		{ID: uuid.New(), Front: "ذلك الكتاب لا ريب فيه هدى للمتقين", Tags: []string{"surat:Al-Baqarah", "ayat:2"}},
		{ID: uuid.New(), Front: "الذين يؤمنون بالغيب", Tags: []string{"surat:Al-Baqarah", "ayat:3"}},
	}
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, len(cards), nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	questions, err := svc.GenerateAyatQuiz(ctx, userID, quizID, GenerateAyatQuizRequest{
		DeckID:       deckID,
		QuestionType: domain.QuizTypeMixedAyat,
		Count:        3,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, questions)
}

func TestQuizService_GenerateAyatQuiz_ContinuationType(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: userID}, nil)

	cards := []domain.Card{
		{ID: uuid.New(), Front: "بسم الله الرحمن الرحيم", Tags: []string{"surat:Al-Fatihah", "ayat:1"}},
		{ID: uuid.New(), Front: "الحمد لله رب العالمين", Tags: []string{"surat:Al-Fatihah", "ayat:2"}},
		{ID: uuid.New(), Front: "الرحمن الرحيم", Tags: []string{"surat:Al-Fatihah", "ayat:3"}},
	}
	cardRepo.On("ListByDeckID", ctx, deckID, domain.CardFilter{}, 10000, 0).Return(cards, 3, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, nil)
	quizRepo.On("BulkCreateQuestions", ctx, mock.AnythingOfType("[]*domain.QuizQuestion")).Return(nil)

	questions, err := svc.GenerateAyatQuiz(ctx, userID, quizID, GenerateAyatQuizRequest{
		DeckID:       deckID,
		QuestionType: domain.QuestionTypeAyatContinuation,
		Count:        2,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, questions)
	for _, q := range questions {
		assert.Equal(t, domain.QuestionTypeAyatContinuation, q.QuestionType)
	}
}

func TestQuizService_GenerateAyatQuiz_DeckForbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	deckRepo.On("GetByID", ctx, deckID).Return(&domain.Deck{ID: deckID, UserID: uuid.New()}, nil) // different owner

	_, err := svc.GenerateAyatQuiz(ctx, userID, quizID, GenerateAyatQuizRequest{
		DeckID: deckID, QuestionType: domain.QuestionTypeAyatCloze, Count: 1,
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestQuizService_GenerateAyatQuiz_QuizForbidden(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	deckID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: uuid.New()}, nil)

	_, err := svc.GenerateAyatQuiz(ctx, uuid.New(), quizID, GenerateAyatQuizRequest{
		DeckID: deckID, QuestionType: domain.QuestionTypeAyatCloze, Count: 1,
	})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// --- applyFSRSFromQuiz: suspended card ---

func TestQuizService_SubmitAnswer_FSRS_SuspendedCard(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()
	cardID := uuid.New()
	now := time.Now()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, QuizID: quizID, UserID: userID, StartedAt: now,
	}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: quizID, CardID: &cardID,
		QuestionType: domain.QuestionTypeFillBlank, CorrectAnswer: "test", Points: 1,
	}, nil)
	attemptRepo.On("GetAnswerByAttemptAndQuestion", ctx, attemptID, questionID).Return(nil, domain.ErrNotFound)
	attemptRepo.On("CreateAnswer", ctx, mock.AnythingOfType("*domain.QuizAnswer")).Return(nil)
	attemptRepo.On("Update", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	// Card is suspended — FSRS should be skipped
	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, IsSuspended: true}, nil)

	result, err := svc.SubmitAnswer(ctx, userID, attemptID, SubmitAnswerRequest{
		QuestionID: questionID,
		Answer:     "test",
		FSRSCard:   &FSRSCardState{Due: now, Stability: 2.5, Difficulty: 5.0, State: 1, LastReview: now},
	})

	require.NoError(t, err)
	assert.True(t, result.IsCorrect)
	assert.False(t, result.CardUpdated) // suspended card skips FSRS
}

// --- applyFSRSFromQuiz: no rating, incorrect answer defaults to Again ---

func TestQuizService_SubmitAnswer_FSRS_DefaultRatingIncorrect(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()
	cardID := uuid.New()
	now := time.Now()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, QuizID: quizID, UserID: userID, StartedAt: now,
	}, nil)
	quizRepo.On("GetQuestionByID", ctx, questionID).Return(&domain.QuizQuestion{
		ID: questionID, QuizID: quizID, CardID: &cardID,
		QuestionType: domain.QuestionTypeFillBlank, CorrectAnswer: "correct", Points: 1,
	}, nil)
	attemptRepo.On("GetAnswerByAttemptAndQuestion", ctx, attemptID, questionID).Return(nil, domain.ErrNotFound)
	attemptRepo.On("CreateAnswer", ctx, mock.AnythingOfType("*domain.QuizAnswer")).Return(nil)
	attemptRepo.On("Update", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	cardRepo.On("GetByID", ctx, cardID).Return(&domain.Card{ID: cardID, State: domain.CardStateReview}, nil)
	cardRepo.On("UpdateFSRS", ctx, mock.AnythingOfType("*domain.Card")).Return(nil)
	reviewRepo.On("Create", ctx, mock.AnythingOfType("*domain.ReviewLog")).Return(nil)

	// No Rating provided + wrong answer → defaults to RatingAgain
	result, err := svc.SubmitAnswer(ctx, userID, attemptID, SubmitAnswerRequest{
		QuestionID: questionID,
		Answer:     "wrong",
		FSRSCard:   &FSRSCardState{Due: now, Stability: 2.5, Difficulty: 5.0, State: 2, LastReview: now},
		// No Rating, No FSRSLog
	})

	require.NoError(t, err)
	assert.False(t, result.IsCorrect)
	assert.True(t, result.CardUpdated)
}

// --- StartAttempt: shuffle enabled ---

func TestQuizService_StartAttempt_WithShuffle(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{
		ID: quizID, UserID: userID, ShuffleQuestions: true,
	}, nil)
	quizRepo.On("ListQuestionsByQuizID", ctx, quizID).Return([]domain.QuizQuestion{
		{ID: uuid.New(), QuizID: quizID, QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Q1", CorrectAnswer: "A1", Points: 1},
		{ID: uuid.New(), QuizID: quizID, QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Q2", CorrectAnswer: "A2", Points: 1},
		{ID: uuid.New(), QuizID: quizID, QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Q3", CorrectAnswer: "A3", Points: 1},
	}, nil)
	attemptRepo.On("Create", ctx, mock.AnythingOfType("*domain.QuizAttempt")).Return(nil)

	resp, err := svc.StartAttempt(ctx, userID, quizID)

	require.NoError(t, err)
	assert.Len(t, resp.Questions, 3)
	assert.Equal(t, 3, resp.Attempt.TotalPoints)
}

// --- GetAttempt: ListAnswers error ---

func TestQuizService_GetAttempt_ListAnswersError(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	attemptID := uuid.New()

	attemptRepo.On("GetByID", ctx, attemptID).Return(&domain.QuizAttempt{
		ID: attemptID, UserID: userID, StartedAt: time.Now(),
	}, nil)
	attemptRepo.On("ListAnswersByAttemptID", ctx, attemptID).Return(nil, assert.AnError)

	_, err := svc.GetAttempt(ctx, userID, attemptID)
	assert.Error(t, err)
}

// --- ListAttempts: quiz not found ---

func TestQuizService_ListAttempts_QuizNotFound(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	quizID := uuid.New()
	quizRepo.On("GetByID", ctx, quizID).Return(nil, domain.ErrNotFound)

	_, _, err := svc.ListAttempts(ctx, uuid.New(), quizID, 20, 0)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- AddQuestion: CountQuestions error ---

func TestQuizService_AddQuestion_CountError(t *testing.T) {
	quizRepo := mockdomain.NewMockQuizRepository(t)
	attemptRepo := mockdomain.NewMockQuizAttemptRepository(t)
	cardRepo := mockdomain.NewMockCardRepository(t)
	deckRepo := mockdomain.NewMockDeckRepository(t)
	reviewRepo := mockdomain.NewMockReviewRepository(t)
	svc := NewQuizService(quizRepo, attemptRepo, cardRepo, deckRepo, reviewRepo)
	ctx := context.Background()

	userID := uuid.New()
	quizID := uuid.New()

	quizRepo.On("GetByID", ctx, quizID).Return(&domain.Quiz{ID: quizID, UserID: userID}, nil)
	quizRepo.On("CountQuestionsByQuizID", ctx, quizID).Return(0, assert.AnError)

	_, err := svc.AddQuestion(ctx, userID, quizID, AddQuestionRequest{
		QuestionType: domain.QuestionTypeFillBlank, QuestionText: "Q", CorrectAnswer: "A",
	})
	assert.Error(t, err)
}

// --- generateAyatCloze: short text (< 3 words) ---

func TestGenerateAyatCloze_ShortText(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	ac := ayatCard{
		Card:    domain.Card{ID: uuid.New(), Front: "كلمتين"},
		Surat:   "Al-Fatihah",
		AyatNum: 1,
	}

	q := svc.generateAyatCloze(rng, []ayatCard{ac})

	require.NotNil(t, q)
	assert.Equal(t, domain.QuestionTypeAyatCloze, q.QuestionType)
	assert.Equal(t, "_____", q.QuestionText)
	assert.Equal(t, "كلمتين", q.CorrectAnswer) // short text uses fill-blank fallback
}

func TestGenerateAyatCloze_LongText(t *testing.T) {
	svc := &QuizService{}
	rng := fixedRng()

	ac := ayatCard{
		Card:    domain.Card{ID: uuid.New(), Front: "بسم الله الرحمن الرحيم الحمد"},
		Surat:   "Al-Fatihah",
		AyatNum: 1,
	}

	q := svc.generateAyatCloze(rng, []ayatCard{ac})

	require.NotNil(t, q)
	assert.Equal(t, domain.QuestionTypeAyatCloze, q.QuestionType)
	assert.Contains(t, q.QuestionText, "_____") // has blank
	assert.NotEqual(t, "_____", q.QuestionText)  // not the short-text fallback
}
