package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/repository/sqlc"
)

// UUID conversions

func uuidToNullable(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func nullableToUUID(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	id := uuid.UUID(p.Bytes)
	return &id
}

// Time conversions

func timeToNullable(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func nullableToTime(p pgtype.Timestamptz) *time.Time {
	if !p.Valid {
		return nil
	}
	return &p.Time
}

// Int conversions

func intToNullable(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

func nullableToInt(p pgtype.Int4) *int {
	if !p.Valid {
		return nil
	}
	v := int(p.Int32)
	return &v
}

// Card converters

func cardFromSqlc(c sqlc.Card) domain.Card {
	return domain.Card{
		ID:            c.ID,
		DeckID:        c.DeckID,
		Front:         c.Front,
		Back:          c.Back,
		ContentType:   c.ContentType,
		Tags:          c.Tags,
		IsSuspended:   c.IsSuspended,
		Position:      int(c.Position),
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
		Due:           c.Due,
		Stability:     float64(c.Stability),
		Difficulty:    float64(c.Difficulty),
		ElapsedDays:   int(c.ElapsedDays),
		ScheduledDays: int(c.ScheduledDays),
		Reps:          int(c.Reps),
		Lapses:        int(c.Lapses),
		State:         domain.CardState(c.State),
		LastReview:    nullableToTime(c.LastReview),
	}
}

func cardsFromSqlc(cards []sqlc.Card) []domain.Card {
	result := make([]domain.Card, len(cards))
	for i, c := range cards {
		result[i] = cardFromSqlc(c)
	}
	return result
}

// User converters

func userFromSqlc(u sqlc.User) domain.User {
	weights := make([]float64, len(u.FsrsWeights))
	for i, w := range u.FsrsWeights {
		weights[i] = float64(w)
	}
	return domain.User{
		ID:               u.ID,
		Email:            u.Email,
		PasswordHash:     u.PasswordHash,
		DisplayName:      u.DisplayName,
		Timezone:         u.Timezone,
		DesiredRetention: float64(u.DesiredRetention),
		DailyNewLimit:    int(u.DailyNewLimit),
		DailyReviewLimit: int(u.DailyReviewLimit),
		FSRSWeights:      weights,
		ReminderEnabled:  u.ReminderEnabled,
		ReminderTime:     pgTimeToString(u.ReminderTime),
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

func pgTimeToString(t pgtype.Time) string {
	if !t.Valid {
		return "08:00"
	}
	hours := t.Microseconds / 3_600_000_000
	minutes := (t.Microseconds % 3_600_000_000) / 60_000_000
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

// Deck converters

func deckFromSqlc(d sqlc.Deck) domain.Deck {
	return domain.Deck{
		ID:             d.ID,
		UserID:         d.UserID,
		Name:           d.Name,
		Description:    d.Description,
		IsArchived:     d.IsArchived,
		NewCardsPerDay: nullableToInt(d.NewCardsPerDay),
		Position:       int(d.Position),
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

// StudySession converters

func sessionFromSqlc(s sqlc.StudySession) domain.StudySession {
	return domain.StudySession{
		ID:              s.ID,
		UserID:          s.UserID,
		DeckID:          nullableToUUID(s.DeckID),
		StartedAt:       s.StartedAt,
		EndedAt:         nullableToTime(s.EndedAt),
		NewCount:        int(s.NewCount),
		ReviewCount:     int(s.ReviewCount),
		RelearnCount:    int(s.RelearnCount),
		TotalDurationMS: int(s.TotalDurationMs),
	}
}

// ReviewLog converters

func reviewLogFromSqlc(r sqlc.ReviewLog) domain.ReviewLog {
	return domain.ReviewLog{
		ID:            r.ID,
		CardID:        r.CardID,
		UserID:        r.UserID,
		Rating:        domain.Rating(r.Rating),
		State:         domain.CardState(r.State),
		ScheduledDays: int(r.ScheduledDays),
		ElapsedDays:   int(r.ElapsedDays),
		Stability:     float64(r.Stability),
		Difficulty:    float64(r.Difficulty),
		DurationMS:    int(r.DurationMs),
		Source:        r.Source,
		ReviewedAt:    r.ReviewedAt,
	}
}

// Quiz converters

func quizFromSqlc(q sqlc.Quiz) domain.Quiz {
	return domain.Quiz{
		ID:               q.ID,
		UserID:           q.UserID,
		DeckID:           nullableToUUID(q.DeckID),
		Title:            q.Title,
		Description:      q.Description,
		QuizType:         q.QuizType,
		TimeLimitSeconds: nullableToInt(q.TimeLimitSeconds),
		ShuffleQuestions: q.ShuffleQuestions,
		IsPublished:      q.IsPublished,
		CreatedAt:        q.CreatedAt,
		UpdatedAt:        q.UpdatedAt,
	}
}

// QuizQuestion converters

func questionFromSqlc(q sqlc.QuizQuestion) domain.QuizQuestion {
	return domain.QuizQuestion{
		ID:            q.ID,
		QuizID:        q.QuizID,
		CardID:        nullableToUUID(q.CardID),
		QuestionType:  q.QuestionType,
		QuestionText:  q.QuestionText,
		Options:       json.RawMessage(q.Options),
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   q.Explanation,
		Position:      int(q.Position),
		Points:        int(q.Points),
		CreatedAt:     q.CreatedAt,
		UpdatedAt:     q.UpdatedAt,
	}
}

// QuizAttempt converters

func attemptFromSqlc(a sqlc.QuizAttempt) domain.QuizAttempt {
	return domain.QuizAttempt{
		ID:             a.ID,
		QuizID:         a.QuizID,
		UserID:         a.UserID,
		StartedAt:      a.StartedAt,
		CompletedAt:    nullableToTime(a.CompletedAt),
		Score:          int(a.Score),
		TotalPoints:    int(a.TotalPoints),
		TotalQuestions: int(a.TotalQuestions),
		CorrectCount:   int(a.CorrectCount),
		DurationMS:     int(a.DurationMs),
		CreatedAt:      a.CreatedAt,
	}
}

// QuizAnswer converters

func answerFromSqlc(a sqlc.QuizAnswer) domain.QuizAnswer {
	return domain.QuizAnswer{
		ID:           a.ID,
		AttemptID:    a.AttemptID,
		QuestionID:   a.QuestionID,
		UserAnswer:   a.UserAnswer,
		IsCorrect:    a.IsCorrect,
		PointsEarned: int(a.PointsEarned),
		DurationMS:   int(a.DurationMs),
		AnsweredAt:   a.AnsweredAt,
	}
}

// Media converters

func mediaFromSqlc(m sqlc.Medium) domain.Media {
	return domain.Media{
		ID:        m.ID,
		UserID:    m.UserID,
		CardID:    nullableToUUID(m.CardID),
		FileName:  m.FileName,
		FileSize:  int(m.FileSize),
		MimeType:  m.MimeType,
		R2Key:     m.R2Key,
		URL:       m.Url,
		CreatedAt: m.CreatedAt,
	}
}

// DeckShare converters

func deckShareFromSqlc(s sqlc.DeckShare) domain.DeckShare {
	return domain.DeckShare{
		ID:        s.ID,
		DeckID:    s.DeckID,
		ShareCode: s.ShareCode,
		IsPublic:  s.IsPublic,
		CreatedAt: s.CreatedAt,
	}
}

// StudyGoal converters

func goalFromSqlc(g sqlc.StudyGoal) domain.StudyGoal {
	return domain.StudyGoal{
		ID:          g.ID,
		UserID:      g.UserID,
		GoalType:    g.GoalType,
		TargetValue: int(g.TargetValue),
		IsActive:    g.IsActive,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}
