-- otp_verifications (App\Models\OtpVerification), as used by OtpAuthController.

-- name: GetRecentOtp :one
-- OtpVerification::where('mobile', $m)->where('created_at', '>', now()->subSeconds(60))->first().
SELECT id, created_at FROM `otp_verifications`
WHERE mobile = ? AND created_at > ?
LIMIT 1;

-- name: DeleteOtpsForMobile :exec
DELETE FROM `otp_verifications` WHERE mobile = ?;

-- name: InsertOtp :exec
INSERT INTO `otp_verifications` (mobile, code, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetLatestUnverifiedOtp :one
-- ->whereNull('verified_at')->latest()->first().
SELECT id, code, expires_at FROM `otp_verifications`
WHERE mobile = ? AND verified_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: ClaimOtpAttempt :execrows
-- Atomic attempt claim: whereKey($id)->where('attempts', '<', max)->increment('attempts').
-- One affected row = a slot was claimed; zero = the cap is reached.
UPDATE `otp_verifications`
SET attempts = attempts + 1, updated_at = ?
WHERE id = ? AND attempts < ?;

-- name: MarkOtpVerified :exec
UPDATE `otp_verifications` SET verified_at = ?, updated_at = ? WHERE id = ?;
