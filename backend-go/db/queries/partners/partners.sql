-- Home lab / screening booking requests (bloom B-N10-04, D-94): the service catalog, the users' requests and the
-- retention sweep. Personal fields live only in the sealed `payload_enc` blob; no query ever selects a lab result.

-- name: ListActiveServices :many
SELECT * FROM `partner_lab_services` WHERE is_active = 1 ORDER BY sort_order, id;

-- name: ListServices :many
SELECT * FROM `partner_lab_services` ORDER BY sort_order, id;

-- name: GetService :one
SELECT * FROM `partner_lab_services` WHERE id = ? LIMIT 1;

-- name: ServiceCodeExists :one
SELECT EXISTS(SELECT 1 FROM `partner_lab_services` WHERE `code` = sqlc.arg(code) AND id <> sqlc.arg(except_id)) AS found;

-- name: CreateService :execlastid
INSERT INTO `partner_lab_services` (`code`, `kind`, `title`, `description`, `partner_name`, `partner_code`, `is_active`,
  `sort_order`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateService :exec
UPDATE `partner_lab_services` SET `code` = ?, `kind` = ?, `title` = ?, `description` = ?, `partner_name` = ?,
  `partner_code` = ?, `is_active` = ?, `sort_order` = ?, `updated_at` = ?
WHERE id = ?;

-- name: CreateRequest :execlastid
INSERT INTO `partner_lab_requests` (`reference`, `user_id`, `service_id`, `status`, `preferred_date`, `preferred_window`,
  `consent_share`, `consent_import`, `payload_enc`, `created_at`, `updated_at`)
VALUES (?, ?, ?, 'requested', ?, ?, ?, ?, ?, ?, ?);

-- name: ListUserRequests :many
SELECT r.*, s.title AS service_title, s.kind AS service_kind, s.partner_name AS partner_name
FROM `partner_lab_requests` r JOIN `partner_lab_services` s ON s.id = r.service_id
WHERE r.user_id = ? ORDER BY r.id DESC LIMIT 50;

-- name: GetUserRequest :one
SELECT r.*, s.title AS service_title, s.kind AS service_kind, s.partner_name AS partner_name
FROM `partner_lab_requests` r JOIN `partner_lab_services` s ON s.id = r.service_id
WHERE r.id = ? AND r.user_id = ? LIMIT 1;

-- name: GetRequest :one
SELECT * FROM `partner_lab_requests` WHERE id = ? LIMIT 1;

-- name: CountOpenUserRequests :one
SELECT COUNT(*) FROM `partner_lab_requests`
WHERE user_id = ? AND status IN ('requested', 'accepted', 'sampled');

-- name: MoveRequestStatus :execrows
-- A conditional transition: 0 rows = the request is no longer in `from` (a concurrent cancel / admin step).
UPDATE `partner_lab_requests` SET `status` = sqlc.arg(to_status), `closed_at` = sqlc.arg(closed_at), `updated_at` = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND `status` = sqlc.arg(from_status);

-- name: SetPartnerRef :exec
UPDATE `partner_lab_requests` SET `partner_ref` = ?, `updated_at` = ? WHERE id = ?;

-- name: SetPartnerStatus :exec
UPDATE `partner_lab_requests` SET `partner_status` = ?, `updated_at` = ? WHERE id = ?;

-- name: ListOpenPartnerRequests :many
-- Requests the partner still has to move forward (the sync loop asks the adapter for their status).
SELECT * FROM `partner_lab_requests`
WHERE status IN ('requested', 'accepted', 'sampled') AND partner_ref IS NOT NULL ORDER BY id LIMIT 200;

-- name: ClearRequestImported :exec
UPDATE `partner_lab_requests` SET `imported_lab_id` = NULL WHERE id = ? AND user_id = ?;

-- name: PurgeRequestPayloads :execrows
-- Retention: the sealed personal blob goes 30 days after the request closed.
UPDATE `partner_lab_requests` SET `payload_enc` = NULL, `purged_at` = sqlc.arg(now)
WHERE `purged_at` IS NULL AND `closed_at` IS NOT NULL AND `closed_at` <= sqlc.arg(cutoff);

-- name: CountAdminRequests :one
SELECT COUNT(*) FROM `partner_lab_requests` r
WHERE (sqlc.arg(status) = '' OR r.status = sqlc.arg(status))
  AND (sqlc.arg(reference) = '' OR r.reference = sqlc.arg(reference));

-- name: ListAdminRequests :many
SELECT r.id, r.reference, r.user_id, r.status, r.preferred_date, r.preferred_window, r.consent_import,
  r.partner_ref, r.imported_lab_id, r.purged_at, r.created_at, r.closed_at,
  s.code AS service_code, s.partner_code AS partner_code, s.title AS service_title, s.partner_name AS partner_name, u.mobile AS user_mobile
FROM `partner_lab_requests` r
JOIN `partner_lab_services` s ON s.id = r.service_id
LEFT JOIN `users` u ON u.id = r.user_id
WHERE (sqlc.arg(status) = '' OR r.status = sqlc.arg(status))
  AND (sqlc.arg(reference) = '' OR r.reference = sqlc.arg(reference))
ORDER BY r.id DESC LIMIT ? OFFSET ?;

-- name: DeleteUnsubmittedRequest :exec
-- Compensation when the partner refused the submission: the row never reached it.
DELETE FROM `partner_lab_requests` WHERE id = ? AND user_id = ? AND partner_ref IS NULL;

-- name: ClaimRequestImport :execrows
-- Claims the one-time import (imported_lab_id 0 = in progress) so two taps never copy the result twice.
UPDATE `partner_lab_requests` SET `imported_lab_id` = 0, `updated_at` = ?
WHERE id = ? AND user_id = ? AND imported_lab_id IS NULL AND status = 'result_ready' AND consent_import = 1;

-- name: SetRequestImportedLab :exec
UPDATE `partner_lab_requests` SET `imported_lab_id` = ?, `updated_at` = ? WHERE id = ? AND user_id = ? AND imported_lab_id = 0;

-- name: GetPartnerStatus :one
SELECT partner_status FROM `partner_lab_requests` WHERE partner_ref = ? LIMIT 1;
