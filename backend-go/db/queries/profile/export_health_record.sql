-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportHealthRecords :many
SELECT `id`, `user_id`, `blood_type`, `allergies`, `allergies_on_emergency_card`, `surgeries`, `family_history`, `created_at`, `updated_at`
FROM `health_records`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportHealthRecordPregnancies :many
SELECT `id`, `user_id`, `outcome`, `ended_on`, `baby_count`, `created_at`, `updated_at`
FROM `health_record_pregnancies`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportRecordDocuments :many
SELECT `id`, `user_id`, `kind`, `title`, `document_date`, `ended_on`, `centre`, `doctor`, `note`, `extracted`, `review_state`, `created_at`, `updated_at`
FROM `record_documents`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportRecordDocumentFiles :many
SELECT `id`, `user_id`, `document_id`, `file_id`, `position`, `created_at`, `updated_at`
FROM `record_document_files`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportRecordDocumentLinks :many
SELECT `id`, `user_id`, `document_id`, `target_type`, `target_id`, `state`, `created_at`, `updated_at`
FROM `record_document_links`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportEmergencyCards :many
SELECT `id`, `user_id`, `show_on_lock_screen`, `show_pregnancy`, `contact_name`, `contact_relation`, `contact_phone`, `insurance_label`, `insurance_last4`, `public_enabled_at`, `public_view_count`, `public_last_viewed_at`, `created_at`, `updated_at`
FROM `emergency_cards`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportHealthShareLinks :many
SELECT `id`, `user_id`, `kind`, `label`, `payload`, `sections`, `include_losses`, `range_from`, `range_to`, `expires_at`, `revoked_at`, `view_count`, `last_viewed_at`, `created_at`, `updated_at`
FROM `health_share_links`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportHealthShareLinkViews :many
SELECT `id`, `share_link_id`, `user_id`, `via`, `device`, `browser`, `viewed_at`
FROM `health_share_link_views`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
