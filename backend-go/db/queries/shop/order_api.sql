-- Cart, addresses, COD checkout, orders and returns API (CB-SHOP-03, internal/shop/order). Every user row is scoped by
-- user_id in the query itself; stock, delivery-slot capacity and discount-code uses change only through guarded
-- UPDATEs (…WHERE stock >= ?, booked < capacity, used_count < max_uses) inside the checkout transaction. No query here
-- joins a health table.

-- name: EnsureCart :exec
INSERT IGNORE INTO `shop_carts` (`user_id`, `created_at`, `updated_at`) VALUES (?, ?, ?);

-- name: LockCartByUser :one
SELECT * FROM `shop_carts` WHERE user_id = ? LIMIT 1 FOR UPDATE;

-- name: SetCartDiscountCode :exec
UPDATE `shop_carts` SET discount_code_id = ?, updated_at = ? WHERE id = ?;

-- name: GetCartItemQuantity :one
SELECT quantity FROM `shop_cart_items` WHERE cart_id = ? AND variant_id = ? LIMIT 1;

-- name: ListCartLines :many
SELECT ci.variant_id, ci.quantity,
       v.product_id, v.size, v.color, v.volume, v.price, v.compare_at_price, v.stock, v.is_active AS variant_active,
       p.title, p.image_path, p.images AS product_images, p.category_id, p.seller_id, p.discreet_packaging, p.is_active AS product_active,
       c.is_active AS category_active, c.parent_id AS category_parent_id
FROM `shop_cart_items` ci
JOIN `shop_product_variants` v ON v.id = ci.variant_id
JOIN `shop_products` p ON p.id = v.product_id
JOIN `shop_categories` c ON c.id = p.category_id
WHERE ci.cart_id = ?
ORDER BY ci.id;

-- name: GetVariantForCart :one
SELECT v.id, v.product_id, v.stock, v.is_active AS variant_active, p.seller_id, p.is_active AS product_active,
       c.is_active AS category_active
FROM `shop_product_variants` v
JOIN `shop_products` p ON p.id = v.product_id
JOIN `shop_categories` c ON c.id = p.category_id
WHERE v.id = ? LIMIT 1;

-- name: GetDiscountCodeByCode :one
SELECT * FROM `shop_discount_codes` WHERE code = ? LIMIT 1;

-- name: GetDiscountCode :one
SELECT * FROM `shop_discount_codes` WHERE id = ? LIMIT 1;

-- name: CountUserOrdersWithDiscount :one
SELECT COUNT(*) FROM `shop_orders` WHERE user_id = ? AND discount_code_id = ? AND status <> 'cancelled';

-- name: UseDiscountCode :execrows
UPDATE `shop_discount_codes` SET used_count = used_count + 1, updated_at = ?
WHERE id = ? AND is_active = 1 AND (max_uses IS NULL OR used_count < max_uses);

-- name: ReleaseDiscountCode :exec
UPDATE `shop_discount_codes` SET used_count = used_count - 1, updated_at = ? WHERE id = ? AND used_count > 0;

-- name: CountAddressesByUser :one
SELECT COUNT(*) FROM `shop_addresses` WHERE user_id = ?;

