# Domain & data inventory (Laravel `backend/`, captured 2026-09-23)

Stack: Laravel 12.41.1, Carbon 3.11.0, PHP 8.4, Passport 13.4. No soft deletes, no encrypted columns, no observers,
no `$appends`. All domain tables have nullable `created_at`/`updated_at`; every `user_id` FK cascades on delete.
utf8mb4_unicode_ci, strict mode.

Headline risks: (1) Tehran wall-clock timestamps; (2) raw Eloquent model JSON shaped by casts; (3) two cycle engines
(legacy `HealthDataEngine` → `calculation`, month, home, messages; v1.1 `CycleStatusResolver` → `cycle_view`);
(4) Passport tokens; (5) code fallback content in PHP.

## 1. Schema (55 migrations → 38 tables + `migrations`)

### Domain tables (26)
| Table | Key columns | Indexes | JSON |
|---|---|---|---|
| users | id, name?, email? unique, email_verified_at, mobile varchar(11)? unique, mobile_verified_at, blocked_at, password?, remember_token | uq email, uq mobile | – |
| otp_verifications | mobile(11) idx, code varchar(4), expires_at, verified_at, attempts tinyint 0 | idx mobile | – |
| user_profiles | user_id, birthday date, weight decimal(5,2), height smallint, period_duration tinyint, cycle_duration tinyint, last_period_start date, user_goal def `non_ttc`, pregnancy_intention, subscription_type def `free`, calculation_status def `pending`, calculation_started_at, calculation_completed_at, calculation_version int 0 | FK only (**no unique user_id**) | chronic_conditions |
| daily_health_logs | user_id, log_date; ~30 nullable varchar enums; ~25 nullable bools; weight decimal(5,2), basal_body_temperature decimal(4,2), heart_rate/systolic/diastolic smallint, blood_sugar decimal(5,1), exercise_duration, notes | **uq(user_id,log_date)** | moods, sexual_activities, medications, exercise_type |
| cycle_histories | user_id, period_start_date, period_end_date?, cycle_length, bleeding_length, is_confirmed 0, is_estimated 0, source def `user_logged` | **uq(user_id,period_start_date)** | data_quality_flags |
| pregnancy_profiles | user_id **unique**; mode flags; age_source, confidence_level; lmp/ultrasound/manual dates/weeks/days; estimated_due_date, estimated_conception_date; uncertainty_days 3; history bools; blood_type, rh_factor, rh_negative_care_flag; fetal movement fields; onboarding_completed(_at) | uq user_id | pre_existing_conditions |
| pregnancy_symptom_logs | 14 × (`has_X` bool, `X_severity`), notes | uq(user_id,log_date) | – |
| pregnancy_weekly_logs | log_date, pregnancy_week, weight/fasting/post-meal sugar decimal(5,2), BP ints, mood | **uq(user_id,pregnancy_week)** | swelling_locations |
| pregnancy_fetal_movements | log_date, pregnancy_week, movement_status, movement_count, first/last_movement_time `time` | uq(user_id,log_date) | – |
| pregnancy_alerts | alert_level, alert_type, title, message, pregnancy_week, is_read, is_dismissed, read_at, dismissed_at | idx(user_id,is_read), idx(user_id,alert_level) | trigger_symptoms, medical_history_flags, recommended_actions |
| pregnancy_weekly_content (singular) | week_number unique; 10 JSON sections | uq | 10 sections |
| task_templates | key uq, category, icon, cycle_phase?, is_active, sort_order | idx(is_active,cycle_phase) | title, description |
| user_task_completions | task_template_id, completion_date, completed_at | uq `user_task_unique_per_day`; idx(user_id,completion_date) | – |
| reminders | type, title, subtitle, notes, scheduled_at, recurrence def `none`, recurrence_time time, starts_on/ends_on, is_active | idx(user_id,type,is_active), idx(user_id,scheduled_at) | meta |
| articles | slug uq, category, read_time_minutes, image_url, image_path, is_published, published_at, sort_order | idx is_published | title, excerpt, body, cycle_phases |
| affirmations | cycle_phase, is_active, sort_order | idx(is_active,cycle_phase) | text |
| challenges | slug? uq, cycle_day_from/to, category, is_active, sort_order | `challenges_active_day_index` | title, description |
| user_challenge_completions | challenge_id, completion_date, completed_at | uq `user_challenge_unique_per_day` | – |
| user_notifications | type, action_url, read_at | idx(user_id,read_at) | title, body, data |
| admins | name, email uq, password bcrypt, role def `editor`, is_active, last_login_at, remember_token | – | – |
| message_contents | group idx, item_key idx, locale(5), label, is_active, is_approved, sort_order | **uq(group,item_key,locale)** | payload |
| banners | image_path, position def `home_top`, link_url(1000), link_type, starts_at, ends_at, is_active, sort_order | idx(position,is_active,starts_at,ends_at) | title |
| phase_contents | phase uq; 9 JSON sections | – | 9 sections |
| recommendations | key? uq, type def `general`, cycle_phase, symptom_trigger, is_active, sort_order | idx(is_active,cycle_phase) | title, text, cycle_subphases |
| info_sections | group(32) def `privacy`, key, link_url, is_active, sort_order | uq(group,key) | heading, body, link_label |
| languages | code(12) uq, name(60), english_name(60), direction(3) def `ltr`, is_active, is_default, sort_order (migration inserts fa default + en) | idx(is_active,sort_order) | – |

