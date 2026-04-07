-- name: CreateCard :one
INSERT INTO cards (deck_id, front, back, content_type, tags, position)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCardByID :one
SELECT * FROM cards WHERE id = $1;

-- name: ListCardsByDeckID :many
SELECT * FROM cards
WHERE deck_id = @deck_id
    AND (sqlc.narg('state')::smallint IS NULL OR state = sqlc.narg('state'))
    AND (sqlc.narg('tag')::text IS NULL OR sqlc.narg('tag') = ANY(tags))
    AND (sqlc.narg('query')::text IS NULL OR front ILIKE '%' || sqlc.narg('query') || '%' OR back ILIKE '%' || sqlc.narg('query') || '%')
ORDER BY position, created_at
LIMIT @row_limit OFFSET @row_offset;

-- name: CountCardsByDeckID :one
SELECT COUNT(*)::int FROM cards
WHERE deck_id = @deck_id
    AND (sqlc.narg('state')::smallint IS NULL OR state = sqlc.narg('state'))
    AND (sqlc.narg('tag')::text IS NULL OR sqlc.narg('tag') = ANY(tags))
    AND (sqlc.narg('query')::text IS NULL OR front ILIKE '%' || sqlc.narg('query') || '%' OR back ILIKE '%' || sqlc.narg('query') || '%');

-- name: UpdateCard :exec
UPDATE cards SET
    front = $2,
    back = $3,
    content_type = $4,
    tags = $5,
    updated_at = now()
WHERE id = $1;

-- name: UpdateCardFSRS :exec
UPDATE cards SET
    due = $2,
    stability = $3,
    difficulty = $4,
    elapsed_days = $5,
    scheduled_days = $6,
    reps = $7,
    lapses = $8,
    state = $9,
    last_review = $10,
    updated_at = now()
WHERE id = $1;

-- name: DeleteCard :exec
DELETE FROM cards WHERE id = $1;

-- name: SetCardSuspended :exec
UPDATE cards SET is_suspended = $2, updated_at = now() WHERE id = $1;

-- name: ResetCardFSRS :exec
UPDATE cards SET
    due = now(), stability = 0, difficulty = 0,
    elapsed_days = 0, scheduled_days = 0, reps = 0, lapses = 0,
    state = 0, last_review = NULL, updated_at = now()
WHERE id = $1;

-- name: GetDueCards :many
SELECT * FROM cards
WHERE deck_id = $1
    AND NOT is_suspended
    AND (
        (state = 0)
        OR (state IN (1, 3) AND due <= $2)
        OR (state = 2 AND due <= $2)
    )
ORDER BY
    CASE WHEN state IN (1, 3) THEN 0
         WHEN state = 2 THEN 1
         WHEN state = 0 THEN 2
    END,
    due ASC
LIMIT $3;

-- name: CountCardsByState :many
SELECT state, COUNT(*)::int AS count FROM cards
WHERE deck_id = $1 AND NOT is_suspended
GROUP BY state;

-- name: CountDueCards :one
SELECT COUNT(*)::int FROM cards
WHERE deck_id = $1
    AND NOT is_suspended
    AND state IN (1, 2, 3)
    AND due <= $2;

-- name: GetDeckMasteryStats :one
SELECT
    COUNT(*)::int AS total_cards,
    COUNT(*) FILTER (WHERE state = 2)::int AS mature_cards,
    COALESCE(AVG(stability), 0)::real AS avg_stability
FROM cards
WHERE deck_id = $1 AND NOT is_suspended;

-- name: GetWeakCards :many
SELECT c.* FROM cards c
JOIN decks d ON d.id = c.deck_id
WHERE d.user_id = $1 AND NOT c.is_suspended
    AND (c.lapses > 2 OR (c.stability < 5 AND c.reps > 0))
ORDER BY c.lapses DESC, c.stability ASC
LIMIT $2;
