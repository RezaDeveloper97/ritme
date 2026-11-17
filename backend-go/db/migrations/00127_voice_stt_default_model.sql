-- 00127_voice_stt_default_model.sql — voice logging: the default speech-to-text model.
--
-- Voice logging runs on two model slots (internal/ai/route.go): `voice_stt` hears (audio), `voice_log` writes JSON.
-- This seeds the default of the first one — GPT-4o Transcribe on the ArvanCloud AI gateway (owner's choice,
-- 2026-11-16) — the same way 00125 seeded Gemini Flash for the first four slots. It is admin data from here on:
-- the model, its prices and the slot's assignment are all editable in the admin panel (هوش مصنوعی).
--
--   ai_models    gpt-4o-transcribe · provider arvan · capability audio. `upstream_model` is the name the gateway
--                knows it by; if the gateway lists it under another name, change it in the panel (or re-sync).
--                Prices are the model's public list prices in milli-USD per 1M tokens (text in 2.50, out 10.00,
--                audio in 6.00) — placeholders for the usage log until the gateway's own prices are entered.
--   ai_features  voice_stt → that model, fallback Gemini Flash (it has audio), so an environment whose Arvan
--                gateway is not configured keeps transcribing. The row is normally created at start-up
--                (credits.Catalog.Sync) without a model; here it is created with one, and a row that already
--                exists without a model gets this one. A model an admin already picked is never replaced.
--
-- Laravel twin: 2026_11_16_000127_seed_voice_stt_model.php (the same rows).

-- +goose Up
INSERT IGNORE INTO `ai_models` (`code`, `provider`, `upstream_model`, `title`, `capabilities`, `input_price_milli`, `output_price_milli`, `audio_price_milli`, `is_active`, `created_at`, `updated_at`) VALUES
  ('gpt-4o-transcribe', 'arvan', 'gpt-4o-transcribe', 'GPT-4o Transcribe', '["audio"]', 2500, 10000, 6000, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

INSERT IGNORE INTO `ai_features` (`key`, `title`, `model_id`, `fallback_model_id`, `credit_cost`, `is_active`, `sort_order`, `created_at`, `updated_at`)
  SELECT 'voice_stt', '{"fa":"تبدیل صدا به متن","en":"Speech to text"}',
         (SELECT `id` FROM `ai_models` WHERE `code` = 'gpt-4o-transcribe'),
         (SELECT `id` FROM `ai_models` WHERE `code` = 'gemini-flash'),
         1, 1, 5,
         CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30');

UPDATE `ai_features`
   SET `model_id` = (SELECT `id` FROM `ai_models` WHERE `code` = 'gpt-4o-transcribe'),
       `fallback_model_id` = COALESCE(`fallback_model_id`, (SELECT `id` FROM `ai_models` WHERE `code` = 'gemini-flash'))
 WHERE `key` = 'voice_stt' AND `model_id` IS NULL;

-- +goose Down
-- The slot goes back to "no model" only where it still points at the seeded one; the row itself stays (start-up
-- would re-create it anyway).
UPDATE `ai_features` SET `model_id` = NULL
 WHERE `key` = 'voice_stt' AND `model_id` = (SELECT `id` FROM `ai_models` WHERE `code` = 'gpt-4o-transcribe');
DELETE FROM `ai_models` WHERE `code` = 'gpt-4o-transcribe';
