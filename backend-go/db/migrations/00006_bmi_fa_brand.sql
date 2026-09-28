-- 00006_bmi_fa_brand.sql — data only (task T-M2-31, deviation D-22): the Laravel-seeded fa
-- `message_contents` rows `bmi_message` / `normal` and `obese` name the app in Latin ("Ritme"); the Go
-- code defaults say «ریتمی» since T-M2-30, but a DB row wins over the code default, so stage (and prod
-- after cutover, T-M2-27) kept showing "Ritme".
--
-- Guarded: a row changes only while its `message` is byte-for-byte the old seeded default (CAST AS
-- BINARY, because utf8mb4_unicode_ci would ignore ZWNJ and case); an admin-edited text is left alone.
-- Re-running is a no-op. `updated_at` is not touched (a data fix, not an edit; and NOW() is not Tehran
-- wall-clock in the containers). Down reverses it with the same guard.
-- No Laravel twin: data only, `make schema-diff` (schema + seed row counts) is unaffected, and the goose
-- baseline seeds no `bmi_message` rows (fresh goose databases: 0 rows changed).
-- Test: db/migrations/bmi_fa_brand_int_test.go (`make test-int PKG=./db/migrations/...`).

-- +goose Up
UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.message', 'بر اساس قد و وزن وارد شده، در محدوده‌ی وزنی طبیعی قرار می‌گیری. این محدوده معمولاً برای سلامت عمومی و سیکل قاعدگی مناسب است. ریتمی تلاش می‌کند به حفظ این وضعیت کمک کند.')
 WHERE `group` = 'bmi_message' AND `item_key` = 'normal' AND `locale` = 'fa'
   AND CAST(JSON_VALUE(`payload`, '$.message') AS BINARY) = CAST('بر اساس قد و وزن وارد شده، در محدوده‌ی وزنی طبیعی قرار می‌گیری. این محدوده معمولاً برای سلامت عمومی و سیکل قاعدگی مناسب است. Ritme تلاش می‌کند به حفظ این وضعیت کمک کند.' AS BINARY);

UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.message', 'بر اساس قد و وزن وارد شده، در محدوده‌ی چاقی قرار می‌گیری. این می‌تواند در طولانی‌مدت روی سلامت قلب، فشار خون، قند و سیکل قاعدگی اثر بگذارد. اگر امکانش را داری، صحبت با پزشک یا کارشناس تغذیه می‌تواند خیلی کمک‌کننده باشد. در ریتمی سعی می‌کنیم با توصیه‌های کوچک قابل‌اجرا، به روند سلامتت کمک کنیم.')
 WHERE `group` = 'bmi_message' AND `item_key` = 'obese' AND `locale` = 'fa'
   AND CAST(JSON_VALUE(`payload`, '$.message') AS BINARY) = CAST('بر اساس قد و وزن وارد شده، در محدوده‌ی چاقی قرار می‌گیری. این می‌تواند در طولانی‌مدت روی سلامت قلب، فشار خون، قند و سیکل قاعدگی اثر بگذارد. اگر امکانش را داری، صحبت با پزشک یا کارشناس تغذیه می‌تواند خیلی کمک‌کننده باشد. در Ritme سعی می‌کنیم با توصیه‌های کوچک قابل‌اجرا، به روند سلامتت کمک کنیم.' AS BINARY);

-- +goose Down
UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.message', 'بر اساس قد و وزن وارد شده، در محدوده‌ی وزنی طبیعی قرار می‌گیری. این محدوده معمولاً برای سلامت عمومی و سیکل قاعدگی مناسب است. Ritme تلاش می‌کند به حفظ این وضعیت کمک کند.')
 WHERE `group` = 'bmi_message' AND `item_key` = 'normal' AND `locale` = 'fa'
   AND CAST(JSON_VALUE(`payload`, '$.message') AS BINARY) = CAST('بر اساس قد و وزن وارد شده، در محدوده‌ی وزنی طبیعی قرار می‌گیری. این محدوده معمولاً برای سلامت عمومی و سیکل قاعدگی مناسب است. ریتمی تلاش می‌کند به حفظ این وضعیت کمک کند.' AS BINARY);

UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.message', 'بر اساس قد و وزن وارد شده، در محدوده‌ی چاقی قرار می‌گیری. این می‌تواند در طولانی‌مدت روی سلامت قلب، فشار خون، قند و سیکل قاعدگی اثر بگذارد. اگر امکانش را داری، صحبت با پزشک یا کارشناس تغذیه می‌تواند خیلی کمک‌کننده باشد. در Ritme سعی می‌کنیم با توصیه‌های کوچک قابل‌اجرا، به روند سلامتت کمک کنیم.')
 WHERE `group` = 'bmi_message' AND `item_key` = 'obese' AND `locale` = 'fa'
   AND CAST(JSON_VALUE(`payload`, '$.message') AS BINARY) = CAST('بر اساس قد و وزن وارد شده، در محدوده‌ی چاقی قرار می‌گیری. این می‌تواند در طولانی‌مدت روی سلامت قلب، فشار خون، قند و سیکل قاعدگی اثر بگذارد. اگر امکانش را داری، صحبت با پزشک یا کارشناس تغذیه می‌تواند خیلی کمک‌کننده باشد. در ریتمی سعی می‌کنیم با توصیه‌های کوچک قابل‌اجرا، به روند سلامتت کمک کنیم.' AS BINARY);
