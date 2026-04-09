package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	mockport "github.com/rekanesiads/backend-quiz/internal/mocks/port"
)

func setupQuizRouter(h *QuizHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/quizzes", h.CreateQuiz)
	r.Get("/quizzes", h.ListQuizzes)
	r.Get("/quizzes/{quizID}", h.GetQuiz)
	r.Put("/quizzes/{quizID}", h.UpdateQuiz)
	r.Delete("/quizzes/{quizID}", h.DeleteQuiz)
	r.Post("/quizzes/{quizID}/questions", h.AddQuestion)
	r.Post("/quizzes/{quizID}/questions/batch", h.BatchAddQuestions)
	r.Post("/quizzes/{quizID}/generate", h.GenerateFromDeck)
	r.Post("/quizzes/{quizID}/generate-ayat", h.GenerateAyatQuiz)
	r.Put("/quizzes/{quizID}/questions/{questionID}", h.UpdateQuestion)
	r.Delete("/quizzes/{quizID}/questions/{questionID}", h.DeleteQuestion)
	r.Post("/quizzes/{quizID}/attempts", h.StartAttempt)
	r.Get("/quizzes/{quizID}/attempts", h.ListAttempts)
	r.Get("/attempts/{attemptID}", h.GetAttempt)
	r.Post("/attempts/{attemptID}/answers", h.SubmitAnswer)
	r.Post("/attempts/{attemptID}/complete", h.CompleteAttempt)
	return r
}

// --- CreateQuiz ---

