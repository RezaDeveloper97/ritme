-- 00009_catalog_items.sql — the admin-editable content catalog (CB-CORE-03, docs/canvas-build/catalog.md):
-- one table for every small clinical/content list of the canvas-build epics (warning signs, missed-pill steps,
-- FAQs, kit items, score items, layette templates, …) instead of one table per list. A row is one item of a
-- `group`, identified by `code` inside it; `title`/`body` are translatable JSON ({"fa": …, "en": …}, the key set
-- grows with `languages`), `audiences` a JSON list of mode/audience codes (NULL = everyone), `meta` free-form JSON
-- per group, `needs_review` the "[needs clinical review]" flag (DECISIONS #7).
--
-- Twin of backend/database/migrations/2026_09_30_000001_create_catalog_items_table.php (Laravel owns the prod
-- schema until T-M2-27); `make schema-diff` proves both build the same table. Spelled the way mariadb-dump prints
-- the Laravel-built table (JSON columns written as `json`, see migrations.md). IF NOT EXISTS: a database whose
-- Laravel half already created the table (prod at cutover) must not fail here.
--
-- Seed convention (each epic appends): a new goose migration `000NN_catalog_<group>.sql` with
-- `INSERT IGNORE INTO catalog_items (...)` keyed on (`group`, `code`) plus a data-only Laravel twin
-- (`insertOrIgnore`) so schema-diff row counts match. Never edit this file to add rows.

-- +goose Up
CREATE TABLE IF NOT EXISTS `catalog_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `group` varchar(64) NOT NULL,
  `code` varchar(64) NOT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `audiences` json DEFAULT NULL,
  `title` json NOT NULL,
  `body` json DEFAULT NULL,
  `meta` json DEFAULT NULL,
  `needs_review` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `catalog_items_group_code_unique` (`group`,`code`),
  KEY `catalog_items_group_is_active_sort_order_index` (`group`,`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `catalog_items`;
