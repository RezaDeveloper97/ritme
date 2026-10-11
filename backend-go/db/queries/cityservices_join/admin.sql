-- CB-DIR-05 — the Ritme team's review queue of business applications (admin API /api/admin/v1/city-service-applications,
-- capability content). NOT owner-scoped on purpose: these run only behind the admin session + capability chain. Drafts
-- are never shown to the team (they are the applicant's private work in progress).

-- name: AdminCountApplications :one
SELECT COUNT(*) FROM `city_service_applications`
WHERE status <> 'draft'
  AND (sqlc.arg(status) = '' OR status = sqlc.arg(status))
  AND (sqlc.arg(q) = '' OR business_name LIKE CONCAT('%', sqlc.arg(q), '%') OR tracking_code = sqlc.arg(q));

-- name: AdminListApplications :many
-- Open work first (submitted, in_review, needs_info), oldest submission first; decided ones after, newest first.
SELECT * FROM `city_service_applications`
WHERE status <> 'draft'
  AND (sqlc.arg(status) = '' OR status = sqlc.arg(status))
  AND (sqlc.arg(q) = '' OR business_name LIKE CONCAT('%', sqlc.arg(q), '%') OR tracking_code = sqlc.arg(q))
ORDER BY (status IN ('submitted', 'in_review', 'needs_info')) DESC,
  CASE WHEN status IN ('submitted', 'in_review', 'needs_info') THEN submitted_at END ASC,
  submitted_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetApplication :one
SELECT * FROM `city_service_applications` WHERE id = ? AND status <> 'draft' LIMIT 1;

-- name: AdminLockApplication :one
SELECT * FROM `city_service_applications` WHERE id = ? AND status <> 'draft' LIMIT 1 FOR UPDATE;

-- name: AdminReviewApplication :execrows
-- Compare-and-set on the status read under the lock (a concurrent resubmission or second reviewer loses).
UPDATE `city_service_applications` SET `status` = ?, `review_note` = ?, `reviewed_at` = ?, `updated_at` = ?
WHERE id = ? AND status = ?;

-- name: AdminApproveApplication :execrows
UPDATE `city_service_applications` SET `status` = 'approved', `listing_id` = ?, `review_note` = ?, `reviewed_at` = ?,
  `updated_at` = ?
WHERE id = ? AND status = ?;

-- name: AdminClearApplicationPhotos :exec
UPDATE `city_service_applications` SET `photo_file_ids` = NULL, `updated_at` = ? WHERE id = ?;
