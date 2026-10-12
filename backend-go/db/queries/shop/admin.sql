-- Shop admin (CB-SHOP-05): sellers, sub-category parents, commerce fields, variants and stock, delivery slots, the
-- shipment queue, returns, review moderation, discount codes and the layette template. No query here joins a health
-- table. Order rows carry recipient PII: the handlers serve them to the `support` capability only and never log them.

-- ─── sellers ───────────────────────────────────────────────────────────────────────────────────────────────────────

-- name: AdminListSellers :many
SELECT s.*, CAST((SELECT COUNT(*) FROM `shop_products` p WHERE p.seller_id = s.id) AS SIGNED) AS products_count
FROM `shop_sellers` s ORDER BY s.sort_order, s.id;

-- name: AdminSellerSlugTaken :one
SELECT COUNT(*) FROM `shop_sellers` WHERE slug = sqlc.arg(slug) AND id <> sqlc.arg(except_id);

-- name: AdminCreateSeller :execlastid
INSERT INTO `shop_sellers` (`slug`, `name`, `is_verified`, `ship_min_days`, `ship_max_days`, `discreet_packaging`,
  `shipping_fee`, `free_shipping_over`, `cod_max_total`, `return_days`, `sort_order`, `is_active`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AdminUpdateSeller :exec
UPDATE `shop_sellers` SET slug = ?, name = ?, is_verified = ?, ship_min_days = ?, ship_max_days = ?,
  discreet_packaging = ?, shipping_fee = ?, free_shipping_over = ?, cod_max_total = ?, return_days = ?, sort_order = ?,
  is_active = ?, updated_at = ?
WHERE id = ?;

-- name: AdminDeleteSeller :execrows
DELETE FROM `shop_sellers` WHERE id = ?;

-- name: AdminCountSellerShipments :one
SELECT COUNT(*) FROM `shop_order_shipments` WHERE seller_id = ?;

-- ─── categories (sub-category parent) ──────────────────────────────────────────────────────────────────────────────

-- name: AdminListCategoryTree :many
SELECT id, slug, title, parent_id, is_active FROM `shop_categories` ORDER BY sort_order, id;

-- name: AdminGetCategory :one
SELECT id, slug, title, parent_id, is_active FROM `shop_categories` WHERE id = ? LIMIT 1;

-- name: AdminCountChildCategories :one
SELECT COUNT(*) FROM `shop_categories` WHERE parent_id = ?;

-- name: AdminSetCategoryParent :exec
UPDATE `shop_categories` SET parent_id = ?, updated_at = ? WHERE id = ?;

-- ─── products (commerce fields) ────────────────────────────────────────────────────────────────────────────────────

-- name: AdminGetProduct :one
SELECT * FROM `shop_products` WHERE id = ? LIMIT 1;

-- name: AdminUpdateProductCommerce :exec
UPDATE `shop_products` SET seller_id = ?, subtitle = ?, description = ?, images = ?, attributes = ?, irc_code = ?,
  expires_on = ?, skin_types = ?, badges = ?, discreet_packaging = ?, updated_at = ?
WHERE id = ?;

-- name: AdminSetProductImages :exec
UPDATE `shop_products` SET images = ?, updated_at = ? WHERE id = ?;

-- ─── variants and stock ────────────────────────────────────────────────────────────────────────────────────────────

-- name: AdminListVariantsByProduct :many
SELECT * FROM `shop_product_variants` WHERE product_id = ? ORDER BY sort_order, id;

-- name: AdminSkuTaken :one
SELECT COUNT(*) FROM `shop_product_variants` WHERE sku = sqlc.arg(sku) AND id <> sqlc.arg(except_id);

-- name: AdminCreateVariant :execlastid
INSERT INTO `shop_product_variants` (`product_id`, `sku`, `size`, `color`, `volume`, `price`, `compare_at_price`,
  `stock`, `sort_order`, `is_active`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AdminUpdateVariant :exec
UPDATE `shop_product_variants` SET sku = ?, size = ?, color = ?, volume = ?, price = ?, compare_at_price = ?,
  stock = ?, sort_order = ?, is_active = ?, updated_at = ?
WHERE id = ?;

-- name: AdminDeleteVariant :execrows
DELETE FROM `shop_product_variants` WHERE id = ?;

-- name: AdminSetVariantStock :exec
UPDATE `shop_product_variants` SET stock = ?, updated_at = ? WHERE id = ?;

-- name: AdminListStock :many
SELECT v.id, v.product_id, v.sku, v.size, v.color, v.volume, v.price, v.stock, v.is_active, p.title AS product_title,
  p.seller_id
FROM `shop_product_variants` v JOIN `shop_products` p ON p.id = v.product_id
WHERE v.stock <= sqlc.arg(max_stock) AND (v.sku LIKE sqlc.arg(sku_pattern) OR p.title LIKE sqlc.arg(title_pattern))
ORDER BY v.stock, v.id
LIMIT ? OFFSET ?;

-- name: AdminCountStock :one
SELECT COUNT(*) FROM `shop_product_variants` v JOIN `shop_products` p ON p.id = v.product_id
WHERE v.stock <= sqlc.arg(max_stock) AND (v.sku LIKE sqlc.arg(sku_pattern) OR p.title LIKE sqlc.arg(title_pattern));

-- ─── delivery slots ────────────────────────────────────────────────────────────────────────────────────────────────

-- name: AdminListSlots :many
SELECT * FROM `shop_delivery_slots`
WHERE seller_id = sqlc.arg(seller_id) AND slot_date BETWEEN sqlc.arg(date_from) AND sqlc.arg(date_to)
ORDER BY slot_date, starts_at, id;

-- name: AdminInsertSlot :execrows
INSERT IGNORE INTO `shop_delivery_slots` (`seller_id`, `slot_date`, `starts_at`, `ends_at`, `capacity`, `booked`,
  `is_active`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, 0, 1, ?, ?);

-- name: AdminGetSlot :one
SELECT * FROM `shop_delivery_slots` WHERE id = ? LIMIT 1;

-- name: AdminUpdateSlot :exec
UPDATE `shop_delivery_slots` SET capacity = ?, is_active = ?, updated_at = ? WHERE id = ?;

-- name: AdminDeleteSlot :execrows
DELETE FROM `shop_delivery_slots` WHERE id = ? AND booked = 0;

-- ─── shipment queue and orders (PII: support capability) ───────────────────────────────────────────────────────────

-- name: AdminListShipments :many
SELECT s.id, s.order_id, s.seller_id, s.slot_date, s.slot_starts_at, s.slot_ends_at, s.status, s.items_total,
  s.shipping_fee, s.tracking_code, s.delivered_at, s.created_at,
  o.code AS order_code, o.user_id AS order_user_id, o.recipient_name, o.pii_enc, o.city, o.payable_total, o.discreet, o.status AS order_status,
  se.name AS seller_name
FROM `shop_order_shipments` s
JOIN `shop_orders` o ON o.id = s.order_id
JOIN `shop_sellers` se ON se.id = s.seller_id
WHERE (sqlc.arg(status) = '' OR s.status = sqlc.arg(status))
  AND (sqlc.arg(seller_id) = 0 OR s.seller_id = sqlc.arg(seller_id))
  AND (sqlc.arg(code) = '' OR o.code = sqlc.arg(code))
ORDER BY s.created_at DESC, s.id DESC
LIMIT ? OFFSET ?;

-- name: AdminCountShipments :one
SELECT COUNT(*) FROM `shop_order_shipments` s JOIN `shop_orders` o ON o.id = s.order_id
WHERE (sqlc.arg(status) = '' OR s.status = sqlc.arg(status))
  AND (sqlc.arg(seller_id) = 0 OR s.seller_id = sqlc.arg(seller_id))
  AND (sqlc.arg(code) = '' OR o.code = sqlc.arg(code));

-- name: AdminGetOrder :one
SELECT * FROM `shop_orders` WHERE id = ? LIMIT 1;

-- name: AdminListReturnsByOrder :many
SELECT * FROM `shop_returns` WHERE order_id = ? ORDER BY created_at, id;

-- name: AdminMoveShipment :execrows
UPDATE `shop_order_shipments` SET status = sqlc.arg(to_status),
  tracking_code = COALESCE(sqlc.narg(tracking_code), tracking_code), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND status = sqlc.arg(from_status);

-- ─── returns (PII: support capability) ─────────────────────────────────────────────────────────────────────────────

-- name: AdminListReturns :many
SELECT r.id, r.order_id, r.order_item_id, r.quantity, r.reason, r.note, r.status, r.created_at, r.updated_at,
  o.code AS order_code, i.title AS item_title, i.variant_label, i.unit_price, i.variant_id
FROM `shop_returns` r
JOIN `shop_orders` o ON o.id = r.order_id
JOIN `shop_order_items` i ON i.id = r.order_item_id
WHERE (sqlc.arg(status) = '' OR r.status = sqlc.arg(status))
ORDER BY r.created_at DESC, r.id DESC
LIMIT ? OFFSET ?;

-- name: AdminCountReturns :one
SELECT COUNT(*) FROM `shop_returns` WHERE (sqlc.arg(status) = '' OR status = sqlc.arg(status));

-- name: AdminGetReturn :one
SELECT r.*, i.variant_id FROM `shop_returns` r JOIN `shop_order_items` i ON i.id = r.order_item_id
WHERE r.id = ? LIMIT 1;

-- name: AdminDecideReturn :execrows
UPDATE `shop_returns` SET status = sqlc.arg(status), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND status = 'requested';

-- ─── reviews ───────────────────────────────────────────────────────────────────────────────────────────────────────

-- name: AdminListReviews :many
SELECT r.id, r.product_id, r.author_name, r.variant_label, r.rating, r.body, r.status, r.created_at,
  p.title AS product_title
FROM `shop_product_reviews` r JOIN `shop_products` p ON p.id = r.product_id
WHERE (sqlc.arg(status) = '' OR r.status = sqlc.arg(status))
ORDER BY r.created_at DESC, r.id DESC
LIMIT ? OFFSET ?;

-- name: AdminCountReviews :one
SELECT COUNT(*) FROM `shop_product_reviews` WHERE (sqlc.arg(status) = '' OR status = sqlc.arg(status));

-- name: AdminGetReview :one
SELECT * FROM `shop_product_reviews` WHERE id = ? LIMIT 1;

-- name: AdminSetReviewStatus :exec
UPDATE `shop_product_reviews` SET status = ?, updated_at = ? WHERE id = ?;

-- name: AdminDeleteReview :execrows
DELETE FROM `shop_product_reviews` WHERE id = ?;

-- ─── discount codes ────────────────────────────────────────────────────────────────────────────────────────────────

-- name: AdminListDiscountCodes :many
SELECT * FROM `shop_discount_codes` ORDER BY id DESC LIMIT ? OFFSET ?;

-- name: AdminCountDiscountCodes :one
SELECT COUNT(*) FROM `shop_discount_codes`;

-- name: AdminDiscountCodeTaken :one
SELECT COUNT(*) FROM `shop_discount_codes` WHERE code = sqlc.arg(code) AND id <> sqlc.arg(except_id);

-- name: AdminCreateDiscountCode :execlastid
INSERT INTO `shop_discount_codes` (`code`, `kind`, `value`, `max_discount`, `min_items_total`, `max_uses`,
  `per_user_limit`, `starts_at`, `ends_at`, `is_active`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AdminUpdateDiscountCode :exec
UPDATE `shop_discount_codes` SET code = ?, kind = ?, value = ?, max_discount = ?, min_items_total = ?, max_uses = ?,
  per_user_limit = ?, starts_at = ?, ends_at = ?, is_active = ?, updated_at = ?
WHERE id = ?;

-- name: AdminDeleteDiscountCode :execrows
DELETE FROM `shop_discount_codes` WHERE id = ?;

-- ─── layette template ──────────────────────────────────────────────────────────────────────────────────────────────

-- name: AdminListLayetteItems :many
SELECT * FROM `shop_layette_items` ORDER BY group_key, sort_order, id;

-- name: AdminGetLayetteItem :one
SELECT * FROM `shop_layette_items` WHERE id = ? LIMIT 1;

-- name: AdminCreateLayetteItem :execlastid
INSERT INTO `shop_layette_items` (`group_key`, `title`, `note`, `suggested_qty`, `is_essential`, `is_first_days`,
  `category_id`, `sort_order`, `is_active`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AdminUpdateLayetteItem :exec
UPDATE `shop_layette_items` SET group_key = ?, title = ?, note = ?, suggested_qty = ?, is_essential = ?,
  is_first_days = ?, category_id = ?, sort_order = ?, is_active = ?, updated_at = ?
WHERE id = ?;

-- name: AdminDeleteLayetteItem :execrows
DELETE FROM `shop_layette_items` WHERE id = ?;
