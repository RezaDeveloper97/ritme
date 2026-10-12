-- Account data export (GET /profile/export, user decision 2026-10-12): the owner's directory rows — bookings, reviews,
-- reports, business applications — and the metadata of the files her applications reference (never the storage
-- path, the hash or the bytes).

-- name: ExportDirectoryBookings :many
SELECT * FROM `city_service_bookings` WHERE user_id = ? ORDER BY id;

-- name: ExportDirectoryReviews :many
SELECT * FROM `city_service_reviews` WHERE user_id = ? ORDER BY id;

-- name: ExportDirectoryReports :many
SELECT * FROM `city_service_reports` WHERE user_id = ? ORDER BY id;

-- name: ExportDirectoryApplications :many
SELECT * FROM `city_service_applications` WHERE user_id = ? ORDER BY id;

-- name: ExportDirectoryApplicationFiles :many
SELECT id, purpose, visibility, mime, size_bytes, created_at FROM `files`
WHERE user_id = ? AND purpose IN ('place_photo', 'place_photo_pending', 'place_licence') ORDER BY id;
