-- PregnancySymptomLog (App\Models\PregnancySymptomLog). Unique (user_id, log_date).
-- from/to are compared as the raw query-string text, like Laravel's where('log_date', '>=', $from).

-- name: ListSymptomLogs :many
SELECT * FROM `pregnancy_symptom_logs`
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.arg(has_from) = 0 OR log_date >= CAST(sqlc.arg(from_value) AS CHAR))
  AND (sqlc.arg(has_to) = 0 OR log_date <= CAST(sqlc.arg(to_value) AS CHAR))
ORDER BY log_date DESC;

-- name: GetSymptomLog :one
SELECT * FROM `pregnancy_symptom_logs` WHERE user_id = ? AND log_date = ? LIMIT 1;

-- name: DeleteSymptomLog :execrows
DELETE FROM `pregnancy_symptom_logs` WHERE user_id = ? AND log_date = ?;

-- name: InsertSymptomLog :execlastid
INSERT INTO `pregnancy_symptom_logs` (
  user_id, log_date,
  has_nausea, nausea_severity, has_vomiting, vomiting_severity, has_fatigue, fatigue_severity,
  has_headache, headache_severity, has_dizziness, dizziness_severity, has_breast_pain, breast_pain_severity,
  has_lower_abdominal_pain, lower_abdominal_pain_severity, has_cramping, cramping_severity,
  has_back_pain, back_pain_severity, has_pelvic_pressure, pelvic_pressure_severity,
  has_spotting, spotting_severity, has_bleeding, bleeding_severity, has_fluid_leakage, fluid_leakage_severity,
  has_severe_sudden_pain, severe_sudden_pain_severity, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateSymptomLog :exec
UPDATE `pregnancy_symptom_logs` SET
  has_nausea = ?, nausea_severity = ?, has_vomiting = ?, vomiting_severity = ?, has_fatigue = ?, fatigue_severity = ?,
  has_headache = ?, headache_severity = ?, has_dizziness = ?, dizziness_severity = ?, has_breast_pain = ?, breast_pain_severity = ?,
  has_lower_abdominal_pain = ?, lower_abdominal_pain_severity = ?, has_cramping = ?, cramping_severity = ?,
  has_back_pain = ?, back_pain_severity = ?, has_pelvic_pressure = ?, pelvic_pressure_severity = ?,
  has_spotting = ?, spotting_severity = ?, has_bleeding = ?, bleeding_severity = ?, has_fluid_leakage = ?, fluid_leakage_severity = ?,
  has_severe_sudden_pain = ?, severe_sudden_pain_severity = ?, notes = ?, updated_at = ?
WHERE id = ?;
