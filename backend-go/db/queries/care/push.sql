-- Web Push for medication reminders (goose 00121): subscriptions and the doses already pushed.

-- name: UpsertPushSubscription :exec
-- One row per browser endpoint; a browser that changes account moves to the new user.
INSERT INTO `push_subscriptions` (user_id, endpoint, endpoint_hash, p256dh, auth, locale, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(endpoint), sqlc.arg(endpoint_hash), sqlc.arg(p256dh), sqlc.arg(auth),
        sqlc.arg(locale), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  user_id = VALUES(user_id), endpoint = VALUES(endpoint), p256dh = VALUES(p256dh), auth = VALUES(auth),
  locale = VALUES(locale), updated_at = VALUES(updated_at);

-- name: CountPushSubscriptions :one
SELECT COUNT(*) FROM `push_subscriptions` WHERE user_id = ?;

-- name: DeleteOldestPushSubscription :exec
DELETE FROM `push_subscriptions` WHERE user_id = ? ORDER BY updated_at ASC, id ASC LIMIT 1;

-- name: DeletePushSubscription :execrows
DELETE FROM `push_subscriptions` WHERE user_id = ? AND endpoint_hash = ?;

-- name: DeletePushSubscriptionByID :exec
-- A push service answered 404 / 410: the subscription is gone.
DELETE FROM `push_subscriptions` WHERE id = ?;

-- name: ListPushSubscriptions :many
SELECT * FROM `push_subscriptions` WHERE user_id = ? ORDER BY id;

-- name: ListPushableMedications :many
-- Active medications of every user with at least one push subscription (the sender's working set).
SELECT r.* FROM `reminders` r
WHERE r.`type` = 'medication' AND r.is_active = 1
  AND r.user_id IN (SELECT user_id FROM `push_subscriptions`)
ORDER BY r.user_id, r.id;

-- name: ClaimReminderPush :execrows
-- 1 = this process pushes the dose; 0 = another one already did.
INSERT IGNORE INTO `reminder_pushes` (user_id, reminder_id, intake_date, slot, sent_at)
VALUES (?, ?, ?, ?, ?);

-- name: PruneReminderPushes :exec
DELETE FROM `reminder_pushes` WHERE sent_at < ?;

-- name: ListIntakesBetween :many
-- The doses the user ticked in a date range (GET /care/upcoming).
SELECT reminder_id, intake_date, slot FROM `reminder_intakes`
WHERE user_id = ? AND intake_date BETWEEN sqlc.arg(from_date) AND sqlc.arg(to_date);

-- name: ListUnpushedNotifications :many
-- Inbox rows the push sender has not handled yet (goose 00123), oldest first, after the cursor.
SELECT id, user_id, `type`, title, body, action_url, read_at, created_at
FROM `user_notifications`
WHERE pushed_at IS NULL AND id > sqlc.arg(after_id)
ORDER BY id
LIMIT ?;

-- name: ClaimNotificationPush :execrows
-- 1 = this process handles the row (pushes it or leaves it silent); 0 = another one already did.
UPDATE `user_notifications` SET pushed_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND pushed_at IS NULL;
