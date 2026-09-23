-- banners (App\Models\Banner), read side of BannerController.

-- name: ListActiveBanners :many
-- Banner::active(): is_active, inside the optional [starts_at, ends_at] window at `now`
-- (bound as Tehran wall-clock 'Y-m-d H:i:s'), ordered sort_order, id DESC. The
-- controller's whereIn('position', …) is applied by the caller.
SELECT id, title, image_path, position, link_url, link_type
FROM `banners`
WHERE is_active = 1
  AND (starts_at IS NULL OR starts_at <= sqlc.arg(now))
  AND (ends_at IS NULL OR ends_at >= sqlc.arg(now))
ORDER BY sort_order, id DESC;
