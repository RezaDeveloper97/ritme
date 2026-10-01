-- Profile & onboarding v2 (B-N2-01, goose 00014): GET/PUT /onboarding*, GET/PUT /profile/life-stage, Go only.
-- One `user_life_profiles` row per user; every statement is scoped by user_id.

-- name: GetLifeProfile :one
SELECT * FROM `user_life_profiles` WHERE user_id = ? LIMIT 1;

-- name: UpsertLifeProfile :exec
-- Writes the whole row (the service loads it, applies one step, writes it back). started/completed keep their
-- first value.
INSERT INTO `user_life_profiles`
  (user_id, gender, life_mode, ivf_iui, track_contraception, chronic_illnesses, gyn_conditions, medications,
   menopause_stage, menopause_last_period, menopause_surgical, menopause_hrt,
   onboarding_started_at, onboarding_completed_at, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.narg(gender), sqlc.narg(life_mode), sqlc.arg(ivf_iui), sqlc.arg(track_contraception),
   sqlc.narg(chronic_illnesses), sqlc.narg(gyn_conditions), sqlc.narg(medications),
   sqlc.narg(menopause_stage), sqlc.narg(menopause_last_period), sqlc.narg(menopause_surgical), sqlc.narg(menopause_hrt),
   sqlc.narg(onboarding_started_at), sqlc.narg(onboarding_completed_at), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  gender = VALUES(gender),
  life_mode = VALUES(life_mode),
  ivf_iui = VALUES(ivf_iui),
  track_contraception = VALUES(track_contraception),
  chronic_illnesses = VALUES(chronic_illnesses),
  gyn_conditions = VALUES(gyn_conditions),
  medications = VALUES(medications),
  menopause_stage = VALUES(menopause_stage),
  menopause_last_period = VALUES(menopause_last_period),
  menopause_surgical = VALUES(menopause_surgical),
  menopause_hrt = VALUES(menopause_hrt),
  onboarding_started_at = COALESCE(onboarding_started_at, VALUES(onboarding_started_at)),
  onboarding_completed_at = COALESCE(onboarding_completed_at, VALUES(onboarding_completed_at)),
  updated_at = VALUES(updated_at);

-- name: PregnancyModeActive :one
-- The legacy mode authority: an active pregnancy profile (pregnancy_mode = 1) wins over the stored life mode.
SELECT EXISTS(SELECT 1 FROM `pregnancy_profiles` WHERE user_id = ? AND pregnancy_mode = 1) AS active;
