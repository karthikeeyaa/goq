-- name: CreateSubscription :one
INSERT INTO subscriptions (
    id,
    operation_name,
    consumer_url,
    secret_key,
    is_active
) VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSubscription :one
SELECT * FROM subscriptions
WHERE id = ? LIMIT 1;

-- name: GetSubscriptionByUniqueKey :one
SELECT * FROM subscriptions
WHERE operation_name = ? AND consumer_url = ? LIMIT 1;

-- name: ListSubscriptions :many
SELECT * FROM subscriptions
ORDER BY created_at DESC;

-- name: ListActiveSubscriptionsForOperation :many
SELECT * FROM subscriptions
WHERE operation_name = ? AND is_active = 1
ORDER BY created_at DESC;

-- name: UpdateSubscription :one
UPDATE subscriptions
SET consumer_url = ?,
    secret_key = ?,
    is_active = ?
WHERE id = ?
RETURNING *;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions
WHERE id = ?;
