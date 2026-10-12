-- Account data export (GET /profile/export, user decision 2026-10-12): every contraception row of the owner.

-- name: ExportContraceptionMethods :many
SELECT * FROM `contraception_methods` WHERE user_id = ? ORDER BY id;

-- name: ExportContraceptionPillLogs :many
SELECT * FROM `contraception_pill_logs` WHERE user_id = ? ORDER BY log_date, id;

-- name: ExportContraceptionReminders :many
SELECT * FROM `contraception_reminders` WHERE user_id = ? ORDER BY id;
