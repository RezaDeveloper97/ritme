-- Placeholder so the `pregnancy` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/pregnancy/.

-- name: SamplePregnancyProfile :one
SELECT * FROM `pregnancy_profiles` WHERE id = ? LIMIT 1;
