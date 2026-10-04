-- Monthly feature usage counters (quotas + the trial sheet's usage lines).

-- name: ListUsage :many
SELECT feature, used FROM plus_usage_counters
WHERE user_id = ? AND period_start = ?;

-- name: IncrementUsage :exec
-- Unlimited features: always count.
INSERT INTO plus_usage_counters (user_id, feature, period_start, used, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(feature), sqlc.arg(period_start), 1, sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at), used = used + 1;

-- name: IncrementUsageWithin :execrows
-- Quota features: count only while used < limit, atomically (1 = inserted, 2 = incremented, 0 = quota reached).
-- updated_at is assigned first because MariaDB evaluates the SET list left to right with the new values.
INSERT INTO plus_usage_counters (user_id, feature, period_start, used, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(feature), sqlc.arg(period_start), 1, sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  updated_at = IF(used < sqlc.arg(quota), VALUES(updated_at), updated_at),
  used = IF(used < sqlc.arg(quota), used + 1, used);

-- name: SumUsageSince :many
-- The trial sheet's «این روزها از پلاس استفاده کردی» lines: uses per feature over every quota month from `since`
-- (the first day of the trial's start month) on — a 7-day trial can straddle two months.
SELECT feature, CAST(SUM(used) AS UNSIGNED) AS used FROM plus_usage_counters
WHERE user_id = sqlc.arg(user_id) AND period_start >= sqlc.arg(since)
GROUP BY feature;

-- name: DecrementUsage :execrows
-- B-N6-05b: give back one reserved use (an AI call reserved at the gate whose provider failed before answering).
-- Never below zero; the period is the one the reservation counted in.
UPDATE plus_usage_counters SET updated_at = sqlc.arg(now), used = used - 1
WHERE user_id = sqlc.arg(user_id) AND feature = sqlc.arg(feature) AND period_start = sqlc.arg(period_start) AND used > 0;
