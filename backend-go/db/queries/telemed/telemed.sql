-- Doctors directory (bloom B-N7-02; internal/telemed): doctors / midwives, accepted insurers, visit types, weekly
-- availability, time off and reviews. Public reads only see active doctors with at least one active visit type; the
-- only user-owned rows are reviews, always scoped by user_id in the query itself (IDOR).

-- ---------------------------------------------------------------------------
-- Public directory

-- name: ListDirectoryDoctors :many
-- Every listed doctor matching the filters (empty string = no filter; insurer 'any' = accepts at least one insurer),
-- in display order, capped by the limit. The "today" filter and the next free slot are computed in Go.
SELECT d.* FROM `telemed_doctors` d
WHERE d.is_active = 1
  AND EXISTS (SELECT 1 FROM `telemed_visit_types` v WHERE v.doctor_id = d.id AND v.is_active = 1
              AND (sqlc.arg(mode) = '' OR v.mode = sqlc.arg(mode)))
  AND (sqlc.arg(kind) = '' OR d.kind = sqlc.arg(kind))
  AND (sqlc.arg(specialty) = '' OR d.specialty = sqlc.arg(specialty))
  AND (sqlc.arg(city) = '' OR d.city = sqlc.arg(city))
  AND (sqlc.arg(insurer) = ''
       OR (sqlc.arg(insurer) = 'any' AND EXISTS (SELECT 1 FROM `telemed_doctor_insurers` i WHERE i.doctor_id = d.id))
       OR EXISTS (SELECT 1 FROM `telemed_doctor_insurers` i WHERE i.doctor_id = d.id AND i.insurer = sqlc.arg(insurer)))
  AND (sqlc.arg(pattern) = '%' OR d.name LIKE CAST(sqlc.arg(pattern) AS CHAR) OR d.name LIKE CAST(sqlc.arg(json_pattern) AS CHAR))
ORDER BY d.sort_order, d.id
LIMIT ?;

-- name: GetListedDoctor :one
-- One active doctor with at least one active visit type (the public profile; anything else is a 404).
SELECT d.* FROM `telemed_doctors` d
WHERE d.id = sqlc.arg(id) AND d.is_active = 1
  AND EXISTS (SELECT 1 FROM `telemed_visit_types` v WHERE v.doctor_id = d.id AND v.is_active = 1)
LIMIT 1;

-- name: ListActiveVisitTypesByDoctors :many
SELECT * FROM `telemed_visit_types`
WHERE doctor_id IN (sqlc.slice(doctor_ids)) AND is_active = 1
ORDER BY doctor_id, FIELD(mode, 'video', 'phone', 'in_person'), id;

-- name: ListRulesByDoctors :many
SELECT * FROM `telemed_availability_rules`
WHERE doctor_id IN (sqlc.slice(doctor_ids))
ORDER BY doctor_id, weekday, start_minute, id;

-- name: ListTimeOffByDoctors :many
-- Absences overlapping [from, to).
SELECT * FROM `telemed_time_off`
WHERE doctor_id IN (sqlc.slice(doctor_ids)) AND ends_at > sqlc.arg(from_at) AND starts_at < sqlc.arg(to_at)
ORDER BY doctor_id, starts_at, id;

-- name: ListInsurersByDoctors :many
SELECT * FROM `telemed_doctor_insurers`
WHERE doctor_id IN (sqlc.slice(doctor_ids))
ORDER BY doctor_id, insurer;

-- name: CountListedBySpecialty :many
SELECT d.specialty, COUNT(*) AS doctors FROM `telemed_doctors` d
WHERE d.is_active = 1
  AND EXISTS (SELECT 1 FROM `telemed_visit_types` v WHERE v.doctor_id = d.id AND v.is_active = 1)
GROUP BY d.specialty;

-- name: CountListedByCity :many
SELECT d.city, COUNT(*) AS doctors FROM `telemed_doctors` d
WHERE d.is_active = 1 AND d.city IS NOT NULL
  AND EXISTS (SELECT 1 FROM `telemed_visit_types` v WHERE v.doctor_id = d.id AND v.is_active = 1)
GROUP BY d.city;

-- name: CountListedByInsurer :many
SELECT i.insurer, COUNT(*) AS doctors FROM `telemed_doctor_insurers` i
JOIN `telemed_doctors` d ON d.id = i.doctor_id
WHERE d.is_active = 1
  AND EXISTS (SELECT 1 FROM `telemed_visit_types` v WHERE v.doctor_id = d.id AND v.is_active = 1)
GROUP BY i.insurer;

-- ---------------------------------------------------------------------------
-- Reviews

