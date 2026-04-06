package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

type TaskHandlers struct {
	logger *slog.Logger
}

func NewTaskHandlers(logger *slog.Logger) *TaskHandlers {
	return &TaskHandlers{logger: logger}
}

func (h *TaskHandlers) HandleDailyStatsAggregate(ctx context.Context, t *asynq.Task) error {
	var payload DailyStatsPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	h.logger.Info("aggregating daily stats",
		slog.String("user_id", payload.UserID.String()),
		slog.String("date", payload.Date),
	)

	// TODO: Implement daily stats aggregation
	return nil
}

func (h *TaskHandlers) HandleDeckStatsRefresh(ctx context.Context, t *asynq.Task) error {
	var payload DeckStatsPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	h.logger.Info("refreshing deck stats",
		slog.String("deck_id", payload.DeckID.String()),
	)

	// TODO: Implement deck stats refresh
	return nil
}

func (h *TaskHandlers) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskDailyStatsAggregate, h.HandleDailyStatsAggregate)
	mux.HandleFunc(TaskDeckStatsRefresh, h.HandleDeckStatsRefresh)
}
