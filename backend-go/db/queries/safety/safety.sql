-- Safety & clinical content dashboard (bloom B-N9-10, D-90). Aggregates and review metadata only: counts, titles, ids,
-- ages — never a user id or a health value.

-- name: CountBPCrisisReadings :one
SELECT COUNT(*) FROM `vital_readings`
WHERE `type` = 'bp' AND measured_at >= sqlc.arg(from_at) AND measured_at < sqlc.arg(to_at)
  AND (systolic > sqlc.arg(systolic) OR diastolic > sqlc.arg(diastolic));

-- name: CountGlucoseUrgentLowReadings :one
SELECT COUNT(*) FROM `vital_readings`
WHERE `type` = 'glucose' AND measured_at >= sqlc.arg(from_at) AND measured_at < sqlc.arg(to_at)
  AND glucose_mg_dl < sqlc.arg(below);

-- name: CountEPDSSelfHarmChecks :one
SELECT COUNT(*) FROM `epds_checks`
WHERE taken_on >= sqlc.arg(from_day) AND taken_on < sqlc.arg(to_day) AND self_harm > 0;

-- name: CountReducedKickSessions :one
SELECT COUNT(*) FROM `pregnancy_kick_sessions`
WHERE ended_at >= sqlc.arg(from_at) AND ended_at < sqlc.arg(to_at)
  AND kicks < sqlc.arg(kick_target)
  AND TIMESTAMPDIFF(MINUTE, started_at, ended_at) >= CAST(sqlc.arg(window_minutes) AS SIGNED);

-- name: CountContraction511Alerts :one
SELECT COUNT(*) FROM `pregnancy_contraction_sessions`
WHERE alert_at >= sqlc.arg(from_at) AND alert_at < sqlc.arg(to_at);

-- name: CountHeavyLochiaDays :one
SELECT COUNT(*) FROM `health_log_entries`
WHERE category = 'bleeding' AND param = 'lochia_amount' AND value_code = 'heavy'
  AND log_date >= sqlc.arg(from_day) AND log_date < sqlc.arg(to_day);

-- name: ListSafetyPendingMessageRules :many
-- Smart messages v2 awaiting the medical approval (B-N9-05).
SELECT r.id, r.title, r.level, r.updated_at, r.created_at, a.name AS submitted_by
FROM `message_rules` r
LEFT JOIN `admins` a ON a.id = r.created_by
WHERE r.status = 'pending_approval'
ORDER BY r.updated_at, r.id
LIMIT 200;

-- name: ListSafetyPendingClinicalConfig :many
-- Clinical threshold proposals waiting for their two medical approvals (B-N9-11).
SELECT v.id, v.config_key, v.version, v.created_at, a.name AS submitted_by,
  (SELECT COUNT(*) FROM `clinical_config_approvals` ap WHERE ap.version_id = v.id) AS approvals
FROM `clinical_config_versions` v
LEFT JOIN `admins` a ON a.id = v.proposed_by
WHERE v.status = 'pending'
ORDER BY v.created_at, v.id
LIMIT 200;

-- name: ListSafetyLessonsToReview :many
-- Published lessons never reviewed or changed since their approval (the B-N8-08 queue's open states).
SELECT l.id, l.title, l.review_status, l.updated_at, l.created_at, i.display_name AS submitted_by,
  CAST(l.review_status = 'approved' AS SIGNED) AS changed
FROM `learning_lessons` l
JOIN `learning_courses` c ON c.id = l.course_id
JOIN `learning_instructors` i ON i.id = c.instructor_id
WHERE l.status = 'published'
  AND (l.review_status = 'pending'
    OR (l.review_status = 'approved' AND (l.reviewed_at IS NULL OR l.updated_at > l.reviewed_at)))
ORDER BY l.updated_at, l.id
LIMIT 200;

-- name: ListSafetyAssistantDepartments :many
-- Assistant departments still flagged «needs clinical review» (B-N7-08).
SELECT id, code, name, updated_at, created_at
FROM `assistant_departments`
WHERE needs_review = 1
ORDER BY id;

-- name: ListSafetyCatalogGroups :many
-- Catalog groups with rows flagged «needs clinical review», one row per group (its first row's id is the claim key).
SELECT `group`, CAST(MIN(id) AS UNSIGNED) AS first_id, COUNT(*) AS items,
  CAST(MIN(COALESCE(updated_at, created_at)) AS DATETIME) AS oldest_at
FROM `catalog_items`
WHERE needs_review = 1
GROUP BY `group`
ORDER BY `group`;

-- name: CountUnapprovedUrgentCopy :one
-- Live urgent-message copy rows that are not the approved version (the «approved-text parity» check).
SELECT COUNT(*) FROM `message_contents`
WHERE `group` IN (sqlc.slice(copy_groups)) AND is_active = 1 AND is_approved = 0;

-- name: ListSafetyClaims :many
-- Claim / release actions of the review queue (stored in the admin audit log, newest last).
SELECT l.target_type, l.target_id, l.action, l.admin_id, l.created_at, a.name AS admin_name
FROM `admin_audit_logs` l
LEFT JOIN `admins` a ON a.id = l.admin_id
WHERE l.action IN ('safety.claim', 'safety.release') AND l.created_at >= sqlc.arg(since)
ORDER BY l.id;
