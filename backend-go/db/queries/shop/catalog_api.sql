-- Shop catalog read API (CB-SHOP-02): home sections, list + filters, product detail, reviews, bought-together.
-- "Catalog product" = an internal product: active, in an active category, with at least one active variant, and no
-- inactive seller. Link-out rows (no variants) stay on bloom's /shop/products (internal/shop). No health table here.

-- name: ListActiveCategories :many
SELECT * FROM `shop_categories` WHERE is_active = 1 ORDER BY sort_order, id;

-- name: ListCatalogProductsInCategories :many
SELECT * FROM `shop_products` p
WHERE p.is_active = 1
  AND p.category_id IN (sqlc.slice('category_ids'))
  AND EXISTS (SELECT 1 FROM `shop_product_variants` v WHERE v.product_id = p.id AND v.is_active = 1)
  AND (p.seller_id IS NULL OR EXISTS (SELECT 1 FROM `shop_sellers` s WHERE s.id = p.seller_id AND s.is_active = 1))
ORDER BY p.purchase_count DESC, p.id DESC
LIMIT ?;

-- name: ListCatalogProductsByIDs :many
SELECT * FROM `shop_products` p
WHERE p.id IN (sqlc.slice('ids'))
  AND p.is_active = 1
  AND EXISTS (SELECT 1 FROM `shop_categories` c WHERE c.id = p.category_id AND c.is_active = 1)
  AND EXISTS (SELECT 1 FROM `shop_product_variants` v WHERE v.product_id = p.id AND v.is_active = 1)
  AND (p.seller_id IS NULL OR EXISTS (SELECT 1 FROM `shop_sellers` s WHERE s.id = p.seller_id AND s.is_active = 1));

-- name: ListActiveVariantsByProducts :many
SELECT * FROM `shop_product_variants` WHERE is_active = 1 AND product_id IN (sqlc.slice('product_ids'))
ORDER BY product_id, sort_order, id;

-- name: ListSellersByIDs :many
SELECT * FROM `shop_sellers` WHERE id IN (sqlc.slice('ids'));

-- name: ListPublishedReviewsPage :many
SELECT * FROM `shop_product_reviews` WHERE product_id = ? AND status = 'published'
ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?;

-- name: CountPublishedReviews :one
SELECT COUNT(*) FROM `shop_product_reviews` WHERE product_id = ? AND status = 'published';

-- name: ListBoughtTogether :many
-- Products that share orders with sqlc.arg(product_id), most shared first (order lines only; no user column read).
SELECT o2.product_id AS product_id, COUNT(DISTINCT o2.order_id) AS orders
FROM `shop_order_items` o1
JOIN `shop_order_items` o2 ON o2.order_id = o1.order_id AND o2.product_id <> o1.product_id
WHERE o1.product_id = sqlc.arg(product_id) AND o2.product_id IS NOT NULL
GROUP BY o2.product_id
ORDER BY orders DESC, o2.product_id
LIMIT ?;
