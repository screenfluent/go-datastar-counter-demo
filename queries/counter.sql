-- name: GetCounter :one
SELECT id, value, updated_at
FROM counters
WHERE id = 1;

-- name: ChangeCounter :one
UPDATE counters
SET value = value + sqlc.arg(delta),
    updated_at = now()
WHERE id = 1
  AND value + sqlc.arg(delta) >= 0
RETURNING id, value, updated_at;

-- name: ResetCounter :one
UPDATE counters
SET value = 0,
    updated_at = now()
WHERE id = 1
RETURNING id, value, updated_at;

