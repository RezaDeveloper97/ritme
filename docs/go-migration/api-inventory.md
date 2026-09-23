# API inventory (Laravel `backend/`, captured 2026-09-23)

74 routes under `/api/v1` (`routes/api.php`, = `php artisan route:list --path=api/v1`). No base controller, trait or
JsonResource: every action builds JSON by hand with `response()->json([...])`. `routes/web.php` = `GET /` only;
`routes/console.php` = `inspire` only; `routes/admin.php` = Blade panel (see infra inventory).

Global middleware: TrustProxies (`at:'*'`, `bootstrap/app.php:42`), HandleCors, PreventRequestsDuringMaintenance,
ValidatePostSize, TrimStrings, ConvertEmptyStringsToNull. `api` group: SubstituteBindings + `SetLocale`
(`bootstrap/app.php:52`). No global API throttle.

References: `app/Http/Controllers/Api/V1/<Controller>.php:<line>`.

## 1. Endpoints by domain

### 1.1 Auth / OTP — M
| Method | Path | Middleware | Action | Notes |
|---|---|---|---|---|
| POST | /auth/send-otp | throttle:5,1 | OtpAuthController@sendOtp :87 | `mobile` required regex `^09[0-9]{9}$`. 429 `{success:false,message,data:{retry_after}}` if OTP sent <60s ago (`config/sms.php:67-71`). Deletes old OTPs, creates `otp_verifications`, dispatches `SendOtpSmsJob` (errors swallowed). `data:{expires_in:120}`. Never reveals `new_user`. |
| POST | /auth/verify-otp | throttle:10,1 | @verifyOtp :244 | `mobile`, `code` size:4. 400 no OTP / expired; 429 after 5 attempts (atomic `UPDATE … WHERE attempts<5`); 422 wrong code (`hash_equals`); 403 blocked user (hard-coded Persian). Success `data:{user,new_user,profile_completed,access_token,token_type:"Bearer"}`. |
| POST | /auth/logout | auth:api | @logout :379 | revoke current token. `{success,message}` |
| GET | /auth/user | auth:api | @user :484 | `data:{user,profile_completed}` |
| POST | /auth/refresh-session | auth:api, throttle:10,1 | @refreshSession :419 | expiry >30d away → `{refreshed:false,expires_at}`; else issue new, revoke old → `{refreshed:true,access_token,token_type,expires_at}`. `expires_at` = `toIso8601String()` (+03:30). |

401 contract (`bootstrap/app.php:75-107`): `{"message":"Unauthenticated.","error_code":"token_expired|token_revoked|unauthenticated"}` compact. `token_expired` only when expiry is the sole violation. Clients drop the session only on these.

### 1.2 Profile / account — M
| Method | Path | Action | Notes |
|---|---|---|---|
| GET | /profile | ProfileController@show :73 | `data:{user,profile,bmi}`; bmi from `BmiService::forProfile` (value, category, category_label, message). Quirk: `data.user` also contains nested `profile`. |
| POST | /profile | @store :162 | unknown keys → 422 `{success:false,message,allowed_fields}` (:176-188). Rules (:190-202): name ≤255; birthday date before:today; weight 20-300; height 50-250; period_duration 1-15; cycle_duration 15-60; last_period_start before_or_equal:today; enums user_goal, subscription_type, pregnancy_intention, chronic_conditions[]. Messages from `lang/{fa,en}/profile.php`; 422 body `{success:false,message:__('profile.validation_failed',{first}),errors}`. Side effects: derive user_goal from pregnancy_intention; seed last_period_start=today for non-pregnant; `syncOnboardingPeriodLog` :462; `markRecalculated`; Telegram notice on name change. Response `data:{user,profile,calculation_status}`; errors caught → 422/500 `{success:false,message,error}`. |
| GET | /profile/export | @export :355 | `data:{exported_at,account{id,name,mobile,created_at},profile,health_logs[],cycle_histories[],pregnancy{profile,symptom_logs,weekly_logs,fetal_movements},reminders[]}` raw models |
| DELETE | /account | @destroyAccount :405 | revoke all tokens, delete user; fa/en message |

### 1.3 Languages / translations / info (public) — S-M
| Method | Path | Action | Notes |
|---|---|---|---|
| GET | /languages | LanguageController@index :48 | `data:{default,languages:[{code,name,english_name,direction,is_default}]}` (LanguageRegistry, cached forever, bootstrap fa/en fallback) |
| GET | /languages/{code}/messages | @messages :100 | never 404 (unknown → default). `data:{locale,direction,messages}`; `TranslationStore::bundle` from `storage/app/translations/<code>/` + default fill |
| GET | /info/{group} | InfoController@show :70 | group ∈ help,privacy,terms,about else `abort(404)`; `?locale=` only fa/en. `data:{group,sections:[InfoSection::toLocalizedArray]}` (`Models/InfoSection.php:112`) |
| GET | /privacy | @privacy :101 | deprecated alias of /info/privacy (keep for old app builds) |

