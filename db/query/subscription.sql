-- name: GetSubscription :one
SELECT * FROM subscriptions
WHERE id = ? LIMIT 1;

-- name: GetSubscriptionByUniqueKey :one
SELECT * FROM subscriptions
WHERE topic_id = ? AND consumer_url = ? LIMIT 1;

-- name: ListSubscriptions :many
SELECT * FROM subscriptions
ORDER BY created_at DESC;

-- name: ListActiveSubscriptionsForTopic :many
SELECT * FROM subscriptions
WHERE topic_id = ? AND is_active = 1
ORDER BY created_at DESC;

-- name: InsertSubscription :one
INSERT INTO subscriptions (
    id,
    topic_id,
    consumer_url,
    secret_key,
    is_active
) VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteAllSubscriptions :exec
DELETE FROM subscriptions;