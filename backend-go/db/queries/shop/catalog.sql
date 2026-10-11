-- Internal shop catalog (CB-SHOP-01, on bloom's 00071): sellers, sub-categories, variants and verified-purchase reviews.
-- No query here joins a health table. Link-out rows (external_url set, no variants) keep using internal/shop's own SQL.

-- name: GetSeller :one
SELECT * FROM `shop_sellers` WHERE id = ? LIMIT 1;

-- name: ListActiveSellers :many
SELECT * FROM `shop_sellers` WHERE is_active = 1 ORDER BY sort_order, id;

-- name: ListSubCategories :many
SELECT * FROM `shop_categories` WHERE parent_id = ? AND is_active = 1 ORDER BY sort_order, id;

-- name: ListActiveVariantsByProduct :many
SELECT * FROM `shop_product_variants` WHERE product_id = ? AND is_active = 1 ORDER BY sort_order, id;

-- name: GetVariant :one
SELECT * FROM `shop_product_variants` WHERE id = ? LIMIT 1;

-- name: CountActiveVariantsByProduct :one
SELECT COUNT(*) FROM `shop_product_variants` WHERE product_id = ? AND is_active = 1;

-- name: DecrementVariantStock :execrows
UPDATE `shop_product_variants` SET stock = stock - sqlc.arg(quantity), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND stock >= sqlc.arg(quantity);

-- name: ListPublishedReviews :many
SELECT * FROM `shop_product_reviews` WHERE product_id = ? AND status = 'published'
ORDER BY created_at DESC, id DESC LIMIT ?;

-- name: RefreshProductRating :exec
UPDATE `shop_products` p SET
  rating_count = (SELECT COUNT(*) FROM `shop_product_reviews` r WHERE r.product_id = p.id AND r.status = 'published'),
  rating_avg = (SELECT ROUND(AVG(r.rating), 2) FROM `shop_product_reviews` r WHERE r.product_id = p.id AND r.status = 'published')
WHERE p.id = ?;
