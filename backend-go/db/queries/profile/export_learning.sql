-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportLearningInstructors :many
SELECT `id`, `user_id`, `display_name`, `title`, `bio`, `status`, `approved_at`, `approved_by`, `revoked_at`, `created_at`, `updated_at`
FROM `learning_instructors`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportLearningGrants :many
SELECT `id`, `instructor_id`, `phone`, `user_id`, `group_id`, `course_id`, `duration`, `duration_days`, `until_date`, `status`, `activated_at`, `expires_at`, `revoked_at`, `seen_at`, `created_at`, `updated_at`
FROM `learning_grants`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportLearningProgress :many
SELECT `id`, `user_id`, `lesson_id`, `course_id`, `position_seconds`, `percent`, `completed_at`, `last_seen_at`, `created_at`, `updated_at`
FROM `learning_progress`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
