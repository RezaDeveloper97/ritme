-- Ritme Plus plans (admin data; B-N2-09 owns the CRUD).

-- name: ListActivePlans :many
SELECT * FROM plus_plans
WHERE is_active = 1
ORDER BY sort_order, id;

-- name: GetActivePlan :one
SELECT * FROM plus_plans
WHERE id = ? AND is_active = 1;

-- name: GetPlan :one
SELECT * FROM plus_plans
WHERE id = ?;
