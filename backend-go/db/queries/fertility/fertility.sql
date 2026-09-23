-- Fertility day log (T-M5-01, docs/fertility-ttc/README.md). Every query is scoped by user_id.

-- name: GetMergedDay :one
-- The merged day in one statement: the day's daily_health_logs columns the fertility screens use
-- and its fertility_logs row, either or both missing (all NULL).
SELECT
  d.basal_body_temperature,
  d.intercourse_type,
  d.ovarian_pain_intensity,
  d.bloating_intensity,
  d.breast_sensitivity_intensity,
  d.spotting,
  d.notes,
  f.lh_test,
  f.cervical_mucus,
  f.bbt_time
FROM (SELECT 1 AS one) AS day
LEFT JOIN `daily_health_logs` d ON d.user_id = sqlc.arg(user_id) AND d.log_date = sqlc.arg(log_date)
LEFT JOIN `fertility_logs` f ON f.user_id = sqlc.arg(user_id) AND f.log_date = sqlc.arg(log_date);

-- name: GetFertilityLog :one
SELECT * FROM `fertility_logs` WHERE user_id = ? AND log_date = ?;

-- name: UpsertFertilityLog :exec
-- One row per (user_id, log_date); created_at is kept on update.
INSERT INTO `fertility_logs` (user_id, log_date, lh_test, cervical_mucus, bbt_time, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(log_date), sqlc.arg(lh_test), sqlc.arg(cervical_mucus), sqlc.arg(bbt_time), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  lh_test = VALUES(lh_test),
  cervical_mucus = VALUES(cervical_mucus),
  bbt_time = VALUES(bbt_time),
  updated_at = VALUES(updated_at);

-- name: DeleteFertilityLog :exec
DELETE FROM `fertility_logs` WHERE user_id = ? AND log_date = ?;
