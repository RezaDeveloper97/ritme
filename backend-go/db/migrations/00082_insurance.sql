-- 00082_insurance.sql — insurance self-tracking (canvas CB-INS-01, DECISIONS #12): the user keeps her own supplementary
-- health insurance policy, coverage caps, claims with a timeline she advances herself, and health questionnaire drafts
-- prefilled from her record. No insurer API: nothing here is ever sent to an insurer. The read-only «how it works» page
-- stays bloom B-N10-06 (info_sections group `insurance`, internal/content). Domain logic: internal/insurance.
--
-- insurance_insurers          admin catalog of insurer names (no logos, no partner adapter).
-- insurance_centres           admin list of contracted centres: type, coordinates, direct («قرارداد مستقیم») or
--                             reimbursed; insurer_id NULL = every insurer.
-- insurance_policies          one policy per user. The policy number and the payout IBAN are never stored in full:
--                             only their last four characters (number_last4, iban_last4).
-- insurance_policy_members    who the policy covers: self / spouse / child (optionally a bloom `children` row of the
--                             owner) / other.
-- insurance_coverages         caps per category (toman), deductible %, waiting period, covered / conditional / not.
-- insurance_claims            one claim: member, cost type, centre, day, invoice amount, status (= latest event),
--                             paid amount; the free-text note is sealed (PRIVATE_NOTE_KEY, notes_enc).
-- insurance_claim_events      the status timeline she maintains (submitted → reviewing → docs_requested + deadline →
--                             paid / rejected / appealed); note sealed.
-- insurance_claim_documents   links to her record documents (record_documents) — no files are stored here.
-- insurance_questionnaires    drafts prefilled from the record (answers JSON with from_record flags), confirmed by her,
--                             exported as a PDF drawn on the device; explanation sealed; timeline JSON she maintains.
--
-- Health / financial data: owner-only (every query is scoped by user_id), never logged. Codes are varchar validated in
-- Go. Twin of backend/database/migrations/2026_10_11_000082_create_insurance_tables.php (Laravel owns the prod schema
-- until T-M2-27), so `make schema-diff` stays green. IF NOT EXISTS so a database whose Laravel half already ran does
-- not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `insurance_insurers` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(40) NOT NULL,
  `name` varchar(120) NOT NULL,
  `website` varchar(255) DEFAULT NULL,
  `phone` varchar(32) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `insurance_insurers_code_unique` (`code`),
  KEY `insurance_insurers_is_active_sort_order_index` (`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `insurance_centres` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `insurer_id` bigint(20) unsigned DEFAULT NULL,
  `name` varchar(160) NOT NULL,
  `type` varchar(16) NOT NULL,
  `services` varchar(160) DEFAULT NULL,
  `city` varchar(80) DEFAULT NULL,
  `address` varchar(300) DEFAULT NULL,
  `phone` varchar(32) DEFAULT NULL,
  `lat` decimal(10,7) DEFAULT NULL,
  `lng` decimal(10,7) DEFAULT NULL,
  `is_direct` tinyint(1) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `insurance_centres_insurer_id_foreign` (`insurer_id`),
  KEY `insurance_centres_is_active_type_index` (`is_active`,`type`),
  CONSTRAINT `insurance_centres_insurer_id_foreign` FOREIGN KEY (`insurer_id`) REFERENCES `insurance_insurers` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `insurance_policies` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `insurer_id` bigint(20) unsigned DEFAULT NULL,
  `insurer_name` varchar(120) DEFAULT NULL,
  `plan_name` varchar(120) DEFAULT NULL,
  `number_last4` varchar(4) DEFAULT NULL,
  `valid_from` date DEFAULT NULL,
  `valid_to` date DEFAULT NULL,
  `claim_deadline_days` smallint(5) unsigned DEFAULT NULL,
  `iban_last4` varchar(4) DEFAULT NULL,
  `not_covered_note` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `insurance_policies_user_id_unique` (`user_id`),
  KEY `insurance_policies_insurer_id_foreign` (`insurer_id`),
  CONSTRAINT `insurance_policies_insurer_id_foreign` FOREIGN KEY (`insurer_id`) REFERENCES `insurance_insurers` (`id`) ON DELETE SET NULL,
  CONSTRAINT `insurance_policies_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `insurance_policy_members` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `policy_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(8) NOT NULL,
  `child_id` bigint(20) unsigned DEFAULT NULL,
  `name` varchar(64) NOT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `insurance_policy_members_child_id_foreign` (`child_id`),
  KEY `insurance_policy_members_policy_id_sort_order_index` (`policy_id`,`sort_order`),
  CONSTRAINT `insurance_policy_members_child_id_foreign` FOREIGN KEY (`child_id`) REFERENCES `children` (`id`) ON DELETE SET NULL,
  CONSTRAINT `insurance_policy_members_policy_id_foreign` FOREIGN KEY (`policy_id`) REFERENCES `insurance_policies` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `insurance_coverages` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `policy_id` bigint(20) unsigned NOT NULL,
  `category` varchar(16) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'covered',
  `cap_amount` bigint(20) unsigned DEFAULT NULL,
  `deductible_pct` tinyint(3) unsigned DEFAULT NULL,
  `waiting_days` smallint(5) unsigned DEFAULT NULL,
  `note` varchar(160) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `insurance_coverages_policy_id_category_unique` (`policy_id`,`category`),
  CONSTRAINT `insurance_coverages_policy_id_foreign` FOREIGN KEY (`policy_id`) REFERENCES `insurance_policies` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `insurance_claims` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `policy_id` bigint(20) unsigned NOT NULL,
  `member_id` bigint(20) unsigned DEFAULT NULL,
  `cost_type` varchar(16) NOT NULL,
  `title` varchar(120) DEFAULT NULL,
  `centre_id` bigint(20) unsigned DEFAULT NULL,
  `centre_name` varchar(160) DEFAULT NULL,
  `service_date` date NOT NULL,
  `amount` bigint(20) unsigned NOT NULL,
  `over_tariff` bigint(20) unsigned NOT NULL DEFAULT 0,
  `paid_amount` bigint(20) unsigned DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'submitted',
  `deadline_on` date DEFAULT NULL,
  `notes_enc` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `insurance_claims_policy_id_foreign` (`policy_id`),
  KEY `insurance_claims_member_id_foreign` (`member_id`),
  KEY `insurance_claims_centre_id_foreign` (`centre_id`),
  KEY `insurance_claims_user_id_service_date_index` (`user_id`,`service_date`),
  CONSTRAINT `insurance_claims_centre_id_foreign` FOREIGN KEY (`centre_id`) REFERENCES `insurance_centres` (`id`) ON DELETE SET NULL,
  CONSTRAINT `insurance_claims_member_id_foreign` FOREIGN KEY (`member_id`) REFERENCES `insurance_policy_members` (`id`) ON DELETE SET NULL,
  CONSTRAINT `insurance_claims_policy_id_foreign` FOREIGN KEY (`policy_id`) REFERENCES `insurance_policies` (`id`) ON DELETE CASCADE,
  CONSTRAINT `insurance_claims_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `insurance_claim_events` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `claim_id` bigint(20) unsigned NOT NULL,
  `status` varchar(16) NOT NULL,
  `occurred_on` date NOT NULL,
  `deadline_on` date DEFAULT NULL,
  `amount` bigint(20) unsigned DEFAULT NULL,
  `note_enc` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `insurance_claim_events_claim_id_occurred_on_index` (`claim_id`,`occurred_on`),
  CONSTRAINT `insurance_claim_events_claim_id_foreign` FOREIGN KEY (`claim_id`) REFERENCES `insurance_claims` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `insurance_claim_documents` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `claim_id` bigint(20) unsigned NOT NULL,
  `document_id` bigint(20) unsigned NOT NULL,
  `label` varchar(16) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `insurance_claim_documents_claim_id_document_id_unique` (`claim_id`,`document_id`),
  KEY `insurance_claim_documents_document_id_foreign` (`document_id`),
  CONSTRAINT `insurance_claim_documents_claim_id_foreign` FOREIGN KEY (`claim_id`) REFERENCES `insurance_claims` (`id`) ON DELETE CASCADE,
  CONSTRAINT `insurance_claim_documents_document_id_foreign` FOREIGN KEY (`document_id`) REFERENCES `record_documents` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `insurance_questionnaires` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `purpose` varchar(16) NOT NULL,
  `label` varchar(80) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'draft',
  `answers` json NOT NULL,
  `explanation_enc` text DEFAULT NULL,
  `timeline` json DEFAULT NULL,
  `confirmed_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `insurance_questionnaires_user_id_created_at_index` (`user_id`,`created_at`),
  CONSTRAINT `insurance_questionnaires_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `insurance_questionnaires`;
DROP TABLE IF EXISTS `insurance_claim_documents`;
DROP TABLE IF EXISTS `insurance_claim_events`;
DROP TABLE IF EXISTS `insurance_claims`;
DROP TABLE IF EXISTS `insurance_coverages`;
DROP TABLE IF EXISTS `insurance_policy_members`;
DROP TABLE IF EXISTS `insurance_policies`;
DROP TABLE IF EXISTS `insurance_centres`;
DROP TABLE IF EXISTS `insurance_insurers`;
