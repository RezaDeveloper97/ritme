-- User-local day (T-M9-26, D-126, X-07). user_profiles.timezone is the user's IANA zone (goose 00098); NULL means
-- unknown → Asia/Tehran.

-- name: GetUserTimezone :one
-- The stored zone, read lazily by the usertz middleware when a request carries no valid X-Timezone header.
SELECT timezone FROM `user_profiles` WHERE user_id = ? LIMIT 1;

-- name: StoreUserTimezone :execrows
-- Stores a validated X-Timezone zone after the response. Only an existing profile row is touched (no profile → 0
-- rows), only when the zone really changed, and updated_at is left alone: the zone moves "today" (which is in the
-- ETag), not the engine state, so there is no revision bump either.
UPDATE `user_profiles` SET timezone = sqlc.arg(timezone)
WHERE user_id = sqlc.arg(user_id) AND (timezone IS NULL OR timezone <> sqlc.arg(timezone));
