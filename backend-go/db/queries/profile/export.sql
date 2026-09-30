-- ProfileController::export: the raw models, in the order Laravel reads them.

-- name: ExportDailyHealthLogs :many
SELECT * FROM `daily_health_logs` WHERE user_id = ? ORDER BY log_date, id;

-- name: ExportPregnancyProfile :one
SELECT * FROM `pregnancy_profiles` WHERE user_id = ? ORDER BY id LIMIT 1;

-- name: ExportPregnancySymptomLogs :many
SELECT * FROM `pregnancy_symptom_logs` WHERE user_id = ? ORDER BY log_date, id;

-- name: ExportPregnancyWeeklyLogs :many
SELECT * FROM `pregnancy_weekly_logs` WHERE user_id = ? ORDER BY pregnancy_week, id;

-- name: ExportPregnancyFetalMovements :many
-- $user->pregnancyFetalMovements()->get() has no ORDER BY; MariaDB answers in the
-- (user_id, log_date) unique-index order.
SELECT * FROM `pregnancy_fetal_movements` WHERE user_id = ? ORDER BY log_date, id;

-- name: ExportReminders :many
-- $user->reminders()->get() has no ORDER BY; MariaDB answers in the
-- (user_id, type, is_active) index order (the contract golden locks it).
SELECT * FROM `reminders` WHERE user_id = ? ORDER BY type, is_active, id;

-- name: ExportUserConsents :many
-- B-N1-12: the consents with their timestamps.
SELECT consent, granted, granted_at, revoked_at, created_at, updated_at
FROM `user_consents` WHERE user_id = ? ORDER BY id;

-- name: ExportSupportReports :many
-- B-N1-12: the user's support reports — no file path, only whether a screenshot was attached.
SELECT id, message, status, CAST(screenshot_path IS NOT NULL AS SIGNED) AS has_screenshot, created_at
FROM `support_reports` WHERE user_id = ? ORDER BY id;
