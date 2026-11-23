-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportHealthLogCustomItems :many
SELECT `id`, `user_id`, `category`, `param`, `label`, `created_at`, `updated_at`, `deleted_at`
FROM `health_log_custom_items`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportHealthLogPreferences :many
SELECT `id`, `user_id`, `mode`, `category_order`, `hidden`, `pinned`, `created_at`, `updated_at`
FROM `health_log_preferences`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
