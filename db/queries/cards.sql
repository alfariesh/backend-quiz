-- name: CreateCard :one
INSERT INTO cards (deck_id, front, back, tags, position)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCardByID :one
SELECT * FROM cards WHERE id = $1;

-- name: ListCardsByDeckID :many
SELECT * FROM cards
WHERE deck_id = $1
    AND ($2::smallint IS NULL OR state = $2)
    AND ($3::text = '' OR $3 = ANY(tags))
    AND ($4::text = '' OR front ILIKE '%' || $4 || '%' OR back ILIKE '%' || $4 || '%')
ORDER BY position, created_at
LIMIT $5 OFFSET $6;

-- name: CountCardsByDeckID :one
SELECT COUNT(*)::int FROM cards
WHERE deck_id = $1
    AND ($2::smallint IS NULL OR state = $2)
    AND ($3::text = '' OR $3 = ANY(tags))
    AND ($4::text = '' OR front ILIKE '%' || $4 || '%' OR back ILIKE '%' || $4 || '%');

-- name: UpdateCard :one
UPDATE cards SET
    front = COALESCE(sqlc.narg('front'), front),
    back = COALESCE(sqlc.narg('back'), back),
    tags = COALESCE(sqlc.narg('tags'), tags),
    updated_at = now()
WHERE id = $1
RETURNING *;

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
    due = now(),
    stability = 0,
    difficulty = 0,
    elapsed_days = 0,
    scheduled_days = 0,
    reps = 0,
    lapses = 0,
    state = 0,
    last_review = NULL,
    updated_at = now()
WHERE id = $1;

-- name: GetDueCards :many
SELECT * FROM cards
WHERE deck_id = $1
    AND NOT is_suspended
    AND (
        (state = 0 AND position <= $4)
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

-- name: GetCardsByDeckIDForExport :many
SELECT * FROM cards WHERE deck_id = $1 ORDER BY position;

-- name: GetMaxPosition :one
SELECT COALESCE(MAX(position), 0)::int FROM cards WHERE deck_id = $1;