Laravel/Passport internals (12): password_reset_tokens, sessions, cache, cache_locks, jobs, job_batches, failed_jobs
(PHP-serialized), oauth_auth_codes, oauth_access_tokens, oauth_refresh_tokens, oauth_clients (uuid PK),
oauth_device_codes. Dropped: cycle_calculations, privacy_sections. Data-rewriting migrations already applied — don't port.

## 2. Models (27 files, ~2,800 lines)
Localization: `Concerns/HasLocalizedContent` → `Support/Translatable::pick` (:75): requested → default (languages table) → first non-empty; `clean()` strips empty locales. Exceptions: `PhaseContent::getLocalizedContent` (:96) `locale→fa→en`; `PregnancyWeeklyContent::getLocalizedContent` (:102) returns the **whole array** if locale missing; `AbstractHomeSection::pick` `locale→fa→en`.

| Cast | Where | JSON |
|---|---|---|
| `date:Y-m-d` | UserProfile birthday/last_period_start, all log_date | `"2026-09-23"` |
| plain `date` | CycleHistory start/end, PregnancyProfile (6 dates), Reminder starts_on/ends_on, completions | `"2026-09-22T20:30:00.000000Z"` (prev day) |
| `datetime`/timestamps | all | `"…000000Z"` |
| manual `toIso8601String()` | `HomeController.php:330`, `OtpAuthController.php:430` | `"…+03:30"` |
| `decimal:2`/`decimal:1` | DailyHealthLog weight/BBT/blood_sugar, PregnancyWeeklyLog weight/sugars | JSON strings |
| `float` | UserProfile weight | number |
| `integer` | UserProfile height/durations | int |
| `array` | JSON cols | list or object |
| enum cast | Language.direction only | string |
| `hashed` | User/Admin password | bcrypt `$2y$` |

`User.$hidden=[password,remember_token]`, same for Admin. Relations: User hasOne profile/pregnancyProfile; hasMany logs, histories, 4 pregnancy tables, completions, reminders, appNotifications. `User::hasCompletedProfile()` = filled(name) && profile exists (:61).

Scopes: `Article::forPhase` (:97, `cycle_phases IS NULL OR JSON_CONTAINS`, order `cycle_phases IS NULL, sort_order`); `Banner::active` (:68, window, order sort_order, id DESC); `Challenge::forCycleDay` (:75) + `appliesToCycleDay` (:116) + Persian `cycleDayLabel` (:101); `Reminder::relevantOn` (:75); `TaskTemplate/Affirmation::forPhase`; `MessageContent::live`; `Language::active/ordered`; `InfoSection::inGroup/ordered/active`; `PregnancyAlert::unread/active/level`; `UserNotification::unread`.

