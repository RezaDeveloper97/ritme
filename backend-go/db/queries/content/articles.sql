-- articles (App\Models\Article), read side of ArticleController.
--
-- The list filters mirror ArticleController::listQuery(): the category filter applies
-- only when has_category, the search only when has_term. The search runs over
-- `title->lang` / `excerpt->lang` for the request locale and fa, compiled the way
-- Laravel's MySQL grammar does: json_unquote(json_extract(col, '$."<lang>"')) like ?.

-- name: CountPublishedArticles :one
SELECT COUNT(*) FROM `articles`
WHERE is_published = 1
  AND (CAST(sqlc.arg(has_category) AS SIGNED) = 0 OR category = sqlc.arg(category))
  AND (CAST(sqlc.arg(has_term) AS SIGNED) = 0 OR (
        JSON_UNQUOTE(JSON_EXTRACT(title, CONCAT('$."', CAST(sqlc.arg(lang_a) AS CHAR), '"'))) LIKE CAST(sqlc.arg(pattern) AS CHAR)
     OR JSON_UNQUOTE(JSON_EXTRACT(excerpt, CONCAT('$."', CAST(sqlc.arg(lang_a) AS CHAR), '"'))) LIKE CAST(sqlc.arg(pattern) AS CHAR)
     OR JSON_UNQUOTE(JSON_EXTRACT(title, CONCAT('$."', CAST(sqlc.arg(lang_b) AS CHAR), '"'))) LIKE CAST(sqlc.arg(pattern) AS CHAR)
     OR JSON_UNQUOTE(JSON_EXTRACT(excerpt, CONCAT('$."', CAST(sqlc.arg(lang_b) AS CHAR), '"'))) LIKE CAST(sqlc.arg(pattern) AS CHAR)));

-- name: ListPublishedArticles :many
-- listQuery()->paginate(): sort_order, published_at DESC, id DESC.
SELECT id, slug, title, excerpt, cycle_phases, category, read_time_minutes, image_url, image_path, published_at
FROM `articles`
WHERE is_published = 1
  AND (CAST(sqlc.arg(has_category) AS SIGNED) = 0 OR category = sqlc.arg(category))
  AND (CAST(sqlc.arg(has_term) AS SIGNED) = 0 OR (
        JSON_UNQUOTE(JSON_EXTRACT(title, CONCAT('$."', CAST(sqlc.arg(lang_a) AS CHAR), '"'))) LIKE CAST(sqlc.arg(pattern) AS CHAR)
     OR JSON_UNQUOTE(JSON_EXTRACT(excerpt, CONCAT('$."', CAST(sqlc.arg(lang_a) AS CHAR), '"'))) LIKE CAST(sqlc.arg(pattern) AS CHAR)
     OR JSON_UNQUOTE(JSON_EXTRACT(title, CONCAT('$."', CAST(sqlc.arg(lang_b) AS CHAR), '"'))) LIKE CAST(sqlc.arg(pattern) AS CHAR)
     OR JSON_UNQUOTE(JSON_EXTRACT(excerpt, CONCAT('$."', CAST(sqlc.arg(lang_b) AS CHAR), '"'))) LIKE CAST(sqlc.arg(pattern) AS CHAR)))
ORDER BY sort_order, published_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: ListPublishedArticleCategories :many
-- ArticleController::categories(): distinct non-empty categories of published articles.
SELECT DISTINCT category FROM `articles`
WHERE is_published = 1 AND category IS NOT NULL AND category != ''
ORDER BY category;

-- name: GetPublishedArticleBySlug :one
SELECT id, slug, title, excerpt, body, cycle_phases, category, read_time_minutes, image_url, image_path, published_at
FROM `articles`
WHERE is_published = 1 AND slug = ?
LIMIT 1;

-- name: ListRelatedArticles :many
-- ArticleController::related(): published, not the article itself, and (same category
-- OR tagged with any of its phases — json_contains per phase, i.e. JSON_OVERLAPS with the
-- phase list); with neither to match on, any article. Newest first, at most 4.
SELECT id, slug, title, excerpt, cycle_phases, category, read_time_minutes, image_url, image_path, published_at
FROM `articles`
WHERE is_published = 1
  AND id <> sqlc.arg(id)
  AND ((CAST(sqlc.arg(has_category) AS SIGNED) = 1 AND category = sqlc.arg(category))
    OR (CAST(sqlc.arg(has_phases) AS SIGNED) = 1 AND JSON_OVERLAPS(cycle_phases, sqlc.arg(phases)))
    OR (CAST(sqlc.arg(has_category) AS SIGNED) = 0 AND CAST(sqlc.arg(has_phases) AS SIGNED) = 0))
ORDER BY published_at DESC, id DESC
LIMIT 4;
