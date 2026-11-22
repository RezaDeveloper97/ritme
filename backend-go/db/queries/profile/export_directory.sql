-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportCityServiceListings :many
SELECT `id`, `category_id`, `city_id`, `name`, `district`, `address`, `phone`, `description`, `price_text`, `hours_text`, `photos`, `booking_mode`, `booking_url`, `sort_order`, `is_active`, `created_at`, `updated_at`, `status`, `verified_at`, `owner_user_id`, `lat`, `lng`, `about`, `age_ranges`, `amenities`, `weekly_hours`, `photo_file_ids`, `website_url`, `cancel_free_hours`, `cancel_policy`, `rules`, `rating_avg`, `rating_count`
FROM `city_service_listings`
WHERE owner_user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportCityServiceServices :many
SELECT `id`, `listing_id`, `kind`, `name`, `description`, `format`, `duration_minutes`, `age_min_months`, `age_max_months`, `capacity`, `price_toman`, `package_sessions`, `package_validity_days`, `sort_order`, `is_active`, `created_at`, `updated_at`
FROM `city_service_services`
WHERE listing_id IN (SELECT id FROM `city_service_listings` WHERE owner_user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportCityServiceSlots :many
SELECT `id`, `listing_id`, `service_id`, `starts_at`, `ends_at`, `capacity`, `booked_count`, `status`, `created_at`, `updated_at`
FROM `city_service_slots`
WHERE listing_id IN (SELECT id FROM `city_service_listings` WHERE owner_user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;