-- name: ListVisibleReviews :many
-- A doctor's visible reviews, newest first, with the reviewer's name (only its first letter leaves the server).
SELECT r.id, r.user_id, r.rating, r.body, r.created_at, u.name AS user_name
FROM `telemed_reviews` r
JOIN `users` u ON u.id = r.user_id
WHERE r.doctor_id = sqlc.arg(doctor_id) AND r.is_visible = 1
ORDER BY r.created_at DESC, r.id DESC
LIMIT ? OFFSET ?;

-- name: CountVisibleReviews :one
SELECT COUNT(*) FROM `telemed_reviews` WHERE doctor_id = sqlc.arg(doctor_id) AND is_visible = 1;

-- name: InsertReview :execlastid
INSERT INTO `telemed_reviews` (doctor_id, user_id, booking_id, rating, body, is_visible, created_at, updated_at)
VALUES (sqlc.arg(doctor_id), sqlc.arg(user_id), sqlc.narg(booking_id), sqlc.arg(rating), sqlc.narg(body), 1,
        sqlc.arg(now), sqlc.arg(now));

-- name: GetUserReview :one
SELECT r.id, r.user_id, r.rating, r.body, r.created_at, u.name AS user_name
FROM `telemed_reviews` r
JOIN `users` u ON u.id = r.user_id
WHERE r.id = sqlc.arg(id) AND r.doctor_id = sqlc.arg(doctor_id) AND r.user_id = sqlc.arg(user_id)
LIMIT 1;

-- name: DeleteUserReview :execrows
DELETE FROM `telemed_reviews`
WHERE id = sqlc.arg(id) AND doctor_id = sqlc.arg(doctor_id) AND user_id = sqlc.arg(user_id);

-- name: RefreshDoctorRating :exec
-- Recomputes the denormalised rating of a doctor from its visible reviews (same transaction as the review write).
UPDATE `telemed_doctors` d
SET d.rating_sum = (SELECT CAST(COALESCE(SUM(r.rating), 0) AS UNSIGNED) FROM `telemed_reviews` r
                    WHERE r.doctor_id = d.id AND r.is_visible = 1),
    d.rating_count = (SELECT COUNT(*) FROM `telemed_reviews` r WHERE r.doctor_id = d.id AND r.is_visible = 1),
    d.positive_count = (SELECT COUNT(*) FROM `telemed_reviews` r
                        WHERE r.doctor_id = d.id AND r.is_visible = 1 AND r.rating >= 4)
WHERE d.id = sqlc.arg(id);

-- ---------------------------------------------------------------------------
-- Admin

-- name: CountAdminDoctors :one
SELECT COUNT(*) FROM `telemed_doctors`
WHERE (name LIKE CAST(sqlc.arg(pattern) AS CHAR) OR name LIKE CAST(sqlc.arg(json_pattern) AS CHAR)
       OR licence_no LIKE CAST(sqlc.arg(pattern) AS CHAR))
  AND is_active >= sqlc.arg(active_min) AND is_active <= sqlc.arg(active_max);

-- name: ListAdminDoctors :many
SELECT * FROM `telemed_doctors`
WHERE (name LIKE CAST(sqlc.arg(pattern) AS CHAR) OR name LIKE CAST(sqlc.arg(json_pattern) AS CHAR)
       OR licence_no LIKE CAST(sqlc.arg(pattern) AS CHAR))
  AND is_active >= sqlc.arg(active_min) AND is_active <= sqlc.arg(active_max)
ORDER BY sort_order, id
LIMIT ? OFFSET ?;

-- name: GetDoctor :one
SELECT * FROM `telemed_doctors` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: NextDoctorSortOrder :one
SELECT CAST(COALESCE(MAX(sort_order), 0) + 1 AS SIGNED) FROM `telemed_doctors`;

-- name: InsertDoctor :execlastid
INSERT INTO `telemed_doctors` (
  kind, name, headline, bio, specialty, city, licence_no, experience_years, photo_path, response_minutes,
  visits_count, rating_sum, rating_count, positive_count, is_active, sort_order, admin_id, created_at, updated_at
) VALUES (
  sqlc.arg(kind), sqlc.arg(name), sqlc.narg(headline), sqlc.narg(bio), sqlc.arg(specialty), sqlc.narg(city),
  sqlc.arg(licence_no), sqlc.narg(experience_years), NULL, sqlc.narg(response_minutes),
  0, 0, 0, 0, sqlc.arg(is_active), sqlc.arg(sort_order), NULL, sqlc.arg(now), sqlc.arg(now)
);

-- name: UpdateDoctor :exec
UPDATE `telemed_doctors`
SET kind = sqlc.arg(kind), name = sqlc.arg(name), headline = sqlc.narg(headline), bio = sqlc.narg(bio),
    specialty = sqlc.arg(specialty), city = sqlc.narg(city), licence_no = sqlc.arg(licence_no),
    experience_years = sqlc.narg(experience_years), response_minutes = sqlc.narg(response_minutes),
    is_active = sqlc.arg(is_active), sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetDoctorPhoto :exec
