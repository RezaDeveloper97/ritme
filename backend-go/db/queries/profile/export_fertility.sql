-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportFertilityLogs :many
SELECT `id`, `user_id`, `log_date`, `lh_test`, `cervical_mucus`, `bbt_time`, `created_at`, `updated_at`
FROM `fertility_logs`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY log_date, id;
