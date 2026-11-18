-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportConsentEvents :many
SELECT `id`, `user_id`, `consent`, `version`, `action`, `scope`, `source`, `created_at`
FROM `consent_events`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportFiles :many
SELECT `id`, `user_id`, `purpose`, `visibility`, `mime`, `size_bytes`, `created_at`, `updated_at`
FROM `files`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