### 1.4 Banners / articles (auth:api) — S (sanitizer M)
| Method | Path | Action | Notes |
|---|---|---|---|
| GET | /banners | BannerController@index :55 | `?position=` (home_top/home_middle/home_bottom; invalid ignored). `data:{positions:{home_top:[...],...}}`, item `{id,title,image_url,position,link_url,link_type}` |
| GET | /articles | ArticleController@index :70 | `$request->validate` page, per_page 1-50 (12), category, q ≤100. Search `title->locale`, `excerpt->locale` + fa. `data:{items,categories,meta:{current_page,last_page,per_page,total}}` |
| GET | /articles/{slug} | @show :132 | 404 `{success:false,message}`. `data:{article: summary+body (HtmlSanitizer::clean), related: ≤4}` |

Article summary (:241): `{id,slug,title,excerpt(plain),image_url,read_time_minutes,category,cycle_phases,published_at(Y-m-d)}`. `image_url` = `APP_URL/storage/<path>` (`Banner.php:55`, `Article.php:70`).

### 1.5 Reminders (auth:api) — S
| Method | Path | Action | Notes |
|---|---|---|---|
| GET | /reminders/enums | ReminderController@enums :48 | `data:{types:[{value,label,icon}],recurrences:[{value,label}]}`; 500 for 3rd locale (D-01) |
| GET | /reminders | @index :96 | `?type=`; `data: Reminder[]` raw |
| POST | /reminders | @store :145 | 201 `data: Reminder` |
| PUT | /reminders/{id} | @update :202 | `sometimes` rules; `data: fresh`; 404 `{success:false,message}` |
| DELETE | /reminders/{id} | @destroy :244 | `{"success":true}` |

Rules (:258): type ∈ doctor,medication,appointment,custom; title ≤255; subtitle; notes ≤2000; scheduled_at date; recurrence ∈ none,daily,weekly,monthly; recurrence_time `H:i`; starts_on; ends_on after_or_equal:starts_on; is_active. 422 `{success:false,message(fa|en),errors}`. `{id}` typed int → non-numeric = 500 (D-02). `recurrence_time` echoes `"16:00"` on create, `"16:00:00"` on read.

### 1.6 Daily health logs (auth:api) — M
| Method | Path | Action | Notes |
|---|---|---|---|
| GET | /health-logs/enums | DailyHealthLogController@enums :424 | `DailyHealthLog::getEnumValues()` (`Models/DailyHealthLog.php:259`) |
| GET | /health-logs | @index :66 | `from_date`/`to_date`; raw LengthAwarePaginator (per_page 30, absolute URLs, links labels `&laquo; Previous` / `Next &raquo;`) |
| POST | /health-logs | @store :198 | `StoreDailyHealthLogRequest` (~70 rules; `prepareForValidation` string exercise_type → array). Upsert (user_id,log_date): **201 created / 200 updated**; optional top-level `warning` (`CycleHistoryService::getSpottingWarning`); side effects `checkAndUpdatePeriodStart`, `markRecalculated`. On create `data` = sent attrs + user_id,id,timestamps; on update full row. |
| GET | /health-logs/{date} | @show :294 | 404 `{success:false,message:"No health log found for this date"}` |
| DELETE | /health-logs/{date} | @destroy :358 | same 404 |

