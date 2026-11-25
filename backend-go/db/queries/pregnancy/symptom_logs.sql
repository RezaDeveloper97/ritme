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

-- name: RecordFlowBleedingSymptom :exec
-- A flow logged in pregnancy on the general day log (healthlog bridge, T-M10-08b) is the day's bleeding symptom: the
-- day's row is created or gets has_bleeding + the severity (its other symptoms are kept). The caller never lowers a
-- severity the user logged herself.
INSERT INTO `pregnancy_symptom_logs` (user_id, log_date, has_bleeding, bleeding_severity, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(log_date), 1, sqlc.arg(bleeding_severity), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE has_bleeding = 1, bleeding_severity = VALUES(bleeding_severity), updated_at = VALUES(updated_at);

-- name: RecordSpottingSymptom :exec
-- Spotting logged in pregnancy on the general day log (healthlog bridge) is the day's spotting symptom: the day's row
-- is created or gets has_spotting (its other symptoms are kept). A severity the user logged herself is kept; a new
-- flag starts at the given severity.
INSERT INTO `pregnancy_symptom_logs` (user_id, log_date, has_spotting, spotting_severity, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(log_date), 1, sqlc.arg(spotting_severity), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  spotting_severity = IF(has_spotting = 1 AND spotting_severity IS NOT NULL, spotting_severity, VALUES(spotting_severity)),
  has_spotting = 1, updated_at = VALUES(updated_at);
