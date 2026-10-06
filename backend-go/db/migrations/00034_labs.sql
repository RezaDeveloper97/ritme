-- 00034_labs.sql — lab analysis «تحلیل آزمایش» (bloom B-N6-06; artboards nbl_Lab_* in c-health-record, nbl_An_Labs):
-- uploaded lab sheets (images / PDF), the markers read from them by the AI extractor (internal/ai, fake provider by
-- default), the user's verification and edits, the plain-language interpretation and per-marker trends. Domain logic:
-- internal/labs. Bloom owns 00034–00039; the canvas queue numbers from 00040.
--
-- lab_reports   one lab sheet of a user. source upload (files + AI extraction) | manual (typed in, no AI).
--               status queued → extracting → needs_review (the user verifies) → interpreting → ready, or failed
--               (error_code). progress 0–100 for the processing screen. quota_at = when the Plus use
--               (plus.lab_ai) was reserved for this upload; set back to NULL once it is refunded (a failed extraction),
--               so it is refunded at most once. interpretation = {summary, source ai|rules, generated_at, context}
--               (the flags, red flags and doctor questions are derived from the markers and the catalog on read).
--               interpret_count caps re-interpretations after edits. feedback 1 | -1 («این تحلیل مفید بود؟»).
-- lab_files     the uploaded pages: AES-256-GCM encrypted at rest (LAB_FILE_KEY, internal/labs/files) under
--               STORAGE_PATH/app/private/labs/<user_id>/ (0700 / 0600), never served publicly; removed with the
--               report, on request, and with the account.
-- lab_markers   one value: name as printed, catalog code (lab_markers group of catalog_items, NULL = not catalogued),
--               numeric value or value_text («Negative»), unit, the sheet's reference range (ref_low / ref_high /
--               ref_text), the extractor's confidence (NULL for typed values) and source extracted | edited | manual.
-- lab_jobs      the DB-backed job queue of the in-process worker (extract | interpret): pending → running (lease
--               locked_until; a crashed worker's job is picked up again once the lease expires) → done | failed,
--               attempts capped in Go. locale = the request language the job answers in (extraction hint,
--               interpretation).
--
-- catalog_items lab_markers (26 rows, fa + en, needs_review): common markers (CBC, iron, thyroid, vitamins, glucose,
--               lipids, hormones) with aliases, unit, typical adult female range, red-flag thresholds, factors, doctor
--               questions and «when to see a doctor». Admin-editable through the catalog admin (CB-CORE-03).
--               [needs clinical review]
--
-- Health data: every row is scoped to its user and never logged. Twin of
-- backend/database/migrations/2026_10_05_000034_create_lab_tables.php (Laravel owns the prod schema until T-M2-27),
-- so `make schema-diff` stays green. IF NOT EXISTS / INSERT IGNORE so a database whose Laravel half already ran
-- does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `lab_reports` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `source` varchar(8) NOT NULL,
  `category` varchar(16) NOT NULL DEFAULT 'blood',
  `title` varchar(120) DEFAULT NULL,
  `taken_on` date DEFAULT NULL,
  `fasting` tinyint(1) DEFAULT NULL,
  `lab_name` varchar(120) DEFAULT NULL,
  `status` varchar(16) NOT NULL,
  `progress` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `error_code` varchar(32) DEFAULT NULL,
  `quota_at` datetime DEFAULT NULL,
  `verified_at` datetime DEFAULT NULL,
  `interpretation` json DEFAULT NULL,
  `interpreted_at` datetime DEFAULT NULL,
  `interpret_count` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `feedback` tinyint(4) DEFAULT NULL,
  `feedback_note` varchar(500) DEFAULT NULL,
  `feedback_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `lab_reports_user_id_created_at_index` (`user_id`,`created_at`),
  CONSTRAINT `lab_reports_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `lab_files` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `lab_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `page` tinyint(3) unsigned NOT NULL,
  `mime` varchar(32) NOT NULL,
  `size_bytes` int(10) unsigned NOT NULL,
  `path` varchar(255) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `lab_files_user_id_foreign` (`user_id`),
  KEY `lab_files_lab_id_page_index` (`lab_id`,`page`),
  CONSTRAINT `lab_files_lab_id_foreign` FOREIGN KEY (`lab_id`) REFERENCES `lab_reports` (`id`) ON DELETE CASCADE,
  CONSTRAINT `lab_files_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `lab_markers` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `lab_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `code` varchar(64) DEFAULT NULL,
  `name` varchar(120) NOT NULL,
  `value` decimal(14,4) DEFAULT NULL,
  `value_text` varchar(32) DEFAULT NULL,
  `unit` varchar(24) DEFAULT NULL,
  `ref_low` decimal(14,4) DEFAULT NULL,
  `ref_high` decimal(14,4) DEFAULT NULL,
  `ref_text` varchar(64) DEFAULT NULL,
  `confidence` decimal(4,3) DEFAULT NULL,
  `source` varchar(10) NOT NULL,
  `sort_order` smallint(5) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `lab_markers_lab_id_sort_order_index` (`lab_id`,`sort_order`),
  KEY `lab_markers_user_id_code_index` (`user_id`,`code`),
  CONSTRAINT `lab_markers_lab_id_foreign` FOREIGN KEY (`lab_id`) REFERENCES `lab_reports` (`id`) ON DELETE CASCADE,
  CONSTRAINT `lab_markers_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `lab_jobs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `lab_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(16) NOT NULL,
  `locale` varchar(12) DEFAULT NULL,
  `status` varchar(16) NOT NULL,
  `attempts` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `available_at` datetime NOT NULL,
  `locked_until` datetime DEFAULT NULL,
  `last_error` varchar(64) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `lab_jobs_lab_id_foreign` (`lab_id`),
  KEY `lab_jobs_status_available_at_index` (`status`,`available_at`),
  CONSTRAINT `lab_jobs_lab_id_foreign` FOREIGN KEY (`lab_id`) REFERENCES `lab_reports` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('lab_markers', 'hemoglobin', 1, 1, NULL,
   '{"fa":"هموگلوبین","en":"Hemoglobin"}',
   '{"fa":"پروتئینی در گلبول‌های قرمز که اکسیژن را به همه بدن می‌رساند. عدد پایین می‌تواند نشانه کم‌خونی باشد.","en":"The protein in red blood cells that carries oxygen around the body. A low value can be a sign of anaemia."}',
   '{"category":"cbc","subtitle":{"fa":"حمل اکسیژن در خون","en":"Oxygen carrier in blood"},"aliases":["hemoglobin","haemoglobin","hb","hgb","هموگلوبین"],"unit":"g/dL","typical":{"low":12,"high":15.5,"text":"12–15.5"},"critical":{"urgent_low":7,"urgent_high":20},"low":{"factors":[{"fa":"خون‌ریزی ماهانه پریود، مخصوصاً پریودهای سنگین","en":"Monthly period blood loss, especially heavy periods"},{"fa":"ذخیره آهن کم یا دریافت کم آهن از غذا","en":"Low iron stores or little iron in the diet"},{"fa":"کمبود ویتامین B12 یا فولات","en":"Low vitamin B12 or folate"}],"questions":[{"fa":"آیا کم‌خونی دارم و علتش چیست؟","en":"Do I have anaemia, and what is causing it?"}],"see_doctor":{"fa":"اگر خستگی شدید، تنگی نفس، سرگیجه یا تپش قلب داری، زودتر به پزشک مراجعه کن.","en":"If you have severe tiredness, shortness of breath, dizziness or palpitations, see a doctor soon."}},"high":{"factors":[{"fa":"کم‌آبی بدن هنگام آزمایش","en":"Being dehydrated at the time of the test"},{"fa":"سیگار کشیدن یا زندگی در ارتفاع بالا","en":"Smoking or living at high altitude"}],"questions":[{"fa":"آیا لازم است آزمایش را تکرار کنم؟","en":"Should I repeat this test?"}],"see_doctor":{"fa":"برای بررسی علت با پزشک مشورت کن.","en":"Talk to a doctor to look into the cause."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'hematocrit', 2, 1, NULL,
   '{"fa":"هماتوکریت","en":"Hematocrit"}',
   '{"fa":"درصدی از حجم خون که گلبول‌های قرمز آن را تشکیل می‌دهند. معمولاً همراه هموگلوبین تفسیر می‌شود.","en":"The share of blood volume made up of red blood cells. It is usually read together with hemoglobin."}',
   '{"category":"cbc","subtitle":{"fa":"سهم گلبول قرمز در خون","en":"Red cell share of blood"},"aliases":["hematocrit","haematocrit","hct","pcv","هماتوکریت"],"unit":"%","typical":{"low":36,"high":46,"text":"36–46"},"critical":{"urgent_low":21},"low":{"factors":[{"fa":"کم‌خونی، مثلاً به دلیل کمبود آهن","en":"Anaemia, for example from low iron"},{"fa":"خون‌ریزی پریود","en":"Period blood loss"}],"questions":[{"fa":"آیا این عدد با هموگلوبین من هم‌خوانی دارد؟","en":"Does this match my hemoglobin result?"}],"see_doctor":{"fa":"همراه با هموگلوبین با پزشک بررسی کن.","en":"Review it with a doctor together with your hemoglobin."}},"high":{"factors":[{"fa":"کم‌آبی بدن","en":"Dehydration"},{"fa":"سیگار کشیدن","en":"Smoking"}],"questions":[{"fa":"آیا لازم است آزمایش را تکرار کنم؟","en":"Should I repeat this test?"}],"see_doctor":{"fa":"برای بررسی علت با پزشک مشورت کن.","en":"Talk to a doctor to look into the cause."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'rbc', 3, 1, NULL,
   '{"fa":"گلبول قرمز","en":"Red blood cells"}',
   '{"fa":"تعداد گلبول‌های قرمز در هر میکرولیتر خون؛ این سلول‌ها اکسیژن را جابه‌جا می‌کنند.","en":"The number of red blood cells per microlitre of blood; these cells carry oxygen."}',
   '{"category":"cbc","subtitle":{"fa":"شمار سلول‌های حامل اکسیژن","en":"Count of oxygen-carrying cells"},"aliases":["rbc","redbloodcells","redbloodcell","redcellcount","erythrocytes","گلبولقرمز"],"unit":"10^6/µL","typical":{"low":4.0,"high":5.2,"text":"4.0–5.2"},"low":{"factors":[{"fa":"کم‌خونی","en":"Anaemia"},{"fa":"کمبود آهن، B12 یا فولات","en":"Low iron, B12 or folate"}],"questions":[{"fa":"آیا به آزمایش آهن یا ویتامین‌ها نیاز دارم؟","en":"Do I need iron or vitamin tests?"}],"see_doctor":{"fa":"همراه با هموگلوبین با پزشک بررسی کن.","en":"Review it with a doctor together with your hemoglobin."}},"high":{"factors":[{"fa":"کم‌آبی بدن","en":"Dehydration"},{"fa":"سیگار کشیدن یا ارتفاع بالا","en":"Smoking or high altitude"}],"questions":[{"fa":"آیا لازم است آزمایش را تکرار کنم؟","en":"Should I repeat this test?"}],"see_doctor":{"fa":"برای بررسی علت با پزشک مشورت کن.","en":"Talk to a doctor to look into the cause."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'wbc', 4, 1, NULL,
   '{"fa":"گلبول سفید","en":"White blood cells"}',
   '{"fa":"سلول‌های دستگاه ایمنی که با عفونت مقابله می‌کنند. عفونت، استرس و برخی داروها این عدد را تغییر می‌دهند.","en":"Immune cells that fight infection. Infections, stress and some medicines change this number."}',
   '{"category":"cbc","subtitle":{"fa":"سلول‌های ایمنی","en":"Immune cells"},"aliases":["wbc","whitebloodcells","whitebloodcell","leukocytes","leucocytes","گلبولسفید"],"unit":"10^3/µL","typical":{"low":4.0,"high":11.0,"text":"4.0–11.0"},"critical":{"urgent_low":2,"urgent_high":30},"low":{"factors":[{"fa":"عفونت‌های ویروسی اخیر","en":"A recent viral infection"},{"fa":"برخی داروها","en":"Some medicines"}],"questions":[{"fa":"آیا باید آزمایش را تکرار کنم؟","en":"Should I repeat this test?"}],"see_doctor":{"fa":"اگر تب یا عفونت‌های مکرر داری، زودتر به پزشک مراجعه کن.","en":"If you have a fever or frequent infections, see a doctor soon."}},"high":{"factors":[{"fa":"عفونت یا التهاب","en":"An infection or inflammation"},{"fa":"استرس، ورزش شدید یا بارداری","en":"Stress, hard exercise or pregnancy"}],"questions":[{"fa":"آیا نشانه عفونت است؟","en":"Could this be a sign of infection?"}],"see_doctor":{"fa":"اگر تب، درد یا علائم عفونت داری، با پزشک تماس بگیر.","en":"If you have a fever, pain or signs of infection, contact a doctor."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'platelets', 5, 1, NULL,
   '{"fa":"پلاکت","en":"Platelets"}',
   '{"fa":"سلول‌های کوچکی که به لخته شدن خون و بند آمدن خون‌ریزی کمک می‌کنند.","en":"Small cells that help blood clot and stop bleeding."}',
   '{"category":"cbc","subtitle":{"fa":"لخته شدن خون","en":"Blood clotting"},"aliases":["platelets","platelet","plt","plateletcount","پلاکت"],"unit":"10^3/µL","typical":{"low":150,"high":450,"text":"150–450"},"critical":{"urgent_low":50,"urgent_high":1000},"low":{"factors":[{"fa":"عفونت‌های ویروسی","en":"Viral infections"},{"fa":"برخی داروها","en":"Some medicines"}],"questions":[{"fa":"آیا خطر خون‌ریزی دارم؟","en":"Am I at risk of bleeding?"}],"see_doctor":{"fa":"اگر کبودی بی‌دلیل، خون‌ریزی لثه یا پریود خیلی سنگین داری، زودتر مراجعه کن.","en":"If you bruise easily, your gums bleed or your periods are very heavy, see a doctor soon."}},"high":{"factors":[{"fa":"کمبود آهن","en":"Low iron"},{"fa":"التهاب یا عفونت","en":"Inflammation or infection"}],"questions":[{"fa":"آیا به آزمایش آهن نیاز دارم؟","en":"Do I need an iron test?"}],"see_doctor":{"fa":"برای بررسی علت با پزشک مشورت کن.","en":"Talk to a doctor to look into the cause."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'mcv', 6, 1, NULL,
   '{"fa":"حجم متوسط گلبول قرمز (MCV)","en":"Mean cell volume (MCV)"}',
   '{"fa":"اندازه متوسط گلبول‌های قرمز. کوچک بودن آن معمولاً با کمبود آهن و بزرگ بودنش با کمبود B12 یا فولات مرتبط است.","en":"The average size of red blood cells. Small cells often go with low iron, large cells with low B12 or folate."}',
   '{"category":"cbc","subtitle":{"fa":"اندازه گلبول قرمز","en":"Red cell size"},"aliases":["mcv","meancellvolume","meancorpuscularvolume"],"unit":"fL","typical":{"low":80,"high":100,"text":"80–100"},"low":{"factors":[{"fa":"کمبود آهن","en":"Low iron"},{"fa":"برخی صفات ارثی خون مثل تالاسمی مینور","en":"Some inherited blood traits such as thalassaemia minor"}],"questions":[{"fa":"آیا به آزمایش فریتین یا الکتروفورز نیاز دارم؟","en":"Do I need a ferritin or haemoglobin electrophoresis test?"}],"see_doctor":{"fa":"همراه با هموگلوبین و فریتین با پزشک بررسی کن.","en":"Review it with a doctor together with hemoglobin and ferritin."}},"high":{"factors":[{"fa":"کمبود ویتامین B12 یا فولات","en":"Low vitamin B12 or folate"},{"fa":"مصرف الکل یا برخی داروها","en":"Alcohol or some medicines"}],"questions":[{"fa":"آیا باید B12 و فولات را بررسی کنم؟","en":"Should I check B12 and folate?"}],"see_doctor":{"fa":"برای بررسی علت با پزشک مشورت کن.","en":"Talk to a doctor to look into the cause."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'ferritin', 7, 1, NULL,
   '{"fa":"فریتین","en":"Ferritin"}',
   '{"fa":"پروتئینی که آهن را در بدن ذخیره می‌کند. عدد پایین معمولاً یعنی ذخیره آهن کم است، حتی اگر هموگلوبین هنوز خیلی پایین نیامده باشد.","en":"A protein that stores iron in the body. A low value usually means low iron stores, even before hemoglobin drops much."}',
   '{"category":"iron","subtitle":{"fa":"ذخیره آهن بدن","en":"Body iron stores"},"aliases":["ferritin","serumferritin","فریتین"],"unit":"ng/mL","typical":{"low":15,"high":150,"text":"15–150"},"low":{"factors":[{"fa":"خون‌ریزی ماهانه پریود","en":"Monthly period blood loss"},{"fa":"دریافت کم آهن از غذا","en":"Little iron in the diet"},{"fa":"جذب کمتر آهن در برخی شرایط گوارشی","en":"Poorer iron absorption in some digestive conditions"}],"questions":[{"fa":"آیا برای ذخیره آهن کم باید مکمل آهن بگیرم؟ چه مقدار و تا کی؟","en":"Should I take an iron supplement for low iron stores? How much and for how long?"},{"fa":"چند وقت دیگر آزمایش را تکرار کنم؟","en":"When should I repeat the test?"}],"see_doctor":{"fa":"برای تصمیم درباره مکمل آهن و دوز آن با پزشک مشورت کن. اگر خستگی شدید، تنگی نفس یا تپش قلب داری، زودتر مراجعه کن.","en":"Talk to a doctor before deciding on an iron supplement and its dose. If you have severe tiredness, shortness of breath or palpitations, go sooner."}},"high":{"factors":[{"fa":"التهاب یا عفونت اخیر","en":"Recent inflammation or infection"},{"fa":"مصرف مکمل آهن","en":"Taking iron supplements"}],"questions":[{"fa":"آیا باید مصرف مکمل آهن را قطع کنم؟","en":"Should I stop my iron supplement?"}],"see_doctor":{"fa":"برای بررسی علت با پزشک مشورت کن.","en":"Talk to a doctor to look into the cause."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'iron', 8, 1, NULL,
   '{"fa":"آهن سرم","en":"Serum iron"}',
   '{"fa":"مقدار آهنی که در لحظه آزمایش در خون است. در طول روز و با غذا تغییر می‌کند و معمولاً همراه فریتین تفسیر می‌شود.","en":"The amount of iron in the blood at the time of the test. It changes through the day and with meals, so it is read with ferritin."}',
   '{"category":"iron","subtitle":{"fa":"آهن در گردش خون","en":"Circulating iron"},"aliases":["iron","serumiron","fe","آهن","آهنسرم"],"unit":"µg/dL","typical":{"low":50,"high":170,"text":"50–170"},"low":{"factors":[{"fa":"ذخیره آهن کم","en":"Low iron stores"},{"fa":"خون‌ریزی پریود","en":"Period blood loss"}],"questions":[{"fa":"آیا فریتین من هم پایین است؟","en":"Is my ferritin low as well?"}],"see_doctor":{"fa":"همراه با فریتین با پزشک بررسی کن.","en":"Review it with a doctor together with ferritin."}},"high":{"factors":[{"fa":"مصرف مکمل آهن پیش از آزمایش","en":"Taking iron before the test"}],"questions":[{"fa":"آیا لازم است آزمایش را ناشتا تکرار کنم؟","en":"Should I repeat the test fasting?"}],"see_doctor":{"fa":"برای بررسی علت با پزشک مشورت کن.","en":"Talk to a doctor to look into the cause."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'tsh', 9, 1, NULL,
   '{"fa":"TSH","en":"TSH"}',
   '{"fa":"هورمونی از غده هیپوفیز که کار تیروئید را تنظیم می‌کند. TSH بالا معمولاً یعنی تیروئید کم‌کار است و TSH پایین یعنی پرکار.","en":"A pituitary hormone that controls the thyroid. A high TSH usually means an underactive thyroid, a low TSH an overactive one."}',
   '{"category":"thyroid","subtitle":{"fa":"تنظیم‌کننده تیروئید","en":"Thyroid regulator"},"aliases":["tsh","thyrotropin","thyroidstimulatinghormone"],"unit":"mIU/L","typical":{"low":0.4,"high":4.0,"text":"0.4–4.0"},"critical":{"soon_high":10,"soon_low":0.1},"low":{"factors":[{"fa":"پرکاری تیروئید","en":"An overactive thyroid"},{"fa":"مصرف زیاد داروی تیروئید","en":"Too high a dose of thyroid medicine"}],"questions":[{"fa":"آیا به آزمایش T4 آزاد نیاز دارم؟","en":"Do I need a free T4 test?"}],"see_doctor":{"fa":"اگر تپش قلب، کاهش وزن بی‌دلیل یا لرزش دست داری، زودتر مراجعه کن.","en":"If you have palpitations, unexplained weight loss or shaky hands, see a doctor soon."}},"high":{"factors":[{"fa":"کم‌کاری تیروئید","en":"An underactive thyroid"},{"fa":"دوز ناکافی داروی تیروئید","en":"Too low a dose of thyroid medicine"}],"questions":[{"fa":"آیا تیروئید من روی سیکل یا اقدام به بارداری اثر دارد؟","en":"Could my thyroid affect my cycle or trying to conceive?"},{"fa":"آیا به آزمایش T4 آزاد و آنتی‌بادی تیروئید نیاز دارم؟","en":"Do I need free T4 and thyroid antibody tests?"}],"see_doctor":{"fa":"با پزشک درباره تکرار آزمایش و نیاز به درمان صحبت کن؛ در اقدام به بارداری یا بارداری زودتر.","en":"Talk to a doctor about repeating the test and whether treatment is needed — sooner if you are trying to conceive or pregnant."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'free_t4', 10, 1, NULL,
   '{"fa":"T4 آزاد","en":"Free T4"}',
   '{"fa":"هورمون اصلی تیروئید که آزادانه در خون می‌چرخد. همراه TSH وضعیت تیروئید را نشان می‌دهد.","en":"The main thyroid hormone circulating freely in the blood. Together with TSH it shows how the thyroid is working."}',
   '{"category":"thyroid","subtitle":{"fa":"هورمون تیروئید","en":"Thyroid hormone"},"aliases":["freet4","ft4","t4free","freethyroxine"],"unit":"ng/dL","typical":{"low":0.8,"high":1.8,"text":"0.8–1.8"},"low":{"factors":[{"fa":"کم‌کاری تیروئید","en":"An underactive thyroid"}],"questions":[{"fa":"آیا به درمان تیروئید نیاز دارم؟","en":"Do I need thyroid treatment?"}],"see_doctor":{"fa":"همراه با TSH با پزشک بررسی کن.","en":"Review it with a doctor together with TSH."}},"high":{"factors":[{"fa":"پرکاری تیروئید","en":"An overactive thyroid"},{"fa":"مصرف زیاد داروی تیروئید","en":"Too high a dose of thyroid medicine"}],"questions":[{"fa":"آیا دوز داروی تیروئید من مناسب است؟","en":"Is my thyroid medicine dose right?"}],"see_doctor":{"fa":"همراه با TSH با پزشک بررسی کن.","en":"Review it with a doctor together with TSH."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'vitamin_d', 11, 1, NULL,
   '{"fa":"ویتامین D","en":"Vitamin D"}',
   '{"fa":"ویتامینی که برای سلامت استخوان، عضله و ایمنی لازم است و بیشتر با نور خورشید ساخته می‌شود.","en":"A vitamin needed for bones, muscles and immunity, made mostly with sunlight."}',
   '{"category":"vitamin","subtitle":{"fa":"استخوان و ایمنی","en":"Bones and immunity"},"aliases":["vitamind","vitd","25ohd","25ohvitamind","25hydroxyvitamind","vitamind25oh","vitamind3","ویتامیند","ویتامیندی"],"unit":"ng/mL","typical":{"low":30,"high":100,"text":"30–100"},"critical":{"soon_low":10},"low":{"factors":[{"fa":"نور کم خورشید یا پوشش زیاد پوست","en":"Little sunlight or covered skin"},{"fa":"دریافت کم از غذا","en":"Little vitamin D in the diet"}],"questions":[{"fa":"چه مقدار ویتامین D مصرف کنم و تا کی؟","en":"How much vitamin D should I take, and for how long?"},{"fa":"قبل از بارداری ویتامین D چقدر باید باشد؟","en":"What should my vitamin D be before pregnancy?"}],"see_doctor":{"fa":"برای انتخاب دوز مکمل با پزشک مشورت کن.","en":"Talk to a doctor to choose a supplement dose."}},"high":{"factors":[{"fa":"مصرف زیاد مکمل ویتامین D","en":"Taking a lot of vitamin D supplements"}],"questions":[{"fa":"آیا باید مکمل را کم یا قطع کنم؟","en":"Should I lower or stop my supplement?"}],"see_doctor":{"fa":"درباره دوز مکمل با پزشک صحبت کن.","en":"Talk to a doctor about your supplement dose."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'b12', 12, 1, NULL,
   '{"fa":"ویتامین B12","en":"Vitamin B12"}',
   '{"fa":"ویتامینی که برای ساخت گلبول قرمز و سلامت اعصاب لازم است و بیشتر در غذاهای حیوانی هست.","en":"A vitamin needed for red blood cells and nerves, found mostly in animal foods."}',
   '{"category":"vitamin","subtitle":{"fa":"خون و اعصاب","en":"Blood and nerves"},"aliases":["b12","vitaminb12","vitb12","cobalamin","cyanocobalamin","ویتامینب12"],"unit":"pg/mL","typical":{"low":200,"high":900,"text":"200–900"},"critical":{"soon_low":150},"low":{"factors":[{"fa":"رژیم گیاه‌خواری","en":"A vegetarian or vegan diet"},{"fa":"جذب کم در برخی بیماری‌های گوارشی یا با برخی داروها مثل متفورمین","en":"Poor absorption in some digestive conditions or with some medicines such as metformin"}],"questions":[{"fa":"آیا به مکمل B12 نیاز دارم؟","en":"Do I need a B12 supplement?"}],"see_doctor":{"fa":"اگر بی‌حسی یا گزگز دست و پا داری، زودتر مراجعه کن.","en":"If you have numbness or tingling in your hands or feet, see a doctor soon."}},"high":{"factors":[{"fa":"مصرف مکمل B12","en":"Taking B12 supplements"}],"questions":[{"fa":"آیا لازم است کاری انجام دهم؟","en":"Do I need to do anything about it?"}],"see_doctor":{"fa":"اگر مکمل مصرف نمی‌کنی، با پزشک مشورت کن.","en":"If you do not take supplements, talk to a doctor."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'folate', 13, 1, NULL,
   '{"fa":"فولات (اسید فولیک)","en":"Folate"}',
   '{"fa":"ویتامین گروه B که برای ساخت سلول‌ها لازم است و پیش از بارداری و در ماه‌های اول آن اهمیت ویژه دارد.","en":"A B vitamin needed to make cells; it matters especially before and early in pregnancy."}',
   '{"category":"vitamin","subtitle":{"fa":"ساخت سلول و بارداری","en":"Cell growth and pregnancy"},"aliases":["folate","folicacid","serumfolate","اسیدفولیک","فولات"],"unit":"ng/mL","typical":{"low":3,"text":"≥ 3"},"low":{"factors":[{"fa":"دریافت کم سبزیجات برگ‌سبز و حبوبات","en":"Few leafy greens and legumes in the diet"},{"fa":"برخی داروها","en":"Some medicines"}],"questions":[{"fa":"چه مقدار اسید فولیک باید مصرف کنم؟","en":"How much folic acid should I take?"}],"see_doctor":{"fa":"مخصوصاً اگر اقدام به بارداری داری، با پزشک درباره مکمل صحبت کن.","en":"Talk to a doctor about a supplement, especially if you are trying to conceive."}},"high":{"factors":[{"fa":"مصرف مکمل","en":"Taking supplements"}],"questions":[{"fa":"آیا لازم است کاری انجام دهم؟","en":"Do I need to do anything about it?"}],"see_doctor":{"fa":"معمولاً نگران‌کننده نیست؛ در صورت شک با پزشک صحبت کن.","en":"Usually not a concern; ask a doctor if unsure."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'glucose_fasting', 14, 1, NULL,
   '{"fa":"قند ناشتا","en":"Fasting glucose"}',
   '{"fa":"مقدار قند خون بعد از حداقل ۸ ساعت ناشتایی. برای بررسی دیابت و پیش‌دیابت استفاده می‌شود.","en":"Blood sugar after at least 8 hours without food. Used to check for diabetes and prediabetes."}',
   '{"category":"glucose","subtitle":{"fa":"قند خون","en":"Blood sugar"},"aliases":["fbs","fastingglucose","fastingbloodsugar","glucosefasting","fastingplasmaglucose","fpg","glucose","bloodsugar","قندناشتا","قندخونناشتا"],"unit":"mg/dL","typical":{"low":70,"high":100,"text":"70–100"},"critical":{"urgent_low":54,"urgent_high":300,"soon_high":126},"low":{"factors":[{"fa":"ناشتایی طولانی یا ورزش شدید","en":"A long fast or hard exercise"},{"fa":"برخی داروهای دیابت","en":"Some diabetes medicines"}],"questions":[{"fa":"آیا افت قند دارم؟","en":"Do I have low blood sugar episodes?"}],"see_doctor":{"fa":"اگر لرزش، تعریق یا گیجی داری، فوراً چیزی شیرین بخور و با پزشک تماس بگیر.","en":"If you feel shaky, sweaty or confused, eat something sugary right away and contact a doctor."}},"high":{"factors":[{"fa":"ناشتا نبودن هنگام آزمایش","en":"Not fasting before the test"},{"fa":"پیش‌دیابت یا دیابت","en":"Prediabetes or diabetes"},{"fa":"سندرم تخمدان پلی‌کیستیک","en":"Polycystic ovary syndrome"}],"questions":[{"fa":"آیا به آزمایش HbA1c نیاز دارم؟","en":"Do I need an HbA1c test?"},{"fa":"چه تغییری در غذا و ورزش کمک می‌کند؟","en":"What changes in food and exercise would help?"}],"see_doctor":{"fa":"برای تکرار آزمایش و بررسی پیش‌دیابت با پزشک مشورت کن.","en":"Talk to a doctor about repeating the test and checking for prediabetes."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'hba1c', 15, 1, NULL,
   '{"fa":"هموگلوبین A1c","en":"HbA1c"}',
   '{"fa":"میانگین قند خون در حدود سه ماه گذشته را نشان می‌دهد و به ناشتایی نیاز ندارد.","en":"Shows your average blood sugar over about the last three months; no fasting needed."}',
   '{"category":"glucose","subtitle":{"fa":"میانگین قند ۳ ماه","en":"3-month sugar average"},"aliases":["hba1c","a1c","glycatedhemoglobin","glycosylatedhemoglobin","hemoglobina1c","haemoglobina1c","هموگلوبینa1c"],"unit":"%","typical":{"low":4.0,"high":5.6,"text":"4.0–5.6"},"critical":{"soon_high":6.5},"low":{"factors":[{"fa":"کم‌خونی یا خون‌ریزی اخیر می‌تواند عدد را کم نشان دهد","en":"Anaemia or recent blood loss can make the value read low"}],"questions":[{"fa":"آیا این عدد برای من قابل اعتماد است؟","en":"Is this result reliable for me?"}],"see_doctor":{"fa":"در صورت شک با پزشک صحبت کن.","en":"Ask a doctor if unsure."}},"high":{"factors":[{"fa":"پیش‌دیابت یا دیابت","en":"Prediabetes or diabetes"}],"questions":[{"fa":"آیا پیش‌دیابت یا دیابت دارم؟","en":"Do I have prediabetes or diabetes?"},{"fa":"پیش از بارداری قندم باید در چه حدی باشد؟","en":"What should my sugar be before pregnancy?"}],"see_doctor":{"fa":"برای بررسی و برنامه کنترل قند با پزشک مشورت کن.","en":"Talk to a doctor about a check-up and a plan to manage blood sugar."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'cholesterol_total', 16, 1, NULL,
   '{"fa":"کلسترول کل","en":"Total cholesterol"}',
   '{"fa":"مجموع چربی‌های کلسترولی خون. همراه LDL و HDL برای ارزیابی سلامت قلب و عروق تفسیر می‌شود.","en":"All the cholesterol in your blood. Read with LDL and HDL to judge heart and blood vessel health."}',
   '{"category":"lipid","subtitle":{"fa":"چربی خون","en":"Blood fats"},"aliases":["cholesterol","totalcholesterol","cholesteroltotal","chol","tc","کلسترول","کلسترولکل"],"unit":"mg/dL","typical":{"high":200,"text":"< 200"},"low":{"factors":[{"fa":"معمولاً نگران‌کننده نیست","en":"Usually not a concern"}],"questions":[{"fa":"آیا لازم است کاری انجام دهم؟","en":"Do I need to do anything about it?"}],"see_doctor":{"fa":"در صورت شک با پزشک صحبت کن.","en":"Ask a doctor if unsure."}},"high":{"factors":[{"fa":"رژیم پرچرب و کم‌تحرکی","en":"A high-fat diet and little exercise"},{"fa":"سابقه خانوادگی","en":"Family history"},{"fa":"کم‌کاری تیروئید","en":"An underactive thyroid"}],"questions":[{"fa":"خطر بیماری قلبی من چقدر است؟","en":"What is my heart disease risk?"}],"see_doctor":{"fa":"برای ارزیابی خطر قلبی و تغییر سبک زندگی با پزشک مشورت کن.","en":"Talk to a doctor about your heart risk and lifestyle changes."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'ldl', 17, 1, NULL,
   '{"fa":"LDL (کلسترول بد)","en":"LDL cholesterol"}',
   '{"fa":"کلسترولی که بالا بودنش با رسوب در رگ‌ها و خطر بیماری قلبی مرتبط است.","en":"Cholesterol that, when high, builds up in arteries and raises heart disease risk."}',
   '{"category":"lipid","subtitle":{"fa":"کلسترول بد","en":"Bad cholesterol"},"aliases":["ldl","ldlc","ldlcholesterol","کلسترولبد"],"unit":"mg/dL","typical":{"high":130,"text":"< 130"},"critical":{"soon_high":190},"low":{"factors":[{"fa":"معمولاً نگران‌کننده نیست","en":"Usually not a concern"}],"questions":[{"fa":"آیا لازم است کاری انجام دهم؟","en":"Do I need to do anything about it?"}],"see_doctor":{"fa":"در صورت شک با پزشک صحبت کن.","en":"Ask a doctor if unsure."}},"high":{"factors":[{"fa":"رژیم پرچرب","en":"A high-fat diet"},{"fa":"سابقه خانوادگی","en":"Family history"}],"questions":[{"fa":"آیا به دارو نیاز دارم یا تغییر سبک زندگی کافی است؟","en":"Do I need medicine, or are lifestyle changes enough?"}],"see_doctor":{"fa":"برای ارزیابی خطر قلبی با پزشک مشورت کن.","en":"Talk to a doctor about your heart risk."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'hdl', 18, 1, NULL,
   '{"fa":"HDL (کلسترول خوب)","en":"HDL cholesterol"}',
   '{"fa":"کلسترولی که به پاک کردن چربی از رگ‌ها کمک می‌کند؛ عدد بالاتر معمولاً بهتر است.","en":"Cholesterol that helps clear fat from arteries; higher is usually better."}',
   '{"category":"lipid","subtitle":{"fa":"کلسترول خوب","en":"Good cholesterol"},"aliases":["hdl","hdlc","hdlcholesterol","کلسترولخوب"],"unit":"mg/dL","typical":{"low":50,"text":"≥ 50"},"low":{"factors":[{"fa":"کم‌تحرکی","en":"Little physical activity"},{"fa":"سیگار کشیدن","en":"Smoking"},{"fa":"سندرم تخمدان پلی‌کیستیک","en":"Polycystic ovary syndrome"}],"questions":[{"fa":"چطور می‌توانم HDL را بالا ببرم؟","en":"How can I raise my HDL?"}],"see_doctor":{"fa":"همراه با بقیه چربی‌ها با پزشک بررسی کن.","en":"Review it with a doctor together with your other lipids."}},"high":{"factors":[{"fa":"معمولاً نشانه خوبی است","en":"Usually a good sign"}],"questions":[{"fa":"آیا لازم است کاری انجام دهم؟","en":"Do I need to do anything about it?"}],"see_doctor":{"fa":"در صورت شک با پزشک صحبت کن.","en":"Ask a doctor if unsure."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'triglycerides', 19, 1, NULL,
   '{"fa":"تری‌گلیسرید","en":"Triglycerides"}',
   '{"fa":"نوعی چربی خون که با غذا، قند و الکل بالا می‌رود. برای آزمایش معمولاً ناشتایی لازم است.","en":"A blood fat that rises with food, sugar and alcohol. Fasting is usually needed for the test."}',
   '{"category":"lipid","subtitle":{"fa":"چربی خون","en":"Blood fats"},"aliases":["triglycerides","triglyceride","tg","trig","تریگلیسرید"],"unit":"mg/dL","typical":{"high":150,"text":"< 150"},"critical":{"soon_high":500,"urgent_high":1000},"low":{"factors":[{"fa":"معمولاً نگران‌کننده نیست","en":"Usually not a concern"}],"questions":[{"fa":"آیا لازم است کاری انجام دهم؟","en":"Do I need to do anything about it?"}],"see_doctor":{"fa":"در صورت شک با پزشک صحبت کن.","en":"Ask a doctor if unsure."}},"high":{"factors":[{"fa":"ناشتا نبودن هنگام آزمایش","en":"Not fasting before the test"},{"fa":"مصرف زیاد قند و کربوهیدرات ساده","en":"A lot of sugar and refined carbohydrates"},{"fa":"مقاومت به انسولین","en":"Insulin resistance"}],"questions":[{"fa":"آیا باید آزمایش را ناشتا تکرار کنم؟","en":"Should I repeat the test fasting?"}],"see_doctor":{"fa":"برای بررسی و برنامه غذایی با پزشک مشورت کن.","en":"Talk to a doctor about a check-up and an eating plan."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'fsh', 20, 1, NULL,
   '{"fa":"FSH","en":"FSH"}',
   '{"fa":"هورمونی از هیپوفیز که رشد فولیکول‌های تخمدان را تحریک می‌کند. مقدارش در روزهای مختلف سیکل فرق دارد و معمولاً روز ۲ تا ۴ سیکل اندازه گرفته می‌شود.","en":"A pituitary hormone that makes ovarian follicles grow. It varies through the cycle and is usually measured on cycle day 2 to 4."}',
   '{"category":"hormone","subtitle":{"fa":"رشد فولیکول تخمدان","en":"Ovarian follicle growth"},"aliases":["fsh","folliclestimulatinghormone"],"unit":"mIU/mL","range_varies":true,"typical":{"low":3.5,"high":12.5,"text":"3.5–12.5 (follicular)"},"low":{"factors":[{"fa":"قرص‌های هورمونی","en":"Hormonal contraceptives"},{"fa":"استرس شدید، کاهش وزن زیاد یا ورزش خیلی سنگین","en":"Severe stress, a lot of weight loss or very hard training"}],"questions":[{"fa":"آیا این عدد روی تخمک‌گذاری من اثر دارد؟","en":"Does this affect my ovulation?"}],"see_doctor":{"fa":"برای تفسیر بر اساس روز سیکل با پزشک زنان صحبت کن.","en":"Talk to a gynaecologist to read it for your cycle day."}},"high":{"factors":[{"fa":"کاهش ذخیره تخمدان","en":"Lower ovarian reserve"},{"fa":"نزدیک شدن به یائسگی","en":"Approaching menopause"},{"fa":"انجام آزمایش در زمان تخمک‌گذاری","en":"Testing around ovulation"}],"questions":[{"fa":"ذخیره تخمدان من چطور است؟","en":"What is my ovarian reserve like?"},{"fa":"آیا آزمایش را باید در روز ۲ تا ۴ سیکل تکرار کنم؟","en":"Should I repeat the test on cycle day 2 to 4?"}],"see_doctor":{"fa":"برای تفسیر بر اساس روز سیکل و سن با پزشک زنان صحبت کن.","en":"Talk to a gynaecologist to read it for your cycle day and age."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'lh', 21, 1, NULL,
   '{"fa":"LH","en":"LH"}',
   '{"fa":"هورمونی از هیپوفیز که جهش آن باعث تخمک‌گذاری می‌شود. مقدارش در طول سیکل خیلی تغییر می‌کند.","en":"A pituitary hormone whose surge triggers ovulation. It changes a lot through the cycle."}',
   '{"category":"hormone","subtitle":{"fa":"محرک تخمک‌گذاری","en":"Ovulation trigger"},"aliases":["lh","luteinizinghormone","luteinisinghormone"],"unit":"mIU/mL","range_varies":true,"typical":{"low":2.4,"high":12.6,"text":"2.4–12.6 (follicular)"},"low":{"factors":[{"fa":"قرص‌های هورمونی","en":"Hormonal contraceptives"},{"fa":"استرس یا کاهش وزن زیاد","en":"Stress or a lot of weight loss"}],"questions":[{"fa":"آیا تخمک‌گذاری من طبیعی است؟","en":"Is my ovulation normal?"}],"see_doctor":{"fa":"برای تفسیر بر اساس روز سیکل با پزشک زنان صحبت کن.","en":"Talk to a gynaecologist to read it for your cycle day."}},"high":{"factors":[{"fa":"آزمایش در زمان جهش LH قبل از تخمک‌گذاری","en":"Testing during the LH surge before ovulation"},{"fa":"سندرم تخمدان پلی‌کیستیک (نسبت LH به FSH بالا)","en":"Polycystic ovary syndrome (high LH to FSH ratio)"},{"fa":"نزدیک شدن به یائسگی","en":"Approaching menopause"}],"questions":[{"fa":"آیا نسبت LH به FSH من نشانه PCOS است؟","en":"Does my LH to FSH ratio suggest PCOS?"}],"see_doctor":{"fa":"برای تفسیر بر اساس روز سیکل با پزشک زنان صحبت کن.","en":"Talk to a gynaecologist to read it for your cycle day."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'estradiol', 22, 1, NULL,
   '{"fa":"استرادیول (E2)","en":"Estradiol (E2)"}',
   '{"fa":"مهم‌ترین هورمون استروژن که تخمدان می‌سازد. مقدارش در طول سیکل بالا و پایین می‌رود.","en":"The main oestrogen made by the ovaries. It rises and falls through the cycle."}',
   '{"category":"hormone","subtitle":{"fa":"استروژن اصلی","en":"Main oestrogen"},"aliases":["estradiol","oestradiol","e2","استرادیول"],"unit":"pg/mL","range_varies":true,"typical":{"low":12.5,"high":166,"text":"12.5–166 (follicular)"},"low":{"factors":[{"fa":"مرحله ابتدایی سیکل","en":"The early part of the cycle"},{"fa":"نزدیک شدن به یائسگی","en":"Approaching menopause"},{"fa":"کاهش وزن زیاد یا ورزش خیلی سنگین","en":"A lot of weight loss or very hard training"}],"questions":[{"fa":"آیا این عدد با روز سیکل من هم‌خوانی دارد؟","en":"Does this fit my cycle day?"}],"see_doctor":{"fa":"برای تفسیر بر اساس روز سیکل با پزشک زنان صحبت کن.","en":"Talk to a gynaecologist to read it for your cycle day."}},"high":{"factors":[{"fa":"نزدیک تخمک‌گذاری","en":"Close to ovulation"},{"fa":"داروهای باروری یا هورمونی","en":"Fertility or hormonal medicines"},{"fa":"بارداری","en":"Pregnancy"}],"questions":[{"fa":"آیا این عدد با روز سیکل من هم‌خوانی دارد؟","en":"Does this fit my cycle day?"}],"see_doctor":{"fa":"برای تفسیر بر اساس روز سیکل با پزشک زنان صحبت کن.","en":"Talk to a gynaecologist to read it for your cycle day."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'progesterone', 23, 1, NULL,
   '{"fa":"پروژسترون","en":"Progesterone"}',
   '{"fa":"هورمونی که بعد از تخمک‌گذاری بالا می‌رود و رحم را برای بارداری آماده می‌کند. آزمایش روز ۲۱ (حدود ۷ روز بعد از تخمک‌گذاری) نشان می‌دهد تخمک‌گذاری رخ داده یا نه.","en":"A hormone that rises after ovulation and prepares the womb for pregnancy. A day-21 test (about 7 days after ovulation) shows whether ovulation happened."}',
   '{"category":"hormone","subtitle":{"fa":"نشانه تخمک‌گذاری","en":"Sign of ovulation"},"aliases":["progesterone","prog","p4","day21progesterone","پروژسترون"],"unit":"ng/mL","range_varies":true,"typical":{"low":1.8,"high":24,"text":"1.8–24 (luteal)"},"low":{"factors":[{"fa":"آزمایش در نیمه اول سیکل","en":"Testing in the first half of the cycle"},{"fa":"تخمک‌گذاری نکردن در آن سیکل","en":"No ovulation that cycle"}],"questions":[{"fa":"آیا در این سیکل تخمک‌گذاری داشته‌ام؟","en":"Did I ovulate this cycle?"},{"fa":"آیا آزمایش در روز درست سیکل انجام شده است؟","en":"Was the test done on the right cycle day?"}],"see_doctor":{"fa":"برای تفسیر بر اساس روز سیکل با پزشک زنان صحبت کن.","en":"Talk to a gynaecologist to read it for your cycle day."}},"high":{"factors":[{"fa":"فاز لوتئال یا بارداری","en":"The luteal phase or pregnancy"},{"fa":"مکمل پروژسترون","en":"Progesterone supplements"}],"questions":[{"fa":"آیا این عدد با روز سیکل من هم‌خوانی دارد؟","en":"Does this fit my cycle day?"}],"see_doctor":{"fa":"برای تفسیر بر اساس روز سیکل با پزشک زنان صحبت کن.","en":"Talk to a gynaecologist to read it for your cycle day."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'prolactin', 24, 1, NULL,
   '{"fa":"پرولاکتین","en":"Prolactin"}',
   '{"fa":"هورمونی که شیرسازی را تحریک می‌کند. بالا بودنش در غیر شیردهی می‌تواند سیکل و تخمک‌گذاری را مختل کند.","en":"A hormone that drives milk production. When high outside breastfeeding it can disturb cycles and ovulation."}',
   '{"category":"hormone","subtitle":{"fa":"هورمون شیرسازی","en":"Milk hormone"},"aliases":["prolactin","prl","پرولاکتین"],"unit":"ng/mL","typical":{"low":4.8,"high":23.3,"text":"4.8–23.3"},"critical":{"soon_high":100},"low":{"factors":[{"fa":"معمولاً نگران‌کننده نیست","en":"Usually not a concern"}],"questions":[{"fa":"آیا لازم است کاری انجام دهم؟","en":"Do I need to do anything about it?"}],"see_doctor":{"fa":"در صورت شک با پزشک صحبت کن.","en":"Ask a doctor if unsure."}},"high":{"factors":[{"fa":"استرس، خواب کم یا آزمایش بلافاصله بعد از بیدار شدن","en":"Stress, poor sleep or testing right after waking"},{"fa":"شیردهی یا بارداری","en":"Breastfeeding or pregnancy"},{"fa":"برخی داروها","en":"Some medicines"}],"questions":[{"fa":"آیا باید آزمایش را در حالت استراحت تکرار کنم؟","en":"Should I repeat the test at rest?"},{"fa":"آیا پرولاکتین بالا روی نظم پریود یا بارداری من اثر دارد؟","en":"Could high prolactin affect my periods or getting pregnant?"}],"see_doctor":{"fa":"اگر ترشح شیر از سینه، سردرد یا اختلال دید داری، زودتر مراجعه کن.","en":"If you have milky nipple discharge, headaches or vision changes, see a doctor soon."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'amh', 25, 1, NULL,
   '{"fa":"AMH","en":"AMH"}',
   '{"fa":"هورمونی که فولیکول‌های کوچک تخمدان می‌سازند و تخمینی از ذخیره تخمدان می‌دهد. با سن کم می‌شود و به‌تنهایی شانس بارداری را تعیین نمی‌کند.","en":"A hormone made by small ovarian follicles that estimates ovarian reserve. It falls with age and does not on its own decide the chance of pregnancy."}',
   '{"category":"hormone","subtitle":{"fa":"ذخیره تخمدان","en":"Ovarian reserve"},"aliases":["amh","antimullerianhormone","antimüllerianhormone","mullerianinhibitinghormone"],"unit":"ng/mL","range_varies":true,"typical":{"low":1.0,"high":3.5,"text":"1.0–3.5 (varies with age)"},"low":{"factors":[{"fa":"افزایش سن","en":"Getting older"},{"fa":"جراحی یا درمان‌های قبلی تخمدان","en":"Past ovarian surgery or treatment"}],"questions":[{"fa":"ذخیره تخمدان من برای سنم چطور است؟","en":"What is my ovarian reserve like for my age?"},{"fa":"آیا باید برای بارداری زودتر اقدام کنم؟","en":"Should I try to conceive sooner?"}],"see_doctor":{"fa":"برای تفسیر بر اساس سن و برنامه بارداری با پزشک زنان یا متخصص ناباروری صحبت کن.","en":"Talk to a gynaecologist or fertility specialist to read it for your age and plans."}},"high":{"factors":[{"fa":"سندرم تخمدان پلی‌کیستیک","en":"Polycystic ovary syndrome"}],"questions":[{"fa":"آیا AMH بالا نشانه PCOS است؟","en":"Does a high AMH suggest PCOS?"}],"see_doctor":{"fa":"همراه با سونوگرافی و علائمت با پزشک زنان بررسی کن.","en":"Review it with a gynaecologist together with an ultrasound and your symptoms."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('lab_markers', 'testosterone', 26, 1, NULL,
   '{"fa":"تستوسترون","en":"Testosterone"}',
   '{"fa":"هورمون مردانه‌ای که در زنان هم به مقدار کم ساخته می‌شود. بالا بودنش می‌تواند با آکنه، موهای زائد و سیکل نامنظم همراه باشد.","en":"A male hormone that women make in small amounts. A high level can go with acne, unwanted hair and irregular cycles."}',
   '{"category":"hormone","subtitle":{"fa":"آندروژن","en":"Androgen"},"aliases":["testosterone","totaltestosterone","testosteronetotal","تستوسترون"],"unit":"ng/dL","typical":{"low":15,"high":70,"text":"15–70"},"low":{"factors":[{"fa":"قرص‌های هورمونی","en":"Hormonal contraceptives"},{"fa":"نزدیک شدن به یائسگی","en":"Approaching menopause"}],"questions":[{"fa":"آیا لازم است کاری انجام دهم؟","en":"Do I need to do anything about it?"}],"see_doctor":{"fa":"در صورت شک با پزشک صحبت کن.","en":"Ask a doctor if unsure."}},"high":{"factors":[{"fa":"سندرم تخمدان پلی‌کیستیک","en":"Polycystic ovary syndrome"},{"fa":"برخی داروها یا مکمل‌ها","en":"Some medicines or supplements"}],"questions":[{"fa":"آیا تستوسترون بالا نشانه PCOS است؟","en":"Does a high testosterone suggest PCOS?"}],"see_doctor":{"fa":"همراه با علائمت با پزشک زنان یا غدد بررسی کن.","en":"Review it with a gynaecologist or endocrinologist together with your symptoms."}}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'lab_markers';
DROP TABLE IF EXISTS `lab_jobs`;
DROP TABLE IF EXISTS `lab_markers`;
DROP TABLE IF EXISTS `lab_files`;
DROP TABLE IF EXISTS `lab_reports`;
