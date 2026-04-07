package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type StudySession struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	DeckID          *uuid.UUID `json:"deck_id,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	NewCount        int        `json:"new_count"`
	ReviewCount     int        `json:"review_count"`
	RelearnCount    int        `json:"relearn_count"`
	TotalDurationMS int        `json:"total_duration_ms"`
}

type StudySessionRepository interface {
	Create(ctx context.Context, session *StudySession) error
	GetByID(ctx context.Context, id uuid.UUID) (*StudySession, error)
	Update(ctx context.Context, session *StudySession) error
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]StudySession, int, error)
}

type DailyStats struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	Date            time.Time `json:"date"`
	NewCards        int       `json:"new_cards"`
	Reviews         int       `json:"reviews"`
	Relearns        int       `json:"relearns"`
	TotalDurationMS int       `json:"total_duration_ms"`
	RetentionRate   *float64  `json:"retention_rate,omitempty"`
}

type LeaderboardEntry struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Streak      int       `json:"streak"`
	Rank        int       `json:"rank"`
}

type StatsRepository interface {
	UpsertDailyStats(ctx context.Context, stats *DailyStats) error
	GetDailyStats(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]DailyStats, error)
	GetStreak(ctx context.Context, userID uuid.UUID) (int, error)
	GetGlobalLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error)
}

type Media struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	CardID    *uuid.UUID `json:"card_id,omitempty"`
	FileName  string     `json:"file_name"`
	FileSize  int        `json:"file_size"`
	MimeType  string     `json:"mime_type"`
	R2Key     string     `json:"r2_key"`
	URL       string     `json:"url"`
	CreatedAt time.Time  `json:"created_at"`
}

type MediaRepository interface {
	Create(ctx context.Context, media *Media) error
	GetByID(ctx context.Context, id uuid.UUID) (*Media, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByCardID(ctx context.Context, cardID uuid.UUID) ([]Media, error)
}
