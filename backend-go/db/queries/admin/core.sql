-- Admin API core (T-M2-20): admin accounts, dashboard, user management.
-- Ported from backend/app/Http/Controllers/Admin/{Auth,Dashboard,User,Admin,Account}Controller.php.

-- ---------------------------------------------------------------------------
-- Admins

-- name: GetAdminByID :one
SELECT * FROM `admins` WHERE id = ? LIMIT 1;

-- name: GetAdminByEmail :one
-- EloquentUserProvider::retrieveByCredentials (email only; is_active is checked in Go).
SELECT * FROM `admins` WHERE email = ? LIMIT 1;

-- name: TouchAdminLastLogin :exec
-- forceFill(['last_login_at' => now()])->save().
UPDATE `admins` SET last_login_at = sqlc.arg(now), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: UpdateAdminPassword :exec
UPDATE `admins` SET password = sqlc.arg(password), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: CountAdmins :one
SELECT COUNT(*) FROM `admins`;

-- name: ListAdmins :many
-- Admin::orderByDesc('id')->paginate(20).
SELECT * FROM `admins` ORDER BY id DESC LIMIT ? OFFSET ?;

-- name: AdminEmailTaken :one
-- unique:admins,email (->ignore($admin) when except_id > 0).
SELECT EXISTS(SELECT 1 FROM `admins` WHERE email = sqlc.arg(email) AND id <> sqlc.arg(except_id)) AS taken;

-- name: CreateAdmin :execresult
INSERT INTO `admins` (name, email, password, role, is_active, created_at, updated_at)
VALUES (sqlc.arg(name), sqlc.arg(email), sqlc.arg(password), sqlc.arg(role), sqlc.arg(is_active), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateAdmin :exec
UPDATE `admins`
SET name = sqlc.arg(name), email = sqlc.arg(email), role = sqlc.arg(role), is_active = sqlc.arg(is_active),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: DeleteAdmin :execresult
DELETE FROM `admins` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Dashboard (DashboardController::index)

-- name: DashboardCounts :one
SELECT
    (SELECT COUNT(*) FROM `users`)                                              AS users,
    (SELECT COUNT(*) FROM `users` ub WHERE ub.blocked_at IS NOT NULL)            AS users_blocked,
    (SELECT COUNT(*) FROM `users` uw WHERE uw.created_at >= sqlc.arg(week_start)) AS users_new_week,
    (SELECT COUNT(*) FROM `users` ut WHERE ut.created_at >= sqlc.arg(today_start)) AS users_new_today,
    (SELECT COUNT(*) FROM `articles`)                                           AS articles,
    (SELECT COUNT(*) FROM `affirmations`)                                       AS affirmations,
    (SELECT COUNT(*) FROM `challenges`)                                         AS challenges,
    (SELECT COUNT(*) FROM `task_templates`)                                     AS task_templates,
    (SELECT COUNT(*) FROM `message_contents`)                                   AS messages,
    (SELECT COUNT(*) FROM `message_contents` mp WHERE mp.is_approved = 0)       AS messages_pending;

-- name: RecentUsers :many
-- User::latest()->take(8).
SELECT id, name, mobile, email, blocked_at, created_at
FROM `users` ORDER BY created_at DESC, id DESC LIMIT ?;

-- ---------------------------------------------------------------------------
-- Users (UserController)

-- name: CountUsers :one
-- Filters: pattern is a LIKE pattern over name/mobile/email ('%' = no search; Go escapes % and _ in
-- the search text). The status filter is a range over blocked_at with NULL mapped to the sentinel
-- 1971-01-01 (sqlc cannot type a bare string flag here; the sentinel must stay inside the TIMESTAMP
-- range): all = [1971-01-01, 2037-12-31], active = [1971-01-01, 1971-01-01],
-- blocked = [1971-01-02, 2037-12-31]. See users.statusRange.
SELECT COUNT(*) FROM `users` u
WHERE (IFNULL(u.name, '') LIKE sqlc.arg(pattern) OR IFNULL(u.mobile, '') LIKE sqlc.arg(pattern) OR IFNULL(u.email, '') LIKE sqlc.arg(pattern))
  AND COALESCE(u.blocked_at, CAST('1971-01-01 00:00:00' AS DATETIME)) >= sqlc.arg(blocked_from)
  AND COALESCE(u.blocked_at, CAST('1971-01-01 00:00:00' AS DATETIME)) <= sqlc.arg(blocked_to);

-- name: ListUsers :many
-- User::with('profile')->…->latest()->paginate(20). The profile is the user's first row (hasOne).
SELECT u.id, u.name, u.mobile, u.email, u.blocked_at, u.created_at,
       p.subscription_type, p.user_goal
FROM `users` u
LEFT JOIN `user_profiles` p ON p.id = (SELECT MIN(p2.id) FROM `user_profiles` p2 WHERE p2.user_id = u.id)
WHERE (IFNULL(u.name, '') LIKE sqlc.arg(pattern) OR IFNULL(u.mobile, '') LIKE sqlc.arg(pattern) OR IFNULL(u.email, '') LIKE sqlc.arg(pattern))
  AND COALESCE(u.blocked_at, CAST('1971-01-01 00:00:00' AS DATETIME)) >= sqlc.arg(blocked_from)
  AND COALESCE(u.blocked_at, CAST('1971-01-01 00:00:00' AS DATETIME)) <= sqlc.arg(blocked_to)
ORDER BY u.created_at DESC, u.id DESC
LIMIT ? OFFSET ?;

-- name: GetUserDetail :one
SELECT u.id, u.name, u.mobile, u.email, u.mobile_verified_at, u.blocked_at, u.created_at, u.updated_at,
       p.id AS profile_id, p.birthday, p.last_period_start, p.cycle_duration, p.period_duration,
       p.subscription_type, p.user_goal, p.pregnancy_intention
FROM `users` u
LEFT JOIN `user_profiles` p ON p.id = (SELECT MIN(p2.id) FROM `user_profiles` p2 WHERE p2.user_id = u.id)
WHERE u.id = ? LIMIT 1;

-- name: UserStats :one
SELECT
    (SELECT COUNT(*) FROM `daily_health_logs` WHERE daily_health_logs.user_id = sqlc.arg(user_id))   AS health_logs,
    (SELECT COUNT(*) FROM `reminders` WHERE reminders.user_id = sqlc.arg(user_id))                   AS reminders,
    (SELECT COUNT(*) FROM `user_notifications` WHERE user_notifications.user_id = sqlc.arg(user_id)) AS notifications;

-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM `users` WHERE id = ?) AS found;

