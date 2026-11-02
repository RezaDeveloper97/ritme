-- Engine v6 input loader (T-M9-25, internal/cycle/inputs): everything engine.Input needs for a batch of users in
-- exactly four queries. Load is LoadMany of one id, so both run these same statements (IN lists of ≤ 500 ids).
-- Health data: every query is scoped by user_id (IDOR). Mapping rules live in Go (internal/cycle/inputs), not here.

-- name: ListEngineInputProfiles :many
-- 1/4 — one row per existing user: the declared data of user_profiles, the «خودکار از داده‌ها» switch
-- (no cycle_preferences row = automatic), the stored life mode and onboarding medications, the active pregnancy
-- (pregnancy_mode = 1 wins the mode), the postpartum delivery date and breastfeeding flag (I-04), the active
-- contraception method (a row exists only while tracking, I-03) with the dates its start is taken from (E-13,
-- T-M9-48d: created_at and the method's own pack/insertion/injection date) and the teen age band. Every table holds
-- at most one row per user (UNIQUE user_id), so the joins never fan out.
SELECT
  u.id AS user_id,
  p.birthday,
  p.period_duration,
  p.cycle_duration,
  p.onboarding_last_period_start,
  p.user_goal,
  COALESCE(cp.lengths_auto, 1) AS lengths_auto,
  lp.life_mode,
  lp.medications,
  CAST(pg.id IS NOT NULL AS UNSIGNED) AS pregnancy_active,
  pp.birth_date AS delivery_date,
  pp.breastfeeding,
  cm.method AS contraception_method,
  cm.created_at AS contraception_created_at,
  cm.pack_started_on AS contraception_pack_started_on,
  cm.inserted_on AS contraception_inserted_on,
  cm.injected_on AS contraception_injected_on,
  tp.age_band
FROM `users` u
LEFT JOIN `user_profiles` p ON p.user_id = u.id
LEFT JOIN `cycle_preferences` cp ON cp.user_id = u.id
LEFT JOIN `user_life_profiles` lp ON lp.user_id = u.id
LEFT JOIN `pregnancy_profiles` pg ON pg.user_id = u.id AND pg.pregnancy_mode = 1
LEFT JOIN `postpartum_profiles` pp ON pp.user_id = u.id
LEFT JOIN `contraception_methods` cm ON cm.user_id = u.id
LEFT JOIN `teen_profiles` tp ON tp.user_id = u.id
WHERE u.id IN (sqlc.slice(user_ids));

-- name: ListEngineInputPeriods :many
-- 2/4 — every cycle_histories row of the users, narrow projection, oldest start first. Estimated rows and
-- user_profile_confirmed seed rows are sorted out in Go (§2.2, I-01, I-02).
SELECT user_id, period_start_date, period_end_date, source, is_confirmed, is_estimated
FROM `cycle_histories`
WHERE user_id IN (sqlc.slice(user_ids))
ORDER BY user_id, period_start_date, id;

-- name: ListEngineInputSkips :many
-- 3/4 — the skipped predicted starts «پریود نداشتم» (E-15, goose 00098), passed to the engine as stored.
SELECT user_id, expected_start
FROM `cycle_skips`
WHERE user_id IN (sqlc.slice(user_ids))
ORDER BY user_id, expected_start;

-- name: ListEngineInputMarkers :many
-- 4/4 — the fertility markers between from_date and to_date (inclusive; Go narrows each user to her own window):
-- taxonomy v2 measurements (lh_test, bbt, pregnancy_test), the /fertility LH log and the legacy BBT column. The
-- kind column says where a row came from: the v2 param, 'fertility_lh' or 'legacy_bbt'.
SELECT e.user_id, e.log_date, e.param AS kind, e.value_code, e.value_num
FROM `health_log_entries` e
WHERE e.user_id IN (sqlc.slice(entry_user_ids))
  AND e.category = 'measurements'
  AND e.param IN ('lh_test', 'bbt', 'pregnancy_test')
  AND e.item = ''
  AND e.log_date >= sqlc.arg(from_date)
  AND e.log_date <= sqlc.arg(to_date)
UNION ALL
SELECT f.user_id, f.log_date, 'fertility_lh', f.lh_test, NULL
FROM `fertility_logs` f
WHERE f.user_id IN (sqlc.slice(fertility_user_ids))
  AND f.lh_test IS NOT NULL
  AND f.log_date >= sqlc.arg(from_date)
  AND f.log_date <= sqlc.arg(to_date)
UNION ALL
SELECT d.user_id, d.log_date, 'legacy_bbt', NULL, d.basal_body_temperature
FROM `daily_health_logs` d
WHERE d.user_id IN (sqlc.slice(daily_user_ids))
  AND d.basal_body_temperature IS NOT NULL
  AND d.log_date >= sqlc.arg(from_date)
  AND d.log_date <= sqlc.arg(to_date);
