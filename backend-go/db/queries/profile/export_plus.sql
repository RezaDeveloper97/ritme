-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportPlusSubscriptions :many
SELECT `id`, `user_id`, `plan_id`, `invoice_id`, `status`, `source`, `starts_at`, `ends_at`, `auto_renew`, `canceled_at`, `created_at`, `updated_at`
FROM `plus_subscriptions`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPlusTrials :many
SELECT `id`, `user_id`, `started_at`, `ends_at`, `created_at`, `updated_at`
FROM `plus_trials`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPlusInvoices :many
SELECT `id`, `reference`, `user_id`, `plan_id`, `kind`, `credit_pack_id`, `credits`, `duration_months`, `duration_minutes`, `status`, `currency`, `subtotal_rials`, `discount_rials`, `vat_rate_bps`, `vat_rials`, `total_rials`, `discount_code_id`, `discount_code`, `gateway`, `expires_at`, `paid_at`, `created_at`, `updated_at`
FROM `plus_invoices`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPlusReceipts :many
SELECT `id`, `invoice_id`, `user_id`, `gateway`, `ref_id`, `card_pan`, `amount_rials`, `paid_at`, `created_at`, `updated_at`
FROM `plus_receipts`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPlusStorePurchases :many
SELECT `id`, `user_id`, `store`, `sku`, `order_id`, `purchase_time`, `created_at`, `updated_at`
FROM `plus_store_purchases`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportPlusUsageCounters :many
SELECT `id`, `user_id`, `feature`, `period_start`, `used`, `created_at`, `updated_at`
FROM `plus_usage_counters`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