func TestQuizHandler_CreateQuiz_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	svc.EXPECT().CreateQuiz(mock.Anything, uid, mock.Anything).Return(
		&domain.Quiz{ID: uuid.New(), UserID: uid, Title: "Test Quiz", QuizType: "mcq"}, nil,
	)

	body := `{"title":"Test Quiz","quiz_type":"mcq"}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp APIResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data := resp.Data.(map[string]any)
	assert.Equal(t, "Test Quiz", data["title"])
}

func TestQuizHandler_CreateQuiz_InvalidBody(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/quizzes", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_CreateQuiz_ValidationError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	body := `{"title":""}` // missing quiz_type, empty title
	req := httptest.NewRequest(http.MethodPost, "/quizzes", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListQuizzes ---

func TestQuizHandler_ListQuizzes_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	svc.EXPECT().ListQuizzes(mock.Anything, uid, 20, 0).Return(
		[]domain.QuizWithCounts{{Quiz: domain.Quiz{Title: "Q1"}}}, 1, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/quizzes", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- GetQuiz ---

func TestQuizHandler_GetQuiz_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().GetQuiz(mock.Anything, uid, quizID).Return(
		&dto.QuizDetail{Quiz: domain.Quiz{ID: quizID, Title: "Test"}, Questions: []domain.QuizQuestion{}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/quizzes/"+quizID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuizHandler_GetQuiz_InvalidID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/quizzes/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_GetQuiz_NotFound(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().GetQuiz(mock.Anything, uid, quizID).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/quizzes/"+quizID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestQuizHandler_GetQuiz_Forbidden(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().GetQuiz(mock.Anything, uid, quizID).Return(nil, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodGet, "/quizzes/"+quizID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- UpdateQuiz ---

func TestQuizHandler_UpdateQuiz_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().UpdateQuiz(mock.Anything, uid, quizID, mock.Anything).Return(
		&domain.Quiz{ID: quizID, UserID: uid, Title: "New Title"}, nil,
	)

	body := `{"title":"New Title"}`
	req := httptest.NewRequest(http.MethodPut, "/quizzes/"+quizID.String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuizHandler_UpdateQuiz_InvalidID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	body := `{"title":"New"}`
	req := httptest.NewRequest(http.MethodPut, "/quizzes/bad-id", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_UpdateQuiz_InvalidBody(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/quizzes/"+quizID.String(), strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- DeleteQuiz ---

func TestQuizHandler_DeleteQuiz_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().DeleteQuiz(mock.Anything, uid, quizID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/quizzes/"+quizID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "quiz deleted")
}

func TestQuizHandler_DeleteQuiz_InvalidID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/quizzes/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- AddQuestion ---

func TestQuizHandler_AddQuestion_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().AddQuestion(mock.Anything, uid, quizID, mock.Anything).Return(
		&domain.QuizQuestion{ID: uuid.New(), QuizID: quizID, QuestionText: "What is X?"}, nil,
	)

	body := `{"question_type":"mcq","question_text":"What is X?","correct_answer":"A","options":[{"text":"A","is_correct":true},{"text":"B","is_correct":false}]}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/questions", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestQuizHandler_AddQuestion_InvalidQuizID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	body := `{"question_type":"mcq","question_text":"Q","correct_answer":"A"}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/bad-id/questions", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_AddQuestion_InvalidBody(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/questions", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_AddQuestion_ValidationError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	body := `{"question_type":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/questions", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- BatchAddQuestions ---

func TestQuizHandler_BatchAddQuestions_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().BatchAddQuestions(mock.Anything, uid, quizID, mock.Anything).Return(
		[]*domain.QuizQuestion{{ID: uuid.New()}}, nil,
	)

	body := `{"questions":[{"question_type":"mcq","question_text":"Q1","correct_answer":"A","options":[{"text":"A","is_correct":true},{"text":"B","is_correct":false}]}]}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/questions/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestQuizHandler_BatchAddQuestions_InvalidBody(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/questions/batch", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_BatchAddQuestions_ValidationError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	body := `{"questions":[]}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/questions/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GenerateFromDeck ---

func TestQuizHandler_GenerateFromDeck_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().GenerateFromDeck(mock.Anything, uid, quizID, mock.Anything).Return(
		[]*domain.QuizQuestion{{ID: uuid.New(), QuizID: quizID, QuestionText: "Generated Q"}}, nil,
	)

	body := `{"deck_id":"` + deckID.String() + `","question_type":"mcq","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/generate", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestQuizHandler_GenerateFromDeck_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().GenerateFromDeck(mock.Anything, uid, quizID, mock.Anything).Return(nil, domain.ErrInsufficientCards)

	body := `{"deck_id":"` + deckID.String() + `","question_type":"mcq","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/generate", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_GenerateFromDeck_InvalidQuizID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	body := `{"deck_id":"` + uuid.New().String() + `","question_type":"mcq","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/bad-id/generate", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_GenerateFromDeck_InvalidBody(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/generate", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_GenerateFromDeck_ValidationError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	body := `{"deck_id":"` + uuid.New().String() + `","question_type":"invalid","count":0}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/generate", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GenerateAyatQuiz ---

func TestQuizHandler_GenerateAyatQuiz_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().GenerateAyatQuiz(mock.Anything, uid, quizID, mock.Anything).Return(
		[]*domain.QuizQuestion{{ID: uuid.New(), QuizID: quizID, QuestionText: "Ayat Q"}}, nil,
	)

	body := `{"deck_id":"` + deckID.String() + `","question_type":"ayat_cloze","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/generate-ayat", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestQuizHandler_GenerateAyatQuiz_ValidationError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	body := `{"deck_id":"` + uuid.New().String() + `","question_type":"invalid","count":0}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/generate-ayat", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_GenerateAyatQuiz_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	deckID := uuid.New()

	svc.EXPECT().GenerateAyatQuiz(mock.Anything, uid, quizID, mock.Anything).Return(nil, domain.ErrInsufficientCards)

	body := `{"deck_id":"` + deckID.String() + `","question_type":"ayat_cloze","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/generate-ayat", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_GenerateAyatQuiz_InvalidQuizID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	body := `{"deck_id":"` + uuid.New().String() + `","question_type":"ayat_cloze","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/bad-id/generate-ayat", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_GenerateAyatQuiz_InvalidBody(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/generate-ayat", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- UpdateQuestion ---

func TestQuizHandler_UpdateQuestion_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	svc.EXPECT().UpdateQuestion(mock.Anything, uid, quizID, questionID, mock.Anything).Return(
		&domain.QuizQuestion{ID: questionID, QuizID: quizID, QuestionText: "Updated Q"}, nil,
	)

	body := `{"question_text":"Updated Q"}`
	req := httptest.NewRequest(http.MethodPut, "/quizzes/"+quizID.String()+"/questions/"+questionID.String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuizHandler_UpdateQuestion_InvalidQuizID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	body := `{"question_text":"Q"}`
	req := httptest.NewRequest(http.MethodPut, "/quizzes/bad-id/questions/"+uuid.New().String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_UpdateQuestion_InvalidQuestionID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	body := `{"question_text":"Q"}`
	req := httptest.NewRequest(http.MethodPut, "/quizzes/"+quizID.String()+"/questions/bad-id", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_UpdateQuestion_InvalidBody(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	questionID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/quizzes/"+quizID.String()+"/questions/"+questionID.String(), strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- DeleteQuestion ---

func TestQuizHandler_DeleteQuestion_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()

	svc.EXPECT().DeleteQuestion(mock.Anything, uid, quizID, questionID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/quizzes/"+quizID.String()+"/questions/"+questionID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "question deleted")
}

func TestQuizHandler_DeleteQuestion_InvalidQuizID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/quizzes/bad-id/questions/"+uuid.New().String(), nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_DeleteQuestion_InvalidQuestionID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	quizID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/quizzes/"+quizID.String()+"/questions/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- StartAttempt ---

func TestQuizHandler_StartAttempt_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()

	svc.EXPECT().StartAttempt(mock.Anything, uid, quizID).Return(
		&dto.StartAttemptResponse{
			Attempt:   domain.QuizAttempt{ID: uuid.New(), QuizID: quizID, UserID: uid, StartedAt: time.Now()},
			Questions: []dto.QuestionForAttempt{{ID: uuid.New(), QuestionText: "Q1"}},
		}, nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/attempts", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestQuizHandler_StartAttempt_InvalidQuizID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/quizzes/bad-id/attempts", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListAttempts ---

func TestQuizHandler_ListAttempts_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()

	svc.EXPECT().ListAttempts(mock.Anything, uid, quizID, 20, 0).Return(
		[]domain.QuizAttempt{{ID: uuid.New()}}, 1, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/quizzes/"+quizID.String()+"/attempts", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuizHandler_ListAttempts_InvalidQuizID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/quizzes/bad-id/attempts", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetAttempt ---

func TestQuizHandler_GetAttempt_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	attemptID := uuid.New()

	svc.EXPECT().GetAttempt(mock.Anything, uid, attemptID).Return(
		&dto.AttemptDetail{
			Attempt: domain.QuizAttempt{ID: attemptID, UserID: uid, StartedAt: time.Now()},
			Answers: []dto.QuizAnswerDetail{},
		}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/attempts/"+attemptID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuizHandler_GetAttempt_InvalidID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/attempts/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_GetAttempt_NotFound(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	attemptID := uuid.New()
	svc.EXPECT().GetAttempt(mock.Anything, uid, attemptID).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/attempts/"+attemptID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- SubmitAnswer ---

func TestQuizHandler_SubmitAnswer_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	attemptID := uuid.New()
	questionID := uuid.New()

	svc.EXPECT().SubmitAnswer(mock.Anything, uid, attemptID, mock.Anything).Return(
		&dto.AnswerResult{IsCorrect: true, CorrectAnswer: "A"}, nil,
	)

	body := `{"question_id":"` + questionID.String() + `","answer":"A"}`
	req := httptest.NewRequest(http.MethodPost, "/attempts/"+attemptID.String()+"/answers", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuizHandler_SubmitAnswer_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	attemptID := uuid.New()
	questionID := uuid.New()

	svc.EXPECT().SubmitAnswer(mock.Anything, uid, attemptID, mock.Anything).Return(nil, domain.ErrAlreadyAnswered)

	body := `{"question_id":"` + questionID.String() + `","answer":"A"}`
	req := httptest.NewRequest(http.MethodPost, "/attempts/"+attemptID.String()+"/answers", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_SubmitAnswer_InvalidAttemptID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	body := `{"question_id":"` + uuid.New().String() + `","answer":"A"}`
	req := httptest.NewRequest(http.MethodPost, "/attempts/bad-id/answers", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_SubmitAnswer_InvalidBody(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	attemptID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/attempts/"+attemptID.String()+"/answers", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_SubmitAnswer_ValidationError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	attemptID := uuid.New()
	body := `{"answer":"A"}` // missing question_id
	req := httptest.NewRequest(http.MethodPost, "/attempts/"+attemptID.String()+"/answers", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- CompleteAttempt ---

func TestQuizHandler_CompleteAttempt_Success(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	attemptID := uuid.New()
	now := time.Now()
	svc.EXPECT().CompleteAttempt(mock.Anything, uid, attemptID).Return(
		&domain.QuizAttempt{ID: attemptID, UserID: uid, CompletedAt: &now}, nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/attempts/"+attemptID.String()+"/complete", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuizHandler_CompleteAttempt_InvalidAttemptID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/attempts/bad-id/complete", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuizHandler_CompleteAttempt_NotFound(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	attemptID := uuid.New()
	svc.EXPECT().CompleteAttempt(mock.Anything, uid, attemptID).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodPost, "/attempts/"+attemptID.String()+"/complete", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- ListQuizzes: service error ---

func TestQuizHandler_ListQuizzes_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	svc.EXPECT().ListQuizzes(mock.Anything, uid, mock.Anything, mock.Anything).Return(nil, 0, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/quizzes", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- DeleteQuiz: service error ---

func TestQuizHandler_DeleteQuiz_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().DeleteQuiz(mock.Anything, uid, quizID).Return(domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodDelete, "/quizzes/"+quizID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- BatchAddQuestions: service error & invalid quizID ---

func TestQuizHandler_BatchAddQuestions_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().BatchAddQuestions(mock.Anything, uid, quizID, mock.Anything).Return(nil, assert.AnError)

	body := `{"questions":[{"question_type":"fill_blank","question_text":"Q?","correct_answer":"A"}]}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/questions/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestQuizHandler_BatchAddQuestions_InvalidQuizID(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	body := `{"questions":[{"question_type":"fill_blank","question_text":"Q?","correct_answer":"A"}]}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/not-a-uuid/questions/batch", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- StartAttempt: service error ---

func TestQuizHandler_StartAttempt_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().StartAttempt(mock.Anything, uid, quizID).Return(nil, domain.ErrQuizNotPublished)

	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/attempts", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.NotEqual(t, http.StatusOK, rec.Code)
}

// --- ListAttempts: service error ---

func TestQuizHandler_ListAttempts_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().ListAttempts(mock.Anything, uid, quizID, mock.Anything, mock.Anything).Return(nil, 0, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/quizzes/"+quizID.String()+"/attempts", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- UpdateQuiz: service error ---

func TestQuizHandler_UpdateQuiz_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().UpdateQuiz(mock.Anything, uid, quizID, mock.Anything).Return(nil, domain.ErrForbidden)

	body := `{"title":"New Title"}`
	req := httptest.NewRequest(http.MethodPut, "/quizzes/"+quizID.String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- AddQuestion: service error ---

func TestQuizHandler_AddQuestion_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	svc.EXPECT().AddQuestion(mock.Anything, uid, quizID, mock.Anything).Return(nil, assert.AnError)

	body := `{"question_type":"fill_blank","question_text":"Q?","correct_answer":"A"}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes/"+quizID.String()+"/questions", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- DeleteQuestion: service error ---

func TestQuizHandler_DeleteQuestion_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()
	svc.EXPECT().DeleteQuestion(mock.Anything, uid, quizID, questionID).Return(domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodDelete, "/quizzes/"+quizID.String()+"/questions/"+questionID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- CreateQuiz: service error ---

func TestQuizHandler_CreateQuiz_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	svc.EXPECT().CreateQuiz(mock.Anything, uid, mock.Anything).Return(nil, domain.ErrForbidden)

	body := `{"title":"My Quiz","quiz_type":"mcq"}`
	req := httptest.NewRequest(http.MethodPost, "/quizzes", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- CompleteAttempt: service error ---

func TestQuizHandler_CompleteAttempt_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	attemptID := uuid.New()
	svc.EXPECT().CompleteAttempt(mock.Anything, uid, attemptID).Return(nil, domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodPost, "/attempts/"+attemptID.String()+"/complete", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- UpdateQuestion: service error ---

func TestQuizHandler_UpdateQuestion_ServiceError(t *testing.T) {
	svc := mockport.NewMockQuizServicer(t)
	h := NewQuizHandler(svc)
	router := setupQuizRouter(h)

	uid := uuid.New()
	quizID := uuid.New()
	questionID := uuid.New()
	svc.EXPECT().UpdateQuestion(mock.Anything, uid, quizID, questionID, mock.Anything).Return(nil, domain.ErrForbidden)

	body := `{"question_text":"Updated?"}`
	req := httptest.NewRequest(http.MethodPut, "/quizzes/"+quizID.String()+"/questions/"+questionID.String(), strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