Accessors: `Article::imageUrl` (:70, image_path overrides image_url), `Banner::imageUrl` (:55). Helpers: `UserProfile::markRecalculated()` (:92, atomic increment calculation_version + status completed — part of cache key); `PregnancyWeeklyLog::hasHighBloodPressure` (≥140/90), `hasHighBloodSugar` (fasting>95, post-meal>140); `PregnancySymptomLog::getCriticalSymptoms`; `Recommendation::appliesTo/toTip`; `InfoSection::toLocalizedArray`; OTP max 5 attempts.

## 3. Enums (57, all string-backed, lowercase snake_case)
Case names ≠ values sometimes (`DataQualityFlag::LONG_PERIOD='long_period_flag'`). Locale-aware `label($locale)` (fa → Persian, **every other locale → English**): CyclePhase, CycleSubphase, MainPhase, FertilityLevel, DataStatus, ConfidenceLevel, CalculationStatus, RegularityStatus, CycleVariability, BmiCategory, TaskCategory, ReminderType, pregnancy enums, MessageMode. Persian-only `label()`: Mood and others. Helpers: `options($locale)`, `labelFor()`, `iconFor()`, `ReminderType::icon`.

Logic enums (port as code): `CycleSubphase` (172 lines, 16 cases; `fertilityLevelV11()`, legacy `fertilityLevel()`, `canonical()`, `contentBacked()`), `CyclePhase::subphases()`, `MainPhase::legacyPhase()`, `CycleVariability::fromStdDev` (≤4 regular, ≤7 semi, else irregular; uncertaintyRange 1/2/3), `RegularityStatus::fromCycleLengths` (≥3 cycles; range ≤7 regular), `CycleWarning::requiresUserInput`, `RecommendationTrigger::matches/activeFor`, `EnergyLevel::score`, `BmiCategory::fromBmi`, `ResolutionSource::isPredicted`, `DataStatus::priority`, `DataQualityFlag::excludesFromPrediction`.

## 4. Services
| Module | Files / lines | Rating |
|---|---|---|
| HealthEngine (cycle) | 19 / 3,607 | **XL** |
| MessageSystem | 14 / 3,593 (~60% fallback content) | L |
| HomePage | 24 / 2,578 | L |
| PregnancyEngine | 2 / 916 | M |
| Language (Registry, TranslationStore, Provisioner) | 594 | M |
| DailyChallengeService | 290 | S-M |
| Sms (+Kavenegar, SmsIr) | 424 | S |
| Content/HtmlSanitizer | 194 | M (bluemonday + custom rules + golden tests) |
| Media/ImageOptimizer (GD, 1080px, webp q82, EXIF, 30MP cap) | 223 | S-M |
| BmiService (`bmi_message` group) | 120 | S |
| TelegramNotifier | 55 | S |
| Support/Translatable | 119 | S |

Repositories: `HealthEngine/RecommendationRepository`, `MessageSystem/Support/MessageContentRepository` (request-scoped singletons, `AppServiceProvider.php:22-27`). The DDD folders in CLAUDE.md don't exist.

### 4.1 HealthEngine (XL)
**`CycleMetricsCalculator` (222):** only `is_confirmed` rows, oldest→newest. Cycle length = diff of consecutive starts (stored `cycle_length` ignored). Valid 21–45 (outlier flags short/long, `only_outliers`). Period duration = end−start+1, valid 2–10, closed only. **Median of last ≤3 valid** (:196); even count `round((a+b)/2)` half-away → 28,29 → **29**. Effective: median → profile (0 = missing) → default 28/5, source `EffectiveSource`. `variabilityRange` = max−min last ≤3 (null if <2); `variability` = population std-dev last ≤6; `regularityStatus` last ≤3.

