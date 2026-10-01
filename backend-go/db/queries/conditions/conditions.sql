-- Condition programs (CB-COND-01, docs/canvas-build/README.md → COND). Health data: every query is scoped by user_id
-- in the statement itself.

-- name: ListEnrolments :many
SELECT * FROM `condition_enrolments` WHERE user_id = ? ORDER BY id;

-- name: Enrol :exec
-- Joining again keeps the first enrolment date.
INSERT INTO `condition_enrolments` (user_id, program, enrolled_on, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(program), sqlc.arg(enrolled_on), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at);

-- name: Leave :exec
DELETE FROM `condition_enrolments` WHERE user_id = ? AND program = ?;

-- name: IsEnrolled :one
SELECT EXISTS (SELECT 1 FROM `condition_enrolments` WHERE user_id = ? AND program = ?) AS enrolled;

-- name: GetPainEntry :one
SELECT * FROM `condition_pain_entries` WHERE user_id = ? AND entry_date = ? LIMIT 1;

-- name: UpsertPainEntry :exec
INSERT INTO `condition_pain_entries` (user_id, entry_date, pain_types, associated, missed_activity, analgesic,
  analgesic_time, analgesic_effect, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(entry_date), sqlc.arg(pain_types), sqlc.arg(associated), sqlc.arg(missed_activity),
  sqlc.arg(analgesic), sqlc.arg(analgesic_time), sqlc.arg(analgesic_effect), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  pain_types = VALUES(pain_types),
  associated = VALUES(associated),
  missed_activity = VALUES(missed_activity),
  analgesic = VALUES(analgesic),
  analgesic_time = VALUES(analgesic_time),
  analgesic_effect = VALUES(analgesic_effect),
  updated_at = VALUES(updated_at);

-- name: DeletePainEntry :exec
DELETE FROM `condition_pain_entries` WHERE user_id = ? AND entry_date = ?;

-- name: ListPmddEntries :many
-- The user's questionnaire days from `from` to `to` (inclusive), oldest first.
SELECT * FROM `pmdd_entries`
WHERE user_id = sqlc.arg(user_id)
  AND entry_date >= sqlc.arg(from_date)
  AND entry_date <= sqlc.arg(to_date)
ORDER BY entry_date;

-- name: UpsertPmddEntry :exec
INSERT INTO `pmdd_entries` (user_id, entry_date, scores, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(entry_date), sqlc.arg(scores), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  scores = VALUES(scores),
  updated_at = VALUES(updated_at);

-- name: DeletePmddEntry :exec
DELETE FROM `pmdd_entries` WHERE user_id = ? AND entry_date = ?;

-- name: ListPbacEntries :many
-- The user's pad-chart days from `from` to `to` (inclusive), oldest first.
SELECT * FROM `pbac_entries`
WHERE user_id = sqlc.arg(user_id)
  AND entry_date >= sqlc.arg(from_date)
  AND entry_date <= sqlc.arg(to_date)
ORDER BY entry_date;

-- name: UpsertPbacEntry :exec
INSERT INTO `pbac_entries` (user_id, entry_date, light_count, medium_count, heavy_count, flooding, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(entry_date), sqlc.arg(light_count), sqlc.arg(medium_count), sqlc.arg(heavy_count),
  sqlc.arg(flooding), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  light_count = VALUES(light_count),
  medium_count = VALUES(medium_count),
  heavy_count = VALUES(heavy_count),
  flooding = VALUES(flooding),
  updated_at = VALUES(updated_at);

-- name: DeletePbacEntry :exec
DELETE FROM `pbac_entries` WHERE user_id = ? AND entry_date = ?;

-- name: ListPeriodStarts :many
-- The user's most recent period starts on or before `day`, newest first (PMDD cycles, PBAC period window).
SELECT period_start_date, period_end_date, bleeding_length
FROM `cycle_histories`
WHERE user_id = sqlc.arg(user_id)
  AND period_start_date <= sqlc.arg(day)
ORDER BY period_start_date DESC
LIMIT 3;

-- name: GetCycleDefaults :one
-- The profile's cycle and period length (the first profile row, as the cycle engine reads it).
SELECT cycle_duration, period_duration FROM `user_profiles` WHERE user_id = ? ORDER BY id LIMIT 1;
