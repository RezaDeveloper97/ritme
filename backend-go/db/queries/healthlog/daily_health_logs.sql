-- Daily health logs (App\Models\DailyHealthLog, DailyHealthLogController).
--
-- Date filters that Laravel passes through as raw request strings (whereDate with a query or
-- route value) are bound as strings too, so MariaDB coerces them exactly as it does for Laravel.

-- name: GetDailyHealthLog :one
SELECT * FROM `daily_health_logs` WHERE id = ? LIMIT 1;

-- name: GetDailyHealthLogOn :one
-- DailyHealthLog::where(['user_id' => $u, 'log_date' => $d])->first() /
-- $user->dailyHealthLogs()->whereDate('log_date', $carbon)->first().
SELECT * FROM `daily_health_logs` WHERE user_id = ? AND log_date = ? LIMIT 1;

-- name: GetDailyHealthLogByDateString :one
-- $user->dailyHealthLogs()->whereDate('log_date', $date)->first() with the raw route value.
SELECT * FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id) AND DATE(log_date) = CAST(sqlc.arg(date) AS CHAR)
LIMIT 1;

-- name: CountDailyHealthLogs :one
-- Total of the index paginator; a NULL bound (?from_date= present but empty) matches nothing.
SELECT COUNT(*) FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.arg(has_from) = 0 OR DATE(log_date) >= CAST(sqlc.narg(from_date) AS CHAR))
  AND (sqlc.arg(has_to) = 0 OR DATE(log_date) <= CAST(sqlc.narg(to_date) AS CHAR));

-- name: ListDailyHealthLogs :many
-- $user->dailyHealthLogs()->orderBy('log_date', 'desc')->paginate(30) (log_date is unique per user).
SELECT * FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.arg(has_from) = 0 OR DATE(log_date) >= CAST(sqlc.narg(from_date) AS CHAR))
  AND (sqlc.arg(has_to) = 0 OR DATE(log_date) <= CAST(sqlc.narg(to_date) AS CHAR))
ORDER BY log_date DESC
LIMIT ? OFFSET ?;

-- name: InsertDailyHealthLog :execlastid
-- Every column is written; the ones the request did not send are NULL (their DB default).
INSERT INTO `daily_health_logs` (
    user_id, log_date, bleeding_intensity, blood_color, has_clots, clots_amount, spotting, 
    bleeding_smell, headache_intensity, stomach_ache_intensity, pelvic_pain_intensity, 
    breast_pain_intensity, back_pain_intensity, ovarian_pain_intensity, nausea_intensity, 
    bloating_intensity, diarrhea, constipation, appetite_change, food_craving, 
    breast_sensitivity_intensity, vaginal_dryness, vaginal_burning, vaginal_burning_intensity, 
    vaginal_itching, vaginal_itching_intensity, vaginal_smell_change, urination_change, 
    urination_burning_intensity, acne, oily_skin, hair_loss, swelling, fatigue, dizziness, hot_flashes, 
    chills, moods, sleep_duration, sleep_quality, exercise_type, exercise_duration, exercise_intensity, 
    sexual_activities, sexual_desire, intercourse_type, weight, basal_body_temperature, heart_rate, 
    systolic_pressure, diastolic_pressure, blood_sugar, energy_level, discharge_color, 
    discharge_texture, discharge_amount, discharge_smell, discharge_itching, discharge_burning, 
    frequent_urination, medications, notes, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateDailyHealthLog :exec
-- fill($validated)->save() when an attribute is dirty (the full row is written back).
UPDATE `daily_health_logs` SET
    `log_date` = ?, `bleeding_intensity` = ?, `blood_color` = ?, `has_clots` = ?, `clots_amount` = ?, 
    `spotting` = ?, `bleeding_smell` = ?, `headache_intensity` = ?, `stomach_ache_intensity` = ?, 
    `pelvic_pain_intensity` = ?, `breast_pain_intensity` = ?, `back_pain_intensity` = ?, 
    `ovarian_pain_intensity` = ?, `nausea_intensity` = ?, `bloating_intensity` = ?, `diarrhea` = ?, 
    `constipation` = ?, `appetite_change` = ?, `food_craving` = ?, `breast_sensitivity_intensity` = ?, 
    `vaginal_dryness` = ?, `vaginal_burning` = ?, `vaginal_burning_intensity` = ?, `vaginal_itching` = 
    ?, `vaginal_itching_intensity` = ?, `vaginal_smell_change` = ?, `urination_change` = ?, 
    `urination_burning_intensity` = ?, `acne` = ?, `oily_skin` = ?, `hair_loss` = ?, `swelling` = ?, 
    `fatigue` = ?, `dizziness` = ?, `hot_flashes` = ?, `chills` = ?, `moods` = ?, `sleep_duration` = ?, 
    `sleep_quality` = ?, `exercise_type` = ?, `exercise_duration` = ?, `exercise_intensity` = ?, 
    `sexual_activities` = ?, `sexual_desire` = ?, `intercourse_type` = ?, `weight` = ?, 
    `basal_body_temperature` = ?, `heart_rate` = ?, `systolic_pressure` = ?, `diastolic_pressure` = ?, 
    `blood_sugar` = ?, `energy_level` = ?, `discharge_color` = ?, `discharge_texture` = ?, 
    `discharge_amount` = ?, `discharge_smell` = ?, `discharge_itching` = ?, `discharge_burning` = ?, 
    `frequent_urination` = ?, `medications` = ?, `notes` = ?, `updated_at` = ?
WHERE id = ?;

-- name: DeleteDailyHealthLog :exec
DELETE FROM `daily_health_logs` WHERE id = ?;

-- name: LastBleedingDateBetween :one
-- CycleHistoryService::updatePreviousPeriodEndDate: the last bleeding day in [from, before).
SELECT log_date FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id)
  AND log_date >= sqlc.arg(from_date)
  AND log_date < sqlc.arg(before_date)
  AND bleeding_intensity IS NOT NULL
ORDER BY log_date DESC
LIMIT 1;
