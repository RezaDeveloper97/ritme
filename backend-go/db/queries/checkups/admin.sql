-- Checkups admin catalog (T-M4-03): /api/admin/v1/checkup-types. Every query is limited to the shared
-- catalog (`user_id IS NULL`); users' custom checkups are never listed, shown, changed or counted here.

-- name: CountAdminCheckupTypes :one
SELECT COUNT(*) FROM `checkup_types`
WHERE user_id IS NULL
  AND (IFNULL(`key`, '') LIKE CAST(sqlc.arg(pattern) AS CHAR)
       OR title LIKE CAST(sqlc.arg(pattern) AS CHAR) OR title LIKE CAST(sqlc.arg(json_pattern) AS CHAR))
  AND is_active >= sqlc.arg(active_min) AND is_active <= sqlc.arg(active_max);

-- name: ListAdminCheckupTypes :many
SELECT * FROM `checkup_types`
WHERE user_id IS NULL
  AND (IFNULL(`key`, '') LIKE CAST(sqlc.arg(pattern) AS CHAR)
       OR title LIKE CAST(sqlc.arg(pattern) AS CHAR) OR title LIKE CAST(sqlc.arg(json_pattern) AS CHAR))
  AND is_active >= sqlc.arg(active_min) AND is_active <= sqlc.arg(active_max)
ORDER BY sort_order, id
LIMIT ? OFFSET ?;

-- name: ListAdminCheckupTypeIDs :many
-- The whole catalog in display order (reorder check, stats).
SELECT id FROM `checkup_types` WHERE user_id IS NULL ORDER BY sort_order, id;

-- name: GetAdminCheckupType :one
SELECT * FROM `checkup_types` WHERE id = ? AND user_id IS NULL LIMIT 1;

-- name: AdminCheckupTypeKeyExists :one
SELECT EXISTS(SELECT 1 FROM `checkup_types` WHERE `key` = ?) AS found;

-- name: NextAdminCheckupTypeSortOrder :one
SELECT CAST(COALESCE(MAX(sort_order), 0) + 1 AS SIGNED) AS next_sort_order
FROM `checkup_types` WHERE user_id IS NULL;

-- name: CreateAdminCheckupType :execresult
INSERT INTO `checkup_types` (`key`, user_id, category, title, subtitle, why, performed_by, icon, tone,
                             interval_months, interval_months_max, age_min, age_max, cycle_day_from, cycle_day_to,
                             remind_lead_days, prep_steps, guide_steps, finding_options, hide_in_pregnancy,
                             is_active, sort_order, source_note, created_at, updated_at)
VALUES (sqlc.arg(type_key), NULL, sqlc.arg(category), sqlc.arg(title), sqlc.narg(subtitle), sqlc.narg(why),
        sqlc.arg(performed_by), sqlc.narg(icon), sqlc.arg(tone), sqlc.arg(interval_months),
        sqlc.narg(interval_months_max), sqlc.narg(age_min), sqlc.narg(age_max), sqlc.narg(cycle_day_from),
        sqlc.narg(cycle_day_to), sqlc.arg(remind_lead_days), sqlc.narg(prep_steps), sqlc.narg(guide_steps),
        sqlc.narg(finding_options), sqlc.arg(hide_in_pregnancy), sqlc.arg(is_active), sqlc.arg(sort_order),
        sqlc.narg(source_note), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateAdminCheckupType :exec
UPDATE `checkup_types`
SET category = sqlc.arg(category), title = sqlc.arg(title), subtitle = sqlc.narg(subtitle), why = sqlc.narg(why),
    performed_by = sqlc.arg(performed_by), icon = sqlc.narg(icon), tone = sqlc.arg(tone),
    interval_months = sqlc.arg(interval_months), interval_months_max = sqlc.narg(interval_months_max),
    age_min = sqlc.narg(age_min), age_max = sqlc.narg(age_max),
    cycle_day_from = sqlc.narg(cycle_day_from), cycle_day_to = sqlc.narg(cycle_day_to),
    remind_lead_days = sqlc.arg(remind_lead_days), prep_steps = sqlc.narg(prep_steps),
    guide_steps = sqlc.narg(guide_steps), finding_options = sqlc.narg(finding_options),
    hide_in_pregnancy = sqlc.arg(hide_in_pregnancy), is_active = sqlc.arg(is_active),
    sort_order = sqlc.arg(sort_order), source_note = sqlc.narg(source_note), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id IS NULL;

-- name: SetAdminCheckupTypeSortOrder :exec
UPDATE `checkup_types` SET sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id IS NULL;

-- name: DeleteUnusedAdminCheckupType :execresult
-- Atomic "refuse if records exist": the FK would otherwise cascade-delete users' records.
DELETE FROM `checkup_types`
WHERE `checkup_types`.id = sqlc.arg(type_id) AND `checkup_types`.user_id IS NULL
  AND NOT EXISTS (SELECT 1 FROM `checkup_records` r WHERE r.checkup_type_id = sqlc.arg(type_id));

-- name: CountCheckupRecordsOfType :one
SELECT COUNT(*) FROM `checkup_records` WHERE checkup_type_id = ?;

-- name: CheckupRecordStatsByType :many
-- Per catalog type with records: all records, distinct users, and records done on or after `since`.
SELECT r.checkup_type_id,
       COUNT(*) AS records_total,
       COUNT(DISTINCT r.user_id) AS users_with_records,
       CAST(SUM(CASE WHEN r.done_on >= sqlc.arg(since) THEN 1 ELSE 0 END) AS SIGNED) AS records_recent
FROM `checkup_records` r
JOIN `checkup_types` t ON t.id = r.checkup_type_id AND t.user_id IS NULL
GROUP BY r.checkup_type_id;

-- name: CheckupOverdueUsersByType :many
-- Per catalog type: users whose latest record (ties → higher id) is past its due-by date, i.e.
-- next_due_on ?? done_on + (interval_months_max ?? interval_months) months (MariaDB clamps to the month
-- end like the engine), strictly before `today`. Users who switched the type off are left out. This is
-- the calendar approximation of the engine's `overdue` (it ignores cycle windows, age and pregnancy).
SELECT r.checkup_type_id, COUNT(DISTINCT r.user_id) AS overdue_users
FROM `checkup_records` r
JOIN `checkup_types` t ON t.id = r.checkup_type_id AND t.user_id IS NULL
WHERE NOT EXISTS (
        SELECT 1 FROM `checkup_records` n
        WHERE n.user_id = r.user_id AND n.checkup_type_id = r.checkup_type_id
          AND (n.done_on > r.done_on OR (n.done_on = r.done_on AND n.id > r.id)))
  AND NOT EXISTS (
        SELECT 1 FROM `user_checkup_settings` s
        WHERE s.user_id = r.user_id AND s.checkup_type_id = r.checkup_type_id AND s.enabled = 0)
  AND COALESCE(r.next_due_on,
               DATE_ADD(r.done_on, INTERVAL COALESCE(t.interval_months_max, t.interval_months) MONTH))
      < sqlc.arg(today)
GROUP BY r.checkup_type_id;
