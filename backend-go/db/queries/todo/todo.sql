-- To-do list (bloom B-N6-08; internal/todo): tasks, list items, cycle-suggestion answers and the suggestion copy.
-- Every user-data query is scoped by user_id in the query itself (IDOR).

-- name: InsertTask :execlastid
INSERT INTO `todo_tasks` (
  user_id, title, note, category, due_date, due_time, remind, done_at, suggestion_key, created_at, updated_at
) VALUES (
  sqlc.arg(user_id), sqlc.arg(title), sqlc.narg(note), sqlc.arg(category), sqlc.narg(due_date), sqlc.narg(due_time),
  sqlc.arg(remind), NULL, sqlc.narg(suggestion_key), sqlc.arg(now), sqlc.arg(now)
);

-- name: GetTask :one
SELECT * FROM `todo_tasks` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: UpdateTask :execrows
UPDATE `todo_tasks`
SET title = sqlc.arg(title), note = sqlc.narg(note), category = sqlc.arg(category), due_date = sqlc.narg(due_date),
    due_time = sqlc.narg(due_time), remind = sqlc.arg(remind), done_at = sqlc.narg(done_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteTask :execrows
DELETE FROM `todo_tasks` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: CountTasks :one
SELECT COUNT(*) FROM `todo_tasks` WHERE user_id = sqlc.arg(user_id);

-- name: ListBoardTasks :many
-- The board of GET /todo: every open task plus the tasks ticked since done_since, with the item counts of each list.
SELECT t.*,
  (SELECT COUNT(*) FROM `todo_items` i WHERE i.task_id = t.id) AS items_total,
  (SELECT COUNT(*) FROM `todo_items` i WHERE i.task_id = t.id AND i.done_at IS NULL) AS items_open
FROM `todo_tasks` t
WHERE t.user_id = sqlc.arg(user_id) AND (t.done_at IS NULL OR t.done_at >= sqlc.arg(done_since))
ORDER BY t.id
LIMIT ?;

-- name: CountItems :one
SELECT COUNT(*) AS total, CAST(COALESCE(SUM(done_at IS NULL), 0) AS SIGNED) AS open_count
FROM `todo_items` WHERE task_id = sqlc.arg(task_id) AND user_id = sqlc.arg(user_id);

-- name: ListItems :many
SELECT * FROM `todo_items` WHERE task_id = sqlc.arg(task_id) AND user_id = sqlc.arg(user_id) ORDER BY sort_order, id;

-- name: MaxItemSort :one
SELECT CAST(COALESCE(MAX(sort_order), 0) AS SIGNED) FROM `todo_items`
WHERE task_id = sqlc.arg(task_id) AND user_id = sqlc.arg(user_id);

-- name: InsertItem :execlastid
INSERT INTO `todo_items` (task_id, user_id, title, due_date, done_at, sort_order, created_at, updated_at)
VALUES (sqlc.arg(task_id), sqlc.arg(user_id), sqlc.arg(title), sqlc.narg(due_date), NULL, sqlc.arg(sort_order),
        sqlc.arg(now), sqlc.arg(now));

-- name: GetItem :one
SELECT * FROM `todo_items`
WHERE id = sqlc.arg(id) AND task_id = sqlc.arg(task_id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: UpdateItem :execrows
UPDATE `todo_items`
SET title = sqlc.arg(title), due_date = sqlc.narg(due_date), done_at = sqlc.narg(done_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND task_id = sqlc.arg(task_id) AND user_id = sqlc.arg(user_id);

-- name: DeleteItem :execrows
DELETE FROM `todo_items` WHERE id = sqlc.arg(id) AND task_id = sqlc.arg(task_id) AND user_id = sqlc.arg(user_id);

-- name: TouchTask :exec
UPDATE `todo_tasks` SET updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: LatestOpenShoppingList :one
-- The newest open shopping task that already holds items: an accepted suggestion lands there as an item.
SELECT t.* FROM `todo_tasks` t
WHERE t.user_id = sqlc.arg(user_id) AND t.category = 'shopping' AND t.done_at IS NULL
  AND EXISTS (SELECT 1 FROM `todo_items` i WHERE i.task_id = t.id)
ORDER BY t.id DESC
LIMIT 1;

-- name: CountSuggestionEvents :one
SELECT COUNT(*) FROM `todo_suggestion_events`
WHERE user_id = sqlc.arg(user_id) AND suggestion_key = sqlc.arg(suggestion_key) AND ref_date = sqlc.arg(ref_date);

-- name: InsertSuggestionEvent :execrows
INSERT IGNORE INTO `todo_suggestion_events` (user_id, suggestion_key, ref_date, action, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(suggestion_key), sqlc.arg(ref_date), sqlc.arg(action), sqlc.arg(now), sqlc.arg(now));

-- name: ListLiveSuggestionCopy :many
-- The live (active + approved) todo_suggestion rows, all locales (admin-editable copy).
SELECT item_key, locale, payload FROM `message_contents`
WHERE `group` = 'todo_suggestion' AND is_active = 1 AND is_approved = 1;
