-- Visit bookings (bloom B-N7-03, goose 00050). Every user-facing read and write is scoped by user_id in the query;
-- doctor-wide reads (busy intervals, overlaps, sweeper) never return user data beyond the booked interval.

-- name: LockDoctor :one
-- Serialises bookings of one doctor: hold, reschedule and settlement take this row lock first.
SELECT id FROM `telemed_doctors` WHERE id = ? FOR UPDATE;

-- name: ExpireDoctorHolds :exec
-- Frees the slots of the doctor's lapsed holds (status expired, slot_key released) before a new hold is placed.
UPDATE `telemed_bookings`
SET status = 'expired', slot_key = NULL, updated_at = sqlc.arg(stamp)
WHERE doctor_id = sqlc.arg(doctor_id) AND status = 'held' AND hold_expires_at <= sqlc.arg(now);

-- name: CountOverlappingBookings :one
-- Active bookings (confirmed, or held and not lapsed) of the doctor overlapping [starts_at, ends_at), except one id.
SELECT COUNT(*) FROM `telemed_bookings`
WHERE doctor_id = sqlc.arg(doctor_id) AND id <> sqlc.arg(except_id)
  AND starts_at < sqlc.arg(ends_at) AND ends_at > sqlc.arg(starts_at)
  AND (status = 'confirmed' OR (status = 'held' AND hold_expires_at > sqlc.arg(now)));

-- name: ListBusyBookings :many
-- The intervals booked or held (not lapsed) for the given doctors overlapping [from_at, to_at), except one id.
SELECT doctor_id, starts_at, ends_at FROM `telemed_bookings`
WHERE doctor_id IN (sqlc.slice(doctor_ids)) AND id <> sqlc.arg(except_id)
  AND starts_at < sqlc.arg(to_at) AND ends_at > sqlc.arg(from_at)
  AND (status = 'confirmed' OR (status = 'held' AND hold_expires_at > sqlc.arg(now)))
ORDER BY doctor_id, starts_at;

-- name: InsertBooking :execlastid
INSERT INTO `telemed_bookings` (
    reference, user_id, doctor_id, mode, duration_minutes, starts_at, ends_at, status, slot_key, hold_expires_at,
    for_whom, child_id, patient_name, reason, note, price_rials, discount_rials, discount_source, total_rials,
    payment_status, gateway, created_at, updated_at
) VALUES (
    sqlc.arg(reference), sqlc.arg(user_id), sqlc.arg(doctor_id), sqlc.arg(mode), sqlc.arg(duration_minutes),
    sqlc.arg(starts_at), sqlc.arg(ends_at), 'held', sqlc.arg(slot_key), sqlc.arg(hold_expires_at),
    sqlc.arg(for_whom), sqlc.narg(child_id), sqlc.narg(patient_name), sqlc.narg(reason), sqlc.narg(note),
    sqlc.arg(price_rials), sqlc.arg(discount_rials), sqlc.narg(discount_source), sqlc.arg(total_rials),
    sqlc.arg(payment_status), sqlc.narg(gateway), sqlc.arg(now), sqlc.arg(now)
);

-- name: GetUserBooking :one
SELECT * FROM `telemed_bookings` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: LockUserBooking :one
SELECT * FROM `telemed_bookings` WHERE id = ? AND user_id = ? LIMIT 1 FOR UPDATE;

-- name: LockUserBookingByReference :one
SELECT * FROM `telemed_bookings` WHERE reference = ? AND user_id = ? LIMIT 1 FOR UPDATE;

-- name: GetBooking :one
-- Unscoped read by id for internal callers that already checked the actor (doctor side, sweeper).
SELECT * FROM `telemed_bookings` WHERE id = ? LIMIT 1;

-- name: ListUserBookings :many
-- The user's bookings that matter to her (no lapsed holds or failed payments), soonest first.
SELECT * FROM `telemed_bookings`
WHERE user_id = ? AND status IN ('held', 'confirmed', 'completed', 'cancelled')
ORDER BY starts_at ASC, id ASC
LIMIT 500;

-- name: NextUserBooking :one
-- The user's next confirmed visit that has not ended yet (the services hub card).
SELECT * FROM `telemed_bookings`
WHERE user_id = sqlc.arg(user_id) AND status = 'confirmed' AND ends_at > sqlc.arg(now)
ORDER BY starts_at ASC, id ASC
LIMIT 1;

-- name: SetBookingAuthority :exec
UPDATE `telemed_bookings` SET authority = ?, payment_status = 'pending', updated_at = ? WHERE id = ?;

