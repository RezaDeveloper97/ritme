-- Companion account and panel home (bloom B-N4-03). Read-only.

-- name: GetUserGender :one
-- user_life_profiles.gender of one account (onboarding v2 Gender step; no row / NULL = never asked → a woman's
-- account, the legacy default). male = a companion account.
SELECT gender FROM `user_life_profiles` WHERE user_id = ? LIMIT 1;

-- name: ListCompanionTipContents :many
-- The live admin copy of the «امروز چه کار کنی؟» tips (message_contents group companion_tip) in the given locales
-- (request locale + default language). Not user data.
SELECT item_key, locale, payload FROM `message_contents`
WHERE `group` = 'companion_tip' AND is_active = 1 AND is_approved = 1 AND locale IN (sqlc.slice(locales));

-- name: ListCompanionArticles :many
-- «برای خواندن» on the companion home: published articles of category `companion`, then articles tagged with the
-- partner's phase (has_phases = 0 skips them); untagged general articles are not companion reading.
SELECT id, slug, title, excerpt, category, read_time_minutes, image_url, image_path
FROM `articles`
WHERE is_published = 1
  AND (category = 'companion'
    OR (CAST(sqlc.arg(has_phases) AS SIGNED) = 1 AND cycle_phases IS NOT NULL AND JSON_OVERLAPS(cycle_phases, sqlc.arg(phases))))
ORDER BY (category = 'companion') DESC, sort_order, id
LIMIT 4;
