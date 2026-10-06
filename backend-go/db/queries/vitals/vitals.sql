-- Vitals (bloom B-N6-01; internal/vitals): timed BP / glucose / heart-rate readings, the weekly measurement plan and
-- the urgent-message copy. Every user-data query is scoped by user_id in the query itself (IDOR).

-- name: InsertReading :execlastid
INSERT INTO `vital_readings` (
  user_id, type, measured_at, systolic, diastolic, pulse, arm, position, glucose_mg_dl, glucose_unit, context, method,
  note, created_at, updated_at
) VALUES (
  sqlc.arg(user_id), sqlc.arg(type), sqlc.arg(measured_at), sqlc.narg(systolic), sqlc.narg(diastolic), sqlc.narg(pulse),
  sqlc.narg(arm), sqlc.narg(position), sqlc.narg(glucose_mg_dl), sqlc.narg(glucose_unit), sqlc.narg(context),
  sqlc.narg(method), sqlc.narg(note), sqlc.arg(now), sqlc.arg(now)
);

-- name: GetReading :one
SELECT * FROM `vital_readings` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: UpdateReading :execrows
UPDATE `vital_readings`
SET type = sqlc.arg(type), measured_at = sqlc.arg(measured_at), systolic = sqlc.narg(systolic),
    diastolic = sqlc.narg(diastolic), pulse = sqlc.narg(pulse), arm = sqlc.narg(arm), position = sqlc.narg(position),
    glucose_mg_dl = sqlc.narg(glucose_mg_dl), glucose_unit = sqlc.narg(glucose_unit), context = sqlc.narg(context),
    method = sqlc.narg(method), note = sqlc.narg(note), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteReading :execrows
DELETE FROM `vital_readings` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListReadings :many
-- Readings of [from, to) newest first; an empty type = every type.
SELECT * FROM `vital_readings`
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.arg(type) = '' OR type = sqlc.arg(type))
  AND measured_at >= sqlc.arg(date_from) AND measured_at < sqlc.arg(date_to)
ORDER BY measured_at DESC, id DESC
LIMIT ?;

-- name: LatestReading :one
SELECT * FROM `vital_readings`
WHERE user_id = sqlc.arg(user_id) AND type = sqlc.arg(type) AND measured_at < sqlc.arg(before)
ORDER BY measured_at DESC, id DESC
LIMIT 1;

-- name: ListLogMeasurements :many
-- The day-level vitals of the log sheet (taxonomy v2 measurements.*, incl. the legacy daily-log projection) in
-- [date_from, date_to], read merged into the vitals views (internal/vitals/merge.go).
SELECT log_date, param, value_num FROM `health_log_entries`
WHERE user_id = sqlc.arg(user_id) AND category = 'measurements'
  AND param IN ('bp_systolic', 'bp_diastolic', 'heart_rate', 'blood_sugar')
  AND item = '' AND value_num IS NOT NULL
  AND log_date >= sqlc.arg(date_from) AND log_date <= sqlc.arg(date_to)
ORDER BY log_date DESC;

-- name: ListPlanItems :many
SELECT * FROM `vital_plan_items` WHERE user_id = sqlc.arg(user_id) ORDER BY sort_order, id;

-- name: DeletePlanItems :exec
DELETE FROM `vital_plan_items` WHERE user_id = sqlc.arg(user_id);

-- name: InsertPlanItem :exec
INSERT INTO `vital_plan_items` (user_id, type, slot, days, remind_at, sort_order, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(type), sqlc.arg(slot), sqlc.arg(days), sqlc.narg(remind_at), sqlc.arg(sort_order),
        sqlc.arg(now), sqlc.arg(now));

-- name: ListLiveAlertCopy :many
-- The live (active + approved) vitals_alert rows, all locales (urgent modal copy).
SELECT item_key, locale, payload FROM `message_contents`
WHERE `group` = 'vitals_alert' AND is_active = 1 AND is_approved = 1
ORDER BY item_key, locale;
