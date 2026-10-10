-- B-N7-08 admin console: bookings list / refund queue and the doctor-delete guard.

-- name: CountAdminBookings :one
SELECT COUNT(*) FROM `telemed_bookings` b
WHERE (sqlc.arg(status) = '' OR b.status = sqlc.arg(status))
  AND (sqlc.arg(payment_status) = '' OR b.payment_status = sqlc.arg(payment_status))
  AND (sqlc.arg(doctor_id) = 0 OR b.doctor_id = sqlc.arg(doctor_id))
  AND b.reference LIKE sqlc.arg(pattern);

-- name: ListAdminBookings :many
-- Newest first. The patient's mobile is returned so the handler can mask it; no notes, reasons or health data.
SELECT b.id, b.reference, b.user_id, b.doctor_id, b.mode, b.status, b.payment_status, b.starts_at, b.ends_at,
       b.total_rials, b.refunded_rials, b.refund_id, b.refunded_at, b.paid_at, b.gateway, b.created_at,
       d.name AS doctor_name, u.mobile AS user_mobile
FROM `telemed_bookings` b
JOIN `telemed_doctors` d ON d.id = b.doctor_id
JOIN `users` u ON u.id = b.user_id
WHERE (sqlc.arg(status) = '' OR b.status = sqlc.arg(status))
  AND (sqlc.arg(payment_status) = '' OR b.payment_status = sqlc.arg(payment_status))
  AND (sqlc.arg(doctor_id) = 0 OR b.doctor_id = sqlc.arg(doctor_id))
  AND b.reference LIKE sqlc.arg(pattern)
ORDER BY b.id DESC
LIMIT ? OFFSET ?;

-- name: CountRefundPendingBookings :one
SELECT COUNT(*) FROM `telemed_bookings` WHERE payment_status = 'refund_pending';

-- name: CountDoctorLiveBookings :one
-- Visits the FK cascade would destroy: open, done, or with money still to settle.
SELECT COUNT(*) FROM `telemed_bookings`
WHERE doctor_id = ? AND (status IN ('held', 'confirmed', 'completed') OR payment_status IN ('paid', 'refund_pending'));
