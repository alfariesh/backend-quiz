package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
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
