-- Fertility day log (T-M5-01, docs/fertility-ttc/README.md). Every query is scoped by user_id.

-- name: GetMergedDay :one
-- The merged day in one statement: the day's daily_health_logs columns the fertility screens use,
-- its fertility_logs row and the log sheet's (taxonomy v2) LH test and mucus consistency, any of
-- them missing (all NULL). B-N3-14b: the v2 values are not synced into fertility_logs, so the day
-- merges both like the TTC analysis does (QUESTIONS #80).
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
  f.bbt_time,
  el.value_code AS entry_lh_test,
  em.value_code AS entry_mucus
FROM (SELECT 1 AS one) AS day
LEFT JOIN `daily_health_logs` d ON d.user_id = sqlc.arg(user_id) AND d.log_date = sqlc.arg(log_date)
LEFT JOIN `fertility_logs` f ON f.user_id = sqlc.arg(user_id) AND f.log_date = sqlc.arg(log_date)
LEFT JOIN `health_log_entries` el ON el.user_id = sqlc.arg(user_id) AND el.log_date = sqlc.arg(log_date)
  AND el.category = 'measurements' AND el.param = 'lh_test' AND el.item = ''
LEFT JOIN `health_log_entries` em ON em.user_id = sqlc.arg(user_id) AND em.log_date = sqlc.arg(log_date)
  AND em.category = 'discharge' AND em.param = 'consistency' AND em.item = '';

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
