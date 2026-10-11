-- banners (App\Models\Banner), read side of BannerController.

-- name: ListActiveBanners :many
-- Banner::active(): is_active, inside the optional [starts_at, ends_at] window at `now`
-- (bound as Tehran wall-clock 'Y-m-d H:i:s'), ordered sort_order, id DESC. The
-- controller's whereIn('position', …) is applied by the caller.
SELECT id, title, image_path, position, link_url, link_type
FROM `banners`
WHERE is_active = 1
  AND kind = 'banner'
  AND (starts_at IS NULL OR starts_at <= sqlc.arg(now))
  AND (ends_at IS NULL OR ends_at >= sqlc.arg(now))
ORDER BY sort_order, id DESC;

-- name: ListActiveAds :many
-- Sponsored slots (kind = 'ad', B-N10-02): the same active window / ordering as ListActiveBanners. The advertiser
-- label is deliberately not selected (admin-only). Mode targeting is applied by the caller.
SELECT id, title, image_path, position, link_url, link_type, cta_label, modes
FROM `banners`
WHERE is_active = 1
  AND kind = 'ad'
  AND (starts_at IS NULL OR starts_at <= sqlc.arg(now))
  AND (ends_at IS NULL OR ends_at >= sqlc.arg(now))
ORDER BY sort_order, id DESC;

-- name: GetActiveAdID :one
SELECT id FROM `banners`
WHERE id = sqlc.arg(id) AND is_active = 1 AND kind = 'ad'
  AND (starts_at IS NULL OR starts_at <= sqlc.arg(now))
  AND (ends_at IS NULL OR ends_at >= sqlc.arg(now))
LIMIT 1;

-- name: BumpBannerImpression :exec
-- Per-day aggregate only: banner id + Tehran calendar day. No user id, no device, no health context.
INSERT INTO `banner_daily_stats` (banner_id, day, impressions, clicks) VALUES (sqlc.arg(banner_id), sqlc.arg(day), 1, 0)
ON DUPLICATE KEY UPDATE impressions = impressions + 1;

-- name: BumpBannerClick :exec
INSERT INTO `banner_daily_stats` (banner_id, day, impressions, clicks) VALUES (sqlc.arg(banner_id), sqlc.arg(day), 0, 1)
ON DUPLICATE KEY UPDATE clicks = clicks + 1;
