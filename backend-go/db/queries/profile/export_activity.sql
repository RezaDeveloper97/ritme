-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportAnalyticsEvents :many
SELECT `id`, `user_id`, `name`, `props`, `occurred_on`, `occurred_at`, `created_at`
FROM `analytics_events`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportUserActivityDays :many
SELECT `user_id`, `day`, `session`, `logged`
FROM `user_activity_days`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY day;
