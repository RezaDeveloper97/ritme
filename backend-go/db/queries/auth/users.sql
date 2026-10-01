-- Users as the auth guard and OtpAuthController see them (App\Models\User).

-- name: GetUserByID :one
-- EloquentUserProvider::retrieveById (the token's `sub`).
SELECT * FROM `users` WHERE id = ? LIMIT 1;

-- name: GetUserByMobile :one
-- User::where('mobile', $mobile)->first().
SELECT * FROM `users` WHERE mobile = ? LIMIT 1;

-- name: CreateUser :execresult
-- User::create(['mobile' => $mobile]): only mobile + timestamps (mobile_verified_at is not
-- fillable, so Laravel never sets it — preserved).
INSERT INTO `users` (mobile, created_at, updated_at) VALUES (?, ?, ?);

-- name: UserHasProfile :one
-- $user->profile()->exists(). B-N2-01: a v2 onboarding that was started and not completed (user_life_profiles)
-- counts as no profile yet; users without that row (everyone created before it) keep the Laravel rule.
SELECT EXISTS(
  SELECT 1 FROM `user_profiles` p
  LEFT JOIN `user_life_profiles` l ON l.user_id = p.user_id
  WHERE p.user_id = ? AND (l.id IS NULL OR l.onboarding_started_at IS NULL OR l.onboarding_completed_at IS NOT NULL)
) AS has_profile;
