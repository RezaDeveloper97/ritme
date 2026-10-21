-- Engine v6 per-user revisions (T-M9-24, DECISIONS.md X-02; table user_revisions, goose 00098). engine_rev is bumped
-- inside every engine-input write transaction (pass the tx to store.New), analysis_rev with every engine or
-- analysis-input write; caches key on them. A user without a row reads as 0 / 0 (sql.ErrNoRows). `now` is the Tehran
-- wall-clock (the session time_zone is never set, so NOW() is not used).

-- name: BumpUserRevisions :exec
-- An engine-input write: both revisions + 1 (the first bump creates the row at 1 / 1). Atomic on the row, so
-- concurrent bumps never lose one.
INSERT INTO `user_revisions` (user_id, engine_rev, analysis_rev, updated_at)
VALUES (sqlc.arg(user_id), 1, 1, sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  engine_rev = engine_rev + 1,
  analysis_rev = analysis_rev + 1,
  updated_at = VALUES(updated_at);

-- name: BumpAnalysisRevision :exec
-- An analysis-only write: analysis_rev + 1, engine_rev unchanged (the first bump creates the row at 0 / 1).
INSERT INTO `user_revisions` (user_id, engine_rev, analysis_rev, updated_at)
VALUES (sqlc.arg(user_id), 0, 1, sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  analysis_rev = analysis_rev + 1,
  updated_at = VALUES(updated_at);

-- name: GetUserRevisions :one
SELECT engine_rev, analysis_rev FROM `user_revisions` WHERE user_id = ? LIMIT 1;
