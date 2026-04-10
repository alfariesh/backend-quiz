package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/alfariesh/backend-quiz/internal/middleware"
)

func ctxWithUserID(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, userID)
}
