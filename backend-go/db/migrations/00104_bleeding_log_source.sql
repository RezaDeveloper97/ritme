-- 00104_bleeding_log_source.sql — periods built from the bleeding log become actual data (T-M9-28; SPEC v6 §1.4, §2.2,
-- §2.9; docs/engine-v6/DECISIONS.md I-02; deviations D-124, D-138). DATA ONLY, no schema change (cycle_histories.source
-- is a varchar), Go-only: no Laravel twin (production runs Laravel and is not touched; `make schema-diff` compares
-- schemas and row counts of a fresh database, where there is nothing to convert). D-138: add `bleeding_log` to
-- Laravel's DataSource.php before any cutover.
--
-- Until T-M9-28 the bleeding-log bridge (Laravel's CycleHistoryService::checkAndUpdatePeriodStart and the Go port, D-55)
-- inserted unconfirmed, non-estimated `user_logged` rows; the period button and every other writer store confirmed
-- rows (T-M9-27 already turned edited onboarding seeds into confirmed user_logged rows). The engine promoted only the
-- newest such row (model.PromoteLoggedStart, removed). The bridge now writes confirmed `bleeding_log` rows and rebuilds
-- only those.
-- Up:   every unconfirmed, non-estimated user_logged row → source bleeding_log, is_confirmed = 1; user_revisions
--       (engine_rev, analysis_rev) + 1 for every user with a converted row.
-- Down: by source — every bleeding_log row (also those the v6 bridge wrote after Up) → user_logged, unconfirmed; the
--       revisions of those users are bumped again (their engine input changed back).
--
-- "now" is the Tehran wall clock (the session time_zone is never set; Iran has had no DST since 2022).

-- +goose Up
INSERT INTO `user_revisions` (user_id, engine_rev, analysis_rev, updated_at)
SELECT DISTINCT user_id, 1, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30') FROM `cycle_histories`
WHERE source = 'user_logged' AND is_confirmed = 0 AND is_estimated = 0
ON DUPLICATE KEY UPDATE
  engine_rev = engine_rev + 1,
  analysis_rev = analysis_rev + 1,
  updated_at = VALUES(updated_at);

UPDATE `cycle_histories` SET source = 'bleeding_log', is_confirmed = 1
WHERE source = 'user_logged' AND is_confirmed = 0 AND is_estimated = 0;

-- +goose Down
INSERT INTO `user_revisions` (user_id, engine_rev, analysis_rev, updated_at)
SELECT DISTINCT user_id, 1, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30') FROM `cycle_histories`
WHERE source = 'bleeding_log'
ON DUPLICATE KEY UPDATE
  engine_rev = engine_rev + 1,
  analysis_rev = analysis_rev + 1,
  updated_at = VALUES(updated_at);

UPDATE `cycle_histories` SET source = 'user_logged', is_confirmed = 0 WHERE source = 'bleeding_log';
