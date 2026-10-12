-- Account data export (GET /profile/export, user decision 2026-10-12): every insurance row of the owner; child rows
-- are reached through her policy / claim (the owner scope is in each statement). Sealed text is opened by the caller.

-- name: ExportInsurancePolicies :many
SELECT * FROM `insurance_policies` WHERE user_id = ? ORDER BY valid_from, id;

-- name: ExportInsurancePolicyMembers :many
SELECT m.* FROM `insurance_policy_members` m
JOIN `insurance_policies` p ON p.id = m.policy_id
WHERE p.user_id = ? ORDER BY m.policy_id, m.sort_order, m.id;

-- name: ExportInsuranceCoverages :many
SELECT cv.* FROM `insurance_coverages` cv
JOIN `insurance_policies` p ON p.id = cv.policy_id
WHERE p.user_id = ? ORDER BY cv.policy_id, cv.sort_order, cv.id;

-- name: ExportInsuranceClaims :many
SELECT * FROM `insurance_claims` WHERE user_id = ? ORDER BY service_date, id;

-- name: ExportInsuranceClaimEvents :many
SELECT e.* FROM `insurance_claim_events` e
JOIN `insurance_claims` c ON c.id = e.claim_id
WHERE c.user_id = ? ORDER BY e.claim_id, e.occurred_on, e.id;

-- name: ExportInsuranceClaimDocuments :many
SELECT d.* FROM `insurance_claim_documents` d
JOIN `insurance_claims` c ON c.id = d.claim_id
WHERE c.user_id = ? ORDER BY d.claim_id, d.id;

-- name: ExportInsuranceQuestionnaires :many
SELECT * FROM `insurance_questionnaires` WHERE user_id = ? ORDER BY id;
