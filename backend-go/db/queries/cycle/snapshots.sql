-- Engine daily snapshots and the prediction log (T-M9-39, D-137; goose 00109; DECISIONS.md A-05, A-06). Analytics
-- only: written by internal/cycle/snapshots, never read by the engine.

-- name: ListActiveSnapshotUsers :many
-- One page (keyset on users.id) of the users active since the cutoff (A-06: any session, log or event in the last
-- 30 days) with their stored zone. Session = a token issued or refreshed, or a session day; log = a daily log, a v2
-- log entry, a period row or a logged day; event = a first-party analytics event. `since` is the Tehran wall clock
-- (the DB timestamps), `since_day` its date.
SELECT u.id AS user_id, p.timezone
FROM `users` u
LEFT JOIN `user_profiles` p ON p.user_id = u.id
WHERE u.id > sqlc.arg(after_id)
  AND u.id IN (
    SELECT t.user_id FROM `oauth_access_tokens` t WHERE t.user_id IS NOT NULL AND t.updated_at >= sqlc.arg(since)
    UNION SELECT a.user_id FROM `user_activity_days` a WHERE a.`day` >= sqlc.arg(since_day)
    UNION SELECT l.user_id FROM `daily_health_logs` l WHERE l.log_date >= sqlc.arg(since_day)
    UNION SELECT e.user_id FROM `health_log_entries` e WHERE e.updated_at >= sqlc.arg(since)
    UNION SELECT c.user_id FROM `cycle_histories` c WHERE c.updated_at >= sqlc.arg(since)
    UNION SELECT v.user_id FROM `analytics_events` v WHERE v.occurred_on >= sqlc.arg(since_day)
  )
ORDER BY u.id
LIMIT ?;

-- name: ListUserEngineRevs :many
-- engine_rev of each user that has a user_revisions row (a missing row is revision 0).
SELECT user_id, engine_rev FROM `user_revisions` WHERE user_id IN (sqlc.slice(user_ids));

-- name: UpsertEngineDailySnapshot :exec
-- One user and day; a rerun the same day overwrites the codes (created_at keeps the first run).
INSERT INTO `engine_daily_snapshots`
  (user_id, `day`, main_phase, subphase, confidence, data_quality, warnings_mask, days_late, `mode`, engine_rev, created_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(day), sqlc.arg(main_phase), sqlc.arg(subphase), sqlc.arg(confidence),
   sqlc.arg(data_quality), sqlc.arg(warnings_mask), sqlc.narg(days_late), sqlc.arg(mode), sqlc.arg(engine_rev),
   sqlc.arg(created_at))
ON DUPLICATE KEY UPDATE
  main_phase = VALUES(main_phase),
  subphase = VALUES(subphase),
  confidence = VALUES(confidence),
  data_quality = VALUES(data_quality),
  warnings_mask = VALUES(warnings_mask),
  days_late = VALUES(days_late),
  `mode` = VALUES(`mode`),
  engine_rev = VALUES(engine_rev);

-- name: PurgeEngineDailySnapshots :execrows
-- Retention (400 days): one chunk of rows older than `before`; the job repeats until a chunk is short.
DELETE FROM `engine_daily_snapshots` WHERE `day` < sqlc.arg(before) LIMIT 5000;

-- name: InsertFirstPrediction :exec
-- The first prediction of a cycle: written once, a rerun keeps it.
INSERT IGNORE INTO `engine_prediction_log`
  (user_id, cycle_start, kind, predicted_next, basis_cycles, ovulation_source, confidence, created_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(cycle_start), 'first', sqlc.arg(predicted_next), sqlc.arg(basis_cycles),
   sqlc.arg(ovulation_source), sqlc.arg(confidence), sqlc.arg(created_at));

-- name: UpsertLastPrediction :exec
-- The last prediction of a cycle before its next real start; a corrected next start replaces it.
INSERT INTO `engine_prediction_log`
  (user_id, cycle_start, kind, predicted_next, basis_cycles, ovulation_source, confidence, created_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(cycle_start), 'last', sqlc.arg(predicted_next), sqlc.arg(basis_cycles),
   sqlc.arg(ovulation_source), sqlc.arg(confidence), sqlc.arg(created_at))
ON DUPLICATE KEY UPDATE
  predicted_next = VALUES(predicted_next),
  basis_cycles = VALUES(basis_cycles),
  ovulation_source = VALUES(ovulation_source),
  confidence = VALUES(confidence);

-- name: ResolvePredictionCycle :execrows
-- The next real start of cycle_start is known: actual_start and error_days (actual − predicted, signed) on its rows.
UPDATE `engine_prediction_log`
SET actual_start = sqlc.arg(actual_start),
    error_days = DATEDIFF(actual_start, predicted_next)
WHERE user_id = sqlc.arg(user_id) AND cycle_start = sqlc.arg(cycle_start);
