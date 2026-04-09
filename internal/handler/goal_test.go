package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	mockport "github.com/rekanesiads/backend-quiz/internal/mocks/port"
)

func setupGoalRouter(h *GoalHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/goals", h.SetGoal)
	r.Get("/goals", h.ListWithProgress)
	r.Delete("/goals/{goalID}", h.DeleteGoal)
	return r
}

// --- SetGoal ---

func TestGoalHandler_SetGoal_Success(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	uid := uuid.New()
	svc.EXPECT().SetGoal(mock.Anything, uid, mock.Anything).Return(
		&domain.StudyGoal{ID: uuid.New(), UserID: uid, GoalType: "daily_reviews", TargetValue: 50}, nil,
	)

	body := `{"goal_type":"daily_reviews","target_value":50}`
	req := httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestGoalHandler_SetGoal_InvalidBody(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(`{bad`))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGoalHandler_SetGoal_ValidationError(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	body := `{"goal_type":"invalid","target_value":50}`
	req := httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(body))
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListWithProgress ---

func TestGoalHandler_ListWithProgress_Success(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	uid := uuid.New()
	svc.EXPECT().ListWithProgress(mock.Anything, uid).Return(
		[]dto.GoalProgress{{
			Goal:         domain.StudyGoal{GoalType: "daily_reviews", TargetValue: 50},
			CurrentValue: 20, Completed: false, Percent: 40,
		}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/goals", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGoalHandler_ListWithProgress_Empty(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	uid := uuid.New()
	svc.EXPECT().ListWithProgress(mock.Anything, uid).Return([]dto.GoalProgress{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/goals", nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "[]")
}

// --- DeleteGoal ---

func TestGoalHandler_DeleteGoal_Success(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	uid := uuid.New()
	goalID := uuid.New()

	svc.EXPECT().DeleteGoal(mock.Anything, uid, goalID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/goals/"+goalID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "goal deleted")
}

func TestGoalHandler_DeleteGoal_InvalidID(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/goals/bad-id", nil)
	req = req.WithContext(ctxWithUserID(uuid.New()))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGoalHandler_DeleteGoal_NotFound(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	uid := uuid.New()
	goalID := uuid.New()

	svc.EXPECT().DeleteGoal(mock.Anything, uid, goalID).Return(domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/goals/"+goalID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGoalHandler_DeleteGoal_Forbidden(t *testing.T) {
	svc := mockport.NewMockGoalServicer(t)
	h := NewGoalHandler(svc)
	router := setupGoalRouter(h)

	uid := uuid.New()
	goalID := uuid.New()

	svc.EXPECT().DeleteGoal(mock.Anything, uid, goalID).Return(domain.ErrForbidden)

	req := httptest.NewRequest(http.MethodDelete, "/goals/"+goalID.String(), nil)
	req = req.WithContext(ctxWithUserID(uid))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
