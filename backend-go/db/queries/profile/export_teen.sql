-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportTeenProfiles :many
SELECT `id`, `user_id`, `age_band`, `menarche`, `parent_note`, `created_at`, `updated_at`
FROM `teen_profiles`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportTeenKitChecks :many
SELECT `id`, `user_id`, `item_code`, `created_at`, `updated_at`
FROM `teen_kit_checks`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
