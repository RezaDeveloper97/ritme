-- Cart, addresses, delivery slots and pay-on-delivery orders (CB-SHOP-01). COD only in the MVP; every order row carries
-- a recipient/address and item-title SNAPSHOT. No query here joins a health table.

-- name: GetCartByUser :one
SELECT * FROM `shop_carts` WHERE user_id = ? LIMIT 1;

-- name: CreateCart :execlastid
INSERT INTO `shop_carts` (`user_id`, `created_at`, `updated_at`) VALUES (?, ?, ?);

-- name: ListCartItems :many
SELECT * FROM `shop_cart_items` WHERE cart_id = ? ORDER BY id;

-- name: UpsertCartItem :exec
INSERT INTO `shop_cart_items` (`cart_id`, `variant_id`, `quantity`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE `quantity` = VALUES(`quantity`), `updated_at` = VALUES(`updated_at`);

-- name: DeleteCartItem :exec
DELETE FROM `shop_cart_items` WHERE cart_id = ? AND variant_id = ?;

-- name: ClearCart :exec
DELETE FROM `shop_cart_items` WHERE cart_id = ?;

-- name: ListAddressesByUser :many
SELECT * FROM `shop_addresses` WHERE user_id = ? ORDER BY is_default DESC, id DESC;

-- name: GetAddressForUser :one
SELECT * FROM `shop_addresses` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: ListOpenDeliverySlots :many
SELECT * FROM `shop_delivery_slots`
WHERE seller_id = sqlc.arg(seller_id) AND is_active = 1 AND slot_date >= sqlc.arg(from_date)
  AND (capacity IS NULL OR booked < capacity)
ORDER BY slot_date, starts_at LIMIT 20;

-- name: BookDeliverySlot :execrows
UPDATE `shop_delivery_slots` SET booked = booked + 1, updated_at = ?
WHERE id = ? AND is_active = 1 AND (capacity IS NULL OR booked < capacity);

-- name: GetOrderForUser :one
SELECT * FROM `shop_orders` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: GetOrderByCodeForUser :one
SELECT * FROM `shop_orders` WHERE code = ? AND user_id = ? LIMIT 1;

-- name: ListOrdersByUser :many
SELECT * FROM `shop_orders` WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT ?;

-- name: ListShipmentsByOrder :many
SELECT * FROM `shop_order_shipments` WHERE order_id = ? ORDER BY id;

-- name: ListOrderItemsByOrder :many
SELECT * FROM `shop_order_items` WHERE order_id = ? ORDER BY shipment_id, id;

-- name: ListOrderEvents :many
SELECT * FROM `shop_order_events` WHERE order_id = ? ORDER BY created_at, id;

-- name: CreateOrderEvent :execlastid
INSERT INTO `shop_order_events` (`order_id`, `shipment_id`, `status`, `note`, `created_at`) VALUES (?, ?, ?, ?, ?);

-- name: ListReturnsByUser :many
SELECT * FROM `shop_returns` WHERE user_id = ? ORDER BY created_at DESC, id DESC;