**Legacy `HealthDataEngine::calculateForDate` (1,132):** empty unless `profile.last_period_start`. Anchor (`findRelevantCycleStart` :311): latest start ≤ date from **all** rows; within ECL → anchor, else roll forward `floor(days/ECL)*ECL`; else profile LMP, before LMP extrapolate `ECL − (daysBefore % ECL)`. Ovulation day `max(ECL−13,7)` (:397). Bleed length from matching row else effective, guard < ECL fallback `min(5,ECL−1)`. Phase (`CyclePhaseMapper.php:19`): ≤bleed menstruation; <O follicular; ≤O+1 ovulation; else luteal. Subphase 12 states (:34). Fertile O−5..O+1; PMS day ≥ ECL−6; both false on period days; period-tomorrow [ECL−u−1, ECL+u−1]; luteal spotting day 15–28 && spotting. Probability: base by day rel. ovulation (−5 .10 … 0 .33, +1 .12) × age (20–29 1.0, 30–34 .85, 35–37 .70, 38–40 .55, 41+ .35, <20 .35) × cycleScore (.60, +.10 regular/+.05 semi, +.10 strong signals; irregular cap .40; max .85) × symptomScore (positives cap 1.4, min negatives, max of two), cap .35; `round(x,4)`, `final_probability=round(p*100,2)`. `text_flags` (:699) interpolate PHP float strings (`12.5`, `12`), bilingual. `daily_tips`: DB recommendations if table has **any** row (even inactive), else hardcoded (:780-965). Calendar mode → `CALENDAR_FIELDS` (:78). **No unit tests.**

**v1.1 `CycleStatusResolver::resolve` (616):** anchor (:413) latest confirmed start ≤ target (keep end) → latest `is_estimated` ("onboarding_declared", end ignored) → profile LMP ≤ target → `unresolved()`. Future targets roll forward by ECL (`predicted_reference`); ≤ today don't roll. Anchors: end = logged end (≥start) else start+EPL−1; next = start+ECL; ovulation = next−14; display fertile = max(O−5,end+1)..O. Caps soft=EPL, warning=max(EPL+3,8), hard=12. Order: menstrual (logged or open split by caps + warnings, poor quality, requires_input) → period_expected (late ≤7; ≤14 overdue warning; beyond → unknown) → fertile (days-to-O ≥3, ≥1, 0) → luteal (O+1 post; P−1..3 pms; P−4..6 late; early/mid by `ceil(len/2)`) → follicular (40/40/20 floor min 1; len 1/2 special) → unknown (insufficient_anchor). Then onboarding/default override, dedup warnings (order kept), `data_quality` (:328 poor→good→partial), `confidence` (:342 unknown→low→high→medium), reasons. Day diff `round(diffInDays)` (:612).

**`CycleDayViewBuilder` (239):** merges `CycleStatus::toApiArray()` (`CycleStatus.php:56`) with legacy keys phase, data_status, predictions, three-layer values, data_quality_details, daily_card. Runs `CyclePredictionService` (anchored on last confirmed start, not rolled), `OpenPeriodEvaluator` (max_expected = max(profile_or_effective+3, 8), hard 12), `loggedPeriodFor` (open period capped at day before next start), `CycleConfidenceCalculator` (≥3 valid & regular → high…, downgraded past hard cap). **`DailyCardBuilder` (386):** title chain actual > incomplete > needs_confirmation > predicted; hardcoded fa/en; Persian digits via `strtr` (:378); non-fa → English.

**`CycleEngineCache` (104):** TTL 86,400s; key `cycle-engine:{uid}:{calc_version}:{locale}:{TehranToday}:{today}:{scope}:{xxh128(serialize(inputs))}`. Go uses its own namespace; correctness never depends on cache.

Unit tests: resolver, metrics, mapper, prediction, confidence, open-period, daily-card (62) — translate 1:1 to Go table tests.

