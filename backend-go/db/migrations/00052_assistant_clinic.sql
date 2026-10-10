-- 00052_assistant_clinic.sql — «کلینیک دستیار ریتمی», the AI assistant clinic (bloom B-N7-06, D-83; artboards
-- nbl_v18_* in d-doctor-assistant). Domain logic: internal/assistant; provider calls through internal/ai.
--
-- assistant_departments       one clinic department: code, icon, translatable name / tagline / intro, translatable
--                             lists does_not_do («این بخش چه کاری نمی‌کند») and quick_replies (starter chips), and the
--                             admin-authored system_prompt + guardrails (English, sent to the model after the Go base
--                             guard). Admin-editable content (B-N7-08); is_active hides a department.
-- assistant_safety_terms      server-side keyword lists checked before the model runs: kind emergency (→ the 115 card,
--                             no model call) | diagnosis (→ the diagnosis guard note), per locale, admin-editable.
-- assistant_profile_settings  which health-record parts the user shares with the assistant (shared = JSON list of keys);
--                             nothing is shared without the assistant_profile consent either.
-- assistant_conversations     one chat of a user in a department: triage_level self_care | non_urgent | soon |
--                             emergency, summary (the «خلاصه گفت‌وگو» document) with the message count it covers.
-- assistant_messages          the turns: role user | assistant | card (emergency 115 card); photo_file_id = a files row
--                             of purpose assistant_photo; chips = quick replies the model suggested.
-- assistant_handoffs          a summary sent to the doctor of one of the user's bookings (needs the booking's active
--                             assistant_summaries consent; doctors read it only through telemed AuthorizeShare).
-- Hard delete: deleting a conversation removes its messages, handoffs and photos (files rows + blobs, in Go).
--
-- Twin of backend/database/migrations/2026_10_08_000052_create_assistant_clinic_tables.php (Laravel owns the prod
-- schema until T-M2-27), so `make schema-diff` stays green. IF NOT EXISTS / INSERT IGNORE.

