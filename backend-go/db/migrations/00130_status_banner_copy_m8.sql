-- 00130_status_banner_copy_m8.sql — data only: the Today status banner copy follows the M8 Persian copy pass.
--
-- The M8 pass (tasks/M8, docs/copy-fa/STYLE.md) rewrote frontend/messages/fa/home.json, which is the source of truth
-- for the banner copy that 00112 seeded into `message_rules` (slot `status_banner`):
--
--   banner.end_missing  a.title  «پایان پریود هنوز ثبت نشده» → «پایان پریود ثبت نشده»            (home.nb.endMissing.title)
--   banner.end_missing  a.body   «…ثبتش کن؛ اگر ادامه دارد…» → «…ثبتش کن. اگر ادامه دارد…»       (home.nb.endMissing.text)
--   banner.overdue      a.body   «…نتیجهٔ آزمایش را ثبت کنی.» → «…نتیجهٔ تست بارداری را هم ثبت کنی.» (home.nb.banner.overdue.text)
--
-- Admin edits win: each field is replaced only where it still holds the exact 00112 text. The English translation,
-- the CTA and every other field are untouched; `updated_at` is not touched (a data fix, not an edit). Re-running is a
-- no-op. Down restores the 00112 text only where the field still holds the M8 text.
-- No Laravel twin: `message_rules` is a Go-only seed table (docs/go-migration/migrations.md, go_only_seeds) and no
-- schema changes. Only JSON_SET / JSON_EXTRACT / JSON_UNQUOTE, so sqlc's parser of the Up section is not affected.
-- Test: internal/messages/feed_int_test.go TestFeedInt_BannerSeed (`make test-int PKG=./internal/messages/...`).

-- +goose Up
UPDATE `message_rules`
   SET `content` = JSON_SET(`content`, '$.a.title', 'پایان پریود ثبت نشده')
 WHERE `rule_key` = 'banner.end_missing'
   AND JSON_UNQUOTE(JSON_EXTRACT(`content`, '$.a.title')) = 'پایان پریود هنوز ثبت نشده';

UPDATE `message_rules`
   SET `content` = JSON_SET(`content`, '$.a.body', 'اگر تمام شده، ثبتش کن. اگر ادامه دارد، کاری لازم نیست.')
 WHERE `rule_key` = 'banner.end_missing'
   AND JSON_UNQUOTE(JSON_EXTRACT(`content`, '$.a.body')) = 'اگر تمام شده، ثبتش کن؛ اگر ادامه دارد، کاری لازم نیست.';

UPDATE `message_rules`
   SET `content` = JSON_SET(`content`, '$.a.body', 'اگر پریودت شروع شده، ثبتش کن. اگر احتمال بارداری هست، می‌توانی نتیجهٔ تست بارداری را هم ثبت کنی.')
 WHERE `rule_key` = 'banner.overdue'
   AND JSON_UNQUOTE(JSON_EXTRACT(`content`, '$.a.body')) = 'اگر پریودت شروع شده، ثبتش کن. اگر احتمال بارداری هست، می‌توانی نتیجهٔ آزمایش را ثبت کنی.';

-- +goose Down
UPDATE `message_rules`
   SET `content` = JSON_SET(`content`, '$.a.title', 'پایان پریود هنوز ثبت نشده')
 WHERE `rule_key` = 'banner.end_missing'
   AND JSON_UNQUOTE(JSON_EXTRACT(`content`, '$.a.title')) = 'پایان پریود ثبت نشده';

UPDATE `message_rules`
   SET `content` = JSON_SET(`content`, '$.a.body', 'اگر تمام شده، ثبتش کن؛ اگر ادامه دارد، کاری لازم نیست.')
 WHERE `rule_key` = 'banner.end_missing'
   AND JSON_UNQUOTE(JSON_EXTRACT(`content`, '$.a.body')) = 'اگر تمام شده، ثبتش کن. اگر ادامه دارد، کاری لازم نیست.';

UPDATE `message_rules`
   SET `content` = JSON_SET(`content`, '$.a.body', 'اگر پریودت شروع شده، ثبتش کن. اگر احتمال بارداری هست، می‌توانی نتیجهٔ آزمایش را ثبت کنی.')
 WHERE `rule_key` = 'banner.overdue'
   AND JSON_UNQUOTE(JSON_EXTRACT(`content`, '$.a.body')) = 'اگر پریودت شروع شده، ثبتش کن. اگر احتمال بارداری هست، می‌توانی نتیجهٔ تست بارداری را هم ثبت کنی.';
