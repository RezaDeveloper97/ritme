-- Account data export (GET /profile/export, user decision 2026-10-12): every IVF row of the owner (cycles with their
-- stage and outcome, meds with the inventory, injections with their sites, scans, two-week-wait moods, reminders).

-- name: ExportIvfCycles :many
SELECT * FROM `ivf_cycles` WHERE user_id = ? ORDER BY id;

-- name: ExportIvfMeds :many
SELECT * FROM `ivf_meds` WHERE user_id = ? ORDER BY id;

-- name: ExportIvfDoseLogs :many
SELECT * FROM `ivf_dose_logs` WHERE user_id = ? ORDER BY dose_date, id;

-- name: ExportIvfScans :many
SELECT * FROM `ivf_scans` WHERE user_id = ? ORDER BY scan_date, id;

-- name: ExportIvfTwwLogs :many
SELECT * FROM `ivf_tww_logs` WHERE user_id = ? ORDER BY log_date, id;

-- name: ExportIvfReminders :many
SELECT * FROM `ivf_reminders` WHERE user_id = ? ORDER BY id;
