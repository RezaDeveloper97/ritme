-- Account data export (GET /profile/export, user decision 2026-10-12): every shop row of the owner; order child rows
-- are reached through her order (the owner scope is in each statement). Buyer PII is opened by the caller.

-- name: ExportShopAddresses :many
SELECT * FROM `shop_addresses` WHERE user_id = ? ORDER BY id;

-- name: ExportShopOrders :many
SELECT * FROM `shop_orders` WHERE user_id = ? ORDER BY created_at, id;

-- name: ExportShopOrderShipments :many
SELECT s.* FROM `shop_order_shipments` s
JOIN `shop_orders` o ON o.id = s.order_id
WHERE o.user_id = ? ORDER BY s.order_id, s.id;

-- name: ExportShopOrderItems :many
SELECT i.* FROM `shop_order_items` i
JOIN `shop_orders` o ON o.id = i.order_id
WHERE o.user_id = ? ORDER BY i.order_id, i.id;

-- name: ExportShopOrderEvents :many
SELECT e.* FROM `shop_order_events` e
JOIN `shop_orders` o ON o.id = e.order_id
WHERE o.user_id = ? ORDER BY e.order_id, e.id;

-- name: ExportShopReturns :many
SELECT * FROM `shop_returns` WHERE user_id = ? ORDER BY id;

-- name: ExportShopProductReviews :many
SELECT * FROM `shop_product_reviews` WHERE user_id = ? ORDER BY id;

-- name: ExportShopChecklistItems :many
SELECT * FROM `shop_user_checklist_items` WHERE user_id = ? ORDER BY group_key, sort_order, id;

-- name: ExportShopSupplyReminders :many
SELECT * FROM `shop_supply_reminders` WHERE user_id = ? ORDER BY user_id;