-- name: CreateAddress :execlastid
INSERT INTO `shop_addresses` (`user_id`, `label`, `recipient_name`, `recipient_phone`, `province`, `city`,
  `address_line`, `postal_code`, `is_default`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateAddressForUser :execrows
UPDATE `shop_addresses` SET label = ?, recipient_name = ?, recipient_phone = ?, province = ?, city = ?,
  address_line = ?, postal_code = ?, is_default = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: ClearDefaultAddress :exec
UPDATE `shop_addresses` SET is_default = 0 WHERE user_id = ? AND id <> ? AND is_default = 1;

-- name: DeleteAddressForUser :execrows
DELETE FROM `shop_addresses` WHERE id = ? AND user_id = ?;

-- name: GetDeliverySlot :one
SELECT * FROM `shop_delivery_slots` WHERE id = ? LIMIT 1;

-- name: ReleaseDeliverySlot :exec
UPDATE `shop_delivery_slots` SET booked = booked - 1, updated_at = ? WHERE id = ? AND booked > 0;

-- name: ReserveVariantStock :execrows
UPDATE `shop_product_variants` SET stock = stock - sqlc.arg(qty), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND is_active = 1 AND stock >= sqlc.arg(qty);

-- name: RestockVariant :exec
UPDATE `shop_product_variants` SET stock = stock + ?, updated_at = ? WHERE id = ?;

-- name: AddProductPurchases :exec
UPDATE `shop_products` SET purchase_count = purchase_count + ? WHERE id = ?;

-- name: SubtractProductPurchases :exec
UPDATE `shop_products` SET purchase_count = purchase_count - LEAST(purchase_count, sqlc.arg(qty)) WHERE id = sqlc.arg(id);

-- name: GetOrderByIdempotencyKey :one
SELECT * FROM `shop_orders` WHERE user_id = ? AND idempotency_key = ? LIMIT 1;

-- name: CreateOrder :execlastid
INSERT INTO `shop_orders` (`code`, `user_id`, `address_id`, `recipient_name`, `recipient_phone`, `city`, `address_text`,
  `postal_code`, `payment_method`, `payment_status`, `status`, `discreet`, `items_total`, `shipping_total`,
  `discount_total`, `payable_total`, `placed_at`, `discount_code_id`, `idempotency_key`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'cod', 'unpaid', 'placed', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateShipment :execlastid
INSERT INTO `shop_order_shipments` (`order_id`, `seller_id`, `delivery_slot_id`, `slot_date`, `slot_starts_at`,
  `slot_ends_at`, `status`, `items_total`, `shipping_fee`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, 'awaiting_seller', ?, ?, ?, ?);

-- name: CreateOrderItem :execlastid
INSERT INTO `shop_order_items` (`order_id`, `shipment_id`, `product_id`, `variant_id`, `title`, `variant_label`,
  `unit_price`, `quantity`, `line_total`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: LockOrderForUser :one
SELECT * FROM `shop_orders` WHERE id = ? AND user_id = ? LIMIT 1 FOR UPDATE;

-- name: CancelOrderForUser :execrows
UPDATE `shop_orders` SET status = 'cancelled', cancelled_at = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND status = 'placed';

-- name: CancelOrderShipments :exec
UPDATE `shop_order_shipments` SET status = 'cancelled', updated_at = ? WHERE order_id = ?;

-- name: ListShipmentsByUser :many
SELECT s.* FROM `shop_order_shipments` s JOIN `shop_orders` o ON o.id = s.order_id
WHERE o.user_id = ? ORDER BY s.order_id, s.id;

-- name: ListOrderItemsByUser :many
SELECT i.* FROM `shop_order_items` i JOIN `shop_orders` o ON o.id = i.order_id
WHERE o.user_id = ? ORDER BY i.order_id, i.shipment_id, i.id;

-- name: ListProductImages :many
SELECT id, image_path, images FROM `shop_products` WHERE id IN (sqlc.slice(ids));

-- name: LockOrder :one
SELECT * FROM `shop_orders` WHERE id = ? LIMIT 1 FOR UPDATE;

-- name: GetShipment :one
SELECT * FROM `shop_order_shipments` WHERE id = ? LIMIT 1;

-- name: MarkShipmentDelivered :execrows
UPDATE `shop_order_shipments` SET status = 'delivered', delivered_at = ?, updated_at = ?
WHERE id = ? AND status NOT IN ('delivered', 'cancelled');

-- name: CountUndeliveredShipments :one
SELECT COUNT(*) FROM `shop_order_shipments` WHERE order_id = ? AND status NOT IN ('delivered', 'cancelled');

-- name: SetOrderStatus :exec
UPDATE `shop_orders` SET status = ?, updated_at = ? WHERE id = ?;

-- name: GetOrderItemForReturn :one
SELECT i.id, i.order_id, i.quantity, s.status AS shipment_status, s.delivered_at, s.seller_id, o.status AS order_status
FROM `shop_order_items` i
JOIN `shop_orders` o ON o.id = i.order_id
JOIN `shop_order_shipments` s ON s.id = i.shipment_id
WHERE i.id = sqlc.arg(item_id) AND i.order_id = sqlc.arg(order_id) AND o.user_id = sqlc.arg(user_id)
LIMIT 1 FOR UPDATE;

-- name: SumOpenReturnQuantity :one
SELECT CAST(COALESCE(SUM(quantity), 0) AS UNSIGNED) FROM `shop_returns` WHERE order_item_id = ? AND status <> 'rejected';

-- name: CreateReturn :execlastid
INSERT INTO `shop_returns` (`order_id`, `order_item_id`, `user_id`, `quantity`, `reason`, `note`, `status`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, 'requested', ?, ?);

-- name: GetReturnForUser :one
SELECT * FROM `shop_returns` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: ListReturnsByOrderForUser :many
SELECT * FROM `shop_returns` WHERE order_id = ? AND user_id = ? ORDER BY created_at, id;

-- name: ListOrderCategories :many
SELECT DISTINCT p.category_id, c.parent_id
FROM `shop_order_items` i
JOIN `shop_products` p ON p.id = i.product_id
JOIN `shop_categories` c ON c.id = p.category_id
WHERE i.order_id = ?;

-- name: ListShipmentCategories :many
SELECT DISTINCT p.category_id, c.parent_id
FROM `shop_order_items` i
JOIN `shop_products` p ON p.id = i.product_id
JOIN `shop_categories` c ON c.id = p.category_id
WHERE i.shipment_id = ?;

-- Layette checklist link (board Shop_Order: «۲ مورد در لیست سیسمونی «در راه» شد / بعد از تحویل خودکار «دارم» می‌شود»).
-- Only the user's EXISTING rows whose template item points at one of the ordered categories move; CB-SHOP-04 owns
-- the checklist itself.

-- name: MarkChecklistOnTheWay :execrows
UPDATE `shop_user_checklist_items` u JOIN `shop_layette_items` l ON l.id = u.layette_item_id
SET u.state = 'on_the_way', u.updated_at = sqlc.arg(updated_at)
WHERE u.user_id = sqlc.arg(user_id) AND u.state IN ('todo', 'in_cart', 'later')
  AND l.category_id IN (sqlc.slice(category_ids));

-- name: MarkChecklistHave :execrows
UPDATE `shop_user_checklist_items` u JOIN `shop_layette_items` l ON l.id = u.layette_item_id
SET u.state = 'have', u.updated_at = sqlc.arg(updated_at)
WHERE u.user_id = sqlc.arg(user_id) AND u.state = 'on_the_way' AND l.category_id IN (sqlc.slice(category_ids));

-- name: RevertChecklistOnTheWay :execrows
UPDATE `shop_user_checklist_items` u JOIN `shop_layette_items` l ON l.id = u.layette_item_id
SET u.state = 'todo', u.updated_at = sqlc.arg(updated_at)
WHERE u.user_id = sqlc.arg(user_id) AND u.state = 'on_the_way' AND l.category_id IN (sqlc.slice(category_ids));

-- name: CountChecklistOnTheWay :one
SELECT COUNT(*) FROM `shop_user_checklist_items` u JOIN `shop_layette_items` l ON l.id = u.layette_item_id
WHERE u.user_id = sqlc.arg(user_id) AND u.state = 'on_the_way' AND l.category_id IN (sqlc.slice(category_ids));

-- Security audit (directory/shop): the open-order cap — her placed orders none of whose shipments has left the seller.
-- name: CountOpenOrdersByUser :one
SELECT COUNT(*) FROM `shop_orders` o
WHERE o.user_id = ? AND o.status = 'placed'
  AND NOT EXISTS (SELECT 1 FROM `shop_order_shipments` s WHERE s.order_id = o.id AND s.status IN ('shipped', 'delivered'));

-- Security audit (directory/shop): categories her OTHER placed orders still carry — their checklist rows stay «در راه»
-- when one order is cancelled.
-- name: ListOtherOpenOrderCategories :many
SELECT DISTINCT p.category_id, c.parent_id
FROM `shop_order_items` i
JOIN `shop_orders` o ON o.id = i.order_id
JOIN `shop_products` p ON p.id = i.product_id
JOIN `shop_categories` c ON c.id = p.category_id
WHERE o.user_id = ? AND o.status = 'placed' AND o.id <> ?;

-- CB-SHOP-12 F-6: a purchase also adds the layette rows she never touched for the ordered categories, straight to
-- «در راه» (MarkChecklistOnTheWay moves the rows she has). A cancel reverts them to «باید بخرم» like any other row,
-- which is what an untouched template row shows anyway. IGNORE only covers a concurrent insert of the same row.
-- name: InsertChecklistOnTheWay :execrows
INSERT IGNORE INTO `shop_user_checklist_items` (`user_id`, `layette_item_id`, `group_key`, `state`, `created_at`, `updated_at`)
SELECT sqlc.arg(user_id), l.id, l.group_key, 'on_the_way', sqlc.arg(now), sqlc.arg(now)
FROM `shop_layette_items` l
WHERE l.is_active = 1 AND l.category_id IN (sqlc.slice(category_ids))
  AND NOT EXISTS (SELECT 1 FROM `shop_user_checklist_items` u WHERE u.user_id = sqlc.arg(user_id) AND u.layette_item_id = l.id);

-- CB-SHOP-12 F-5: the user's own product review — verified purchase, one per order item (unique order_item_id),
-- only after the item's shipment is delivered, `pending` until an admin moderates it.
-- name: GetOrderItemForReview :one
SELECT i.id, i.product_id, i.variant_label, s.status AS shipment_status
FROM `shop_order_items` i
JOIN `shop_orders` o ON o.id = i.order_id
JOIN `shop_order_shipments` s ON s.id = i.shipment_id
WHERE i.id = sqlc.arg(item_id) AND i.order_id = sqlc.arg(order_id) AND o.user_id = sqlc.arg(user_id)
LIMIT 1 FOR UPDATE;

-- name: GetUserDisplayName :one
SELECT name FROM `users` WHERE id = ? LIMIT 1;

-- name: CountReviewsByOrderItem :one
SELECT COUNT(*) FROM `shop_product_reviews` WHERE order_item_id = ?;

-- name: CreateUserProductReview :execlastid
INSERT INTO `shop_product_reviews` (`product_id`, `user_id`, `order_item_id`, `author_name`, `variant_label`, `rating`,
  `body`, `status`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?);

-- name: GetProductReviewForUser :one
SELECT * FROM `shop_product_reviews` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: ListReviewsByOrderForUser :many
SELECT r.order_item_id, r.status FROM `shop_product_reviews` r
JOIN `shop_order_items` i ON i.id = r.order_item_id
WHERE i.order_id = sqlc.arg(order_id) AND r.user_id = sqlc.arg(user_id);
