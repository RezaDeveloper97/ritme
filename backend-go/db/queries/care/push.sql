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
-- 1 = this process pushes the dose; 0 = another one already did (or released it: ReclaimReminderPush).
INSERT IGNORE INTO `reminder_pushes` (user_id, reminder_id, intake_date, slot, sent_at)
VALUES (?, ?, ?, ?, ?);

-- Durable retry (T-M12-22b, internal/care/push_retry.go): a dose no browser accepted because of a passing failure
-- is released — its sent_at becomes the marker releaseEpoch + <failed tries> seconds (year 2000, before any real
-- claim) — so any process takes it again on a later tick with a compare-and-set on that exact marker.

-- name: ListReleasedReminderPushes :many
-- The released doses among the reminders due in this tick (the unique key's reminder_id prefix).
SELECT reminder_id, intake_date, slot, sent_at FROM `reminder_pushes`
WHERE reminder_id IN (sqlc.slice(reminder_ids)) AND intake_date >= sqlc.arg(from_date)
  AND sent_at < sqlc.arg(released_before);

-- name: ReclaimReminderPush :execrows
-- 1 = this process tries the released dose again; 0 = another one took it first.
UPDATE `reminder_pushes` SET sent_at = sqlc.arg(now)
WHERE reminder_id = sqlc.arg(reminder_id) AND intake_date = sqlc.arg(intake_date) AND slot = sqlc.arg(slot)
  AND sent_at = sqlc.arg(released_at);

-- name: ReleaseReminderPush :exec
-- The claiming process gives the dose back after a passing failure (released_at = the marker of its tries).
UPDATE `reminder_pushes` SET sent_at = sqlc.arg(released_at)
WHERE reminder_id = sqlc.arg(reminder_id) AND intake_date = sqlc.arg(intake_date) AND slot = sqlc.arg(slot);

-- name: PruneReminderPushes :exec
-- Old claims; a released marker is old by construction, so it waits until its dose day is past every lookback.
DELETE FROM `reminder_pushes` WHERE sent_at < sqlc.arg(sent_before) AND intake_date < sqlc.arg(day_before);

-- name: ListIntakesBetween :many
-- The doses the user ticked in a date range (GET /care/upcoming).
SELECT reminder_id, intake_date, slot FROM `reminder_intakes`
WHERE user_id = ? AND intake_date BETWEEN sqlc.arg(from_date) AND sqlc.arg(to_date);

-- name: ListUnpushedNotifications :many
-- Inbox rows the push sender has not handled yet (goose 00123) or released after a passing failure (pushed_at =
-- the year-2000 marker, T-M12-22b), oldest first, after the cursor.
SELECT id, user_id, `type`, title, body, action_url, read_at, created_at, pushed_at
FROM `user_notifications`
WHERE (pushed_at IS NULL OR pushed_at < sqlc.arg(released_before)) AND id > sqlc.arg(after_id)
ORDER BY id
LIMIT ?;

-- name: ClaimNotificationPush :execrows
-- 1 = this process handles the row (pushes it or leaves it silent); 0 = another one already did.
UPDATE `user_notifications` SET pushed_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND pushed_at IS NULL;

-- name: ReclaimNotificationPush :execrows
-- 1 = this process handles the released row again; 0 = another one took it first.
UPDATE `user_notifications` SET pushed_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND pushed_at = sqlc.arg(released_at);

-- name: ReleaseNotificationPush :exec
-- The claiming process gives the row back after a passing failure (released_at = the marker of its tries).
UPDATE `user_notifications` SET pushed_at = sqlc.arg(released_at) WHERE id = sqlc.arg(id);
