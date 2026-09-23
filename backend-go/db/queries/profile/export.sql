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