-- name: ConfirmBooking :exec
-- Paid (or free): the hold becomes a confirmed booking; slot_key is (re)claimed for a hold that lapsed meanwhile.
UPDATE `telemed_bookings`
SET status = 'confirmed', slot_key = sqlc.arg(slot_key), hold_expires_at = NULL, payment_status = sqlc.arg(payment_status),
    ref_id = sqlc.narg(ref_id), card_pan = sqlc.narg(card_pan), paid_at = sqlc.narg(paid_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetBookingPaid :exec
-- Paid but not confirmable (slot gone, booking closed): the payment is recorded before it is refunded.
UPDATE `telemed_bookings`
SET payment_status = 'paid', ref_id = sqlc.narg(ref_id), card_pan = sqlc.narg(card_pan), paid_at = sqlc.narg(paid_at),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetBookingAppointment :exec
UPDATE `telemed_bookings` SET appointment_id = ?, updated_at = ? WHERE id = ?;

-- name: CloseBooking :exec
-- held/confirmed → cancelled | expired | failed: the slot is released.
UPDATE `telemed_bookings`
SET status = sqlc.arg(status), slot_key = NULL, hold_expires_at = NULL, cancelled_at = sqlc.narg(cancelled_at),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetBookingRefund :exec
UPDATE `telemed_bookings`
SET payment_status = sqlc.arg(payment_status), refund_id = sqlc.narg(refund_id), refunded_rials = sqlc.arg(refunded_rials),
    refunded_at = sqlc.narg(refunded_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: MarkPaymentFailed :exec
-- A declined or mismatched payment of a hold: the hold closes as failed.
UPDATE `telemed_bookings`
SET status = 'failed', slot_key = NULL, hold_expires_at = NULL, payment_status = 'none', updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status IN ('held', 'expired');

-- name: RescheduleBooking :exec
UPDATE `telemed_bookings`
SET starts_at = sqlc.arg(starts_at), ends_at = sqlc.arg(ends_at), slot_key = sqlc.arg(slot_key),
    reschedules = reschedules + 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'confirmed';

-- name: ListLapsedHolds :many
-- Holds whose payment window ended (the sweeper expires them).
SELECT id, user_id, doctor_id FROM `telemed_bookings`
WHERE status = 'held' AND hold_expires_at <= sqlc.arg(now)
ORDER BY id
LIMIT 500;

-- name: ExpireHold :execrows
UPDATE `telemed_bookings`
SET status = 'expired', slot_key = NULL, updated_at = sqlc.arg(stamp)
WHERE id = sqlc.arg(id) AND status = 'held' AND hold_expires_at <= sqlc.arg(now);

-- name: ListEndedConfirmed :many
-- Confirmed visits that ended (the sweeper completes them).
SELECT id, doctor_id FROM `telemed_bookings`
WHERE status = 'confirmed' AND ends_at <= sqlc.arg(now)
ORDER BY id
LIMIT 500;

-- name: CompleteBooking :execrows
UPDATE `telemed_bookings`
SET status = 'completed', slot_key = NULL, completed_at = sqlc.arg(now), updated_at = sqlc.arg(stamp)
WHERE id = sqlc.arg(id) AND status = 'confirmed';

-- name: IncrementDoctorVisits :exec
UPDATE `telemed_doctors` SET visits_count = visits_count + 1 WHERE id = ?;

-- name: ReviewableBooking :one
-- The user's oldest completed visit with the doctor that has no review yet.
SELECT b.id FROM `telemed_bookings` b
WHERE b.user_id = sqlc.arg(user_id) AND b.doctor_id = sqlc.arg(doctor_id) AND b.status = 'completed'
  AND NOT EXISTS (SELECT 1 FROM `telemed_reviews` r WHERE r.booking_id = b.id)
ORDER BY b.starts_at ASC, b.id ASC
LIMIT 1;

-- name: GetOwnedChild :one
-- A child of the user (the «برای چه کسی؟» chip); spouse-shared children are not bookable by the spouse.
SELECT id, name, birth_date FROM `children` WHERE id = ? AND owner_id = ? LIMIT 1;

-- name: ListOwnedChildrenForBooking :many
SELECT id, name, birth_date FROM `children` WHERE owner_id = ? ORDER BY birth_date DESC, id;

-- name: ListBookingConsents :many
SELECT * FROM `telemed_booking_consents` WHERE booking_id = ? ORDER BY id;

-- name: UpsertConsent :exec
-- Grant (again): a revoked scope becomes active with a fresh granted_at.
INSERT INTO `telemed_booking_consents` (booking_id, scope, granted_at, revoked_at, created_at, updated_at)
VALUES (sqlc.arg(booking_id), sqlc.arg(scope), sqlc.arg(granted_at), NULL, sqlc.arg(stamp), sqlc.arg(stamp))
ON DUPLICATE KEY UPDATE
  granted_at = IF(revoked_at IS NULL, granted_at, VALUES(granted_at)),
  revoked_at = NULL,
  updated_at = VALUES(updated_at);

-- name: RevokeConsent :exec
UPDATE `telemed_booking_consents`
SET revoked_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE booking_id = sqlc.arg(booking_id) AND scope = sqlc.arg(scope) AND revoked_at IS NULL;

-- name: RevokeAllConsents :exec
UPDATE `telemed_booking_consents`
SET revoked_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE booking_id = sqlc.arg(booking_id) AND revoked_at IS NULL;

-- name: GetActiveConsent :one
-- The active consent of one scope on a booking of this doctor (the doctor-side share check).
SELECT c.* FROM `telemed_booking_consents` c
JOIN `telemed_bookings` b ON b.id = c.booking_id
WHERE c.booking_id = sqlc.arg(booking_id) AND b.doctor_id = sqlc.arg(doctor_id) AND c.scope = sqlc.arg(scope)
  AND c.revoked_at IS NULL
LIMIT 1;
