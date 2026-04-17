package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserDataExportRepository returns a per-table dump of user-owned data.
// Format is map[table_name] -> slice of row maps, keeping the shape generic so
// new tables automatically show up once their query is registered below.
type UserDataExportRepository struct {
	pool *pgxpool.Pool
}

func NewUserDataExportRepository(pool *pgxpool.Pool) *UserDataExportRepository {
	return &UserDataExportRepository{pool: pool}
}

// tables we dump per-user. password_hash and token_hash columns are explicitly
// excluded — we export the data the user provided, not our secret material.
var exportQueries = []struct {
	name string
	sql  string
}{
	{"user", `SELECT id, email, display_name, timezone, desired_retention,
		daily_new_limit, daily_review_limit, fsrs_weights, reminder_enabled,
		reminder_time, email_verified_at, deletion_requested_at, created_at, updated_at
		FROM users WHERE id = $1`},

	{"oauth_accounts", `SELECT id, provider, provider_id, email, avatar_url, created_at
		FROM oauth_accounts WHERE user_id = $1 ORDER BY created_at`},

	{"decks", `SELECT id, name, description, is_archived, new_cards_per_day, position, created_at, updated_at
		FROM decks WHERE user_id = $1 ORDER BY created_at`},

	{"deck_shares", `SELECT s.id, s.deck_id, s.share_code, s.is_public, s.created_at
		FROM deck_shares s JOIN decks d ON d.id = s.deck_id WHERE d.user_id = $1 ORDER BY s.created_at`},

	{"cards", `SELECT c.id, c.deck_id, c.front, c.back, c.tags, c.content_type, c.state,
		c.due, c.stability, c.difficulty, c.elapsed_days, c.scheduled_days, c.reps, c.lapses,
		c.last_review, c.is_suspended, c.position, c.created_at, c.updated_at
		FROM cards c JOIN decks d ON d.id = c.deck_id WHERE d.user_id = $1 ORDER BY c.created_at`},

	{"review_logs", `SELECT id, card_id, rating, state, elapsed_days, scheduled_days,
		stability, difficulty, duration_ms, reviewed_at, source
		FROM review_logs WHERE user_id = $1 ORDER BY reviewed_at`},

	{"study_sessions", `SELECT id, deck_id, started_at, ended_at, new_count, review_count, relearn_count, total_duration_ms
		FROM study_sessions WHERE user_id = $1 ORDER BY started_at`},

	{"study_goals", `SELECT id, goal_type, target_value, is_active, created_at, updated_at
		FROM study_goals WHERE user_id = $1 ORDER BY created_at`},

	{"daily_stats", `SELECT id, date, new_cards, reviews, relearns, total_duration_ms, retention_rate
		FROM daily_stats WHERE user_id = $1 ORDER BY date`},

	{"quizzes", `SELECT id, deck_id, title, description, quiz_type, time_limit_seconds,
		shuffle_questions, is_published, created_at, updated_at
		FROM quizzes WHERE user_id = $1 ORDER BY created_at`},

	{"quiz_questions", `SELECT q.id, q.quiz_id, q.card_id, q.question_type, q.question_text,
		q.options, q.correct_answer, q.explanation, q.position, q.points, q.created_at, q.updated_at
		FROM quiz_questions q JOIN quizzes z ON z.id = q.quiz_id WHERE z.user_id = $1 ORDER BY q.created_at`},

	{"quiz_attempts", `SELECT id, quiz_id, started_at, completed_at, score, total_points,
		total_questions, correct_count, duration_ms, created_at
		FROM quiz_attempts WHERE user_id = $1 ORDER BY started_at`},

	{"quiz_answers", `SELECT a.id, a.attempt_id, a.question_id, a.user_answer, a.is_correct,
		a.points_earned, a.duration_ms, a.answered_at
		FROM quiz_answers a JOIN quiz_attempts t ON t.id = a.attempt_id WHERE t.user_id = $1 ORDER BY a.answered_at`},

	{"media", `SELECT id, card_id, file_name, file_size, mime_type, r2_key, url, created_at
		FROM media WHERE user_id = $1 ORDER BY created_at`},

	{"auth_audit_logs", `SELECT id, event, ip_address, user_agent, metadata, created_at
		FROM auth_audit_logs WHERE user_id = $1 ORDER BY created_at`},
}

func (r *UserDataExportRepository) Dump(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	out := make(map[string]any, len(exportQueries))
	for _, q := range exportQueries {
		rows, err := r.pool.Query(ctx, q.sql, userID)
		if err != nil {
			return nil, err
		}
		collected, err := rowsToMaps(rows)
		if err != nil {
			return nil, err
		}
		// Single-row tables (user) are flattened to an object for convenience.
		if q.name == "user" && len(collected) == 1 {
			out[q.name] = collected[0]
		} else {
			out[q.name] = collected
		}
	}
	return out, nil
}

func rowsToMaps(rows pgx.Rows) ([]map[string]any, error) {
	defer rows.Close()
	fds := rows.FieldDescriptions()
	names := make([]string, len(fds))
	for i, fd := range fds {
		names[i] = string(fd.Name)
	}
	var out []map[string]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(vals))
		for i, v := range vals {
			row[names[i]] = v
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
