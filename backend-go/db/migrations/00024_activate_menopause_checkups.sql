-- CB-MENO-01b: activate the nine menopause checkups seeded inactive by 00022_menopause.
--
-- 00022 seeded them with audiences ["menopause"] and is_active = 0 because nothing read checkup_types.audiences yet.
-- Since CB-MENO-01b the checkups user API (internal/checkups: plan, home card, detail, records, search) lists only
-- types whose audiences is NULL or contains the user's resolved life mode, so active rows are shown to
-- menopause-mode users only. Data only (no schema change): no Laravel twin, `make schema-diff` compares schema and
-- row counts, not values.

-- +goose Up
UPDATE `checkup_types` SET `is_active` = 1
WHERE `user_id` IS NULL
  AND `key` IN ('meno_blood_pressure', 'meno_blood_sugar', 'meno_lipids', 'meno_weight_waist', 'meno_bone_density',
                'meno_vitamin_d_calcium', 'meno_colon_screening', 'meno_thyroid', 'meno_eye_exam');

-- +goose Down
UPDATE `checkup_types` SET `is_active` = 0
WHERE `user_id` IS NULL
  AND `key` IN ('meno_blood_pressure', 'meno_blood_sugar', 'meno_lipids', 'meno_weight_waist', 'meno_bone_density',
                'meno_vitamin_d_calcium', 'meno_colon_screening', 'meno_thyroid', 'meno_eye_exam');
