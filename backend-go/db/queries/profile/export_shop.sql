-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportShopCarts :many
SELECT `id`, `user_id`, `created_at`, `updated_at`, `discount_code_id`
FROM `shop_carts`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportShopCartItems :many
SELECT `id`, `cart_id`, `variant_id`, `quantity`, `created_at`, `updated_at`
FROM `shop_cart_items`
WHERE cart_id IN (SELECT id FROM `shop_carts` WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;
