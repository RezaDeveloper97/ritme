-- Checkups user API (T-M4-02, /api/v1/checkups/*). Every query is scoped by user_id; a catalog
-- read returns the shared rows plus the user's own custom checkups only.

-- name: GetCheckupUserContext :one
-- The profile columns the cycle engine and the age rules read, plus pregnancy mode, in one read.
SELECT
  p.id AS profile_id,
  p.birthday,
  p.last_period_start,
  p.cycle_duration,
  p.period_duration,
  p.user_goal,
  CAST(EXISTS (
    SELECT 1 FROM `pregnancy_profiles` pp WHERE pp.user_id = u.id AND pp.pregnancy_mode = 1
  ) AS SIGNED) AS pregnant
FROM `users` u
LEFT JOIN `user_profiles` p ON p.user_id = u.id
WHERE u.id = ?
LIMIT 1;

-- name: ListCheckupPlanRows :many
-- The active catalog with the user's setting and latest record per type, in one read
-- (latest = highest done_on, ties → higher id, like the engine).
SELECT
  sqlc.embed(t),
  s.enabled AS setting_enabled,
  s.remind AS setting_remind,
  r.id AS record_id,
  r.done_on AS record_done_on,
  r.next_due_on AS record_next_due_on
FROM `checkup_types` t
LEFT JOIN `user_checkup_settings` s
  ON s.checkup_type_id = t.id AND s.user_id = CAST(sqlc.arg(user_id) AS UNSIGNED)
LEFT JOIN `checkup_records` r ON r.id = (
  SELECT r2.id FROM `checkup_records` r2
  WHERE r2.user_id = CAST(sqlc.arg(user_id) AS UNSIGNED) AND r2.checkup_type_id = t.id
  ORDER BY r2.done_on DESC, r2.id DESC
  LIMIT 1
)
WHERE t.is_active = 1 AND (t.user_id IS NULL OR t.user_id = CAST(sqlc.arg(user_id) AS UNSIGNED))
ORDER BY t.sort_order, t.id;

-- name: GetCheckupTypeForUser :one
-- An active type the user can see: a shared catalog row or her own custom checkup.
SELECT * FROM `checkup_types`
WHERE id = sqlc.arg(id) AND is_active = 1 AND (user_id IS NULL OR user_id = CAST(sqlc.arg(user_id) AS UNSIGNED))
LIMIT 1;

-- name: ListCheckupRecordsOfType :many
SELECT * FROM `checkup_records`
WHERE user_id = ? AND checkup_type_id = ?
ORDER BY done_on DESC, id DESC
LIMIT ?;

-- name: CountCheckupRecordHistory :one
SELECT COUNT(*) FROM `checkup_records` r
WHERE r.user_id = sqlc.arg(user_id)
  AND (sqlc.arg(type_id) = 0 OR r.checkup_type_id = sqlc.arg(type_id))
  AND r.done_on >= sqlc.arg(from_date)
  AND (CAST(sqlc.arg(with_attachment) AS SIGNED) = 0 OR r.has_attachment = 1);

-- name: ListCheckupRecordHistory :many
-- The History timeline, newest first, with the checkup's title/icon/tone.
SELECT
  sqlc.embed(r),
  t.title AS checkup_title,
  t.`key` AS checkup_key,
  t.icon AS checkup_icon,
  t.tone AS checkup_tone
FROM `checkup_records` r
JOIN `checkup_types` t ON t.id = r.checkup_type_id
WHERE r.user_id = sqlc.arg(user_id)
  AND (sqlc.arg(type_id) = 0 OR r.checkup_type_id = sqlc.arg(type_id))
  AND r.done_on >= sqlc.arg(from_date)
  AND (CAST(sqlc.arg(with_attachment) AS SIGNED) = 0 OR r.has_attachment = 1)
ORDER BY r.done_on DESC, r.id DESC
LIMIT ? OFFSET ?;

-- name: GetCheckupRecord :one
SELECT * FROM `checkup_records`
WHERE id = ? AND user_id = ?
LIMIT 1;

-- name: InsertCheckupRecord :execlastid
INSERT INTO `checkup_records` (
  user_id, checkup_type_id, done_on, result, findings, note, has_attachment, next_due_on, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateCheckupRecord :exec
UPDATE `checkup_records`
SET done_on = ?, result = ?, findings = ?, note = ?, has_attachment = ?, next_due_on = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteCheckupRecord :execrows
DELETE FROM `checkup_records`
WHERE id = ? AND user_id = ?;

-- name: GetCustomCheckupType :one
SELECT * FROM `checkup_types`
WHERE id = ? AND user_id = ? AND category = 'custom'
LIMIT 1;

-- name: InsertCustomCheckupType :execlastid
INSERT INTO `checkup_types` (
  user_id, category, title, subtitle, performed_by, icon, tone, interval_months, remind_lead_days,
  is_active, sort_order, created_at, updated_at
) VALUES (?, 'custom', ?, ?, ?, ?, 'neutral', ?, ?, 1, ?, ?, ?);

-- name: UpdateCustomCheckupType :exec
UPDATE `checkup_types`
SET title = ?, subtitle = ?, performed_by = ?, icon = ?, interval_months = ?, remind_lead_days = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND category = 'custom';

-- name: DeleteCustomCheckupType :execrows
DELETE FROM `checkup_types`
WHERE id = ? AND user_id = ? AND category = 'custom';

-- name: GetCheckupSetting :one
SELECT * FROM `user_checkup_settings`
WHERE user_id = ? AND checkup_type_id = ?
LIMIT 1;

-- name: UpsertCheckupSetting :exec
-- One row per (user_id, checkup_type_id); created_at is kept on update.
INSERT INTO `user_checkup_settings` (user_id, checkup_type_id, enabled, remind, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(checkup_type_id), sqlc.arg(enabled), sqlc.arg(remind), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  enabled = VALUES(enabled),
  remind = VALUES(remind),
  updated_at = VALUES(updated_at);
