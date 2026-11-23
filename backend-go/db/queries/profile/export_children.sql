-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportChildren :many
SELECT `id`, `owner_id`, `name`, `birth_date`, `sex`, `birth_weight_kg`, `birth_length_cm`, `birth_head_cm`, `delivery_type`, `created_at`, `updated_at`, `created_by`, `updated_by`
FROM `children`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportChildGuardians :many
SELECT `id`, `child_id`, `user_id`, `level`, `sub_scopes`, `status`, `invited_by`, `link_id`, `created_at`, `updated_at`
FROM `child_guardians`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED) OR child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportChildMeasurements :many
SELECT `id`, `child_id`, `measured_on`, `weight_kg`, `length_cm`, `head_cm`, `created_at`, `updated_at`, `created_by`, `updated_by`
FROM `child_measurements`
WHERE child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportChildMilestoneChecks :many
SELECT `id`, `child_id`, `milestone_code`, `checked_on`, `created_at`, `updated_at`, `created_by`, `updated_by`
FROM `child_milestone_checks`
WHERE child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportChildVaccineDoses :many
SELECT `id`, `child_id`, `dose_code`, `given_on`, `note`, `created_at`, `updated_at`, `created_by`, `updated_by`
FROM `child_vaccine_doses`
WHERE child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportBabyFeeds :many
SELECT `id`, `child_id`, `type`, `started_at`, `ended_at`, `active_side`, `side_started_at`, `last_side`, `left_seconds`, `right_seconds`, `duration_seconds`, `amount_ml`, `note`, `active_lock`, `created_at`, `updated_at`, `created_by`, `updated_by`
FROM `baby_feeds`
WHERE child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportBabySleeps :many
SELECT `id`, `child_id`, `started_at`, `ended_at`, `note`, `active_lock`, `created_at`, `updated_at`, `created_by`, `updated_by`
FROM `baby_sleeps`
WHERE child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportBabyDiapers :many
SELECT `id`, `child_id`, `changed_at`, `kind`, `note`, `created_at`, `updated_at`, `created_by`, `updated_by`
FROM `baby_diapers`
WHERE child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportBabyMedDoses :many
SELECT `id`, `child_id`, `name`, `dose_text`, `given_at`, `note`, `created_by`, `updated_by`, `created_at`, `updated_at`
FROM `baby_med_doses`
WHERE child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;
