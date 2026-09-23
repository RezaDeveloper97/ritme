-- PregnancyWeeklyContent (table pregnancy_weekly_content).

-- name: GetWeeklyContent :one
SELECT * FROM `pregnancy_weekly_content` WHERE week_number = ? LIMIT 1;
