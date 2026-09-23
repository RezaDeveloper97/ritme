-- PregnancyWeeklyLog (App\Models\PregnancyWeeklyLog). Unique (user_id, pregnancy_week).

-- name: ListWeeklyLogs :many
SELECT * FROM `pregnancy_weekly_logs` WHERE user_id = ? ORDER BY pregnancy_week DESC;

-- name: GetWeeklyLog :one
SELECT * FROM `pregnancy_weekly_logs` WHERE user_id = ? AND pregnancy_week = ? LIMIT 1;

-- name: InsertWeeklyLog :execlastid
INSERT INTO `pregnancy_weekly_logs` (
  user_id, log_date, pregnancy_week, weight, swelling_locations, has_swelling, has_shortness_of_breath,
  has_blood_pressure_device, systolic_pressure, diastolic_pressure, fasting_blood_sugar, post_meal_blood_sugar,
  overall_mood, has_anxiety, anxiety_severity, has_mood_swings, mood_swings_severity,
  has_depression_feelings, depression_severity, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateWeeklyLog :exec
UPDATE `pregnancy_weekly_logs` SET
  log_date = ?, weight = ?, swelling_locations = ?, has_swelling = ?, has_shortness_of_breath = ?,
  has_blood_pressure_device = ?, systolic_pressure = ?, diastolic_pressure = ?, fasting_blood_sugar = ?,
  post_meal_blood_sugar = ?, overall_mood = ?, has_anxiety = ?, anxiety_severity = ?, has_mood_swings = ?,
  mood_swings_severity = ?, has_depression_feelings = ?, depression_severity = ?, notes = ?, updated_at = ?
WHERE id = ?;