UPDATE `telemed_doctors` SET photo_path = sqlc.narg(photo_path), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteDoctor :execrows
DELETE FROM `telemed_doctors` WHERE id = sqlc.arg(id);

-- name: ListDoctorInsurers :many
SELECT insurer FROM `telemed_doctor_insurers` WHERE doctor_id = sqlc.arg(doctor_id) ORDER BY insurer;

-- name: DeleteDoctorInsurers :exec
DELETE FROM `telemed_doctor_insurers` WHERE doctor_id = sqlc.arg(doctor_id);

-- name: InsertDoctorInsurer :exec
INSERT IGNORE INTO `telemed_doctor_insurers` (doctor_id, insurer) VALUES (sqlc.arg(doctor_id), sqlc.arg(insurer));

-- name: ListDoctorVisitTypes :many
-- Every visit type of a doctor, inactive ones included (admin).
SELECT * FROM `telemed_visit_types` WHERE doctor_id = sqlc.arg(doctor_id)
ORDER BY FIELD(mode, 'video', 'phone', 'in_person'), id;

-- name: UpsertVisitType :exec
INSERT INTO `telemed_visit_types` (doctor_id, mode, duration_minutes, price_rials, note, address, is_active, created_at, updated_at)
VALUES (sqlc.arg(doctor_id), sqlc.arg(mode), sqlc.arg(duration_minutes), sqlc.arg(price_rials), sqlc.narg(note),
        sqlc.narg(address), sqlc.arg(is_active), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE duration_minutes = VALUES(duration_minutes), price_rials = VALUES(price_rials),
  note = VALUES(note), address = VALUES(address), is_active = VALUES(is_active), updated_at = VALUES(updated_at);

-- name: DeleteVisitType :exec
DELETE FROM `telemed_visit_types` WHERE doctor_id = sqlc.arg(doctor_id) AND mode = sqlc.arg(mode);

-- name: ListDoctorRules :many
SELECT * FROM `telemed_availability_rules` WHERE doctor_id = sqlc.arg(doctor_id) ORDER BY weekday, start_minute, id;

-- name: DeleteDoctorRules :exec
DELETE FROM `telemed_availability_rules` WHERE doctor_id = sqlc.arg(doctor_id);

-- name: InsertRule :exec
INSERT INTO `telemed_availability_rules` (doctor_id, weekday, start_minute, end_minute, slot_minutes, modes, created_at, updated_at)
VALUES (sqlc.arg(doctor_id), sqlc.arg(weekday), sqlc.arg(start_minute), sqlc.arg(end_minute), sqlc.arg(slot_minutes),
        sqlc.narg(modes), sqlc.arg(now), sqlc.arg(now));

-- name: ListDoctorTimeOff :many
-- A doctor's absences that end after since (the admin list hides past ones).
SELECT * FROM `telemed_time_off` WHERE doctor_id = sqlc.arg(doctor_id) AND ends_at > sqlc.arg(since)
ORDER BY starts_at, id;

-- name: CountDoctorTimeOff :one
SELECT COUNT(*) FROM `telemed_time_off` WHERE doctor_id = sqlc.arg(doctor_id) AND ends_at > sqlc.arg(since);

-- name: InsertTimeOff :execlastid
INSERT INTO `telemed_time_off` (doctor_id, starts_at, ends_at, note, created_at, updated_at)
VALUES (sqlc.arg(doctor_id), sqlc.arg(starts_at), sqlc.arg(ends_at), sqlc.narg(note), sqlc.arg(now), sqlc.arg(now));

-- name: DeleteTimeOff :execrows
DELETE FROM `telemed_time_off` WHERE id = sqlc.arg(id) AND doctor_id = sqlc.arg(doctor_id);

-- name: CountAdminReviews :one
SELECT COUNT(*) FROM `telemed_reviews`
WHERE (sqlc.arg(doctor_id) = 0 OR doctor_id = sqlc.arg(doctor_id))
  AND is_visible >= sqlc.arg(visible_min) AND is_visible <= sqlc.arg(visible_max);

-- name: ListAdminReviews :many
SELECT * FROM `telemed_reviews`
WHERE (sqlc.arg(doctor_id) = 0 OR doctor_id = sqlc.arg(doctor_id))
  AND is_visible >= sqlc.arg(visible_min) AND is_visible <= sqlc.arg(visible_max)
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: GetReview :one
SELECT * FROM `telemed_reviews` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: SetReviewVisibility :exec
UPDATE `telemed_reviews` SET is_visible = sqlc.arg(is_visible), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);
