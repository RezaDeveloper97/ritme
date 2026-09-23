-- 00001_baseline.sql — the Ritme schema as of the end of M1 (55 Laravel migrations, frozen during M2).
--
-- Source: `mariadb-dump --no-data --skip-comments` of a MariaDB 11.4 database migrated by
-- `backend/database/migrations` (DB_CONNECTION=mysql, the grammar prod and stage use), with:
--   * Laravel's `migrations` table left out (goose tracks itself in `goose_db_version`);
--   * MariaDB's expansion of JSON columns (`longtext COLLATE utf8mb4_bin CHECK (json_valid(col))`)
--     written back as `json` — MariaDB expands it identically, and sqlc needs the `json` type;
--   * `users` moved first so the foreign keys resolve; AUTO_INCREMENT counters dropped.
-- `make schema-diff` proves this file and the Laravel migrations still produce the same schema.
--
-- Who runs this: tests (internal/platform/db/testdb) and brand-new environments only.
-- Stage/prod already have these tables; at cutover (T-M2-27) they are version-stamped, not migrated.
-- See docs/go-migration/migrations.md.

-- +goose Up
CREATE TABLE `users` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `mobile` varchar(11) DEFAULT NULL,
  `mobile_verified_at` timestamp NULL DEFAULT NULL,
  `blocked_at` timestamp NULL DEFAULT NULL,
  `email_verified_at` timestamp NULL DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `remember_token` varchar(100) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `users_email_unique` (`email`),
  UNIQUE KEY `users_mobile_unique` (`mobile`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `admins` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `role` varchar(255) NOT NULL DEFAULT 'editor',
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `last_login_at` timestamp NULL DEFAULT NULL,
  `remember_token` varchar(100) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `admins_email_unique` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `affirmations` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `text` json NOT NULL,
  `cycle_phase` varchar(255) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `affirmations_is_active_cycle_phase_index` (`is_active`,`cycle_phase`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `articles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `slug` varchar(255) NOT NULL,
  `title` json NOT NULL,
  `excerpt` json DEFAULT NULL,
  `body` json DEFAULT NULL,
  `cycle_phases` json DEFAULT NULL,
  `category` varchar(255) DEFAULT NULL,
  `read_time_minutes` smallint(5) unsigned DEFAULT NULL,
  `image_url` varchar(255) DEFAULT NULL,
  `image_path` varchar(255) DEFAULT NULL,
  `is_published` tinyint(1) NOT NULL DEFAULT 1,
  `published_at` timestamp NULL DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `articles_slug_unique` (`slug`),
  KEY `articles_is_published_index` (`is_published`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `banners` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `title` json DEFAULT NULL,
  `image_path` varchar(255) NOT NULL,
  `position` varchar(255) NOT NULL DEFAULT 'home_top',
  `link_url` varchar(1000) DEFAULT NULL,
  `link_type` varchar(255) DEFAULT NULL,
  `starts_at` timestamp NULL DEFAULT NULL,
  `ends_at` timestamp NULL DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `banners_position_is_active_starts_at_ends_at_index` (`position`,`is_active`,`starts_at`,`ends_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `cache` (
  `key` varchar(255) NOT NULL,
  `value` mediumtext NOT NULL,
  `expiration` int(11) NOT NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `cache_locks` (
  `key` varchar(255) NOT NULL,
  `owner` varchar(255) NOT NULL,
  `expiration` int(11) NOT NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `challenges` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `slug` varchar(255) DEFAULT NULL,
  `title` json NOT NULL,
  `description` json DEFAULT NULL,
  `cycle_day_from` tinyint(3) unsigned DEFAULT NULL,
  `cycle_day_to` tinyint(3) unsigned DEFAULT NULL,
  `category` varchar(255) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `challenges_slug_unique` (`slug`),
  KEY `challenges_active_day_index` (`is_active`,`cycle_day_from`,`cycle_day_to`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `cycle_histories` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `period_start_date` date NOT NULL,
  `period_end_date` date DEFAULT NULL,
  `cycle_length` int(11) DEFAULT NULL,
  `bleeding_length` int(11) DEFAULT NULL,
  `is_confirmed` tinyint(1) NOT NULL DEFAULT 0,
  `is_estimated` tinyint(1) NOT NULL DEFAULT 0,
  `source` varchar(255) NOT NULL DEFAULT 'user_logged',
  `data_quality_flags` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `cycle_histories_user_id_period_start_date_unique` (`user_id`,`period_start_date`),
  CONSTRAINT `cycle_histories_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `daily_health_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `bleeding_intensity` varchar(255) DEFAULT NULL,
  `blood_color` varchar(255) DEFAULT NULL,
  `has_clots` tinyint(1) DEFAULT NULL,
  `clots_amount` varchar(255) DEFAULT NULL,
  `spotting` tinyint(1) DEFAULT NULL,
  `bleeding_smell` varchar(255) DEFAULT NULL,
  `headache_intensity` varchar(255) DEFAULT NULL,
  `stomach_ache_intensity` varchar(255) DEFAULT NULL,
  `pelvic_pain_intensity` varchar(255) DEFAULT NULL,
  `breast_pain_intensity` varchar(255) DEFAULT NULL,
  `back_pain_intensity` varchar(255) DEFAULT NULL,
  `ovarian_pain_intensity` varchar(255) DEFAULT NULL,
  `nausea_intensity` varchar(255) DEFAULT NULL,
  `bloating_intensity` varchar(255) DEFAULT NULL,
  `diarrhea` tinyint(1) DEFAULT NULL,
  `constipation` tinyint(1) DEFAULT NULL,
  `appetite_change` varchar(255) DEFAULT NULL,
  `food_craving` tinyint(1) DEFAULT NULL,
  `breast_sensitivity_intensity` varchar(255) DEFAULT NULL,
  `vaginal_dryness` tinyint(1) DEFAULT NULL,
  `vaginal_burning` tinyint(1) DEFAULT NULL,
  `vaginal_burning_intensity` varchar(255) DEFAULT NULL,
  `vaginal_itching` tinyint(1) DEFAULT NULL,
  `vaginal_itching_intensity` varchar(255) DEFAULT NULL,
  `vaginal_smell_change` tinyint(1) DEFAULT NULL,
  `urination_change` varchar(255) DEFAULT NULL,
  `urination_burning_intensity` varchar(255) DEFAULT NULL,
  `acne` tinyint(1) DEFAULT NULL,
  `oily_skin` tinyint(1) DEFAULT NULL,
  `hair_loss` tinyint(1) DEFAULT NULL,
  `swelling` tinyint(1) DEFAULT NULL,
  `fatigue` tinyint(1) DEFAULT NULL,
  `dizziness` tinyint(1) DEFAULT NULL,
  `hot_flashes` tinyint(1) DEFAULT NULL,
  `chills` tinyint(1) DEFAULT NULL,
  `moods` json DEFAULT NULL,
  `sleep_duration` varchar(255) DEFAULT NULL,
  `sleep_quality` varchar(255) DEFAULT NULL,
  `exercise_type` json DEFAULT NULL,
  `exercise_duration` smallint(5) unsigned DEFAULT NULL,
  `exercise_intensity` varchar(255) DEFAULT NULL,
  `sexual_activities` json DEFAULT NULL,
  `sexual_desire` varchar(255) DEFAULT NULL,
  `intercourse_type` varchar(255) DEFAULT NULL,
  `weight` decimal(5,2) DEFAULT NULL,
  `basal_body_temperature` decimal(4,2) DEFAULT NULL,
  `heart_rate` smallint(5) unsigned DEFAULT NULL,
  `systolic_pressure` smallint(5) unsigned DEFAULT NULL,
  `diastolic_pressure` smallint(5) unsigned DEFAULT NULL,
  `blood_sugar` decimal(5,1) DEFAULT NULL,
  `energy_level` varchar(255) DEFAULT NULL,
  `discharge_color` varchar(255) DEFAULT NULL,
  `discharge_texture` varchar(255) DEFAULT NULL,
  `discharge_amount` varchar(255) DEFAULT NULL,
  `discharge_smell` varchar(255) DEFAULT NULL,
  `discharge_itching` tinyint(1) DEFAULT NULL,
  `discharge_burning` tinyint(1) DEFAULT NULL,
  `frequent_urination` tinyint(1) DEFAULT NULL,
  `medications` json DEFAULT NULL,
  `notes` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `daily_health_logs_user_id_log_date_unique` (`user_id`,`log_date`),
  CONSTRAINT `daily_health_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `failed_jobs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `uuid` varchar(255) NOT NULL,
  `connection` text NOT NULL,
  `queue` text NOT NULL,
  `payload` longtext NOT NULL,
  `exception` longtext NOT NULL,
  `failed_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `failed_jobs_uuid_unique` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `info_sections` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `group` varchar(32) NOT NULL DEFAULT 'privacy',
  `key` varchar(255) DEFAULT NULL,
  `heading` json NOT NULL,
  `body` json NOT NULL,
  `link_label` json DEFAULT NULL,
  `link_url` varchar(255) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `info_sections_group_key_unique` (`group`,`key`),
  KEY `info_sections_group_is_active_sort_order_index` (`group`,`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `job_batches` (
  `id` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL,
  `total_jobs` int(11) NOT NULL,
  `pending_jobs` int(11) NOT NULL,
  `failed_jobs` int(11) NOT NULL,
  `failed_job_ids` longtext NOT NULL,
  `options` mediumtext DEFAULT NULL,
  `cancelled_at` int(11) DEFAULT NULL,
  `created_at` int(11) NOT NULL,
  `finished_at` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `jobs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `queue` varchar(255) NOT NULL,
  `payload` longtext NOT NULL,
  `attempts` tinyint(3) unsigned NOT NULL,
  `reserved_at` int(10) unsigned DEFAULT NULL,
  `available_at` int(10) unsigned NOT NULL,
  `created_at` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `jobs_queue_index` (`queue`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `languages` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(12) NOT NULL,
  `name` varchar(60) NOT NULL,
  `english_name` varchar(60) NOT NULL,
  `direction` varchar(3) NOT NULL DEFAULT 'ltr',
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `is_default` tinyint(1) NOT NULL DEFAULT 0,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `languages_code_unique` (`code`),
  KEY `languages_is_active_sort_order_index` (`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `message_contents` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `group` varchar(255) NOT NULL,
  `item_key` varchar(255) NOT NULL,
  `locale` varchar(5) NOT NULL,
  `label` varchar(255) DEFAULT NULL,
  `payload` json NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `is_approved` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(10) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `message_contents_group_item_key_locale_unique` (`group`,`item_key`,`locale`),
  KEY `message_contents_group_index` (`group`),
  KEY `message_contents_item_key_index` (`item_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oauth_access_tokens` (
  `id` char(80) NOT NULL,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `client_id` char(36) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `scopes` text DEFAULT NULL,
  `revoked` tinyint(1) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `expires_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `oauth_access_tokens_user_id_index` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oauth_auth_codes` (
  `id` char(80) NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `client_id` char(36) NOT NULL,
  `scopes` text DEFAULT NULL,
  `revoked` tinyint(1) NOT NULL,
  `expires_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `oauth_auth_codes_user_id_index` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oauth_clients` (
  `id` char(36) NOT NULL,
  `owner_type` varchar(255) DEFAULT NULL,
  `owner_id` bigint(20) unsigned DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `secret` varchar(255) DEFAULT NULL,
  `provider` varchar(255) DEFAULT NULL,
  `redirect_uris` text NOT NULL,
  `grant_types` text NOT NULL,
  `revoked` tinyint(1) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `oauth_clients_owner_type_owner_id_index` (`owner_type`,`owner_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oauth_device_codes` (
  `id` char(80) NOT NULL,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `client_id` char(36) NOT NULL,
  `user_code` char(8) NOT NULL,
  `scopes` text NOT NULL,
  `revoked` tinyint(1) NOT NULL,
  `user_approved_at` datetime DEFAULT NULL,
  `last_polled_at` datetime DEFAULT NULL,
  `expires_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `oauth_device_codes_user_code_unique` (`user_code`),
  KEY `oauth_device_codes_user_id_index` (`user_id`),
  KEY `oauth_device_codes_client_id_index` (`client_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oauth_refresh_tokens` (
  `id` char(80) NOT NULL,
  `access_token_id` char(80) NOT NULL,
  `revoked` tinyint(1) NOT NULL,
  `expires_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `oauth_refresh_tokens_access_token_id_index` (`access_token_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `otp_verifications` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `mobile` varchar(11) NOT NULL,
  `code` varchar(4) NOT NULL,
  `expires_at` timestamp NOT NULL,
  `verified_at` timestamp NULL DEFAULT NULL,
  `attempts` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `otp_verifications_mobile_index` (`mobile`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `password_reset_tokens` (
  `email` varchar(255) NOT NULL,
  `token` varchar(255) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `phase_contents` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `phase` varchar(255) NOT NULL,
  `symptom_prediction` json DEFAULT NULL,
  `vaginal_discharge` json DEFAULT NULL,
  `fertility` json DEFAULT NULL,
  `hormonal_changes` json DEFAULT NULL,
  `sex_tips` json DEFAULT NULL,
  `nutrition` json DEFAULT NULL,
  `exercise` json DEFAULT NULL,
  `skin_care` json DEFAULT NULL,
  `sleep` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `phase_contents_phase_unique` (`phase`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `pregnancy_alerts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `alert_level` varchar(255) NOT NULL,
  `alert_type` varchar(255) NOT NULL,
  `title` varchar(255) NOT NULL,
  `message` text NOT NULL,
  `pregnancy_week` int(11) DEFAULT NULL,
  `trigger_symptoms` json DEFAULT NULL,
  `medical_history_flags` json DEFAULT NULL,
  `is_read` tinyint(1) NOT NULL DEFAULT 0,
  `is_dismissed` tinyint(1) NOT NULL DEFAULT 0,
  `read_at` timestamp NULL DEFAULT NULL,
  `dismissed_at` timestamp NULL DEFAULT NULL,
  `recommended_actions` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `pregnancy_alerts_user_id_is_read_index` (`user_id`,`is_read`),
  KEY `pregnancy_alerts_user_id_alert_level_index` (`user_id`,`alert_level`),
  CONSTRAINT `pregnancy_alerts_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `pregnancy_fetal_movements` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `pregnancy_week` int(11) NOT NULL,
  `movement_status` varchar(255) NOT NULL,
  `movement_count` int(11) DEFAULT NULL,
  `first_movement_time` time DEFAULT NULL,
  `last_movement_time` time DEFAULT NULL,
  `notes` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_fetal_movements_user_id_log_date_unique` (`user_id`,`log_date`),
  CONSTRAINT `pregnancy_fetal_movements_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `pregnancy_profiles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `pregnancy_mode` tinyint(1) NOT NULL DEFAULT 1,
  `cycle_mode` tinyint(1) NOT NULL DEFAULT 0,
  `is_locked` tinyint(1) NOT NULL DEFAULT 0,
  `age_source` varchar(255) DEFAULT NULL,
  `confidence_level` varchar(255) DEFAULT NULL,
  `lmp_date` date DEFAULT NULL,
  `ultrasound_date` date DEFAULT NULL,
  `ultrasound_weeks` int(11) DEFAULT NULL,
  `ultrasound_days` int(11) DEFAULT NULL,
  `manual_weeks` int(11) DEFAULT NULL,
  `manual_days` int(11) DEFAULT NULL,
  `manual_entry_date` date DEFAULT NULL,
  `estimated_due_date` date DEFAULT NULL,
  `estimated_conception_date` date DEFAULT NULL,
  `uncertainty_days` int(11) NOT NULL DEFAULT 3,
  `has_miscarriage_history` tinyint(1) DEFAULT NULL,
  `has_high_risk_history` tinyint(1) DEFAULT NULL,
  `pre_existing_conditions` json DEFAULT NULL,
  `blood_type` varchar(255) DEFAULT NULL,
  `rh_factor` varchar(255) DEFAULT NULL,
  `rh_negative_care_flag` tinyint(1) NOT NULL DEFAULT 0,
  `first_fetal_movement_date` date DEFAULT NULL,
  `fetal_movement_felt` tinyint(1) NOT NULL DEFAULT 0,
  `onboarding_completed` tinyint(1) NOT NULL DEFAULT 0,
  `onboarding_completed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_profiles_user_id_unique` (`user_id`),
  CONSTRAINT `pregnancy_profiles_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `pregnancy_symptom_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `has_nausea` tinyint(1) DEFAULT NULL,
  `nausea_severity` varchar(255) DEFAULT NULL,
  `has_vomiting` tinyint(1) DEFAULT NULL,
  `vomiting_severity` varchar(255) DEFAULT NULL,
  `has_fatigue` tinyint(1) DEFAULT NULL,
  `fatigue_severity` varchar(255) DEFAULT NULL,
  `has_headache` tinyint(1) DEFAULT NULL,
  `headache_severity` varchar(255) DEFAULT NULL,
  `has_dizziness` tinyint(1) DEFAULT NULL,
  `dizziness_severity` varchar(255) DEFAULT NULL,
  `has_breast_pain` tinyint(1) DEFAULT NULL,
  `breast_pain_severity` varchar(255) DEFAULT NULL,
  `has_lower_abdominal_pain` tinyint(1) DEFAULT NULL,
  `lower_abdominal_pain_severity` varchar(255) DEFAULT NULL,
  `has_cramping` tinyint(1) DEFAULT NULL,
  `cramping_severity` varchar(255) DEFAULT NULL,
  `has_back_pain` tinyint(1) DEFAULT NULL,
  `back_pain_severity` varchar(255) DEFAULT NULL,
  `has_pelvic_pressure` tinyint(1) DEFAULT NULL,
  `pelvic_pressure_severity` varchar(255) DEFAULT NULL,
  `has_spotting` tinyint(1) DEFAULT NULL,
  `spotting_severity` varchar(255) DEFAULT NULL,
  `has_bleeding` tinyint(1) DEFAULT NULL,
  `bleeding_severity` varchar(255) DEFAULT NULL,
  `has_fluid_leakage` tinyint(1) DEFAULT NULL,
  `fluid_leakage_severity` varchar(255) DEFAULT NULL,
  `has_severe_sudden_pain` tinyint(1) DEFAULT NULL,
  `severe_sudden_pain_severity` varchar(255) DEFAULT NULL,
  `notes` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_symptom_logs_user_id_log_date_unique` (`user_id`,`log_date`),
  CONSTRAINT `pregnancy_symptom_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `pregnancy_weekly_content` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `week_number` int(11) NOT NULL,
  `fetal_development` json DEFAULT NULL,
  `mother_body_changes` json DEFAULT NULL,
  `dos_and_donts` json DEFAULT NULL,
  `care_plan` json DEFAULT NULL,
  `body_adaptation` json DEFAULT NULL,
  `emotional_status` json DEFAULT NULL,
  `key_nutrition` json DEFAULT NULL,
  `physical_activity` json DEFAULT NULL,
  `tests_and_checkups` json DEFAULT NULL,
  `faq` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_weekly_content_week_number_unique` (`week_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `pregnancy_weekly_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `pregnancy_week` int(11) NOT NULL,
  `weight` decimal(5,2) DEFAULT NULL,
  `swelling_locations` json DEFAULT NULL,
  `has_swelling` tinyint(1) DEFAULT NULL,
  `has_shortness_of_breath` tinyint(1) DEFAULT NULL,
  `has_blood_pressure_device` tinyint(1) NOT NULL DEFAULT 0,
  `systolic_pressure` int(11) DEFAULT NULL,
  `diastolic_pressure` int(11) DEFAULT NULL,
  `fasting_blood_sugar` decimal(5,2) DEFAULT NULL,
  `post_meal_blood_sugar` decimal(5,2) DEFAULT NULL,
  `overall_mood` varchar(255) DEFAULT NULL,
  `has_anxiety` tinyint(1) DEFAULT NULL,
  `anxiety_severity` varchar(255) DEFAULT NULL,
  `has_mood_swings` tinyint(1) DEFAULT NULL,
  `mood_swings_severity` varchar(255) DEFAULT NULL,
  `has_depression_feelings` tinyint(1) DEFAULT NULL,
  `depression_severity` varchar(255) DEFAULT NULL,
  `notes` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_weekly_logs_user_id_pregnancy_week_unique` (`user_id`,`pregnancy_week`),
  CONSTRAINT `pregnancy_weekly_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `recommendations` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(255) DEFAULT NULL,
  `type` varchar(255) NOT NULL DEFAULT 'general',
  `title` json DEFAULT NULL,
  `text` json NOT NULL,
  `cycle_phase` varchar(255) DEFAULT NULL,
  `cycle_subphases` json DEFAULT NULL,
  `symptom_trigger` varchar(255) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `recommendations_key_unique` (`key`),
  KEY `recommendations_is_active_cycle_phase_index` (`is_active`,`cycle_phase`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `reminders` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `type` varchar(255) NOT NULL,
  `title` varchar(255) NOT NULL,
  `subtitle` varchar(255) DEFAULT NULL,
  `notes` text DEFAULT NULL,
  `scheduled_at` datetime DEFAULT NULL,
  `recurrence` varchar(255) NOT NULL DEFAULT 'none',
  `recurrence_time` time DEFAULT NULL,
  `starts_on` date DEFAULT NULL,
  `ends_on` date DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `meta` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `reminders_user_id_type_is_active_index` (`user_id`,`type`,`is_active`),
  KEY `reminders_user_id_scheduled_at_index` (`user_id`,`scheduled_at`),
  CONSTRAINT `reminders_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `sessions` (
  `id` varchar(255) NOT NULL,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `ip_address` varchar(45) DEFAULT NULL,
  `user_agent` text DEFAULT NULL,
  `payload` longtext NOT NULL,
  `last_activity` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `sessions_user_id_index` (`user_id`),
  KEY `sessions_last_activity_index` (`last_activity`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `task_templates` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(255) NOT NULL,
  `title` json NOT NULL,
  `description` json DEFAULT NULL,
  `category` varchar(255) NOT NULL,
  `icon` varchar(255) DEFAULT NULL,
  `cycle_phase` varchar(255) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `task_templates_key_unique` (`key`),
  KEY `task_templates_is_active_cycle_phase_index` (`is_active`,`cycle_phase`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_challenge_completions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `challenge_id` bigint(20) unsigned NOT NULL,
  `completion_date` date NOT NULL,
  `completed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_challenge_unique_per_day` (`user_id`,`challenge_id`,`completion_date`),
  KEY `user_challenge_completions_challenge_id_foreign` (`challenge_id`),
  KEY `user_challenge_completions_user_id_completion_date_index` (`user_id`,`completion_date`),
  CONSTRAINT `user_challenge_completions_challenge_id_foreign` FOREIGN KEY (`challenge_id`) REFERENCES `challenges` (`id`) ON DELETE CASCADE,
  CONSTRAINT `user_challenge_completions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_notifications` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `type` varchar(255) NOT NULL,
  `title` json NOT NULL,
  `body` json DEFAULT NULL,
  `action_url` varchar(255) DEFAULT NULL,
  `data` json DEFAULT NULL,
  `read_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_notifications_user_id_read_at_index` (`user_id`,`read_at`),
  CONSTRAINT `user_notifications_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_profiles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `birthday` date DEFAULT NULL,
  `weight` decimal(5,2) DEFAULT NULL,
  `height` smallint(5) unsigned DEFAULT NULL,
  `period_duration` tinyint(3) unsigned DEFAULT NULL,
  `cycle_duration` tinyint(3) unsigned DEFAULT NULL,
  `last_period_start` date DEFAULT NULL,
  `user_goal` varchar(255) NOT NULL DEFAULT 'non_ttc',
  `pregnancy_intention` varchar(255) DEFAULT NULL,
  `chronic_conditions` json DEFAULT NULL,
  `subscription_type` varchar(255) NOT NULL DEFAULT 'free',
  `calculation_status` varchar(255) NOT NULL DEFAULT 'pending',
  `calculation_started_at` timestamp NULL DEFAULT NULL,
  `calculation_completed_at` timestamp NULL DEFAULT NULL,
  `calculation_version` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_profiles_user_id_foreign` (`user_id`),
  CONSTRAINT `user_profiles_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_task_completions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `task_template_id` bigint(20) unsigned NOT NULL,
  `completion_date` date NOT NULL,
  `completed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_task_unique_per_day` (`user_id`,`task_template_id`,`completion_date`),
  KEY `user_task_completions_task_template_id_foreign` (`task_template_id`),
  KEY `user_task_completions_user_id_completion_date_index` (`user_id`,`completion_date`),
  CONSTRAINT `user_task_completions_task_template_id_foreign` FOREIGN KEY (`task_template_id`) REFERENCES `task_templates` (`id`) ON DELETE CASCADE,
  CONSTRAINT `user_task_completions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- The Laravel migration 2026_08_22_000001_create_languages_table seeds the registry (LanguageRegistry::BOOTSTRAP)
-- so it is never empty. Timestamps are Tehran wall-clock like every other row (Iran has no DST since 2022).
INSERT IGNORE INTO `languages` (`code`, `name`, `english_name`, `direction`, `is_active`, `is_default`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('fa', 'فارسی', 'Persian', 'rtl', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('en', 'English', 'English', 'ltr', 1, 0, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
-- Unsupported: the baseline is the whole production schema. Dropping it would drop every user's data,
-- so rolling back past it is refused on purpose. Use a fresh database instead.
SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'goose: rolling back 00001_baseline is not supported';
