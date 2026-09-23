-- Admin API II (T-M2-21): content CRUD, smart messages and languages.
-- Ported from backend/app/Http/Controllers/Admin/{Article,Affirmation,Challenge,ChallengeCompletion,
-- Recommendation,Banner,TaskTemplate,InfoSection,PregnancyWeek,PhaseContent,MessageContent,Language}Controller.php
-- and App\Services\Language\LanguageProvisioner.
--
-- Optional filters use patterns instead of flags (sqlc cannot type bare flags for MySQL):
--   * text filters are LIKE patterns: '%' = no filter, an escaped literal = exact match;
--   * boolean filters are ranges: [false, true] = all, [true, true] = only true, [false, false] = only false;
--   * numeric filters use 0 = none (ids and days are ≥ 1).

-- ---------------------------------------------------------------------------
-- Articles

-- name: CountAdminArticles :one
SELECT COUNT(*) FROM `articles`;

-- name: ListAdminArticles :many
-- Article::orderBy('sort_order')->orderByDesc('id')->paginate(20).
SELECT * FROM `articles` ORDER BY sort_order, id DESC LIMIT ? OFFSET ?;

-- name: GetArticle :one
SELECT * FROM `articles` WHERE id = ? LIMIT 1;

-- name: ArticleSlugTaken :one
-- unique:articles,slug (->ignore($article) when except_id > 0).
SELECT EXISTS(SELECT 1 FROM `articles` WHERE slug = sqlc.arg(slug) AND id <> sqlc.arg(except_id)) AS taken;

-- name: CreateArticle :execresult
INSERT INTO `articles` (slug, title, excerpt, body, cycle_phases, category, read_time_minutes, image_url, image_path,
                        is_published, published_at, sort_order, created_at, updated_at)
