-- Postpartum mode (bloom B-N5-01, goose 00030). Every query is scoped by user_id (health data, IDOR).

-- name: GetPostpartumProfile :one
SELECT * FROM `postpartum_profiles` WHERE user_id = ? LIMIT 1;

-- name: UpsertPostpartumProfile :exec
INSERT INTO `postpartum_profiles`
  (user_id, birth_date, delivery_type, baby_count, source, pregnancy_profile_id, pregnancy_closed_at, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(birth_date), sqlc.narg(delivery_type), sqlc.arg(baby_count), sqlc.arg(source),
   sqlc.narg(pregnancy_profile_id), sqlc.narg(pregnancy_closed_at), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  birth_date = VALUES(birth_date),
  delivery_type = VALUES(delivery_type),
  baby_count = VALUES(baby_count),
  source = VALUES(source),
  pregnancy_profile_id = VALUES(pregnancy_profile_id),
  pregnancy_closed_at = VALUES(pregnancy_closed_at),
  updated_at = VALUES(updated_at);

-- name: GetPostpartumLifeProfile :one
-- The stored life mode and gender (user_life_profiles, bloom B-N2-01).
SELECT life_mode, gender FROM `user_life_profiles` WHERE user_id = ? LIMIT 1;

-- name: GetPostpartumUserGoal :one
SELECT user_goal FROM `user_profiles` WHERE user_id = ? LIMIT 1;

-- name: GetActivePregnancyProfile :one
-- The user's active pregnancy (pregnancy_mode = 1): the effective mode is then pregnancy.
SELECT id FROM `pregnancy_profiles` WHERE user_id = ? AND pregnancy_mode = 1 LIMIT 1;

-- name: LockActivePregnancyProfile :one
-- The same row, locked for the activation transaction.
SELECT id FROM `pregnancy_profiles` WHERE user_id = ? AND pregnancy_mode = 1 LIMIT 1 FOR UPDATE;

-- name: ClosePregnancyAsDelivered :exec
-- The pregnancy ended in a birth: the same flags as POST /pregnancy/deactivate.
UPDATE `pregnancy_profiles` SET pregnancy_mode = 0, cycle_mode = 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: SetLifeModePostpartum :exec
INSERT INTO `user_life_profiles` (user_id, life_mode, ivf_iui, track_contraception, created_at, updated_at)
VALUES (sqlc.arg(user_id), 'postpartum', 0, 0, sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE life_mode = 'postpartum', updated_at = VALUES(updated_at);

-- name: SyncPostpartumLegacyGoal :exec
-- user_profiles kept in step with a non-TTC mode (PUT /profile/life-stage does the same): user_goal non_ttc, a
-- trying / pregnant intention dropped.
UPDATE `user_profiles`
SET user_goal = 'non_ttc',
    pregnancy_intention = CASE WHEN pregnancy_intention IN ('trying', 'pregnant') THEN NULL ELSE pregnancy_intention END,
    updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id);

-- name: UpsertEpdsCheck :exec
INSERT INTO `epds_checks` (user_id, kind, taken_on, answers, total, self_harm, urgent, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(kind), sqlc.arg(taken_on), sqlc.arg(answers), sqlc.arg(total), sqlc.narg(self_harm),
        sqlc.arg(urgent), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  answers = VALUES(answers),
  total = VALUES(total),
  self_harm = VALUES(self_harm),
  urgent = VALUES(urgent),
  updated_at = VALUES(updated_at);

-- name: GetEpdsCheckOn :one
SELECT id, kind, taken_on, total, self_harm, urgent FROM `epds_checks`
WHERE user_id = ? AND kind = ? AND taken_on = ? LIMIT 1;

-- name: ListEpdsChecks :many
-- Newest first; the answers stay in the table (history shows totals only).
SELECT id, kind, taken_on, total, self_harm, urgent FROM `epds_checks`
WHERE user_id = ?
ORDER BY taken_on DESC, id DESC
LIMIT ?;

-- name: GetLastEpdsCheck :one
SELECT id, kind, taken_on, total, self_harm, urgent FROM `epds_checks`
WHERE user_id = ? ORDER BY taken_on DESC, id DESC LIMIT 1;

-- name: GetLastEpdsCheckOfKind :one
SELECT id, kind, taken_on, total, self_harm, urgent FROM `epds_checks`
WHERE user_id = ? AND kind = ? ORDER BY taken_on DESC, id DESC LIMIT 1;
