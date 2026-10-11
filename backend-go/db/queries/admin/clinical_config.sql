-- Versioned clinical thresholds & config with two medical approvals (bloom B-N9-11, migration 00063, D-88).

-- name: ListApprovedClinicalConfigVersions :many
-- Every approved version (the internal/clinicalconfig snapshot; v1 is the code default and never stored).
SELECT config_key, version, value, effective_from FROM `clinical_config_versions`
WHERE status = 'approved' ORDER BY config_key, effective_from, version;

-- name: ListClinicalConfigVersions :many
SELECT v.*, p.name AS proposer_name
FROM `clinical_config_versions` v
LEFT JOIN `admins` p ON p.id = v.proposed_by
ORDER BY v.config_key, v.version DESC;

-- name: ListClinicalConfigVersionsOfKey :many
SELECT v.*, p.name AS proposer_name
FROM `clinical_config_versions` v
LEFT JOIN `admins` p ON p.id = v.proposed_by
WHERE v.config_key = ? ORDER BY v.version DESC;

-- name: GetClinicalConfigVersion :one
SELECT * FROM `clinical_config_versions` WHERE id = ? LIMIT 1;

-- name: GetClinicalConfigVersionForUpdate :one
SELECT * FROM `clinical_config_versions` WHERE id = ? LIMIT 1 FOR UPDATE;

-- name: MaxClinicalConfigVersion :one
SELECT CAST(COALESCE(MAX(version), 1) AS SIGNED) AS version FROM `clinical_config_versions` WHERE config_key = ?;

-- name: CountPendingClinicalConfig :one
SELECT COUNT(*) FROM `clinical_config_versions` WHERE config_key = ? AND status = 'pending';

-- name: InsertClinicalConfigVersion :execlastid
INSERT INTO `clinical_config_versions` (config_key, version, value, source, note, effective_from, status, proposed_by,
                                        created_at, updated_at)
VALUES (sqlc.arg(config_key), sqlc.arg(version), sqlc.arg(value), sqlc.arg(source), sqlc.arg(note),
        sqlc.arg(effective_from), 'pending', sqlc.arg(proposed_by), sqlc.arg(now), sqlc.arg(now));

-- name: DecideClinicalConfigVersion :execrows
UPDATE `clinical_config_versions`
SET status = sqlc.arg(status), decided_by = sqlc.arg(decided_by), decided_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: InsertClinicalConfigApproval :exec
INSERT INTO `clinical_config_approvals` (version_id, admin_id, created_at)
VALUES (sqlc.arg(version_id), sqlc.arg(admin_id), sqlc.arg(now));

-- name: ListClinicalConfigApprovals :many
SELECT ap.version_id, ap.admin_id, ap.created_at, a.name AS admin_name
FROM `clinical_config_approvals` ap
LEFT JOIN `admins` a ON a.id = ap.admin_id
ORDER BY ap.version_id, ap.id;
