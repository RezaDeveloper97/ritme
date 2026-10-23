-- Fixture for TestManifest_FailsOnUnlistedWrite (T-M9-30): a write query on an engine table that the manifest does
-- not list. Never compiled by sqlc (sqlc reads db/queries only).

-- name: FakeReadHistories :many
SELECT * FROM `cycle_histories` WHERE user_id = ?;

-- name: FakeTouchHistories :exec
-- A comment line inside the block is ignored.
UPDATE `cycle_histories` SET updated_at = ? WHERE user_id = ?;

-- name: FakeUpsertRevision :exec
INSERT INTO `user_revisions` (user_id, engine_rev, analysis_rev, updated_at) VALUES (?, 1, 1, ?)
ON DUPLICATE KEY UPDATE engine_rev = engine_rev + 1;

-- name: FakeDeleteAliased :exec
DELETE h FROM `cycle_skips` h JOIN `users` u ON u.id = h.user_id WHERE u.id = ?;
