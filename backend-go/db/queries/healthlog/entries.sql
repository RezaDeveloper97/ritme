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

-- name: InsertLogEntriesBatch :exec
-- T-M12-31 (P-06): every row of one day save in one multi-row INSERT. batch is a JSON array of rows
-- [category, param, item, value_code|null, value_num|null, value_text|null, source], kept in array order (ids follow
-- it, as the per-row inserts did). sqlc's parser has neither JSON_TABLE nor recursive CTEs, so the array is walked by
-- an index 0…999 built from three digit tables (as InsertMessageEventsBatch); the caller sends at most 1000 rows.
INSERT INTO `health_log_entries` (
    user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at
)
SELECT sqlc.arg(user_id), sqlc.arg(log_date),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][0]')),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][1]')),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][2]')),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][3]')),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][4]')),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][5]')),
  JSON_VALUE(sqlc.arg(batch), CONCAT('$[', n.i, '][6]')),
  sqlc.arg(now), sqlc.arg(now)
FROM (
  SELECT d0.d + 10 * d1.d + 100 * d2.d AS i
  FROM (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
    UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) AS d0
  CROSS JOIN (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
    UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) AS d1
  CROSS JOIN (SELECT 0 AS d UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
    UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) AS d2
) AS n
WHERE n.i < JSON_LENGTH(sqlc.arg(batch))
ORDER BY n.i;

-- name: DeleteLogEntriesOn :exec
DELETE FROM `health_log_entries` WHERE user_id = ? AND log_date = ?;

-- name: DeleteLogEntryParam :exec
-- One param of a day (every item of a multi / items / text_items param).
DELETE FROM `health_log_entries` WHERE user_id = ? AND log_date = ? AND category = ? AND param = ?;

-- name: DeleteLogEntryParams :exec
-- T-M12-31 (P-06): every slot of the given params of a day in one statement — the (category, param) IN (…) of a
-- save. sqlc drops a sqlc.slice whose left side is a tuple or an expression (no type to infer), so the pairs go as
-- one comma-separated "category.param" list (taxonomy codes are [a-z0-9_], never a comma; the caller checks).
-- The (user_id, log_date) prefix of the slot index narrows the scan to the day's rows first.
DELETE FROM `health_log_entries`
WHERE user_id = sqlc.arg(user_id) AND log_date = sqlc.arg(log_date)
  AND FIND_IN_SET(CONCAT(category, '.', param), sqlc.arg(param_keys)) > 0;

-- name: DeleteLogEntrySlot :exec
DELETE FROM `health_log_entries`
WHERE user_id = ? AND log_date = ? AND category = ? AND param = ? AND item = ?;

-- name: GetLogLifeMode :one
-- enums.ResolveLifeMode input: the stored life mode (no row / NULL = legacy detection).
SELECT life_mode FROM `user_life_profiles` WHERE user_id = ? LIMIT 1;

-- name: LogPregnancyActive :one
-- enums.ResolveLifeMode input: an active pregnancy profile wins over the stored mode.
SELECT EXISTS(SELECT 1 FROM `pregnancy_profiles` WHERE user_id = ? AND pregnancy_mode = 1) AS active;
