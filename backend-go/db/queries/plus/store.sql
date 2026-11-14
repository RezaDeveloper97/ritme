-- Store in-app billing (Café Bazaar / Myket): the SKU map and the claimed purchase tokens (00124).

-- name: GetStoreProduct :one
SELECT * FROM plus_store_products
WHERE store = ? AND sku = ?;

-- name: ListStoreProducts :many
SELECT * FROM plus_store_products
ORDER BY store, id;

-- name: LockStorePurchase :one
-- Serialises concurrent verifies of one token.
SELECT * FROM plus_store_purchases
WHERE store = ? AND purchase_token = ?
FOR UPDATE;

-- name: InsertStorePurchase :exec
-- UNIQUE(store, purchase_token): a token belongs to the first account that claims it.
INSERT INTO plus_store_purchases (user_id, store, sku, purchase_token, order_id, purchase_time, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: TouchStorePurchase :exec
UPDATE plus_store_purchases SET purchase_time = sqlc.arg(purchase_time), order_id = sqlc.arg(order_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ReceiptExists :one
SELECT EXISTS(SELECT 1 FROM plus_receipts WHERE gateway = ? AND ref_id = ?) AS found;
