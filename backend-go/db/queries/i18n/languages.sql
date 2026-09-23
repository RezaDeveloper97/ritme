-- LanguageRegistry::load(): Language::query()->active()->ordered()->get().

-- name: ListActiveLanguages :many
SELECT code, name, english_name, direction, is_default
FROM languages
WHERE is_active = 1
ORDER BY sort_order, id;