### 1.7 Cycle engine + period log (auth:api) — L
| Method | Path | Action | Notes |
|---|---|---|---|
| GET | /cycle/status | CycleCalculationController@status :350 | `data:{status,status_label,is_processing,version,started_at,completed_at}` |
| GET | /cycle/today | @today :104 | `data:{calculation(localized),cycle_view(CycleDayViewBuilder),calculation_status,is_recalculating}` |
| GET | /cycle/date/{date} | @forDate :161 | `Carbon::parse`; failure 422 `{success:false,message}` |
| GET | /cycle/month/{year}/{month} | @month :252 | `?view=full`(default)/`calendar`, else 422. `data:{calculations[],calculation_status,is_recalculating,month_summary{fertile_days,period_days,pms_days}}`. **JSON_UNESCAPED_UNICODE**. In full view `text_flags`/`daily_tips` stay raw `{en,fa}`. |
| POST | /cycle/recalculate | @recalculate :425 | 400 no profile; `data:{version,status:"completed"}` |
| GET | /cycle/period/status | PeriodLogController@status :67 | `data:{active,period_start_date,period_end_date}` |
| POST | /cycle/period/start | @start :106 | `date` nullable ≤today. 422 with `code: previous_period_open` (+`data.open_period_start`) or `period_overlap`. `data:{active,period_start_date,period_end_date,warnings[{type,message}]}` |
| POST | /cycle/period/end | @end :346 | 422 no ongoing period / end<start |
| GET | /cycle/period/history | @history :441 | `data:{periods:[{id,period_start_date,period_end_date,source}]}` |
| POST | /cycle/period | @store :241 | start_date required ≤today; end_date ≥start (may be future). `data:{id,active,...,warnings}` |
| PUT | /cycle/period/{period} | @update :489 | 404 `{success:false,message}`; no `active` key |
| DELETE | /cycle/period/{period} | @destroy :577 | `data:{deleted:true}` |
| GET | /cycle/phase-content/{phase} | PhaseContentController@show :75 | invalid subphase 422; missing 404. `data:{phase,phase_label,sections}`; locale clamped fa/en (default fa) |

Period-log validation errors use `message = first error`. Engine cache 24h, per user (`CycleEngineCache.php`), cold at cutover.

### 1.8 Pregnancy (auth:api) — M-L
| Method | Path | Action | Notes |
|---|---|---|---|
| POST | /pregnancy/activate | PregnancyProfileController@activate :58 | `data:{pregnancy_mode,cycle_mode,onboarding_required}` |
| POST | /pregnancy/deactivate | @deactivate :115 | |
| POST | /pregnancy/onboarding | @onboarding :196 | `StorePregnancyProfileRequest` (age_source + required_if; Persian messages). **201** `data:{profile,gestational_age,estimated_due_date}` |
| GET | /pregnancy/profile | @show :297 | 404 if missing; `data:{profile,status}` |
| PUT | /pregnancy/profile | @update :376 | `UpdatePregnancyProfileRequest` |
| POST | /pregnancy/confirm | @confirm :472 | `data:{is_locked:true}` |
| GET | /pregnancy/status | @status :537 | `getPregnancyStatus()`; raw `{en,fa}` in `formatted`, `flags` |
| GET | /pregnancy/enums | @enums :579 | age_sources, confidence_levels, blood_types, rh_factors, pre_existing_conditions, alert_levels |
| GET | /pregnancy/symptoms/enums | PregnancySymptomController@enums :364 | |
| GET | /pregnancy/symptoms | @index :65 | `?from&to`; `data:{logs,count}` |
| POST | /pregnancy/symptoms | @store :159 | **201** `data:{log,alerts}` (`PregnancyAlertService::processSymptomLog`) |
| GET/DELETE | /pregnancy/symptoms/{date} | @show :227 / @destroy :300 | show adds `pregnancy_week` |
| GET | /pregnancy/weekly/enums | PregnancyWeeklyController@enums :508 | |
| GET | /pregnancy/weekly | @index :57 | |
| POST | /pregnancy/weekly | @store :132 | upsert by week, **201** |
| GET | /pregnancy/weekly/{week} | @show :199 | int; outside 1-42 → 422 |
| GET/POST | /pregnancy/fetal-movement | @indexFetalMovement :362 / @storeFetalMovement :277 | POST **201**, updates `fetal_movement_felt` |
| GET | /pregnancy/content/{week} | @content :432 | 1-40 else 422; locale clamped fa/en (default **en**) |
| GET | /pregnancy/alerts/summary | PregnancyAlertController@summary :378 | |
| GET | /pregnancy/alerts | @index :68 | `?level&unread_only`; `data:{alerts,counts{total,unread,emergency,warning,info}}` |
| GET | /pregnancy/alerts/{id} | @show :143 | |
| POST | /pregnancy/alerts/{id}/read | @markAsRead :203 | |
| POST | /pregnancy/alerts/read-all | @markAllAsRead :258 | |
| POST | /pregnancy/alerts/{id}/dismiss | @dismiss :315 | |

### 1.9 Messages (auth:api) — L
| Method | Path | Action | Notes |
|---|---|---|---|
| GET | /messages/daily | MessageController@daily :111 | query `date` Y-m-d, `mode` ∈ cycle,pregnancy; 422 `{success,message,errors}`; 400 incomplete profile. `data: MessageResult::toArray()` = `{mode,date,user_goal,subscription_type,context_info,primary_message,correlations,patterns,supplements{nutrition,sleep,exercise},tips}` |
| GET | /messages/mode | @mode :217 | `{mode,mode_label,user_goal,subscription_type,is_ttc,is_premium}` |

