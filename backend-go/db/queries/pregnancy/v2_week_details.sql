-- Pregnancy v2 week details (table pregnancy_week_details, T-M7-01, docs/pregnancy-v2/README.md § Data).
-- One row per week 1–42, admin-defined; translatable columns are JSON keyed by language code.

-- name: GetWeekDetails :one
SELECT * FROM `pregnancy_week_details` WHERE week_number = ? LIMIT 1;

-- name: ListWeekDetailsRange :many
-- Weeks from..to inclusive (the Today carousel reads prev/current/next in one query).
SELECT * FROM `pregnancy_week_details`
WHERE week_number >= sqlc.arg(week_from) AND week_number <= sqlc.arg(week_to)
ORDER BY week_number;

-- name: ListWeekDetails :many
SELECT * FROM `pregnancy_week_details` ORDER BY week_number;

-- name: UpsertWeekDetails :exec
-- Admin editor (T-M7-06): one row per week_number; created_at is kept on update.
INSERT INTO `pregnancy_week_details` (
  week_number, size_label, illustration_key, length_cm, weight_g, heart_rate, headline, highlights,
  body_symptoms, body_text, tasks, warning, reviewer_name, reviewed_at, sources, created_at, updated_at
) VALUES (
  sqlc.arg(week_number), sqlc.arg(size_label), sqlc.arg(illustration_key), sqlc.arg(length_cm), sqlc.arg(weight_g),
  sqlc.arg(heart_rate), sqlc.arg(headline), sqlc.arg(highlights), sqlc.arg(body_symptoms), sqlc.arg(body_text),
  sqlc.arg(tasks), sqlc.arg(warning), sqlc.arg(reviewer_name), sqlc.arg(reviewed_at), sqlc.arg(sources),
  sqlc.arg(now), sqlc.arg(now)
)
ON DUPLICATE KEY UPDATE
  size_label = VALUES(size_label),
  illustration_key = VALUES(illustration_key),
  length_cm = VALUES(length_cm),
  weight_g = VALUES(weight_g),
  heart_rate = VALUES(heart_rate),
  headline = VALUES(headline),
  highlights = VALUES(highlights),
  body_symptoms = VALUES(body_symptoms),
  body_text = VALUES(body_text),
  tasks = VALUES(tasks),
  warning = VALUES(warning),
  reviewer_name = VALUES(reviewer_name),
  reviewed_at = VALUES(reviewed_at),
  sources = VALUES(sources),
  updated_at = VALUES(updated_at);
