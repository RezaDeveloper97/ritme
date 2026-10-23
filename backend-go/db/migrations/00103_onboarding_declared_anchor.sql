-- 00103_onboarding_declared_anchor.sql — the declared onboarding LMP leaves cycle_histories (T-M9-27; SPEC v6 §2.2,
-- §2.9, §2.40, §2.41; docs/engine-v6/DECISIONS.md I-01; deviation D-123). DATA ONLY, no schema change, Go-only: no
-- Laravel twin (production runs Laravel and is not touched; `make schema-diff` compares schemas and row counts of a
-- fresh database, where there is nothing to move).
--
-- Until T-M9-27 profile.Save turned the declared LMP into a confirmed `user_profile_confirmed` cycle_histories row with
-- an assumed end (syncOnboardingPeriodLog, removed). The declared anchor now lives in
-- user_profiles.onboarding_last_period_start (00098). Every seed row is classified with the old re-sync rule:
--   d = GREATEST(1, COALESCE(NULLIF(period_duration, 0), 5))   (the profile's period duration, 5 when unknown)
--   untouched = confirmed, not estimated, and
--     (end = start + d − 1 AND bleeding_length = d)          the re-sync of a past period
--     OR (end IS NULL AND bleeding_length IS NULL)          the re-sync while the period was still running
--   and the start is the profile's last_period_start — unless a later non-seed period exists: logging a period moves
--   last_period_start to the latest actual start, so then the start is not compared.
--   cycle_length is not compared (the neighbour recompute writes it).
-- Up:
--   1. untouched seed → its start is copied into onboarding_last_period_start (the newest one when there are several;
--      a value already there is kept) and the row is deleted;
--   2. any other seed (edited by the user, by the bleeding-log bridge, or with no profile) → source = user_logged,
--      is_confirmed = 1: the user said it is real;
--   3. user_revisions (engine_rev, analysis_rev) + 1 for every user that had a seed, in the same migration.
-- Down: the seeds are re-created from onboarding_last_period_start with the old re-sync rule (end and
-- bleeding_length NULL while start + d − 1 is after today, Asia/Tehran), unless a row with that start already exists;
-- the column goes back to NULL (nothing wrote it before 00103), and the revisions of those users are bumped. Converted
-- rows stay user_logged/confirmed (the conversion is not recorded, so it cannot be undone), and a re-created seed has
-- the old rule's end, not the one the deleted row had (which was the same rule's output by definition, except for an
-- ongoing seed that has finished since).
--
-- "now" is the Tehran wall clock (the session time_zone is never set; Iran has had no DST since 2022).

-- +goose Up
CREATE TEMPORARY TABLE `t_m9_27_seeds` (
  `id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `start` date NOT NULL,
  `untouched` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`)
);

INSERT INTO `t_m9_27_seeds` (id, user_id, start, untouched)
SELECT h.id, h.user_id, h.period_start_date,
  p.id IS NOT NULL AND h.is_confirmed = 1 AND h.is_estimated = 0
  AND (
    (h.period_end_date = DATE_ADD(h.period_start_date, INTERVAL GREATEST(1, COALESCE(NULLIF(p.period_duration, 0), 5)) - 1 DAY)
      AND h.bleeding_length = GREATEST(1, COALESCE(NULLIF(p.period_duration, 0), 5)))
    OR (h.period_end_date IS NULL AND h.bleeding_length IS NULL)
  )
  AND (
    h.period_start_date <=> p.last_period_start
    OR EXISTS (SELECT 1 FROM `cycle_histories` l WHERE l.user_id = h.user_id AND l.source <> 'user_profile_confirmed'
               AND l.is_estimated = 0 AND l.period_start_date > h.period_start_date)
  )
FROM `cycle_histories` h
LEFT JOIN `user_profiles` p ON p.user_id = h.user_id
WHERE h.source = 'user_profile_confirmed';

INSERT INTO `user_revisions` (user_id, engine_rev, analysis_rev, updated_at)
SELECT DISTINCT user_id, 1, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30') FROM `t_m9_27_seeds`
ON DUPLICATE KEY UPDATE
  engine_rev = engine_rev + 1,
  analysis_rev = analysis_rev + 1,
  updated_at = VALUES(updated_at);

UPDATE `user_profiles` p
JOIN (SELECT user_id, MAX(start) AS start FROM `t_m9_27_seeds` WHERE untouched = 1 GROUP BY user_id) s ON s.user_id = p.user_id
SET p.onboarding_last_period_start = s.start
WHERE p.onboarding_last_period_start IS NULL;

DELETE h FROM `cycle_histories` h JOIN `t_m9_27_seeds` s ON s.id = h.id WHERE s.untouched = 1;

UPDATE `cycle_histories` h JOIN `t_m9_27_seeds` s ON s.id = h.id
SET h.source = 'user_logged', h.is_confirmed = 1
WHERE s.untouched = 0;

DROP TEMPORARY TABLE `t_m9_27_seeds`;

-- +goose Down
INSERT INTO `user_revisions` (user_id, engine_rev, analysis_rev, updated_at)
SELECT user_id, 1, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30') FROM `user_profiles`
WHERE onboarding_last_period_start IS NOT NULL
ON DUPLICATE KEY UPDATE
  engine_rev = engine_rev + 1,
  analysis_rev = analysis_rev + 1,
  updated_at = VALUES(updated_at);

INSERT INTO `cycle_histories`
  (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source, created_at, updated_at)
SELECT p.user_id, p.onboarding_last_period_start,
  IF(e.end_date > e.today, NULL, e.end_date),
  IF(e.end_date > e.today, NULL, e.d),
  1, 0, 'user_profile_confirmed', e.now, e.now
FROM `user_profiles` p
JOIN (
  SELECT p2.id,
    GREATEST(1, COALESCE(NULLIF(p2.period_duration, 0), 5)) AS d,
    DATE_ADD(p2.onboarding_last_period_start, INTERVAL GREATEST(1, COALESCE(NULLIF(p2.period_duration, 0), 5)) - 1 DAY) AS end_date,
    DATE(CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')) AS today,
    CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30') AS now
  FROM `user_profiles` p2 WHERE p2.onboarding_last_period_start IS NOT NULL
) e ON e.id = p.id
WHERE NOT EXISTS (SELECT 1 FROM `cycle_histories` h WHERE h.user_id = p.user_id AND h.period_start_date = p.onboarding_last_period_start);

UPDATE `user_profiles` SET onboarding_last_period_start = NULL WHERE onboarding_last_period_start IS NOT NULL;
