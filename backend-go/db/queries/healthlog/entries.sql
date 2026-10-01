-- Log taxonomy v2 entries (B-N3-01, health_log_entries; internal/healthlog/taxonomy). Every query is
-- scoped by user_id.

-- name: ListLogEntriesOn :many
SELECT * FROM `health_log_entries` WHERE user_id = ? AND log_date = ? ORDER BY id;

-- name: ListLogEntriesBetween :many
SELECT * FROM `health_log_entries`
WHERE user_id = sqlc.arg(user_id) AND log_date >= sqlc.arg(date_from) AND log_date <= sqlc.arg(date_to)
ORDER BY log_date, id;

-- name: InsertLogEntry :exec
INSERT INTO `health_log_entries` (
    user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteLogEntriesOn :exec
DELETE FROM `health_log_entries` WHERE user_id = ? AND log_date = ?;

-- name: DeleteLogEntryParam :exec
-- One param of a day (every item of a multi / items / text_items param).
DELETE FROM `health_log_entries` WHERE user_id = ? AND log_date = ? AND category = ? AND param = ?;

-- name: DeleteLogEntrySlot :exec
DELETE FROM `health_log_entries`
WHERE user_id = ? AND log_date = ? AND category = ? AND param = ? AND item = ?;

-- name: GetLogLifeMode :one
-- enums.ResolveLifeMode input: the stored life mode (no row / NULL = legacy detection).
SELECT life_mode FROM `user_life_profiles` WHERE user_id = ? LIMIT 1;

-- name: LogPregnancyActive :one
-- enums.ResolveLifeMode input: an active pregnancy profile wins over the stored mode.
SELECT EXISTS(SELECT 1 FROM `pregnancy_profiles` WHERE user_id = ? AND pregnancy_mode = 1) AS active;
