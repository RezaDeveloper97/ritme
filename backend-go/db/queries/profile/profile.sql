-- ProfileController (backend/app/Http/Controllers/Api/V1/ProfileController.php) and UserProfile.

-- name: GetUser :one
-- $user->fresh().
SELECT * FROM `users` WHERE id = ? LIMIT 1;

-- name: UpdateUserName :exec
-- $user->update(['name' => …]) when the name is dirty.
UPDATE `users` SET name = sqlc.narg(name), updated_at = sqlc.arg(updated_at) WHERE id = sqlc.arg(id);

-- name: DeleteUser :execrows
-- $user->delete(); every user-owned table cascades (FK ON DELETE CASCADE).
DELETE FROM `users` WHERE id = ?;

-- name: DeleteUserRefreshTokens :exec
-- DELETE /account (D-25, T-M2-34): the refresh tokens of the user's access tokens (no FK, no user_id).
DELETE rt FROM `oauth_refresh_tokens` rt
JOIN `oauth_access_tokens` a ON a.id = rt.access_token_id
WHERE a.user_id = sqlc.arg(user_id);

-- name: DeleteUserAccessTokens :exec
-- DELETE /account (D-25): the user's access tokens (no FK to users). A deleted token id still
-- answers the auth 401 `token_revoked` (passport verifier: row missing).
DELETE FROM `oauth_access_tokens` WHERE user_id = sqlc.arg(user_id);

-- name: GetProfileByUserID :one
-- $user->profile (hasOne: the first row in index order).
SELECT * FROM `user_profiles` WHERE user_id = ? ORDER BY id LIMIT 1;

-- name: GetProfileByID :one
-- $profile->fresh().
SELECT * FROM `user_profiles` WHERE id = ? LIMIT 1;

-- name: InsertProfile :execresult
-- new UserProfile(['user_id' => …])->save(): the row starts with the DB defaults; the
-- request's attributes follow in UpdateProfileAttributes inside the same transaction.
-- T-M9-23 (D-127): user_id is UNIQUE. A concurrent first save that lost the race changes
-- nothing on the existing row and gets its id back as LastInsertId (id = LAST_INSERT_ID(id)),
-- so the caller's attribute UPDATE lands on the one profile instead of failing with 1062.
INSERT INTO `user_profiles` (user_id, created_at, updated_at) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id);

-- name: UpdateProfileAttributes :exec
-- $profile->fill($data)->save(): only the dirty attributes are written (set_* flags). The
-- values are bound as the raw request strings, as PDO binds them, so MariaDB does the same
-- conversions (and rejects the same values) as it does for Laravel.
UPDATE `user_profiles` SET
    birthday            = IF(sqlc.arg(set_birthday), sqlc.narg(birthday), birthday),
    weight              = IF(sqlc.arg(set_weight), sqlc.narg(weight), weight),
    height              = IF(sqlc.arg(set_height), sqlc.narg(height), height),
    period_duration     = IF(sqlc.arg(set_period_duration), sqlc.narg(period_duration), period_duration),
    cycle_duration      = IF(sqlc.arg(set_cycle_duration), sqlc.narg(cycle_duration), cycle_duration),
    last_period_start   = IF(sqlc.arg(set_last_period_start), sqlc.narg(last_period_start), last_period_start),
    user_goal           = IF(sqlc.arg(set_user_goal), sqlc.narg(user_goal), user_goal),
    subscription_type   = IF(sqlc.arg(set_subscription_type), sqlc.narg(subscription_type), subscription_type),
    pregnancy_intention = IF(sqlc.arg(set_pregnancy_intention), sqlc.narg(pregnancy_intention), pregnancy_intention),
    chronic_conditions  = IF(sqlc.arg(set_chronic_conditions), sqlc.narg(chronic_conditions), chronic_conditions),
    updated_at          = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: MarkProfileRecalculated :exec
-- UserProfile::markRecalculated(): one atomic increment, so concurrent writes never lose a bump.
UPDATE `user_profiles` SET
    calculation_version      = calculation_version + 1,
    calculation_status       = 'completed',
    calculation_started_at   = sqlc.arg(now),
    calculation_completed_at = sqlc.arg(now),
    updated_at               = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: GetLiveMessagePayload :one
-- MessageContentRepository::payload(): a live (active + approved) row's raw JSON payload.
SELECT payload FROM `message_contents`
WHERE `group` = ? AND item_key = ? AND locale = ? AND is_active = 1 AND is_approved = 1
LIMIT 1;

-- name: GetProfileEngineInputs :one
-- Engine v6 (T-M9-24, goose 00098): the declared onboarding LMP (I-01) and the stored IANA time zone (X-07).
SELECT onboarding_last_period_start, timezone FROM `user_profiles` WHERE user_id = ? LIMIT 1;

-- name: SetOnboardingLastPeriodStart :exec
-- I-01: only profile.Save (the onboarding cycle step and the profile form) writes the declared anchor (T-M9-27,
-- D-123). user_id is UNIQUE (00096), so a missing profile row is created with the DB defaults and an existing one
-- keeps everything else.
INSERT INTO `user_profiles` (user_id, onboarding_last_period_start, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.narg(onboarding_last_period_start), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  onboarding_last_period_start = VALUES(onboarding_last_period_start),
  updated_at = VALUES(updated_at);

-- name: SetProfileTimezone :exec
-- X-07: the validated X-Timezone zone (the caller decides when it changed). Same upsert as above.
INSERT INTO `user_profiles` (user_id, timezone, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.narg(timezone), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  timezone = VALUES(timezone),
  updated_at = VALUES(updated_at);