### 1.10 Home + notifications (auth:api) — L
| Method | Path | Action | Notes |
|---|---|---|---|
| GET | /home | HomeController@index :75 | `?date` framework 422. `data:{mode,mode_label,date,locale,profile_complete,sections[]}` |
| GET | /home/sections/{section} | @section :153 | unknown → 404 `{success:false,message,available_sections}`; `data:{section|null}` |
| POST | /home/tasks/{task}/toggle | @toggleTask :209 | model binding (framework 404). `data:{task_id,is_completed,progress{completed,total,percent}}` |
| POST | /home/challenges/{challenge}/toggle | @toggleChallenge :271 | `data:{challenge_id,is_completed}` |
| GET | /home/notifications | @notifications :313 | per_page 1-100 (20). `data:{unread_count,items[{id,type,title,body,action_url,is_read,read_at,created_at}],pagination{current_page,last_page,per_page,total}}` (ISO +03:30) |
| POST | /home/notifications/read-all | @markAllNotificationsRead :416 | `data:{marked}` |
| POST | /home/notifications/{notification}/read | @markNotificationRead :375 | other user's → abort(404). `data:{unread_count}` |

`HomeSection::toArray` = `{key,type,title,subtitle,order,action,data,meta}` (meta null when empty). 17 sections (see domain inventory §4.4). A throwing section is logged and omitted.

### 1.11 Non-v1 routes
`GET /up`; `GET /storage/{path}`; l5-swagger `/api/documentation`, `/docs`, `/docs/asset/{asset}`, `/api/oauth2-callback` behind `swagger.auth` Basic auth (`Middleware/SwaggerBasicAuth.php`); Passport `/oauth/*` (unused by clients).

## 2. Envelopes and errors
**A. Controller bodies (compact):** success `{"success":true[,"message"],"data":...}` (key order literal, e.g. health-logs POST `success,message,data[,warning]`); error `{"success":false,"message":...}` + optional `errors`, `code`, `data.retry_after`, `data.open_period_start`, `allowed_fields`, `available_sections`, `error`.

**B. Framework bodies (no `success`):**
- 422 from FormRequest / `$request->validate` (health-logs POST, pregnancy onboarding/PUT profile/symptoms/weekly/fetal-movement POST, GET /articles, GET /home, /home/sections, task toggle): `{"message":"<first error> (and N more errors)","errors":{field:[...]}}`. Messages from `lang/fa/validation.php` or framework en; `:attribute` humanised (`log_date` → "log date"). JSON only when `Accept: application/json` (both clients send it: `frontend/src/shared/api/apiClient.ts:121`, `application/.../HttpClient.kt:75`).
- 401 — §1.1.
- Pretty-printed (`JSON_PRETTY_PRINT|JSON_UNESCAPED_SLASHES`, 4-space; `Foundation/Exceptions/Handler.php:1002-1028`): 404 unknown route `"The route api/v1/x could not be found."`; binding miss `"No query results for model [App\\Models\\TaskTemplate] 99"`; `abort(404)` (capture golden); 405 `"The GET method is not supported for route ... Supported methods: POST."`; 429 `{"message":"Too Many Attempts."}` + `Retry-After`, `X-RateLimit-Limit/Remaining/Reset` (successful throttled responses carry Limit/Remaining; key = user id if authed else IP); 500 `{"message":"Server Error"}`; 503 maintenance.

**Pagination shapes:** (1) raw Laravel paginator (`/health-logs`), (2) `data.pagination{current_page,last_page,per_page,total}` (notifications), (3) `data.meta{...}` (articles).

## 3. Middleware, CORS, locale, rate limits
- Locale (`Middleware/SetLocale.php`, `Controllers/Concerns/ResolvesLocale.php`, `LanguageRegistry::resolve`): `?locale=` → `Accept-Language` (split `,`, strip `;q`, lowercase, `_`→`-`, exact or base match) → default (fa). Many strings are `$locale==='fa' ? fa : en`. Web sends Accept-Language + `?locale=` for phase-content/info/pregnancy content; **Android sends no Accept-Language → default fa**.
- CORS (`config/cors.php`): `api/*`, methods/headers `*`, origins `CORS_ALLOWED_ORIGINS` (default `https://web.ritme.app`, `http://localhost:3000`, `http://127.0.0.1:3000`), max_age 3600, no credentials; echo origin + `Vary: Origin`, preflight 204.
- Throttles only on the 3 auth routes.
- TrimStrings + ConvertEmptyStringsToNull matter for `nullable`/`has()`/`filled()`.
- Trust any proxy (XFF/Host/Port/Proto) — affects paginator URLs.