VALUES (sqlc.arg(slug), sqlc.arg(title), sqlc.narg(excerpt), sqlc.narg(body), sqlc.narg(cycle_phases), sqlc.narg(category),
        sqlc.narg(read_time_minutes), sqlc.narg(image_url), sqlc.narg(image_path), sqlc.arg(is_published),
        sqlc.narg(published_at), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateArticle :exec
UPDATE `articles`
SET slug = sqlc.arg(slug), title = sqlc.arg(title), excerpt = sqlc.narg(excerpt), body = sqlc.narg(body),
    cycle_phases = sqlc.narg(cycle_phases), category = sqlc.narg(category),
    read_time_minutes = sqlc.narg(read_time_minutes), image_url = sqlc.narg(image_url),
    image_path = sqlc.narg(image_path), is_published = sqlc.arg(is_published), published_at = sqlc.narg(published_at),
    sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetArticlePublished :exec
UPDATE `articles`
SET is_published = sqlc.arg(is_published), published_at = sqlc.narg(published_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: DeleteArticle :execresult
DELETE FROM `articles` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Affirmations

-- name: CountAdminAffirmations :one
SELECT COUNT(*) FROM `affirmations`;

-- name: ListAdminAffirmations :many
SELECT * FROM `affirmations` ORDER BY sort_order, id DESC LIMIT ? OFFSET ?;

-- name: GetAffirmation :one
SELECT * FROM `affirmations` WHERE id = ? LIMIT 1;

-- name: CreateAffirmation :execresult
INSERT INTO `affirmations` (text, cycle_phase, is_active, sort_order, created_at, updated_at)
VALUES (sqlc.arg(text), sqlc.narg(cycle_phase), sqlc.arg(is_active), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateAffirmation :exec
UPDATE `affirmations`
SET text = sqlc.arg(text), cycle_phase = sqlc.narg(cycle_phase), is_active = sqlc.arg(is_active),
    sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ToggleAffirmation :exec
UPDATE `affirmations` SET is_active = NOT is_active, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteAffirmation :execresult
DELETE FROM `affirmations` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Challenges (ChallengeController::filtered). day = 0: no cycle-day filter; otherwise
-- Challenge::scopeForCycleDay (untargeted rows match every day).

-- name: CountAdminChallenges :one
SELECT COUNT(*) FROM `challenges`
WHERE (IFNULL(category, '') LIKE CAST(sqlc.arg(pattern) AS CHAR)
       OR title LIKE CAST(sqlc.arg(pattern) AS CHAR) OR title LIKE CAST(sqlc.arg(json_pattern) AS CHAR)
       OR IFNULL(description, '') LIKE CAST(sqlc.arg(pattern) AS CHAR)
       OR IFNULL(description, '') LIKE CAST(sqlc.arg(json_pattern) AS CHAR))
  AND is_active >= sqlc.arg(active_min) AND is_active <= sqlc.arg(active_max)
  AND (((cycle_day_from IS NULL OR cycle_day_from <= sqlc.arg(day)) AND (cycle_day_to IS NULL OR cycle_day_to >= sqlc.arg(day)))
       OR (cycle_day_from IS NULL AND cycle_day_to IS NULL)
       OR sqlc.arg(day) = 0);

-- name: ListAdminChallenges :many
SELECT * FROM `challenges`
WHERE (IFNULL(category, '') LIKE CAST(sqlc.arg(pattern) AS CHAR)
       OR title LIKE CAST(sqlc.arg(pattern) AS CHAR) OR title LIKE CAST(sqlc.arg(json_pattern) AS CHAR)
       OR IFNULL(description, '') LIKE CAST(sqlc.arg(pattern) AS CHAR)
       OR IFNULL(description, '') LIKE CAST(sqlc.arg(json_pattern) AS CHAR))
  AND is_active >= sqlc.arg(active_min) AND is_active <= sqlc.arg(active_max)
  AND (((cycle_day_from IS NULL OR cycle_day_from <= sqlc.arg(day)) AND (cycle_day_to IS NULL OR cycle_day_to >= sqlc.arg(day)))
       OR (cycle_day_from IS NULL AND cycle_day_to IS NULL)
       OR sqlc.arg(day) = 0)
ORDER BY sort_order, id DESC
LIMIT ? OFFSET ?;

-- name: GetChallenge :one
SELECT * FROM `challenges` WHERE id = ? LIMIT 1;

-- name: ChallengeExists :one
SELECT EXISTS(SELECT 1 FROM `challenges` WHERE id = ?) AS found;

-- name: CreateChallenge :execresult
INSERT INTO `challenges` (title, description, cycle_day_from, cycle_day_to, category, is_active, sort_order,
                          created_at, updated_at)
VALUES (sqlc.arg(title), sqlc.narg(description), sqlc.narg(cycle_day_from), sqlc.narg(cycle_day_to),
        sqlc.narg(category), sqlc.arg(is_active), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateChallenge :exec
UPDATE `challenges`
SET title = sqlc.arg(title), description = sqlc.narg(description), cycle_day_from = sqlc.narg(cycle_day_from),
    cycle_day_to = sqlc.narg(cycle_day_to), category = sqlc.narg(category), is_active = sqlc.arg(is_active),
    sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ToggleChallenge :exec
UPDATE `challenges` SET is_active = NOT is_active, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteChallenge :execresult
DELETE FROM `challenges` WHERE id = ?;

-- name: ListChallengeTitles :many
-- Challenge::orderBy('sort_order')->orderBy('id')->get(['id', 'title']) (report filter).
SELECT id, title FROM `challenges` ORDER BY sort_order, id;

-- ---------------------------------------------------------------------------
-- Challenge completions report (ChallengeCompletionController). challenge_id = 0: all;
-- the date range is inclusive; pattern searches the user's name and mobile.

-- name: CompletionTotals :one
SELECT COUNT(*) AS total, COUNT(DISTINCT c.user_id) AS users
FROM `user_challenge_completions` c
WHERE (c.challenge_id = sqlc.arg(challenge_id) OR sqlc.arg(challenge_id) = 0)
  AND c.completion_date >= sqlc.arg(date_from) AND c.completion_date <= sqlc.arg(date_to)
  AND (sqlc.arg(pattern) = '%' OR EXISTS (
        SELECT 1 FROM `users` u WHERE u.id = c.user_id
          AND (IFNULL(u.name, '') LIKE sqlc.arg(pattern) OR IFNULL(u.mobile, '') LIKE sqlc.arg(pattern))));

-- name: ListCompletions :many
SELECT c.id, c.user_id, c.challenge_id, c.completion_date, c.completed_at,
       u.name AS user_name, u.mobile AS user_mobile, ch.title AS challenge_title
FROM `user_challenge_completions` c
LEFT JOIN `users` u ON u.id = c.user_id
LEFT JOIN `challenges` ch ON ch.id = c.challenge_id
WHERE (c.challenge_id = sqlc.arg(challenge_id) OR sqlc.arg(challenge_id) = 0)
  AND c.completion_date >= sqlc.arg(date_from) AND c.completion_date <= sqlc.arg(date_to)
  AND (sqlc.arg(pattern) = '%' OR EXISTS (
        SELECT 1 FROM `users` su WHERE su.id = c.user_id
          AND (IFNULL(su.name, '') LIKE sqlc.arg(pattern) OR IFNULL(su.mobile, '') LIKE sqlc.arg(pattern))))
ORDER BY c.completion_date DESC, c.id DESC
LIMIT ? OFFSET ?;

-- name: CompletionsPerChallenge :many
SELECT c.challenge_id, ch.title AS challenge_title, COUNT(*) AS completions, COUNT(DISTINCT c.user_id) AS users
FROM `user_challenge_completions` c
LEFT JOIN `challenges` ch ON ch.id = c.challenge_id
WHERE (c.challenge_id = sqlc.arg(challenge_id) OR sqlc.arg(challenge_id) = 0)
  AND c.completion_date >= sqlc.arg(date_from) AND c.completion_date <= sqlc.arg(date_to)
  AND (sqlc.arg(pattern) = '%' OR EXISTS (
        SELECT 1 FROM `users` u WHERE u.id = c.user_id
          AND (IFNULL(u.name, '') LIKE sqlc.arg(pattern) OR IFNULL(u.mobile, '') LIKE sqlc.arg(pattern))))
GROUP BY c.challenge_id, ch.title
ORDER BY completions DESC, c.challenge_id;

-- name: CountCompletionsOn :one
SELECT COUNT(*) FROM `user_challenge_completions` WHERE completion_date = ?;

-- ---------------------------------------------------------------------------
-- Recommendations. phase_pattern over IFNULL(cycle_phase, ''): '%' = all, '' = general (NULL),
-- a value = that phase. type_pattern: '%' = all.

-- name: CountAdminRecommendations :one
SELECT COUNT(*) FROM `recommendations`
WHERE IFNULL(cycle_phase, '') LIKE sqlc.arg(phase_pattern) AND type LIKE sqlc.arg(type_pattern);

-- name: ListAdminRecommendations :many
SELECT * FROM `recommendations`
WHERE IFNULL(cycle_phase, '') LIKE sqlc.arg(phase_pattern) AND type LIKE sqlc.arg(type_pattern)
ORDER BY cycle_phase IS NULL, cycle_phase, sort_order, id
LIMIT ? OFFSET ?;

-- name: GetRecommendation :one
SELECT * FROM `recommendations` WHERE id = ? LIMIT 1;

-- name: CreateRecommendation :execresult
INSERT INTO `recommendations` (type, title, text, cycle_phase, cycle_subphases, symptom_trigger, is_active, sort_order,
                               created_at, updated_at)
VALUES (sqlc.arg(type), sqlc.narg(title), sqlc.arg(text), sqlc.narg(cycle_phase), sqlc.narg(cycle_subphases),
        sqlc.narg(symptom_trigger), sqlc.arg(is_active), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateRecommendation :exec
UPDATE `recommendations`
SET type = sqlc.arg(type), title = sqlc.narg(title), text = sqlc.arg(text), cycle_phase = sqlc.narg(cycle_phase),
    cycle_subphases = sqlc.narg(cycle_subphases), symptom_trigger = sqlc.narg(symptom_trigger),
    is_active = sqlc.arg(is_active), sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ToggleRecommendation :exec
UPDATE `recommendations` SET is_active = NOT is_active, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteRecommendation :execresult
DELETE FROM `recommendations` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Banners

-- name: CountAdminBanners :one
SELECT COUNT(*) FROM `banners`;

-- name: ListAdminBanners :many
SELECT * FROM `banners` ORDER BY position, sort_order, id DESC LIMIT ? OFFSET ?;

-- name: GetBanner :one
SELECT * FROM `banners` WHERE id = ? LIMIT 1;

-- name: CreateBanner :execresult
INSERT INTO `banners` (title, image_path, position, link_url, link_type, starts_at, ends_at, is_active, sort_order,
                       created_at, updated_at)
VALUES (sqlc.narg(title), sqlc.arg(image_path), sqlc.arg(position), sqlc.narg(link_url), sqlc.narg(link_type),
        sqlc.narg(starts_at), sqlc.narg(ends_at), sqlc.arg(is_active), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateBanner :exec
UPDATE `banners`
SET title = sqlc.narg(title), image_path = sqlc.arg(image_path), position = sqlc.arg(position),
    link_url = sqlc.narg(link_url), link_type = sqlc.narg(link_type), starts_at = sqlc.narg(starts_at),
    ends_at = sqlc.narg(ends_at), is_active = sqlc.arg(is_active), sort_order = sqlc.arg(sort_order),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ToggleBanner :exec
UPDATE `banners` SET is_active = NOT is_active, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteBanner :execresult
DELETE FROM `banners` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Task templates

-- name: CountAdminTaskTemplates :one
SELECT COUNT(*) FROM `task_templates`;

-- name: ListAdminTaskTemplates :many
SELECT * FROM `task_templates` ORDER BY sort_order, id DESC LIMIT ? OFFSET ?;

-- name: GetTaskTemplate :one
SELECT * FROM `task_templates` WHERE id = ? LIMIT 1;

-- name: TaskTemplateKeyTaken :one
SELECT EXISTS(SELECT 1 FROM `task_templates` WHERE `key` = sqlc.arg(template_key) AND id <> sqlc.arg(except_id)) AS taken;

-- name: CreateTaskTemplate :execresult
INSERT INTO `task_templates` (`key`, title, description, category, icon, cycle_phase, is_active, sort_order,
                              created_at, updated_at)
VALUES (sqlc.arg(template_key), sqlc.arg(title), sqlc.narg(description), sqlc.arg(category), sqlc.narg(icon),
        sqlc.narg(cycle_phase), sqlc.arg(is_active), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateTaskTemplate :exec
UPDATE `task_templates`
SET `key` = sqlc.arg(template_key), title = sqlc.arg(title), description = sqlc.narg(description),
    category = sqlc.arg(category), icon = sqlc.narg(icon), cycle_phase = sqlc.narg(cycle_phase),
    is_active = sqlc.arg(is_active), sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ToggleTaskTemplate :exec
UPDATE `task_templates` SET is_active = NOT is_active, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteTaskTemplate :execresult
DELETE FROM `task_templates` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Info sections (InfoSection::inGroup($group)->ordered())

-- name: CountAdminInfoSections :one
SELECT COUNT(*) FROM `info_sections` WHERE `group` = ?;

-- name: ListAdminInfoSections :many
SELECT * FROM `info_sections` WHERE `group` = ? ORDER BY sort_order, id LIMIT ? OFFSET ?;

-- name: MaxInfoSectionSortOrder :one
SELECT CAST(IFNULL(MAX(sort_order), 0) AS SIGNED) AS max_sort FROM `info_sections` WHERE `group` = ?;

-- name: GetInfoSection :one
SELECT * FROM `info_sections` WHERE id = ? LIMIT 1;

-- name: CreateInfoSection :execresult
INSERT INTO `info_sections` (`group`, heading, body, link_label, link_url, is_active, sort_order, created_at, updated_at)
VALUES (sqlc.arg(section_group), sqlc.arg(heading), sqlc.arg(body), sqlc.narg(link_label), sqlc.narg(link_url),
        sqlc.arg(is_active), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateInfoSection :exec
UPDATE `info_sections`
SET `group` = sqlc.arg(section_group), heading = sqlc.arg(heading), body = sqlc.arg(body),
    link_label = sqlc.narg(link_label), link_url = sqlc.narg(link_url), is_active = sqlc.arg(is_active),
    sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ToggleInfoSection :exec
UPDATE `info_sections` SET is_active = NOT is_active, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteInfoSection :execresult
DELETE FROM `info_sections` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Pregnancy weeks (pregnancy_weekly_content)

-- name: ListPregnancyWeekIDs :many
SELECT id, week_number FROM `pregnancy_weekly_content` ORDER BY week_number, id;

-- name: GetPregnancyWeek :one
SELECT * FROM `pregnancy_weekly_content` WHERE id = ? LIMIT 1;

-- name: PregnancyWeekTaken :one
SELECT EXISTS(SELECT 1 FROM `pregnancy_weekly_content`
              WHERE week_number = sqlc.arg(week_number) AND id <> sqlc.arg(except_id)) AS taken;

-- name: CreatePregnancyWeek :execresult
INSERT INTO `pregnancy_weekly_content` (week_number, fetal_development, mother_body_changes, dos_and_donts, care_plan,
                                        body_adaptation, emotional_status, key_nutrition, physical_activity,
                                        tests_and_checkups, faq, created_at, updated_at)
VALUES (sqlc.arg(week_number), sqlc.narg(fetal_development), sqlc.narg(mother_body_changes), sqlc.narg(dos_and_donts),
        sqlc.narg(care_plan), sqlc.narg(body_adaptation), sqlc.narg(emotional_status), sqlc.narg(key_nutrition),
        sqlc.narg(physical_activity), sqlc.narg(tests_and_checkups), sqlc.narg(faq), sqlc.arg(now), sqlc.arg(now));

-- name: UpdatePregnancyWeek :exec
UPDATE `pregnancy_weekly_content`
SET week_number = sqlc.arg(week_number), fetal_development = sqlc.narg(fetal_development),
    mother_body_changes = sqlc.narg(mother_body_changes), dos_and_donts = sqlc.narg(dos_and_donts),
    care_plan = sqlc.narg(care_plan), body_adaptation = sqlc.narg(body_adaptation),
    emotional_status = sqlc.narg(emotional_status), key_nutrition = sqlc.narg(key_nutrition),
    physical_activity = sqlc.narg(physical_activity), tests_and_checkups = sqlc.narg(tests_and_checkups),
    faq = sqlc.narg(faq), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: DeletePregnancyWeek :execresult
DELETE FROM `pregnancy_weekly_content` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Phase contents

-- name: ListPhaseContentIDs :many
SELECT id, phase FROM `phase_contents` ORDER BY id;

-- name: GetPhaseContentByID :one
SELECT * FROM `phase_contents` WHERE id = ? LIMIT 1;

-- name: PhaseContentTaken :one
SELECT EXISTS(SELECT 1 FROM `phase_contents` WHERE phase = sqlc.arg(phase) AND id <> sqlc.arg(except_id)) AS taken;

-- name: CreatePhaseContent :execresult
INSERT INTO `phase_contents` (phase, symptom_prediction, vaginal_discharge, fertility, hormonal_changes, sex_tips,
                              nutrition, exercise, skin_care, sleep, created_at, updated_at)
VALUES (sqlc.arg(phase), sqlc.narg(symptom_prediction), sqlc.narg(vaginal_discharge), sqlc.narg(fertility),
        sqlc.narg(hormonal_changes), sqlc.narg(sex_tips), sqlc.narg(nutrition), sqlc.narg(exercise),
        sqlc.narg(skin_care), sqlc.narg(sleep), sqlc.arg(now), sqlc.arg(now));

-- name: UpdatePhaseContent :exec
UPDATE `phase_contents`
SET phase = sqlc.arg(phase), symptom_prediction = sqlc.narg(symptom_prediction),
    vaginal_discharge = sqlc.narg(vaginal_discharge), fertility = sqlc.narg(fertility),
    hormonal_changes = sqlc.narg(hormonal_changes), sex_tips = sqlc.narg(sex_tips), nutrition = sqlc.narg(nutrition),
    exercise = sqlc.narg(exercise), skin_care = sqlc.narg(skin_care), sleep = sqlc.narg(sleep),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: DeletePhaseContent :execresult
DELETE FROM `phase_contents` WHERE id = ?;

-- ---------------------------------------------------------------------------
-- Smart messages (message_contents). group/locale patterns: '%' = all.

-- name: ListMessageGroups :many
SELECT DISTINCT `group` FROM `message_contents` ORDER BY `group`;

-- name: ListMessageLocales :many
SELECT DISTINCT locale FROM `message_contents` ORDER BY locale;

-- name: CountAdminMessages :one
SELECT COUNT(*) FROM `message_contents`
WHERE `group` LIKE sqlc.arg(group_pattern) AND locale LIKE sqlc.arg(locale_pattern)
  AND is_approved >= sqlc.arg(approved_min) AND is_approved <= sqlc.arg(approved_max);

-- name: ListAdminMessages :many
SELECT * FROM `message_contents`
WHERE `group` LIKE sqlc.arg(group_pattern) AND locale LIKE sqlc.arg(locale_pattern)
  AND is_approved >= sqlc.arg(approved_min) AND is_approved <= sqlc.arg(approved_max)
ORDER BY `group`, item_key, locale
LIMIT ? OFFSET ?;

-- name: GetMessageContent :one
SELECT * FROM `message_contents` WHERE id = ? LIMIT 1;

-- name: UpdateMessageContent :exec
UPDATE `message_contents`
SET payload = sqlc.arg(payload), label = sqlc.narg(label), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ToggleMessageApproved :exec
UPDATE `message_contents` SET is_approved = NOT is_approved, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: ToggleMessageActive :exec
UPDATE `message_contents` SET is_active = NOT is_active, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: MessageLocaleExists :one
SELECT EXISTS(SELECT 1 FROM `message_contents` WHERE locale = ?) AS found;

-- name: CloneMessageContents :execresult
-- LanguageProvisioner::cloneMessageContents: the source locale's rows, unapproved; rows that
-- already exist (unique group+item_key+locale) are skipped.
INSERT IGNORE INTO `message_contents` (`group`, item_key, locale, label, payload, is_active, is_approved, sort_order,
                                       created_at, updated_at)
SELECT s.`group`, s.item_key, sqlc.arg(code), s.label, s.payload, s.is_active, 0, s.sort_order, sqlc.arg(now), sqlc.arg(now)
FROM `message_contents` s
WHERE s.locale = sqlc.arg(source)
ORDER BY s.id;

-- name: DeleteMessageContentsForLocale :exec
DELETE FROM `message_contents` WHERE locale = ?;

-- ---------------------------------------------------------------------------
-- Languages (every write must flush the registry caches)

-- name: ListAllLanguages :many
-- Language::query()->ordered()->get() (active and inactive).
SELECT * FROM `languages` ORDER BY sort_order, id;

-- name: GetLanguage :one
SELECT * FROM `languages` WHERE id = ? LIMIT 1;

-- name: LanguageCodeTaken :one
SELECT EXISTS(SELECT 1 FROM `languages` WHERE code = sqlc.arg(code) AND id <> sqlc.arg(except_id)) AS taken;

-- name: MaxLanguageSortOrder :one
SELECT CAST(IFNULL(MAX(sort_order), 0) AS SIGNED) AS max_sort FROM `languages`;

-- name: CreateLanguage :execresult
INSERT INTO `languages` (code, name, english_name, direction, is_active, is_default, sort_order, created_at, updated_at)
VALUES (sqlc.arg(code), sqlc.arg(name), sqlc.arg(english_name), sqlc.arg(direction), sqlc.arg(is_active),
        sqlc.arg(is_default), sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateLanguage :exec
UPDATE `languages`
SET code = sqlc.arg(code), name = sqlc.arg(name), english_name = sqlc.arg(english_name),
    direction = sqlc.arg(direction), is_active = sqlc.arg(is_active), is_default = sqlc.arg(is_default),
    sort_order = sqlc.arg(sort_order), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetLanguageActive :exec
UPDATE `languages` SET is_active = sqlc.arg(is_active), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: SetLanguageDefault :exec
UPDATE `languages` SET is_default = 1, updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: ClearOtherDefaultLanguages :exec
UPDATE `languages` SET is_default = 0, updated_at = sqlc.arg(now) WHERE id <> sqlc.arg(id) AND is_default = 1;

-- name: AnyDefaultLanguage :one
SELECT EXISTS(SELECT 1 FROM `languages` WHERE is_default = 1) AS found;

-- name: FirstActiveLanguageID :one
SELECT id FROM `languages` WHERE is_active = 1 ORDER BY sort_order, id LIMIT 1;

-- name: DeleteLanguage :execresult
DELETE FROM `languages` WHERE id = ?;
