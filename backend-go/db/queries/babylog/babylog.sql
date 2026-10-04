-- Baby logs (bloom B-N5-03; internal/babylog): feeds, sleeps and diapers of a child. Every query is scoped by
-- child_id; the caller resolved the child through children.Service.Access (owner, or spouse read-only) first.
-- active_lock = 1 marks the running session (UNIQUE(child_id, active_lock)); NULL once it ended.

-- name: InsertFeed :execlastid
INSERT INTO `baby_feeds` (
  child_id, type, started_at, ended_at, active_side, side_started_at, last_side, left_seconds, right_seconds,
  duration_seconds, amount_ml, note, active_lock, created_at, updated_at
) VALUES (
  sqlc.arg(child_id), sqlc.arg(type), sqlc.arg(started_at), sqlc.narg(ended_at), sqlc.narg(active_side),
  sqlc.narg(side_started_at), sqlc.narg(last_side), sqlc.arg(left_seconds), sqlc.arg(right_seconds),
  sqlc.narg(duration_seconds), sqlc.narg(amount_ml), sqlc.narg(note), sqlc.narg(active_lock), sqlc.arg(now), sqlc.arg(now)
);

-- name: GetFeed :one
SELECT * FROM `baby_feeds` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id) LIMIT 1;

-- name: LockFeed :one
-- The feed row inside a write transaction (side switch / stop / edit read-modify-write).
SELECT * FROM `baby_feeds` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id) LIMIT 1 FOR UPDATE;

-- name: GetActiveFeed :one
SELECT * FROM `baby_feeds` WHERE child_id = sqlc.arg(child_id) AND active_lock = 1 LIMIT 1;

-- name: UpdateFeed :exec
UPDATE `baby_feeds`
SET type = sqlc.arg(type), started_at = sqlc.arg(started_at), ended_at = sqlc.narg(ended_at),
    active_side = sqlc.narg(active_side), side_started_at = sqlc.narg(side_started_at), last_side = sqlc.narg(last_side),
    left_seconds = sqlc.arg(left_seconds), right_seconds = sqlc.arg(right_seconds),
    duration_seconds = sqlc.narg(duration_seconds), amount_ml = sqlc.narg(amount_ml), note = sqlc.narg(note),
    active_lock = sqlc.narg(active_lock), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id);

-- name: DeleteFeed :execrows
DELETE FROM `baby_feeds` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id);

-- name: ListFeedsBetween :many
-- Feeds started in [date_from, date_to), newest first.
SELECT * FROM `baby_feeds`
WHERE child_id = sqlc.arg(child_id) AND started_at >= sqlc.arg(date_from) AND started_at < sqlc.arg(date_to)
ORDER BY started_at DESC, id DESC;

-- name: LastEndedFeed :one
SELECT * FROM `baby_feeds`
WHERE child_id = sqlc.arg(child_id) AND ended_at IS NOT NULL
ORDER BY ended_at DESC, id DESC LIMIT 1;

-- name: LastBreastSide :one
-- The side the last ended breast feed ended on (the side to offer next is the other one).
SELECT last_side FROM `baby_feeds`
WHERE child_id = sqlc.arg(child_id) AND ended_at IS NOT NULL AND type = 'breast' AND last_side IS NOT NULL
ORDER BY ended_at DESC, id DESC LIMIT 1;

-- name: CountOwnerFeedsBetween :one
-- Feeds of every child of owner_id started in [date_from, date_to) (the mother's baby.feeds_count).
SELECT COUNT(*) FROM `baby_feeds` f JOIN `children` c ON c.id = f.child_id
WHERE c.owner_id = sqlc.arg(owner_id) AND f.started_at >= sqlc.arg(date_from) AND f.started_at < sqlc.arg(date_to);

-- name: InsertSleep :execlastid
INSERT INTO `baby_sleeps` (child_id, started_at, ended_at, note, active_lock, created_at, updated_at)
VALUES (sqlc.arg(child_id), sqlc.arg(started_at), sqlc.narg(ended_at), sqlc.narg(note), sqlc.narg(active_lock), sqlc.arg(now), sqlc.arg(now));

-- name: GetSleep :one
SELECT * FROM `baby_sleeps` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id) LIMIT 1;

-- name: GetActiveSleep :one
SELECT * FROM `baby_sleeps` WHERE child_id = sqlc.arg(child_id) AND active_lock = 1 LIMIT 1;

-- name: UpdateSleep :exec
UPDATE `baby_sleeps`
SET started_at = sqlc.arg(started_at), ended_at = sqlc.narg(ended_at), note = sqlc.narg(note),
    active_lock = sqlc.narg(active_lock), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id);

-- name: StopSleep :execrows
-- Ends the running sleep (0 rows = it ended meanwhile).
UPDATE `baby_sleeps`
SET ended_at = sqlc.arg(ended_at), note = COALESCE(sqlc.narg(note), note), active_lock = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id) AND active_lock = 1;

-- name: DeleteSleep :execrows
DELETE FROM `baby_sleeps` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id);

-- name: ListSleepsOverlapping :many
-- Sleeps overlapping [date_from, date_to) (running ones included), newest first.
SELECT * FROM `baby_sleeps`
WHERE child_id = sqlc.arg(child_id) AND started_at < sqlc.arg(date_to)
  AND (ended_at IS NULL OR ended_at > sqlc.arg(date_from))
ORDER BY started_at DESC, id DESC;

-- name: InsertDiaper :execlastid
INSERT INTO `baby_diapers` (child_id, changed_at, kind, note, created_at, updated_at)
VALUES (sqlc.arg(child_id), sqlc.arg(changed_at), sqlc.arg(kind), sqlc.narg(note), sqlc.arg(now), sqlc.arg(now));

-- name: GetDiaper :one
SELECT * FROM `baby_diapers` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id) LIMIT 1;

-- name: UpdateDiaper :exec
UPDATE `baby_diapers`
SET changed_at = sqlc.arg(changed_at), kind = sqlc.arg(kind), note = sqlc.narg(note), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id);

-- name: DeleteDiaper :execrows
DELETE FROM `baby_diapers` WHERE id = sqlc.arg(id) AND child_id = sqlc.arg(child_id);

-- name: ListDiapersBetween :many
SELECT * FROM `baby_diapers`
WHERE child_id = sqlc.arg(child_id) AND changed_at >= sqlc.arg(date_from) AND changed_at < sqlc.arg(date_to)
ORDER BY changed_at DESC, id DESC;

-- name: HasPostpartumProfile :one
-- The mother's baby.feeds_count is written only while she has a postpartum profile.
SELECT EXISTS(SELECT 1 FROM `postpartum_profiles` WHERE user_id = sqlc.arg(user_id)) AS active;

-- name: UpsertFeedsCount :exec
-- The owner's taxonomy v2 slot baby.feeds_count of a day, owned by the feeding sessions (source baby_log).
INSERT INTO `health_log_entries` (user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(log_date), 'baby', 'feeds_count', '', NULL, sqlc.arg(value_num), NULL, 'baby_log', sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  value_code = NULL, value_num = VALUES(value_num), value_text = NULL, source = 'baby_log', updated_at = VALUES(updated_at);

-- name: DeleteFeedsCount :exec
-- Only the slot the sessions wrote (a manual value from the log sheet stays).
DELETE FROM `health_log_entries`
WHERE user_id = sqlc.arg(user_id) AND log_date = sqlc.arg(log_date) AND category = 'baby' AND param = 'feeds_count'
  AND item = '' AND source = 'baby_log';
