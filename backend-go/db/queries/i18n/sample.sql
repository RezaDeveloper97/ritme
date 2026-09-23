-- Placeholder so the `i18n` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/i18n/.

-- name: SampleI18nLanguage :one
SELECT * FROM `languages` WHERE id = ? LIMIT 1;
