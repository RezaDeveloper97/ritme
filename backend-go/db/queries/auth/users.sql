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
-- $user->profile()->exists().
SELECT EXISTS(SELECT 1 FROM `user_profiles` WHERE user_id = ?) AS has_profile;