-- +goose Up
CREATE TABLE IF NOT EXISTS `assistant_departments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(32) NOT NULL,
  `icon` varchar(32) DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`name`)),
  `tagline` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`tagline`)),
  `intro` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`intro`)),
  `does_not_do` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`does_not_do`)),
  `quick_replies` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`quick_replies`)),
  `system_prompt` text NOT NULL,
  `guardrails` text DEFAULT NULL,
  `needs_review` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `assistant_departments_code_unique` (`code`),
  KEY `assistant_departments_is_active_sort_order_index` (`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `assistant_safety_terms` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `kind` varchar(16) NOT NULL,
  `locale` varchar(8) NOT NULL,
  `term` varchar(100) NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `assistant_safety_terms_kind_locale_term_unique` (`kind`,`locale`,`term`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `assistant_profile_settings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `shared` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`shared`)),
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `assistant_profile_settings_user_id_unique` (`user_id`),
  CONSTRAINT `assistant_profile_settings_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `assistant_conversations` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `department_id` bigint(20) unsigned NOT NULL,
  `title` varchar(120) DEFAULT NULL,
  `triage_level` varchar(16) DEFAULT NULL,
  `emergency` tinyint(1) NOT NULL DEFAULT 0,
  `message_count` int(10) unsigned NOT NULL DEFAULT 0,
  `summary` text DEFAULT NULL,
  `summary_at` datetime DEFAULT NULL,
  `summary_message_count` int(10) unsigned NOT NULL DEFAULT 0,
  `last_message_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `assistant_conversations_department_id_foreign` (`department_id`),
  KEY `assistant_conversations_user_id_last_message_at_index` (`user_id`,`last_message_at`),
  CONSTRAINT `assistant_conversations_department_id_foreign` FOREIGN KEY (`department_id`) REFERENCES `assistant_departments` (`id`),
  CONSTRAINT `assistant_conversations_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `assistant_messages` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `conversation_id` bigint(20) unsigned NOT NULL,
  `role` varchar(16) NOT NULL,
  `body` text NOT NULL,
  `photo_file_id` bigint(20) unsigned DEFAULT NULL,
  `card` varchar(16) DEFAULT NULL,
  `triage_level` varchar(16) DEFAULT NULL,
  `guardrail` varchar(16) DEFAULT NULL,
  `chips` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`chips`)),
  `status` varchar(12) NOT NULL DEFAULT 'complete',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `assistant_messages_conversation_id_foreign` (`conversation_id`),
  KEY `assistant_messages_photo_file_id_foreign` (`photo_file_id`),
  CONSTRAINT `assistant_messages_conversation_id_foreign` FOREIGN KEY (`conversation_id`) REFERENCES `assistant_conversations` (`id`) ON DELETE CASCADE,
  CONSTRAINT `assistant_messages_photo_file_id_foreign` FOREIGN KEY (`photo_file_id`) REFERENCES `files` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `assistant_handoffs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `conversation_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `booking_id` bigint(20) unsigned NOT NULL,
  `summary` text NOT NULL,
  `triage_level` varchar(16) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `assistant_handoffs_conversation_id_booking_id_unique` (`conversation_id`,`booking_id`),
  KEY `assistant_handoffs_user_id_foreign` (`user_id`),
  KEY `assistant_handoffs_booking_id_foreign` (`booking_id`),
  CONSTRAINT `assistant_handoffs_booking_id_foreign` FOREIGN KEY (`booking_id`) REFERENCES `telemed_bookings` (`id`) ON DELETE CASCADE,
  CONSTRAINT `assistant_handoffs_conversation_id_foreign` FOREIGN KEY (`conversation_id`) REFERENCES `assistant_conversations` (`id`) ON DELETE CASCADE,
  CONSTRAINT `assistant_handoffs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `assistant_departments` (`code`, `icon`, `sort_order`, `is_active`, `name`, `tagline`, `intro`, `does_not_do`, `quick_replies`, `system_prompt`, `guardrails`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('general', 'stethoscope', 1, 1, '{"fa":"مشاور پزشکی عمومی","en":"General health advisor"}', '{"fa":"علائم، دارو، آزمایش","en":"Symptoms, medicines, lab tests"}', '{"fa":"راهنمایی کلی درباره علائم روزمره، داروهایی که مصرف می‌کنی و معنای ساده نتایج آزمایش. کمک می‌کند بفهمی چه زمانی باید به پزشک مراجعه کنی.","en":"General guidance on everyday symptoms, the medicines you take and what lab results mean in plain words. Helps you decide when to see a doctor."}', '{"fa":["تشخیص بیماری","تجویز دارو یا تعیین دوز","جایگزین ویزیت پزشک"],"en":["Diagnosing a disease","Prescribing medicines or doses","Replacing a doctor''s visit"]}', '{"fa":["سردرد","تب","سرماخوردگی","تفسیر آزمایش","چیز دیگر"],"en":["Headache","Fever","Cold","Lab results","Something else"]}', 'You are the general health advisor of the Ritme women''s health app. Help with everyday symptoms, how common medicines are usually used, and what lab values mean in plain words. Ask one or two short clarifying questions before giving guidance. Suggest when a doctor visit is sensible and how soon.', 'Never name a diagnosis as a fact; describe patterns as possibilities. Never give a medicine dose or start/stop advice for prescription drugs. Pregnancy or breastfeeding changes medicine safety: always say so when relevant.', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('gynecology', 'flower', 2, 1, '{"fa":"مشاوره زنان","en":"Gynecology advisor"}', '{"fa":"سیکل، باروری، بارداری، یائسگی","en":"Cycle, fertility, pregnancy, menopause"}', '{"fa":"پاسخ به پرسش‌هایت درباره سیکل قاعدگی، لکه‌بینی، باروری، بارداری و یائسگی با در نظر گرفتن داده‌هایی که خودت اجازه داده‌ای.","en":"Answers about your menstrual cycle, spotting, fertility, pregnancy and menopause, taking into account the data you chose to share."}', '{"fa":["تشخیص بیماری یا بارداری","تجویز دارو یا هورمون","تفسیر قطعی سونوگرافی"],"en":["Diagnosing a disease or a pregnancy","Prescribing medicines or hormones","Definitive ultrasound readings"]}', '{"fa":["پریود نامنظم","لکه‌بینی","درد پریود","اقدام به بارداری","چیز دیگر"],"en":["Irregular period","Spotting","Period pain","Trying to conceive","Something else"]}', 'You are the gynecology advisor of the Ritme women''s health app. Help with menstrual cycle questions, spotting, period pain, fertility, pregnancy and menopause. Use the shared cycle and life-stage data when relevant. Ask short clarifying questions (cycle day, duration, severity) before guidance.', 'Heavy bleeding, severe one-sided pelvic pain, fever with pelvic pain, or bleeding in pregnancy need urgent care: say so clearly. Never confirm or exclude a pregnancy; suggest a test. Never recommend hormones or contraceptive brands.', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('sexual_health', 'heart', 3, 1, '{"fa":"مشاوره جنسی","en":"Sexual health advisor"}', '{"fa":"رابطه، میل، درد، پیشگیری","en":"Intimacy, desire, pain, contraception"}', '{"fa":"فضایی امن و بدون قضاوت برای پرسش درباره میل جنسی، درد هنگام رابطه، پیشگیری و عفونت‌ها.","en":"A safe, non-judgemental space for questions about desire, pain during sex, contraception and infections."}', '{"fa":["تشخیص عفونت","تجویز دارو یا روش پیشگیری","قضاوت درباره رابطه‌ات"],"en":["Diagnosing an infection","Prescribing medicines or a contraceptive method","Judging your relationship"]}', '{"fa":["درد هنگام رابطه","کاهش میل","پیشگیری","ترشحات","چیز دیگر"],"en":["Pain during sex","Low desire","Contraception","Discharge","Something else"]}', 'You are the sexual health advisor of the Ritme women''s health app. Be warm, respectful and non-judgemental. Help with desire, pain during intercourse, contraception options in general terms, and signs of infection that need a check-up.', 'Never ask for explicit details beyond what is medically useful. Sexual violence or coercion: respond with care and point to professional help. Possible infection: recommend testing, never a specific drug.', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('skin_beauty', 'sparkle', 4, 1, '{"fa":"زیبایی و پوست","en":"Skin & beauty"}', '{"fa":"آکنه، لک، مراقبت پوست و مو","en":"Acne, dark spots, skin and hair care"}', '{"fa":"راهنمایی روتین مراقبت، آکنه، لک، خشکی، ریزش مو و اثر هورمون‌ها و سیکل روی پوست. با عکس هم می‌توانی توضیح بدهی.","en":"Guidance on care routines, acne, dark spots, dryness, hair loss and how hormones and your cycle affect your skin. You can also explain with a photo."}', '{"fa":["تشخیص بیماری پوستی","تجویز دارو","تفسیر قطعی عکس"],"en":["Diagnosing a skin disease","Prescribing medicines","Definitive photo readings"]}', '{"fa":["آکنه","لک و تیرگی","خشکی","ریزش مو","چیز دیگر"],"en":["Acne","Dark spots","Dryness","Hair loss","Something else"]}', 'You are the skin and beauty advisor of the Ritme women''s health app. Help with skincare routines, acne, pigmentation, dryness and hair loss, and how the cycle, pregnancy and hormones affect skin. A photo may be attached: describe only what is visible and say a photo is not a diagnosis.', 'If the user is pregnant, trying to conceive or breastfeeding, warn against retinoids, hydroquinone and other ingredients unsafe in pregnancy unless a doctor approves. A changing mole, a bleeding lesion or a fast-spreading rash needs a doctor soon.', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('nutrition', 'leaf', 5, 1, '{"fa":"تغذیه","en":"Nutrition"}', '{"fa":"رژیم، مکمل، وزن","en":"Diet, supplements, weight"}', '{"fa":"پیشنهادهای ساده برای تغذیه سالم در هر مرحله از زندگی: سیکل، اقدام به بارداری، بارداری، شیردهی و یائسگی.","en":"Simple ideas for healthy eating at every life stage: cycle, trying to conceive, pregnancy, breastfeeding and menopause."}', '{"fa":["رژیم درمانی برای بیماری","تعیین دوز مکمل","برنامه کاهش وزن شدید"],"en":["Therapeutic diets for a disease","Supplement doses","Extreme weight-loss plans"]}', '{"fa":["تغذیه در بارداری","آهن و کم‌خونی","کاهش وزن","مکمل‌ها","چیز دیگر"],"en":["Pregnancy nutrition","Iron and anaemia","Weight loss","Supplements","Something else"]}', 'You are the nutrition advisor of the Ritme women''s health app. Give practical, culturally familiar (Iranian) food suggestions for the user''s life stage. Keep advice balanced and sustainable.', 'Never give supplement doses; say a doctor or dietitian should set them. Signs of an eating disorder: respond with care and suggest professional help. No diets below safe energy intake.', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('mental_health', 'moon', 6, 1, '{"fa":"سلامت روان","en":"Mental health"}', '{"fa":"خلق، خواب، استرس","en":"Mood, sleep, stress"}', '{"fa":"گفت‌وگویی آرام درباره خلق، استرس، خواب و ارتباط آن‌ها با سیکل و مراحل زندگی، همراه با تمرین‌های ساده.","en":"A calm conversation about mood, stress and sleep and how they relate to your cycle and life stage, with simple exercises."}', '{"fa":["تشخیص اختلال روانی","تجویز دارو","جایگزین روان‌درمانگر یا اورژانس"],"en":["Diagnosing a mental disorder","Prescribing medicines","Replacing a therapist or emergency care"]}', '{"fa":["استرس","بی‌خوابی","نوسان خلق قبل از پریود","غم بعد از زایمان","چیز دیگر"],"en":["Stress","Insomnia","Mood swings before my period","Sadness after birth","Something else"]}', 'You are the mental health companion of the Ritme women''s health app. Listen first, reflect feelings, and offer simple evidence-based coping exercises (breathing, sleep hygiene, journaling). Consider cycle-related mood changes, postpartum mood and menopause.', 'Any sign of self-harm or suicidal thoughts: respond with empathy, urge contacting emergency services (115) or a trusted person right now, and do not continue with routine advice. Never name a psychiatric diagnosis or a medicine.', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

INSERT IGNORE INTO `assistant_safety_terms` (`kind`, `locale`, `term`, `is_active`, `created_at`, `updated_at`) VALUES
  ('emergency', 'fa', 'خونریزی شدید', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'خونریزی زیاد در بارداری', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'بیهوش', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'از هوش رفتم', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'تشنج', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'درد قفسه سینه', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'درد سینه شدید', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'نفس نمی‌کشم', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'تنگی نفس شدید', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'خودکشی', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'خودم را بکشم', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'خودمو بکشم', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'درد شدید شکم', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'مسمومیت', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'اوردوز', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'حرکت جنین را حس نمی‌کنم', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'fa', 'کیسه آب پاره', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'heavy bleeding', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'unconscious', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'fainted', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'seizure', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'chest pain', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'can''t breathe', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'cannot breathe', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'suicide', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'kill myself', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'severe abdominal pain', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'overdose', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'poisoning', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'baby stopped moving', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('emergency', 'en', 'water broke', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'تشخیص بده', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'تشخیص بدی', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'چه بیماری', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'چه مریضی', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'چه دارویی', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'چه قرصی', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'نسخه بنویس', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'دوز', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'fa', 'چند میلی', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'diagnose', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'what disease', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'what illness', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'which medicine', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'what medicine', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'prescribe', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'prescription', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'dosage', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('diagnosis', 'en', 'what dose', 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));
-- +goose Down
DROP TABLE IF EXISTS `assistant_handoffs`;
DROP TABLE IF EXISTS `assistant_messages`;
DROP TABLE IF EXISTS `assistant_conversations`;
DROP TABLE IF EXISTS `assistant_profile_settings`;
DROP TABLE IF EXISTS `assistant_safety_terms`;
DROP TABLE IF EXISTS `assistant_departments`;
