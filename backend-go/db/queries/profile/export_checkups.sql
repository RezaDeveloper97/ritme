-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportCheckupTypes :many
SELECT `id`, `key`, `user_id`, `category`, `title`, `subtitle`, `why`, `performed_by`, `icon`, `tone`, `interval_months`, `interval_months_max`, `age_min`, `age_max`, `cycle_day_from`, `cycle_day_to`, `remind_lead_days`, `prep_steps`, `guide_steps`, `finding_options`, `hide_in_pregnancy`, `audiences`, `is_active`, `sort_order`, `source_note`, `created_at`, `updated_at`, `informational`
FROM `checkup_types`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportCheckupRecords :many
SELECT `id`, `user_id`, `checkup_type_id`, `done_on`, `result`, `findings`, `note`, `has_attachment`, `next_due_on`, `created_at`, `updated_at`, `test_kind`
FROM `checkup_records`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportUserCheckupSettings :many
SELECT `id`, `user_id`, `checkup_type_id`, `enabled`, `remind`, `created_at`, `updated_at`
FROM `user_checkup_settings`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