-- name: UpdateUserName :exec
-- $user->update(['name' => …]): updated_at only moves when the name changed (Eloquent dirty check).
-- MariaDB evaluates SET left to right, so updated_at is compared against the old name.
UPDATE `users`
SET updated_at = CASE WHEN name <=> sqlc.narg(name) THEN updated_at ELSE sqlc.arg(now) END,
    name = sqlc.narg(name)
WHERE id = sqlc.arg(id);

-- name: FirstProfileID :one
SELECT id FROM `user_profiles` WHERE user_id = ? ORDER BY id LIMIT 1;

-- name: UpdateProfilePlan :exec
UPDATE `user_profiles`
SET updated_at = CASE WHEN subscription_type = sqlc.arg(subscription_type) AND user_goal = sqlc.arg(user_goal)
                      THEN updated_at ELSE sqlc.arg(now) END,
    subscription_type = sqlc.arg(subscription_type),
    user_goal = sqlc.arg(user_goal)
WHERE id = sqlc.arg(id);

-- name: CreateProfilePlan :exec
INSERT INTO `user_profiles` (user_id, subscription_type, user_goal, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(subscription_type), sqlc.arg(user_goal), sqlc.arg(now), sqlc.arg(now));

-- name: SetUserBlockedAt :exec
-- forceFill(['blocked_at' => now()|null])->save().
UPDATE `users` SET blocked_at = sqlc.narg(blocked_at), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteUser :execresult
DELETE FROM `users` WHERE id = ?;