### 4.2 PregnancyEngine (M)
`PregnancyCalculationService`: GA from LMP (medium ±3), ultrasound (high ±1), manual (low ±5); weeks=floor(d/7), days=d%7; trimester ≤12, ≤27; EDD = LMP+280; conception = today − (total−14); `getCurrentWeek = clamp(weeks+1,1,40)` (null weeks → 1); fetal movement tracking from week 18, required 24; high risk = miscarriage/high-risk history, Rh−, chronic hypertension/diabetes; `formatPersianDate` (:416) Gregorian with Persian month names. `PregnancyAlertService`: rule alerts (bleeding, spotting, fluid leakage, severe pain, 3+ severe symptoms, high BP/sugar, severe depression, reduced/no movement after 24); titles stored localized in request locale; `medical_history_flags` `[]` when empty else object; sugar values in trigger_symptoms are decimal strings.

### 4.3 MessageSystem (L)
`Core/MessageManager` (354): mode pregnancy if `pregnancy_mode` else cycle; context = legacy `calculateForDate` + 90 days of logs `->toArray()`; engine base → symptom override merge → CorrelationLayer (premium-only filtered for free) → PatternLayer (premium) → Nutrition/Sleep/Exercise modules. Dead-field bugs (preserve, D-05): `extractSymptoms` (:289) reads non-existent columns, sleep check expects poor/very_poor vs stored good/medium/bad — only `low_energy` fires; most PatternLayer detectors (:139-280) dead — only `insufficient_data` (<14 logs), `ttc_tracking`, `chronic_fatigue` (≥15 low-energy days) fire; pregnancy context uses 0-based weeks (week 0 falsy → default). Content: each class `contentDefaults()` fa/en via `MessageContentRepository::resolve(group,item_key,locale,fallback)` (loads all live rows for locale once/request; fallback locale → fa). Groups: cycle_base_non_ttc, cycle_base_ttc, cycle_override, pregnancy_base/trimester/week/override, correlation_*, pattern, nutrition_*, sleep_*, exercise_*, bmi_message. Week milestone = closest within ±2 weeks.

### 4.4 HomePage (L)
`HomePageService`: 17 sections, try/catch each (drop on error), sorted by order: header 10, week_calendar 20*, next_period 30*, cycle_prediction 40*, recommendations 50*, tasks 60, challenge 70, doctor_reminder 80, medication_reminder 90, smart_tip 100, affirmation 110, weekly_summary 120, vitals 130, status_charts 140, articles 150, my_cycles 160*, cycle_summary 170* (*cycle mode only). `HomeSection::toArray` meta null when empty (`HomeSection.php:48`). `HomeContext` (360): per-request memo; log window date−29 → end of Saturday-start week (:262); Persian digits `num()` (stringifies floats PHP-style). Affirmation = `dayOfYear % count` (`AffirmationSection.php:37`); challenge = `crc32("{uid}:{Y-m-d}") % count` (`DailyChallengeService.php:288`) after cycle-day filter, excluding last-14-day completions, preferring signal category. Support: `CycleHistoryDigest` (round avgs), `HealthMetricScorer`, `MoonPhase` (fmod, cos, round 2), `CyclePhasePalette`.

### 4.5 Language (M)
`LanguageRegistry` `rememberForever('languages.registry')` (:47), `flush()`, BOOTSTRAP fallback, `resolve()` Accept-Language parsing. `TranslationStore`: JSON bundles `storage/app/translations/<code>/*.json` seeded from `resources/translations/{fa,en}` (23 namespaces). `LanguageProvisioner`: copies `lang/` files + clones message_contents (is_approved=false).

## 5. Dates / timezone
No Jalali library. Carbon 3 `diffInDays` = signed float, cast `(int)` (truncate) or `round()` — safe because startOfDay. Today = `Carbon::today()` / `Carbon::now('Asia/Tehran')` (`CycleCalculationController.php:464`, `CycleEngineCache.php:93`). Week starts **Saturday**; `dayOfWeekIso` Mon=1..Sun=7. DB session has no tz → Go DSN `loc=Asia%2FTehran&parseTime=true`, never set `time_zone`. Iran had DST until 2022 → use a civil-date type, never `Sub()/24h`.

