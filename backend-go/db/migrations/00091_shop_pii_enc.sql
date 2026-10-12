-- 00091_shop_pii_enc.sql — shop buyer addresses and recipient phones encrypted at rest (user decision 2026-10-12).
--
-- shop_addresses  +pii_enc   ONE sealed blob (PRIVATE_NOTE_KEY AES-256-GCM, the internal/loss NoteBox, bound to the
--                            owner: "shop-address:<user_id>") holding label, recipient name, recipient phone, address
--                            line and postal code — the B-N10-04 partner lab request `payload_enc` pattern.
-- shop_orders     +pii_enc   the recipient/address SNAPSHOT the same way ("shop-order:<user_id>"): recipient name,
--                            phone, address text and postal code.
--
-- The app writes ONLY the sealed blob: the plaintext columns are written '' / NULL on every new or edited row (they
-- stay NOT NULL so this twin needs no column change). Reads open the blob first and fall back to the plaintext
-- columns for legacy rows written before this migration (no backfill here; an address is re-sealed when it is next
-- edited). city / province / is_default stay plain (coarse, non-identifying, used for shipping and the queue).
-- Without PRIVATE_NOTE_KEY outside local/testing the address, checkout and order-detail paths answer 503.
--
-- Twin of backend/database/migrations/2026_10_12_000091_add_pii_enc_to_shop_tables.php so `make schema-diff` stays
-- green; IF NOT EXISTS / IF EXISTS so it is a no-op where the Laravel twin already ran.

-- +goose Up
ALTER TABLE `shop_addresses` ADD COLUMN IF NOT EXISTS `pii_enc` longtext DEFAULT NULL AFTER `postal_code`;
ALTER TABLE `shop_orders` ADD COLUMN IF NOT EXISTS `pii_enc` longtext DEFAULT NULL AFTER `postal_code`;

-- +goose Down
ALTER TABLE `shop_orders` DROP COLUMN IF EXISTS `pii_enc`;
ALTER TABLE `shop_addresses` DROP COLUMN IF EXISTS `pii_enc`;
