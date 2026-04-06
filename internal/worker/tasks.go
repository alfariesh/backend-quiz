package worker

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TaskDailyStatsAggregate = "stats:daily_aggregate"
	TaskDeckStatsRefresh    = "deck:stats_refresh"
)

type DailyStatsPayload struct {
	UserID uuid.UUID `json:"user_id"`
	Date   string    `json:"date"`
}

type DeckStatsPayload struct {
	DeckID uuid.UUID `json:"deck_id"`
}

func NewDailyStatsTask(userID uuid.UUID, date string) (*asynq.Task, error) {
	payload, err := json.Marshal(DailyStatsPayload{UserID: userID, Date: date})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskDailyStatsAggregate, payload), nil
}

func NewDeckStatsTask(deckID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(DeckStatsPayload{DeckID: deckID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskDeckStatsRefresh, payload), nil
}
