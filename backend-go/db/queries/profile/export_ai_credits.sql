-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportAiCreditGrants :many
SELECT `id`, `user_id`, `kind`, `subscription_id`, `trial_id`, `invoice_id`, `amount`, `remaining`, `note`, `created_at`, `updated_at`
FROM `ai_credit_grants`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportAiCreditSpends :many
SELECT `id`, `user_id`, `feature`, `credits`, `plan_credits`, `extra_credits`, `allocations`, `status`, `created_at`, `updated_at`
FROM `ai_credit_spends`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
