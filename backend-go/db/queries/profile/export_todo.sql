-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportTodoTasks :many
SELECT `id`, `user_id`, `title`, `note`, `category`, `due_date`, `due_time`, `remind`, `done_at`, `suggestion_key`, `created_at`, `updated_at`, `created_by`, `updated_by`, `scope`, `assigned_to`
FROM `todo_tasks`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportTodoItems :many
SELECT `id`, `task_id`, `user_id`, `title`, `due_date`, `done_at`, `sort_order`, `created_at`, `updated_at`, `created_by`, `updated_by`
FROM `todo_items`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportTodoSuggestionEvents :many
SELECT `id`, `user_id`, `suggestion_key`, `ref_date`, `action`, `created_at`, `updated_at`
FROM `todo_suggestion_events`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
