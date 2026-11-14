-- AI credits (goose 00125). Every user-facing query is scoped by user_id (IDOR).
--
-- A grant is usable at `now` when it is an extra / admin bucket, or when the subscription period / trial it belongs
-- to covers `now`. The period is read by JOIN (never copied onto the grant), so an admin extend or refund of the
-- subscription moves the credits with it.

-- name: CurrentSubscription :one
SELECT id, plan_id, starts_at, ends_at FROM plus_subscriptions
WHERE user_id = sqlc.arg(user_id)
  AND status IN ('active', 'canceled')
  AND starts_at <= sqlc.arg(now)
  AND ends_at > sqlc.arg(now)
ORDER BY ends_at DESC, id DESC
LIMIT 1;

-- name: RunningTrial :one
SELECT id, started_at, ends_at FROM plus_trials
WHERE user_id = sqlc.arg(user_id) AND started_at <= sqlc.arg(now) AND ends_at > sqlc.arg(now);

-- name: GetSetting :one
SELECT value FROM plus_settings WHERE `key` = ?;

-- name: EnsurePlanGrants :execrows
-- One bucket per running subscription period that has a plan with credits. UNIQUE(subscription_id) makes it
-- idempotent and race-free.
INSERT IGNORE INTO ai_credit_grants (user_id, kind, subscription_id, amount, remaining, created_at, updated_at)
SELECT s.user_id, 'plan', s.id, p.ai_credits, p.ai_credits, sqlc.arg(stamp), sqlc.arg(stamp)
FROM plus_subscriptions s
JOIN plus_plans p ON p.id = s.plan_id
WHERE s.user_id = sqlc.arg(user_id)
  AND s.status IN ('active', 'canceled')
  AND s.starts_at <= sqlc.arg(now)
  AND s.ends_at > sqlc.arg(now)
  AND p.ai_credits > 0;

-- name: EnsureTrialGrant :execrows
INSERT IGNORE INTO ai_credit_grants (user_id, kind, trial_id, amount, remaining, created_at, updated_at)
SELECT t.user_id, 'trial', t.id, sqlc.arg(amount), sqlc.arg(amount), sqlc.arg(stamp), sqlc.arg(stamp)
FROM plus_trials t
WHERE t.user_id = sqlc.arg(user_id) AND t.started_at <= sqlc.arg(now) AND t.ends_at > sqlc.arg(now);

-- name: LockUserGrants :many
-- Serialises every spend / refund / adjustment of one user (single-table row locks only).
SELECT id FROM ai_credit_grants
WHERE user_id = ?
FOR UPDATE;

-- name: ValidGrants :many
-- The buckets that count at `now`, in spending order: the period's credits first (earliest end first), then the
-- never-expiring ones (oldest first).
SELECT g.id, g.kind, g.amount, g.remaining, s.ends_at AS subscription_ends_at, t.ends_at AS trial_ends_at
FROM ai_credit_grants g
LEFT JOIN plus_subscriptions s ON s.id = g.subscription_id
LEFT JOIN plus_trials t ON t.id = g.trial_id
WHERE g.user_id = sqlc.arg(user_id)
  AND (
    (g.kind = 'plan' AND s.status IN ('active', 'canceled') AND s.starts_at <= sqlc.arg(now) AND s.ends_at > sqlc.arg(now))
    OR (g.kind = 'trial' AND t.started_at <= sqlc.arg(now) AND t.ends_at > sqlc.arg(now))
    OR g.kind IN ('extra', 'admin')
  )
ORDER BY (g.kind IN ('extra', 'admin')), COALESCE(s.ends_at, t.ends_at), g.id;

-- name: TakeFromGrant :execrows
UPDATE ai_credit_grants SET remaining = remaining - sqlc.arg(credits), updated_at = sqlc.arg(stamp)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND remaining >= sqlc.arg(credits);

-- name: ReturnToGrant :execrows
-- A refund never lifts a bucket above what it was granted.
UPDATE ai_credit_grants SET remaining = LEAST(amount, remaining + sqlc.arg(credits)), updated_at = sqlc.arg(stamp)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: InsertSpend :execlastid
INSERT INTO ai_credit_spends (user_id, feature, credits, plan_credits, extra_credits, allocations, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, 'spent', ?, ?);

-- name: LockSpend :one
SELECT * FROM ai_credit_spends
WHERE id = ? AND user_id = ?
FOR UPDATE;

-- name: MarkSpendRefunded :execrows
UPDATE ai_credit_spends SET status = 'refunded', updated_at = sqlc.arg(stamp)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND status = 'spent';

-- name: ListUserSpends :many
-- The usage history, newest first (keyset on id). Refunded reservations never happened for the user.
SELECT id, feature, credits, plan_credits, extra_credits, created_at FROM ai_credit_spends
WHERE user_id = sqlc.arg(user_id) AND status = 'spent'
  AND (sqlc.arg(before_id) = 0 OR id < sqlc.arg(before_id))
ORDER BY id DESC
LIMIT ?;

-- name: UserSpendByFeature :many
SELECT feature, COUNT(*) AS uses, CAST(COALESCE(SUM(credits), 0) AS UNSIGNED) AS credits
FROM ai_credit_spends
WHERE user_id = sqlc.arg(user_id) AND status = 'spent' AND created_at >= sqlc.arg(since)
GROUP BY feature
ORDER BY credits DESC, feature;

-- name: UserSpendByDay :many
SELECT CAST(DATE(created_at) AS CHAR(10)) AS day, CAST(COALESCE(SUM(credits), 0) AS UNSIGNED) AS credits
FROM ai_credit_spends
WHERE user_id = sqlc.arg(user_id) AND status = 'spent' AND created_at >= sqlc.arg(since)
GROUP BY DATE(created_at)
ORDER BY day;

-- name: ListActivePacks :many
SELECT * FROM ai_credit_packs
WHERE is_active = 1
ORDER BY sort_order, id;

-- name: ListFeatures :many
SELECT * FROM ai_features
ORDER BY sort_order, id;

-- name: ListModels :many
SELECT * FROM ai_models
ORDER BY id;

-- name: InsertFeatureIfMissing :exec
-- Boot-time registry sync: a feature key declared in code gets its row (no model, 1 credit) so the admin can set
-- it up; an existing row is never touched.
INSERT IGNORE INTO ai_features (`key`, title, credit_cost, is_active, sort_order, created_at, updated_at)
VALUES (?, ?, 1, 1, ?, ?, ?);

-- name: InsertAdminGrant :execlastid
INSERT INTO ai_credit_grants (user_id, kind, amount, remaining, note, created_at, updated_at)
VALUES (?, 'admin', sqlc.arg(amount), sqlc.arg(amount), ?, ?, ?);

-- name: LockSpendAt :one
-- A reservation found by when it was made (async features keep that moment on their own row and refund later).
SELECT * FROM ai_credit_spends
WHERE user_id = sqlc.arg(user_id) AND feature = sqlc.arg(feature) AND status = 'spent' AND created_at = sqlc.arg(reserved_at)
ORDER BY id
LIMIT 1
FOR UPDATE;
