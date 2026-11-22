-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportCyclePreferences :many
SELECT `id`, `user_id`, `lengths_auto`, `created_at`, `updated_at`
FROM `cycle_preferences`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportCycleSkips :many
SELECT `id`, `user_id`, `expected_start`, `created_at`, `updated_at`
FROM `cycle_skips`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportEnginePredictionLog :many
SELECT `id`, `user_id`, `cycle_start`, `kind`, `predicted_next`, `basis_cycles`, `ovulation_source`, `confidence`, `actual_start`, `error_days`, `created_at`
FROM `engine_prediction_log`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
