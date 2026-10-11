-- Business applications — the 4-step «ثبت مجموعه» join (canvas CB-DIR-04, table city_service_applications of goose
-- 00081_directory). Every applicant query is scoped by user_id in the SQL itself (IDOR): another account's row
-- behaves exactly like a missing one. Kept apart from bloom's raw-SQL cityservices store and from the public listing
-- queries (CB-DIR-02) on purpose: own directory, own generated package (internal/cityservices/join/store).

-- name: CountUserApplications :one
SELECT COUNT(*) FROM `city_service_applications` WHERE user_id = ?;

-- name: GetUserDraft :one
SELECT * FROM `city_service_applications` WHERE user_id = ? AND status = 'draft' ORDER BY id DESC LIMIT 1;

-- name: CreateApplication :execlastid
INSERT INTO `city_service_applications` (`user_id`, `status`, `step`, `created_at`, `updated_at`)
VALUES (?, 'draft', 1, ?, ?);

-- name: ListUserApplications :many
SELECT * FROM `city_service_applications` WHERE user_id = ? ORDER BY id DESC LIMIT 50;

-- name: GetUserApplication :one
SELECT * FROM `city_service_applications` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: LockUserApplication :one
SELECT * FROM `city_service_applications` WHERE id = ? AND user_id = ? LIMIT 1 FOR UPDATE;

-- name: SaveStep1 :exec
UPDATE `city_service_applications` SET `business_name` = ?, `category_slugs` = ?, `city_id` = ?, `district` = ?,
  `contact_name` = ?, `contact_phone` = ?, `owner_confirmed` = ?, `step` = GREATEST(`step`, 1), `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: SaveStep2 :exec
UPDATE `city_service_applications` SET `lat` = ?, `lng` = ?, `address` = ?, `about` = ?, `age_ranges` = ?,
  `amenities` = ?, `weekly_hours` = ?, `step` = GREATEST(`step`, 2), `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: SaveStep3 :exec
UPDATE `city_service_applications` SET `services` = ?, `step` = GREATEST(`step`, 3), `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: SaveStep4 :exec
UPDATE `city_service_applications` SET `booking_mode` = ?, `terms_accepted_at` = ?, `step` = GREATEST(`step`, 4),
  `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: SetApplicationPhotos :exec
UPDATE `city_service_applications` SET `photo_file_ids` = ?, `updated_at` = ? WHERE id = ? AND user_id = ?;

-- name: SetApplicationDocuments :exec
UPDATE `city_service_applications` SET `documents` = ?, `updated_at` = ? WHERE id = ? AND user_id = ?;

-- name: SubmitApplication :exec
-- A resubmission after needs_info keeps its tracking code (COALESCE) and clears the previous review note.
UPDATE `city_service_applications` SET `status` = 'submitted', `tracking_code` = COALESCE(`tracking_code`, ?),
  `submitted_at` = ?, `review_note` = NULL, `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: TrackingCodeTaken :one
SELECT EXISTS(SELECT 1 FROM `city_service_applications` WHERE `tracking_code` = ?) AS taken;

-- name: DeleteUserDraft :execrows
DELETE FROM `city_service_applications` WHERE id = ? AND user_id = ? AND status = 'draft';

-- name: ActiveCategorySlugs :many
SELECT `slug` FROM `city_service_categories` WHERE is_active = 1 ORDER BY sort_order, id;

-- name: ActiveCityExists :one
SELECT EXISTS(SELECT 1 FROM `city_service_cities` WHERE id = ? AND is_active = 1) AS found;

-- name: ActiveCatalogCodes :many
SELECT `code` FROM `catalog_items` WHERE `group` = ? AND is_active = 1 ORDER BY sort_order, id;

-- name: UserMobile :one
-- The submit SMS goes to the account's OTP-verified login mobile, never to the free-text step-1 contact_phone.
SELECT `mobile` FROM `users` WHERE id = ?;
