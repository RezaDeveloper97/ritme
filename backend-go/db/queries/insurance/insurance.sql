-- Insurance self-tracking (canvas CB-INS-01): catalog (insurers, centres), the user's policies (history, 00093) + members + coverages,
-- claims with their timeline and record-document links, questionnaire drafts. Every user query is scoped by user_id
-- in the SQL itself (IDOR): another user's row behaves exactly like a missing one.

-- ---------------------------------------------------------------------------------------------------------------
-- Catalog

-- name: ListActiveInsurers :many
SELECT * FROM `insurance_insurers` WHERE is_active = 1 ORDER BY sort_order, id;

-- name: ListInsurers :many
SELECT * FROM `insurance_insurers` ORDER BY sort_order, id;

-- name: GetInsurer :one
SELECT * FROM `insurance_insurers` WHERE id = ? LIMIT 1;

-- name: InsurerCodeExists :one
SELECT EXISTS(SELECT 1 FROM `insurance_insurers` WHERE `code` = sqlc.arg(code) AND id <> sqlc.arg(except_id)) AS found;

-- name: CreateInsurer :execlastid
INSERT INTO `insurance_insurers` (`code`, `name`, `website`, `phone`, `is_active`, `sort_order`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateInsurer :exec
UPDATE `insurance_insurers` SET `code` = ?, `name` = ?, `website` = ?, `phone` = ?, `is_active` = ?, `sort_order` = ?,
  `updated_at` = ?
WHERE id = ?;

-- name: DeleteInsurer :execrows
DELETE FROM `insurance_insurers` WHERE id = ?;

-- name: ListActiveCentres :many
-- The contracted centres a user sees: active, of the given type ('' = all), matching q ('' = all), for her insurer
-- (insurer_id NULL rows are for every insurer; insurer 0 = she has none, so only those).
SELECT c.*, i.name AS insurer_name FROM `insurance_centres` c
LEFT JOIN `insurance_insurers` i ON i.id = c.insurer_id
WHERE c.is_active = 1
  AND (sqlc.arg(type) = '' OR c.type = sqlc.arg(type))
  AND (c.insurer_id IS NULL OR c.insurer_id = sqlc.arg(insurer_id))
  AND (sqlc.arg(q) = '' OR c.name LIKE CONCAT('%', sqlc.arg(q), '%') OR c.services LIKE CONCAT('%', sqlc.arg(q), '%'))
ORDER BY c.sort_order, c.id LIMIT 200;

-- name: ListCentres :many
SELECT c.*, i.name AS insurer_name FROM `insurance_centres` c
LEFT JOIN `insurance_insurers` i ON i.id = c.insurer_id
ORDER BY c.sort_order, c.id;

-- name: GetCentre :one
SELECT c.*, i.name AS insurer_name FROM `insurance_centres` c
LEFT JOIN `insurance_insurers` i ON i.id = c.insurer_id
WHERE c.id = ? LIMIT 1;

-- name: GetActiveCentre :one
SELECT * FROM `insurance_centres` WHERE id = ? AND is_active = 1 LIMIT 1;

-- name: CreateCentre :execlastid
INSERT INTO `insurance_centres` (`insurer_id`, `name`, `type`, `services`, `city`, `address`, `phone`, `lat`, `lng`,
  `is_direct`, `is_active`, `sort_order`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateCentre :exec
UPDATE `insurance_centres` SET `insurer_id` = ?, `name` = ?, `type` = ?, `services` = ?, `city` = ?, `address` = ?,
  `phone` = ?, `lat` = ?, `lng` = ?, `is_direct` = ?, `is_active` = ?, `sort_order` = ?, `updated_at` = ?
WHERE id = ?;

-- name: DeleteCentre :execrows
DELETE FROM `insurance_centres` WHERE id = ?;

-- ---------------------------------------------------------------------------------------------------------------
-- Policy

-- name: ListUserPolicies :many
-- Her policy history (00093: several per user), newest period first. «Current» is chosen in Go (CurrentPolicy).
SELECT p.*, i.name AS catalog_insurer_name FROM `insurance_policies` p
LEFT JOIN `insurance_insurers` i ON i.id = p.insurer_id
WHERE p.user_id = ? ORDER BY p.valid_from DESC, p.id DESC LIMIT 100;

-- name: GetUserPolicyByID :one
SELECT p.*, i.name AS catalog_insurer_name FROM `insurance_policies` p
LEFT JOIN `insurance_insurers` i ON i.id = p.insurer_id
WHERE p.id = ? AND p.user_id = ? LIMIT 1;

-- name: UserHasPolicy :one
SELECT EXISTS(SELECT 1 FROM `insurance_policies` WHERE user_id = ?) AS found;

-- name: CreatePolicy :execlastid
INSERT INTO `insurance_policies` (`user_id`, `insurer_id`, `insurer_name`, `plan_name`, `number_last4`, `valid_from`,
  `valid_to`, `claim_deadline_days`, `iban_last4`, `not_covered_note`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePolicy :exec
UPDATE `insurance_policies` SET `insurer_id` = ?, `insurer_name` = ?, `plan_name` = ?, `number_last4` = ?,
  `valid_from` = ?, `valid_to` = ?, `claim_deadline_days` = ?, `iban_last4` = ?, `not_covered_note` = ?, `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteUserPolicy :execrows
DELETE FROM `insurance_policies` WHERE id = ? AND user_id = ?;

-- name: ListPolicyMembers :many
SELECT * FROM `insurance_policy_members` WHERE policy_id = ? ORDER BY sort_order, id;

-- name: CreateMember :execlastid
INSERT INTO `insurance_policy_members` (`policy_id`, `kind`, `child_id`, `name`, `sort_order`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateMember :exec
-- Child-table mutations are scoped to the owner through the parent row (defence in depth on top of the service checks).
UPDATE `insurance_policy_members` SET `kind` = sqlc.arg(kind), `child_id` = sqlc.arg(child_id), `name` = sqlc.arg(name),
  `sort_order` = sqlc.arg(sort_order), `updated_at` = sqlc.arg(updated_at)
WHERE `insurance_policy_members`.id = sqlc.arg(id) AND `insurance_policy_members`.policy_id = sqlc.arg(policy_id) AND `insurance_policy_members`.`policy_id` IN (SELECT p.id FROM `insurance_policies` p WHERE p.user_id = sqlc.arg(user_id));

-- name: DeleteMember :exec
DELETE FROM `insurance_policy_members`
WHERE `insurance_policy_members`.id = sqlc.arg(id) AND `insurance_policy_members`.policy_id = sqlc.arg(policy_id) AND `insurance_policy_members`.`policy_id` IN (SELECT p.id FROM `insurance_policies` p WHERE p.user_id = sqlc.arg(user_id));

-- name: UserOwnsChild :one
SELECT EXISTS(SELECT 1 FROM `children` WHERE id = sqlc.arg(child_id) AND owner_id = sqlc.arg(user_id)) AS found;

-- name: ListUserChildren :many
SELECT id, name FROM `children` WHERE owner_id = ? ORDER BY birth_date, id;

-- name: ListPolicyCoverages :many
SELECT * FROM `insurance_coverages` WHERE policy_id = ? ORDER BY sort_order, id;

-- name: DeletePolicyCoverages :exec
DELETE FROM `insurance_coverages` WHERE policy_id = sqlc.arg(policy_id) AND `policy_id` IN (SELECT p.id FROM `insurance_policies` p WHERE p.user_id = sqlc.arg(user_id));

-- name: CreateCoverage :exec
INSERT INTO `insurance_coverages` (`policy_id`, `category`, `status`, `cap_amount`, `deductible_pct`, `waiting_days`,
  `note`, `sort_order`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- ---------------------------------------------------------------------------------------------------------------
-- Claims

-- name: ListUserClaims :many
SELECT * FROM `insurance_claims` WHERE user_id = ? ORDER BY service_date DESC, id DESC LIMIT 500;

-- name: ListPolicyClaims :many
-- The claims of one of her policies (coverage usage is per policy period).
SELECT * FROM `insurance_claims` WHERE policy_id = ? AND user_id = ? ORDER BY service_date DESC, id DESC LIMIT 500;

-- name: GetUserClaim :one
SELECT * FROM `insurance_claims` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: CreateClaim :execlastid
INSERT INTO `insurance_claims` (`user_id`, `policy_id`, `member_id`, `cost_type`, `title`, `centre_id`, `centre_name`,
  `service_date`, `amount`, `over_tariff`, `status`, `notes_enc`, `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'submitted', ?, ?, ?);

-- name: UpdateClaim :exec
UPDATE `insurance_claims` SET `member_id` = ?, `cost_type` = ?, `title` = ?, `centre_id` = ?, `centre_name` = ?,
  `service_date` = ?, `amount` = ?, `over_tariff` = ?, `notes_enc` = ?, `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: SetClaimState :exec
-- The status, paid amount and deadline mirror the latest event of the timeline.
UPDATE `insurance_claims` SET `status` = ?, `paid_amount` = ?, `deadline_on` = ?, `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteUserClaim :execrows
DELETE FROM `insurance_claims` WHERE id = ? AND user_id = ?;

-- name: ListClaimEvents :many
SELECT * FROM `insurance_claim_events` WHERE claim_id = ? ORDER BY occurred_on, id;

-- name: CreateClaimEvent :execlastid
INSERT INTO `insurance_claim_events` (`claim_id`, `status`, `occurred_on`, `deadline_on`, `amount`, `note_enc`,
  `created_at`, `updated_at`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteClaimEvent :execrows
DELETE FROM `insurance_claim_events` WHERE `insurance_claim_events`.id = sqlc.arg(id) AND `insurance_claim_events`.claim_id = sqlc.arg(claim_id) AND `insurance_claim_events`.`claim_id` IN (SELECT c.id FROM `insurance_claims` c WHERE c.user_id = sqlc.arg(user_id));

-- name: ListClaimDocuments :many
-- Only documents that still belong to the claim's owner (record_documents.user_id) are listed.
SELECT cd.id, cd.document_id, cd.label, d.kind AS document_kind, d.title AS document_title, d.document_date
FROM `insurance_claim_documents` cd
JOIN `record_documents` d ON d.id = cd.document_id AND d.user_id = sqlc.arg(user_id)
WHERE cd.claim_id = sqlc.arg(claim_id)
ORDER BY cd.id;

-- name: UserOwnsDocument :one
SELECT EXISTS(SELECT 1 FROM `record_documents` WHERE id = sqlc.arg(document_id) AND user_id = sqlc.arg(user_id)) AS found;

-- name: AttachClaimDocument :exec
-- INSERT … SELECT: nothing is linked unless the claim is hers.
INSERT INTO `insurance_claim_documents` (`claim_id`, `document_id`, `label`, `created_at`, `updated_at`)
SELECT c.id, sqlc.arg(document_id), sqlc.arg(label), sqlc.arg(created_at), sqlc.arg(updated_at)
FROM `insurance_claims` c WHERE c.id = sqlc.arg(claim_id) AND c.user_id = sqlc.arg(user_id)
ON DUPLICATE KEY UPDATE `label` = VALUES(`label`), `updated_at` = VALUES(`updated_at`);

-- name: DetachClaimDocument :execrows
DELETE FROM `insurance_claim_documents`
WHERE claim_id = sqlc.arg(claim_id) AND document_id = sqlc.arg(document_id) AND `claim_id` IN (SELECT c.id FROM `insurance_claims` c WHERE c.user_id = sqlc.arg(user_id));

-- name: DeleteClaimDocuments :exec
DELETE FROM `insurance_claim_documents` WHERE claim_id = sqlc.arg(claim_id) AND `claim_id` IN (SELECT c.id FROM `insurance_claims` c WHERE c.user_id = sqlc.arg(user_id));

-- ---------------------------------------------------------------------------------------------------------------
-- Questionnaires

-- name: ListUserQuestionnaires :many
SELECT * FROM `insurance_questionnaires` WHERE user_id = ? ORDER BY id DESC LIMIT 50;

-- name: GetUserQuestionnaire :one
SELECT * FROM `insurance_questionnaires` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: CountUserQuestionnaires :one
SELECT COUNT(*) FROM `insurance_questionnaires` WHERE user_id = ?;

-- name: CreateQuestionnaire :execlastid
-- `answers` carries no evidence: the record items behind the suggestions are sealed in `evidence_enc` (00093).
INSERT INTO `insurance_questionnaires` (`user_id`, `purpose`, `label`, `status`, `answers`, `evidence_enc`, `created_at`,
  `updated_at`)
VALUES (?, ?, ?, 'draft', ?, ?, ?, ?);

-- name: UpdateQuestionnaire :exec
UPDATE `insurance_questionnaires` SET `answers` = ?, `evidence_enc` = ?, `explanation_enc` = ?, `label` = ?, `status` = ?,
  `confirmed_at` = ?, `timeline` = ?, `updated_at` = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteUserQuestionnaire :execrows
DELETE FROM `insurance_questionnaires` WHERE id = ? AND user_id = ?;

-- ---------------------------------------------------------------------------------------------------------------
-- Record reads for the questionnaire prefill (read-only, owner-scoped)

-- name: PrefillLifeProfile :one
SELECT life_mode, ivf_iui, chronic_illnesses, medications, menopause_surgical FROM `user_life_profiles`
WHERE user_id = ? LIMIT 1;

-- name: PrefillHealthRecord :one
SELECT surgeries, family_history FROM `health_records` WHERE user_id = ? LIMIT 1;

-- name: PrefillPregnancy :one
SELECT pregnancy_mode FROM `pregnancy_profiles` WHERE user_id = ? LIMIT 1;

-- name: PrefillCesareans :many
SELECT ended_on FROM `health_record_pregnancies` WHERE user_id = ? AND outcome = 'cesarean' ORDER BY ended_on;

-- name: PrefillHospitalDocuments :many
SELECT title, document_date FROM `record_documents`
WHERE user_id = sqlc.arg(user_id) AND kind = 'hospital' AND (document_date IS NULL OR document_date >= sqlc.arg(since))
ORDER BY document_date;

-- name: PrefillOpenIVF :one
SELECT COUNT(*) FROM `ivf_cycles` WHERE user_id = ? AND closed_at IS NULL;
