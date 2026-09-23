-- PregnancyProfile (App\Models\PregnancyProfile). One row per user (unique user_id).

-- name: GetProfileByUser :one
SELECT * FROM `pregnancy_profiles` WHERE user_id = ? LIMIT 1;

-- name: InsertProfile :execlastid
INSERT INTO `pregnancy_profiles` (
  user_id, pregnancy_mode, cycle_mode, is_locked, age_source, confidence_level,
  lmp_date, ultrasound_date, ultrasound_weeks, ultrasound_days, manual_weeks, manual_days, manual_entry_date,
  estimated_due_date, estimated_conception_date, uncertainty_days,
  has_miscarriage_history, has_high_risk_history, pre_existing_conditions,
  blood_type, rh_factor, rh_negative_care_flag, first_fetal_movement_date, fetal_movement_felt,
  onboarding_completed, onboarding_completed_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProfile :exec
UPDATE `pregnancy_profiles` SET
  pregnancy_mode = ?, cycle_mode = ?, is_locked = ?, age_source = ?, confidence_level = ?,
  lmp_date = ?, ultrasound_date = ?, ultrasound_weeks = ?, ultrasound_days = ?, manual_weeks = ?, manual_days = ?,
  manual_entry_date = ?, estimated_due_date = ?, estimated_conception_date = ?, uncertainty_days = ?,
  has_miscarriage_history = ?, has_high_risk_history = ?, pre_existing_conditions = ?,
  blood_type = ?, rh_factor = ?, rh_negative_care_flag = ?, first_fetal_movement_date = ?, fetal_movement_felt = ?,
  onboarding_completed = ?, onboarding_completed_at = ?, updated_at = ?
WHERE id = ?;