## 4. OpenAPI
`backend/storage/api-docs/api-docs.json` (OpenAPI 3.0, ~400 KB, **not in git**), 63 paths / 74 ops. Use as a route/field checklist only — misses framework errors, date formats, decimal strings, 201/200; several examples stale.

## 5. Dates, numbers, translatable fields
App tz `Asia/Tehran`; DB stores Tehran wall-clock; prod MariaDB 11.4.

| Source | Output | Where |
|---|---|---|
| `datetime`/timestamps | `2026-09-23T09:30:00.000000Z` | created/updated_at, mobile_verified_at, reminders.scheduled_at, alert read_at, onboarding_completed_at, /cycle/status started/completed_at |
| plain `date` | midnight Tehran → UTC = previous day `2026-09-22T20:30:00.000000Z` (pre-2022 DST summer `19:30Z`) | PregnancyProfile dates, Reminder starts_on/ends_on, CycleHistory in export |
| `date:Y-m-d` | `2026-09-23` | UserProfile birthday/last_period_start, all `log_date` |
| `toIso8601String()` | `2026-09-23T13:00:00+03:30` | notifications, export, refresh-session expires_at, doctor-reminder section |
| `toDateString()` | `Y-m-d` | period endpoints, articles published_at, home date, EDD date |

No Jalali anywhere (Persian EDD text = Gregorian months in Persian, `PregnancyCalculationService.php:414`); Western digits. `decimal:N` → strings (`"65.50"`, blood_sugar `"98.5"`); UserProfile weight `float` → number. Translatable JSON `{fa,en,...}` via `Translatable::pick`, except raw blobs: `/cycle/month?view=full` text_flags/daily_tips/source_*, `/pregnancy/status` formatted/flags, onboarding gestational_age.formatted/estimated_due_date.formatted.

## 6. Client usage
- Web: `frontend/src/shared/api/apiClient.ts` (+ raw fetch in `shared/i18n/registry.ts:107`, `shared/i18n/messages.ts:72`); envelope type `shared/api/envelope.ts`; month `?view=calendar`; home sections used: articles, challenge, weekly_summary, my_cycles, cycle_summary.
- Android: `application/app/src/main/java/ir/ritmeapp/ritme/adapter/outbound/network/ApiConfig.kt` + `*GatewayAdapter.kt`; month full view; checks `success` everywhere.
- Unused by both clients (check nginx logs before dropping; old Android builds): GET /home, POST /home/tasks/{task}/toggle, GET /health-logs (list), DELETE /health-logs/{date}, PUT /pregnancy/profile, POST /pregnancy/confirm, GET /pregnancy/symptoms, DELETE /pregnancy/symptoms/{date}, GET /pregnancy/weekly, GET /pregnancy/alerts/{id}, GET /privacy.
- Android calls missing `POST /api/diagnostics/crash-reports` (404) — D-07.

## 7. Must reproduce
1. JSON encoding: PHP flags 0 (`\/`, `\uXXXX`, `<>&` unescaped → Go `SetEscapeHTML(false)`); `/cycle/month` raw UTF-8; framework errors pretty + unescaped slashes; empty PHP arrays `[]`; `10.0` → `10`. (Harness normalises escaping; types/values must match.)
2. Both error families incl. 422 summary text and fa/en default messages; pretty 404/405/429/500/503 + headers.
3. Raw model serialization (User, UserProfile, DailyHealthLog, Reminder, Pregnancy*, PregnancyAlert, CycleHistory): all columns, `$hidden`, casts, loaded relations nested; fresh-created models only set attrs.
4. Status codes: 201 for health-log create, reminder create, pregnancy onboarding/symptoms/weekly/fetal-movement.
5. Passport JWT compatibility, revoked checks, 401 error codes, 365-day lifetime, 30-day refresh window.
6. Locale algorithm, per-endpoint fa/en clamps, raw bilingual blobs.
7. Tehran time + real tz database.
8. Static routes before param routes (`enums`, `summary`, `read-all`); GET answers HEAD.
9. Write side effects: markRecalculated, cycle-history reconciliation, onboarding period seed, alert generation, OTP queue, Telegram.

## 8. Complexity
Auth/OTP M · Profile M · Languages/Info S-M · Banners/Articles S (sanitizer M) · Reminders S · Daily logs M · Cycle + period **L** · Phase content S · Pregnancy M-L · Messages **L** · Home **L** (notifications S).
