package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/service"
	"github.com/rekanesiads/backend-quiz/pkg/pagination"
	"github.com/rekanesiads/backend-quiz/pkg/validate"
)

type QuizHandler struct {
	quizSvc *service.QuizService
}

func NewQuizHandler(quizSvc *service.QuizService) *QuizHandler {
	return &QuizHandler{quizSvc: quizSvc}
}

func (h *QuizHandler) CreateQuiz(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req service.CreateQuizRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	quiz, err := h.quizSvc.CreateQuiz(r.Context(), userID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, quiz)
}

func (h *QuizHandler) ListQuizzes(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	p := ParsePagination(r)

	quizzes, total, err := h.quizSvc.ListQuizzes(r.Context(), userID, p.Limit, p.Offset)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, pagination.NewResponse(quizzes, total, p.Limit, p.Offset))
}

func (h *QuizHandler) GetQuiz(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}

	detail, err := h.quizSvc.GetQuiz(r.Context(), userID, quizID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, detail)
}

func (h *QuizHandler) UpdateQuiz(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}

	var req service.UpdateQuizRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	quiz, err := h.quizSvc.UpdateQuiz(r.Context(), userID, quizID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, quiz)
}

func (h *QuizHandler) DeleteQuiz(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}

	if err := h.quizSvc.DeleteQuiz(r.Context(), userID, quizID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "quiz deleted")
}

// Question handlers

func (h *QuizHandler) AddQuestion(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}

	var req service.AddQuestionRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	question, err := h.quizSvc.AddQuestion(r.Context(), userID, quizID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, question)
}

func (h *QuizHandler) BatchAddQuestions(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}

	var req service.BatchAddQuestionsRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	questions, err := h.quizSvc.BatchAddQuestions(r.Context(), userID, quizID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, questions)
}

func (h *QuizHandler) GenerateFromDeck(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}

	var req service.GenerateFromDeckRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	questions, err := h.quizSvc.GenerateFromDeck(r.Context(), userID, quizID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, questions)
}

func (h *QuizHandler) GenerateAyatQuiz(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}

	var req service.GenerateAyatQuizRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	questions, err := h.quizSvc.GenerateAyatQuiz(r.Context(), userID, quizID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, questions)
}

func (h *QuizHandler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}
	questionID, err := uuid.Parse(chi.URLParam(r, "questionID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid question id")
		return
	}

	var req service.UpdateQuestionRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	question, err := h.quizSvc.UpdateQuestion(r.Context(), userID, quizID, questionID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, question)
}

func (h *QuizHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}
	questionID, err := uuid.Parse(chi.URLParam(r, "questionID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid question id")
		return
	}

	if err := h.quizSvc.DeleteQuestion(r.Context(), userID, quizID, questionID); err != nil {
		HandleError(w, err)
		return
	}
	JSONMessage(w, http.StatusOK, "question deleted")
}

// Attempt handlers

func (h *QuizHandler) StartAttempt(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}

	resp, err := h.quizSvc.StartAttempt(r.Context(), userID, quizID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusCreated, resp)
}

func (h *QuizHandler) ListAttempts(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	quizID, err := uuid.Parse(chi.URLParam(r, "quizID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid quiz id")
		return
	}
	p := ParsePagination(r)

	attempts, total, err := h.quizSvc.ListAttempts(r.Context(), userID, quizID, p.Limit, p.Offset)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, pagination.NewResponse(attempts, total, p.Limit, p.Offset))
}

func (h *QuizHandler) GetAttempt(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	attemptID, err := uuid.Parse(chi.URLParam(r, "attemptID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid attempt id")
		return
	}

	detail, err := h.quizSvc.GetAttempt(r.Context(), userID, attemptID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, detail)
}

func (h *QuizHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	attemptID, err := uuid.Parse(chi.URLParam(r, "attemptID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid attempt id")
		return
	}

	var req service.SubmitAnswerRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.quizSvc.SubmitAnswer(r.Context(), userID, attemptID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, result)
}

func (h *QuizHandler) CompleteAttempt(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	attemptID, err := uuid.Parse(chi.URLParam(r, "attemptID"))
	if err != nil {
		JSONError(w, http.StatusBadRequest, "invalid attempt id")
		return
	}

	attempt, err := h.quizSvc.CompleteAttempt(r.Context(), userID, attemptID)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, attempt)
}
