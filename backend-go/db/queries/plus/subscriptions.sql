-- Trials and subscription periods. Every query is scoped by user_id (IDOR).

-- name: GetTrial :one
SELECT * FROM plus_trials
WHERE user_id = ?;

-- name: InsertTrial :exec
-- UNIQUE(user_id) makes a second (or concurrent) start fail with 1062.
INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: HasAnySubscription :one
SELECT EXISTS(SELECT 1 FROM plus_subscriptions WHERE user_id = ?) AS has_any;

-- name: CurrentSubscription :one
-- The period that covers `now` (canceled = auto-renew off, still valid); the latest-ending one wins.
SELECT * FROM plus_subscriptions
WHERE user_id = sqlc.arg(user_id)
  AND status IN ('active', 'canceled')
  AND starts_at <= sqlc.arg(now)
  AND ends_at > sqlc.arg(now)
ORDER BY ends_at DESC, id DESC
LIMIT 1;

-- name: LatestValidSubscription :one
-- Where a new purchase starts: the last still-valid period ending after now (current or queued).
SELECT * FROM plus_subscriptions
WHERE user_id = sqlc.arg(user_id)
  AND status IN ('active', 'canceled')
  AND ends_at > sqlc.arg(now)
ORDER BY ends_at DESC, id DESC
LIMIT 1;

-- name: InsertSubscription :execlastid
INSERT INTO plus_subscriptions (user_id, plan_id, invoice_id, status, source, starts_at, ends_at, auto_renew, created_at, updated_at)
VALUES (?, ?, ?, 'active', ?, ?, ?, 1, ?, ?);

-- name: CancelSubscriptions :execrows
-- Turn auto-renew off on every valid (current or queued) period of the user; they stay valid until ends_at.
UPDATE plus_subscriptions
SET status = 'canceled', auto_renew = 0, canceled_at = sqlc.arg(stamp), updated_at = sqlc.arg(stamp)
WHERE user_id = sqlc.arg(user_id)
  AND status = 'active'
  AND ends_at > sqlc.arg(now);

-- name: InsertGrantedSubscription :execlastid
-- A free period (challenge reward, B-N9-07): no plan, no invoice, auto-renew off.
INSERT INTO plus_subscriptions (user_id, plan_id, invoice_id, status, source, starts_at, ends_at, auto_renew, created_at, updated_at)
VALUES (?, NULL, NULL, 'active', ?, ?, ?, 0, ?, ?);

-- name: ListPlusUserIDs :many
-- Every user on Plus right now: a period that covers `now`, or a running trial (the reminder SMS sender's set).
SELECT s.user_id FROM plus_subscriptions s
WHERE s.status IN ('active', 'canceled') AND s.starts_at <= sqlc.arg(now) AND s.ends_at > sqlc.arg(now)
UNION
SELECT t.user_id FROM plus_trials t
WHERE t.started_at <= sqlc.arg(now) AND t.ends_at > sqlc.arg(now);
