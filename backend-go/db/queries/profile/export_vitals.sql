-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportVitalPlanItems :many
SELECT `id`, `user_id`, `type`, `slot`, `days`, `remind_at`, `sort_order`, `created_at`, `updated_at`
FROM `vital_plan_items`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportVitalReadings :many
SELECT `id`, `user_id`, `type`, `measured_at`, `systolic`, `diastolic`, `pulse`, `arm`, `position`, `glucose_mg_dl`, `glucose_unit`, `context`, `method`, `note`, `created_at`, `updated_at`
FROM `vital_readings`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY measured_at, id;

-- name: AcctExportVitalTargets :many
SELECT `id`, `user_id`, `context`, `low_mgdl`, `high_mgdl`, `set_by`, `note`, `created_at`, `updated_at`
FROM `vital_targets`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
