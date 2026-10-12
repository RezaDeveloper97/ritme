-- Account data export (GET /profile/export, user decision 2026-10-12): every menopause row of the owner. The profile
-- is user_life_profiles.menopause_*; the daily symptom log is health_log_entries (B-N3-01), which the menopause
-- screens read and write.

-- name: ExportMenopauseProfile :many
SELECT user_id, menopause_stage, menopause_last_period, menopause_surgical, menopause_hrt, created_at, updated_at
FROM `user_life_profiles` WHERE user_id = ? ORDER BY user_id;

-- name: ExportHealthLogEntries :many
SELECT * FROM `health_log_entries` WHERE user_id = ? ORDER BY log_date, id;

-- name: ExportHotFlashes :many
SELECT * FROM `hot_flashes` WHERE user_id = ? ORDER BY started_at, id;

-- name: ExportMenopauseScores :many
SELECT * FROM `menopause_scores` WHERE user_id = ? ORDER BY month, id;

-- name: ExportTreatmentItems :many
SELECT * FROM `treatment_items` WHERE user_id = ? ORDER BY sort_order, id;

-- name: ExportTreatmentIntakes :many
SELECT * FROM `treatment_intakes` WHERE user_id = ? ORDER BY intake_date, id;

-- name: ExportSideEffectLogs :many
SELECT * FROM `side_effect_logs` WHERE user_id = ? ORDER BY log_date, id;
