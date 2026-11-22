-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportUserNotifications :many
SELECT `id`, `user_id`, `type`, `title`, `body`, `action_url`, `data`, `read_at`, `created_at`, `updated_at`, `link_id`, `pushed_at`
FROM `user_notifications`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportMessageInbox :many
SELECT `id`, `user_id`, `rule_id`, `version`, `rule_key`, `slot`, `kind`, `level`, `state`, `open_flag`, `evidence_key`, `hit_count`, `evidence`, `first_at`, `last_at`, `acked_at`, `noted_at`, `resolved_at`, `archived_at`, `legacy_alert_id`, `created_at`, `updated_at`
FROM `message_inbox`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportMessageEvents :many
SELECT `id`, `rule_id`, `user_id`, `variant`, `event`, `created_at`, `version`, `slot`
FROM `message_events`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
