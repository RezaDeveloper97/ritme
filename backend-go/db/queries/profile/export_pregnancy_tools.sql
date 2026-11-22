-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportPregnancyAlerts :many
SELECT `id`, `user_id`, `alert_level`, `alert_type`, `title`, `message`, `pregnancy_week`, `trigger_symptoms`, `medical_history_flags`, `is_read`, `is_dismissed`, `read_at`, `dismissed_at`, `recommended_actions`, `created_at`, `updated_at`
FROM `pregnancy_alerts`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPregnancyContractionSessions :many
SELECT `id`, `user_id`, `started_at`, `ended_at`, `alert_at`, `active_lock`, `created_at`, `updated_at`
FROM `pregnancy_contraction_sessions`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPregnancyContractions :many
SELECT `id`, `session_id`, `user_id`, `started_at`, `ended_at`, `active_lock`, `created_at`, `updated_at`
FROM `pregnancy_contractions`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPregnancyDailyExtras :many
SELECT `id`, `user_id`, `log_date`, `mood`, `water_glasses`, `heartburn_severity`, `constipation_severity`, `visit_note`, `created_at`, `updated_at`
FROM `pregnancy_daily_extras`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY log_date, id;

-- name: AcctExportPregnancyKickSessions :many
SELECT `id`, `user_id`, `started_at`, `ended_at`, `kicks`, `tenth_kick_at`, `last_kick_at`, `pregnancy_week`, `active_lock`, `created_at`, `updated_at`
FROM `pregnancy_kick_sessions`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPregnancyWeekUserState :many
SELECT `id`, `user_id`, `week`, `done_task_keys`, `created_at`, `updated_at`
FROM `pregnancy_week_user_state`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY week, id;
