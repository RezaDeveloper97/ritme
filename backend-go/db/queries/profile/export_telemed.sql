-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportTelemedBookings :many
SELECT `id`, `reference`, `user_id`, `doctor_id`, `mode`, `duration_minutes`, `starts_at`, `ends_at`, `status`, `slot_key`, `hold_expires_at`, `for_whom`, `child_id`, `patient_name`, `reason`, `note`, `price_rials`, `discount_rials`, `discount_source`, `total_rials`, `payment_status`, `gateway`, `ref_id`, `card_pan`, `paid_at`, `refund_id`, `refunded_rials`, `refunded_at`, `appointment_id`, `reschedules`, `cancelled_at`, `completed_at`, `created_at`, `updated_at`
FROM `telemed_bookings`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportTelemedBookingConsents :many
SELECT `id`, `booking_id`, `scope`, `granted_at`, `revoked_at`, `created_at`, `updated_at`
FROM `telemed_booking_consents`
WHERE booking_id IN (SELECT id FROM `telemed_bookings` WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportTelemedMessages :many
SELECT `id`, `booking_id`, `sender`, `admin_id`, `body`, `file_id`, `read_at`, `created_at`, `updated_at`
FROM `telemed_messages`
WHERE booking_id IN (SELECT id FROM `telemed_bookings` WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY booking_id, id;

-- name: AcctExportTelemedPrescriptionItems :many
SELECT `id`, `booking_id`, `drug`, `dose`, `form`, `times`, `duration_days`, `instructions`, `reminder_id`, `added_at`, `created_at`, `updated_at`
FROM `telemed_prescription_items`
WHERE booking_id IN (SELECT id FROM `telemed_bookings` WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportTelemedReviews :many
SELECT `id`, `doctor_id`, `user_id`, `booking_id`, `rating`, `body`, `is_visible`, `created_at`, `updated_at`
FROM `telemed_reviews`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
