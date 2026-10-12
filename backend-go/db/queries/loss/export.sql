-- Account data export (GET /profile/export, user decision 2026-10-12): every loss path row of the owner (the private
-- note is opened for her by the caller).

-- name: ExportPregnancyLosses :many
SELECT * FROM `pregnancy_losses` WHERE user_id = ? ORDER BY id;

-- name: ExportPregnancyLossMoods :many
SELECT * FROM `pregnancy_loss_moods` WHERE user_id = ? ORDER BY log_date, id;
