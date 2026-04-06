-- name: CreateDeck :one
INSERT INTO decks (user_id, name, description, new_cards_per_day, position)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDeckByID :one
SELECT * FROM decks WHERE id = $1;

-- name: ListDecksByUserID :many
SELECT d.*,
    COALESCE(card_counts.total, 0)::int AS card_count,
    COALESCE(card_counts.new_count, 0)::int AS new_count,
    COALESCE(card_counts.due_count, 0)::int AS due_count,
    COALESCE(card_counts.learn_count, 0)::int AS learn_count,
    COALESCE(card_counts.relearn_count, 0)::int AS relearn_count
FROM decks d
LEFT JOIN LATERAL (
    SELECT
        COUNT(*)::int AS total,
        COUNT(*) FILTER (WHERE c.state = 0 AND NOT c.is_suspended)::int AS new_count,
        COUNT(*) FILTER (WHERE c.state = 2 AND c.due <= now() AND NOT c.is_suspended)::int AS due_count,
        COUNT(*) FILTER (WHERE c.state = 1 AND NOT c.is_suspended)::int AS learn_count,
        COUNT(*) FILTER (WHERE c.state = 3 AND NOT c.is_suspended)::int AS relearn_count
    FROM cards c WHERE c.deck_id = d.id
) card_counts ON true
WHERE d.user_id = $1
ORDER BY d.position, d.created_at
LIMIT $2 OFFSET $3;

-- name: CountDecksByUserID :one
SELECT COUNT(*)::int FROM decks WHERE user_id = $1;

-- name: UpdateDeck :one
UPDATE decks SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    is_archived = COALESCE(sqlc.narg('is_archived'), is_archived),
    new_cards_per_day = COALESCE(sqlc.narg('new_cards_per_day'), new_cards_per_day),
    position = COALESCE(sqlc.narg('position'), position),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteDeck :exec
DELETE FROM decks WHERE id = $1;

-- name: CreateDeckShare :one
INSERT INTO deck_shares (deck_id, share_code, is_public)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetDeckShareByDeckID :one
SELECT * FROM deck_shares WHERE deck_id = $1;

-- name: GetDeckShareByCode :one
SELECT * FROM deck_shares WHERE share_code = $1;

-- name: DeleteDeckShare :exec
DELETE FROM deck_shares WHERE deck_id = $1;

-- name: ListPublicDecks :many
SELECT d.*,
    COALESCE(card_counts.total, 0)::int AS card_count,
    0::int AS new_count,
    0::int AS due_count,
    0::int AS learn_count,
    0::int AS relearn_count
FROM decks d
INNER JOIN deck_shares ds ON ds.deck_id = d.id AND ds.is_public = true
LEFT JOIN LATERAL (
    SELECT COUNT(*)::int AS total
    FROM cards c WHERE c.deck_id = d.id
) card_counts ON true
WHERE ($1::text = '' OR d.name ILIKE '%' || $1 || '%')
ORDER BY d.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPublicDecks :one
SELECT COUNT(*)::int
FROM decks d
INNER JOIN deck_shares ds ON ds.deck_id = d.id AND ds.is_public = true
WHERE ($1::text = '' OR d.name ILIKE '%' || $1 || '%');
