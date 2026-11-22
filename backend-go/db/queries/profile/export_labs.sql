-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportLabReports :many
SELECT `id`, `user_id`, `source`, `category`, `title`, `taken_on`, `fasting`, `lab_name`, `status`, `progress`, `error_code`, `quota_at`, `verified_at`, `interpretation`, `interpreted_at`, `interpret_count`, `feedback`, `feedback_note`, `feedback_at`, `created_at`, `updated_at`
FROM `lab_reports`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportLabMarkers :many
SELECT `id`, `lab_id`, `user_id`, `code`, `name`, `value`, `value_text`, `unit`, `ref_low`, `ref_high`, `ref_text`, `confidence`, `source`, `sort_order`, `created_at`, `updated_at`, `confirmed_at`
FROM `lab_markers`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY lab_id, sort_order, id;

-- name: AcctExportLabFiles :many
SELECT `id`, `lab_id`, `user_id`, `page`, `mime`, `size_bytes`, `created_at`, `updated_at`
FROM `lab_files`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPartnerLabRequests :many
SELECT `id`, `reference`, `user_id`, `service_id`, `status`, `partner_ref`, `partner_status`, `preferred_date`, `preferred_window`, `consent_share`, `consent_import`, `payload_enc`, `imported_lab_id`, `closed_at`, `purged_at`, `created_at`, `updated_at`
FROM `partner_lab_requests`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
