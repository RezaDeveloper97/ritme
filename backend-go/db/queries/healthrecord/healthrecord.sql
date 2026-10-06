-- Health record «پرونده سلامت من» (bloom B-N6-03; internal/healthrecord): the user-owned rows (health_records,
-- health_record_pregnancies) and the read-only views of other domains' tables the record aggregates. Every query is
-- scoped by user_id in the query itself (IDOR). pregnancy_losses is only ever counted: no column of a loss (type,
-- date, mood, the encrypted note) is read here.

-- name: GetHealthRecord :one
SELECT * FROM `health_records` WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- name: UpsertHealthRecord :exec
INSERT INTO `health_records` (user_id, blood_type, allergies, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.narg(blood_type), sqlc.narg(allergies), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE blood_type = VALUES(blood_type), allergies = VALUES(allergies), updated_at = VALUES(updated_at);

-- name: ListManualPregnancies :many
SELECT * FROM `health_record_pregnancies`
WHERE user_id = sqlc.arg(user_id)
ORDER BY ended_on IS NULL, ended_on DESC, id DESC;

-- name: CountManualPregnancies :one
SELECT COUNT(*) FROM `health_record_pregnancies` WHERE user_id = sqlc.arg(user_id);

-- name: GetManualPregnancy :one
SELECT * FROM `health_record_pregnancies` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: InsertManualPregnancy :execlastid
INSERT INTO `health_record_pregnancies` (user_id, outcome, ended_on, baby_count, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(outcome), sqlc.narg(ended_on), sqlc.narg(baby_count), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateManualPregnancy :execrows
UPDATE `health_record_pregnancies`
SET outcome = sqlc.arg(outcome), ended_on = sqlc.narg(ended_on), baby_count = sqlc.narg(baby_count),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteManualPregnancy :execrows
DELETE FROM `health_record_pregnancies` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: GetRecordUser :one
SELECT name FROM `users` WHERE id = sqlc.arg(user_id) LIMIT 1;

-- name: GetRecordProfile :one
-- $user->profile (hasOne: the first row in index order), only the columns the record shows.
SELECT birthday, weight, height, user_goal, updated_at FROM `user_profiles`
WHERE user_id = sqlc.arg(user_id) ORDER BY id LIMIT 1;

-- name: GetRecordLifeProfile :one
SELECT gender, life_mode, chronic_illnesses, gyn_conditions, medications, updated_at FROM `user_life_profiles`
WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- name: GetRecordPregnancy :one
-- The pregnancy profile (one per user): an active pregnancy is «ongoing»; blood_type + rh_factor are the fallback of
-- health_records.blood_type.
SELECT id, pregnancy_mode, estimated_due_date, blood_type, rh_factor FROM `pregnancy_profiles`
WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- name: GetRecordBirth :one
-- The birth postpartum mode counts from (one per user).
SELECT birth_date, delivery_type, baby_count FROM `postpartum_profiles` WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- name: CountRecordLosses :one
-- A pregnancy that ended in a loss shows only as «ended»: the count is all the record reads of pregnancy_losses.
SELECT COUNT(*) FROM `pregnancy_losses` WHERE user_id = sqlc.arg(user_id);

-- name: ListRecordCheckups :many
-- The latest done checkups with the checkup's title / key / icon (custom checkups included: they are the user's own).
SELECT r.id, r.done_on, r.result, t.title AS checkup_title, t.`key` AS checkup_key, t.icon AS checkup_icon
FROM `checkup_records` r
JOIN `checkup_types` t ON t.id = r.checkup_type_id
WHERE r.user_id = sqlc.arg(user_id)
ORDER BY r.done_on DESC, r.id DESC
LIMIT ?;
