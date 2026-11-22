-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportAssistantConversations :many
SELECT `id`, `user_id`, `department_id`, `title`, `triage_level`, `emergency`, `message_count`, `summary`, `summary_at`, `summary_message_count`, `last_message_at`, `created_at`, `updated_at`
FROM `assistant_conversations`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportAssistantMessages :many
SELECT `id`, `conversation_id`, `role`, `body`, `photo_file_id`, `card`, `triage_level`, `guardrail`, `chips`, `status`, `created_at`, `updated_at`
FROM `assistant_messages`
WHERE conversation_id IN (SELECT id FROM `assistant_conversations` WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY conversation_id, id;

-- name: AcctExportAssistantHandoffs :many
SELECT `id`, `conversation_id`, `user_id`, `booking_id`, `summary`, `triage_level`, `created_at`, `updated_at`
FROM `assistant_handoffs`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportAssistantProfileSettings :many
SELECT `id`, `user_id`, `shared`, `created_at`, `updated_at`
FROM `assistant_profile_settings`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
