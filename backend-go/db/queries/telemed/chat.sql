-- Doctor chat, e-prescription and the doctor's own bookings (bloom B-N7-04, goose 00051). Callers resolve the actor
-- first (the user's own booking, or the booking of the doctor linked to the admin); message bodies are ciphertext.

-- name: GetDoctorByAdmin :one
SELECT * FROM `telemed_doctors` WHERE admin_id = ? LIMIT 1;

-- name: GetDoctorBooking :one
SELECT * FROM `telemed_bookings` WHERE id = ? AND doctor_id = ? LIMIT 1;

-- name: ListDoctorBookings :many
-- The doctor's confirmed and completed visits, newest first.
SELECT * FROM `telemed_bookings`
WHERE doctor_id = ? AND status IN ('confirmed', 'completed')
ORDER BY starts_at DESC, id DESC
LIMIT 500;

-- name: InsertMessage :execlastid
INSERT INTO `telemed_messages` (booking_id, sender, admin_id, body, file_id, created_at, updated_at)
VALUES (sqlc.arg(booking_id), sqlc.arg(sender), sqlc.narg(admin_id), sqlc.arg(body), sqlc.narg(file_id),
        sqlc.arg(now), sqlc.arg(now));

-- name: GetMessage :one
SELECT * FROM `telemed_messages` WHERE id = ? AND booking_id = ? LIMIT 1;

-- name: ListMessages :many
-- Messages of a booking after an id (0 = from the start), oldest first.
SELECT * FROM `telemed_messages`
WHERE booking_id = sqlc.arg(booking_id) AND id > sqlc.arg(after_id)
ORDER BY id ASC
LIMIT 200;

-- name: CountMessages :one
SELECT COUNT(*) FROM `telemed_messages` WHERE booking_id = ?;

-- name: MarkMessagesRead :exec
-- The other party's unread messages of a booking become read.
UPDATE `telemed_messages` SET read_at = sqlc.arg(now)
WHERE booking_id = sqlc.arg(booking_id) AND sender <> sqlc.arg(reader) AND read_at IS NULL;

-- name: InsertPrescriptionItem :execlastid
INSERT INTO `telemed_prescription_items` (booking_id, drug, dose, form, times, duration_days, instructions,
                                          created_at, updated_at)
VALUES (sqlc.arg(booking_id), sqlc.arg(drug), sqlc.narg(dose), sqlc.arg(form), sqlc.arg(times),
        sqlc.narg(duration_days), sqlc.narg(instructions), sqlc.arg(now), sqlc.arg(now));

-- name: ListPrescriptionItems :many
SELECT * FROM `telemed_prescription_items` WHERE booking_id = ? ORDER BY id ASC;

-- name: CountPrescriptionItems :one
SELECT COUNT(*) FROM `telemed_prescription_items` WHERE booking_id = ?;

-- name: GetPrescriptionItem :one
SELECT * FROM `telemed_prescription_items` WHERE id = ? AND booking_id = ? LIMIT 1;

-- name: LockPrescriptionItem :one
SELECT * FROM `telemed_prescription_items` WHERE id = ? AND booking_id = ? LIMIT 1 FOR UPDATE;

-- name: DeletePrescriptionItem :execrows
DELETE FROM `telemed_prescription_items` WHERE id = ? AND booking_id = ?;

-- name: MarkPrescriptionAdded :exec
UPDATE `telemed_prescription_items`
SET reminder_id = sqlc.arg(reminder_id), added_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);