## 6. Seeders
Run on every boot (`docker/entrypoint.sh:71-74`, firstOrCreate): LanguageSeeder, AdminSeeder (ADMIN_SEED_*), MessageContentSeeder (from `contentDefaults()`), RecommendationSeeder (309). Production content seeded once: PregnancyWeeklyContent (2,007 lines, 40 weeks), PhaseContent, InfoSection, Challenge, TaskTemplate, Article, Affirmation. Dev only: HomeDemoSeeder, UserFactory, DatabaseSeeder test user. Code fallbacks (message engines, HealthDataEngine tips, BmiService, DailyCardBuilder) are runtime-relevant → embed in Go (export once to JSON).

## 7. Tests (39 files, ~340 methods; sqlite :memory:, CACHE_STORE=array)
Unit (62): CycleStatusResolverTest 18, DailyCardBuilderTest 15, CycleMetricsCalculatorTest 9, OpenPeriodEvaluatorTest 7, CycleConfidenceCalculatorTest 5, CyclePhaseMapperTest 4, CyclePredictionServiceTest 3. Feature: PeriodLogApiTest 30, RecommendationTest 23, DailyChallengeTest 19, PregnancyApiTest 18, MultiLanguageTest 15, InfoSectionTest 13, ArticleApiTest 12, BannerTest 12, DailyHealthLogApiTest 12, SessionTokenTest 12, CycleRecalculationStatusTest 11, CycleViewApiTest 8, PhaseContentApiTest 8, CycleEngineCacheTest 8, CycleCalculationApiTest 6, OnboardingPeriodLogTest 6, DailyMessagesApiTest 6, Home*SectionTests, Profile tests, OtpNoBypassTest, OtpProfileCompletionTest. Not portable: HotEndpointQueryBudgetTest, AdminPanelTest, AdminArticleTest, MessageContentTest, CycleHistorySectionsTest. Blockers for a language-agnostic suite: only 4 tests freeze time → add test clock to both stacks; `actingAs` → real OTP login fixture; sqlite → MariaDB.

## 8. PHP → Go pitfalls
1. Float formatting (`65.0`, interpolated `12.5`/`12` — `HealthDataEngine.php:720`, `HomeContext::num`).
2. `round()` half-away-from-zero with decimal precision — `math.Round(x*100)/100` differs (1.955). Tested helper.
3. Decimal casts → JSON strings; nullable → null.
4. Plain `date` cast → previous-day UTC.
5. Empty array vs object; nil slice → `null` in Go. Decide per field.
6. Locale-dependent types: `localizeCalculation` returns string or whole `{en,fa}` object (e.g. `ar`); same for PregnancyWeeklyContent.
7. "fa or English" hardcoding vs registry fallback — keep both.
8. Carbon 3 signed float diff + `(int)` vs `round`.
9. PHP truthiness/null arithmetic (`null+1=1`, `!$week`, `?:`, `empty([])`).
10. Missing attributes read as null (dead-field reads).
11. `array_unique` keeps first; `usort` stable on PHP 8.4; `sortByDesc` ties.
12. Paginator JSON shape.
13. Enum strings exact.
14. No month overflow risk (only day arithmetic, `createFromDate(y,m,1)`, `->age`).
15. PHP-serialized cache/jobs tables; LanguageRegistry cache stale if Go writes `languages` while PHP runs; `crc32` = Go `crc32.ChecksumIEEE`.

## 9. Go package mapping
`platform/civildate`, `platform/db|cache|clock`, `platform/jsonx`, `auth`, `i18n`, `enums`, `cycle/{metrics,resolver,legacy,view,periods}` (periods = `PeriodLogController` 783 lines + `ProfileController::syncOnboardingPeriodLog`), `recommendation`, `healthlog`, `pregnancy/{calc,alerts,content}`, `messages`, `home`, `content` (articles, banners, info, phase, affirmations, tasks, challenges, reminders, notifications, sanitizer, image optimizer), `sms`, `notify`, `admin`.
