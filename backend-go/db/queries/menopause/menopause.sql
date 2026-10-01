-- Menopause (CB-MENO-01, goose 00022): the base reads/writes of the menopause tables for the API tasks
-- (CB-MENO-02 hot flashes + score, CB-MENO-03 treatment). Health data: every statement is scoped by user_id in the
-- statement itself. Profile = user_life_profiles.menopause_* (B-N2-01); daily symptoms = health_log_entries (B-N3-01).

-- name: CreateHotFlash :execlastid
INSERT INTO `hot_flashes` (user_id, started_at, duration_s, severity, night, sweat, triggers, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(started_at), sqlc.narg(duration_s), sqlc.narg(severity), sqlc.arg(night),
        sqlc.arg(sweat), sqlc.narg(triggers), sqlc.arg(now), sqlc.arg(now));

-- name: GetRunningHotFlash :one
-- The timer still running (duration_s NULL), newest first.
SELECT * FROM `hot_flashes`
WHERE user_id = ? AND duration_s IS NULL
ORDER BY started_at DESC, id DESC
LIMIT 1;

-- name: ListHotFlashesInRange :many
-- started_at in [from, to).
SELECT * FROM `hot_flashes`
WHERE user_id = sqlc.arg(user_id) AND started_at >= sqlc.arg(from_at) AND started_at < sqlc.arg(to_at)
ORDER BY started_at DESC, id DESC;

-- name: UpsertMenopauseScore :exec
-- One questionnaire per (user, month); filling it again replaces the answers.
INSERT INTO `menopause_scores`
  (user_id, month, answers, total, somatic, psychological, urogenital, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(month), sqlc.arg(answers), sqlc.arg(total), sqlc.arg(somatic),
   sqlc.arg(psychological), sqlc.arg(urogenital), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  answers = VALUES(answers),
  total = VALUES(total),
  somatic = VALUES(somatic),
  psychological = VALUES(psychological),
  urogenital = VALUES(urogenital),
  updated_at = VALUES(updated_at);

-- name: ListMenopauseScoresSince :many
SELECT * FROM `menopause_scores`
WHERE user_id = sqlc.arg(user_id) AND month >= sqlc.arg(from_month)
ORDER BY month DESC;

-- name: ListTreatmentItems :many
SELECT * FROM `treatment_items`
WHERE user_id = ?
ORDER BY kind, sort_order, id;

-- name: ListTreatmentIntakesInRange :many
-- intake_date in [from, to].
SELECT * FROM `treatment_intakes`
WHERE user_id = sqlc.arg(user_id) AND intake_date >= sqlc.arg(from_date) AND intake_date <= sqlc.arg(to_date)
ORDER BY intake_date, treatment_item_id;

-- name: ListSideEffectLogsInRange :many
-- log_date in [from, to].
SELECT * FROM `side_effect_logs`
WHERE user_id = sqlc.arg(user_id) AND log_date >= sqlc.arg(from_date) AND log_date <= sqlc.arg(to_date)
ORDER BY log_date, code;
