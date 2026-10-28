-- Account data export (GET /profile/export, D-96 extended by T-M9-06c, user decision 2026-10-23): the owner's
-- postpartum profile and her red-flag / visit recovery logs. EPDS checks are never exported (D-54: neither the answers
-- nor the totals), so there is deliberately no epds_checks statement here.

-- name: ExportPostpartumProfile :many
SELECT * FROM `postpartum_profiles` WHERE user_id = ? ORDER BY id;

-- name: ExportPostpartumRecoveryLogs :many
SELECT * FROM `postpartum_recovery_logs` WHERE user_id = ? ORDER BY log_date, id;
