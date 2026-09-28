# Periodic checkups (M4) — screening plan, records, self-exam, admin catalog

Source design: the Design canvas (claude.ai artifact `LQgpWxuwxq5oEuwmhCYsvf`), v14 series. Local copies, light
(`nbl_`) and dark (`nbd_`), are in [`docs/design/checkups-v14/`](../design/checkups-v14/). The inline styles are the
spec (sizes, radii, weights, copy).

| Artboard | What | Frontend route / slot |
|---|---|---|
| `v14_Main` → «چکاپ‌های دوره‌ای» card | progress ring `x/y`, counts line, up to 2 action rows (due = rose tint + «راهنما», overdue = amber tint + «ثبت نوبت»), disclaimer | widget `checkups-card` on the **cycle home** (not in pregnancy mode) |
| `v14_Checkups` | header «بر اساس سن N سال», status card (stacked bar + legend), tabs همه / نیاز به اقدام / انجام‌شده, sections این ماه / عقب‌افتاده / سالانه / هر ۶ ماه / بر اساس سن, info note, «افزودن چکاپ سفارشی» | `/[locale]/checkups` |
| `v14_CheckupDetail` | hero (category chip, status pill, next due, «انجام دادم» + «ثبت نوبت»), چرا مهم است؟, قبل از رفتن (numbered), سابقه (last 2 + «همه»), disclaimer | `/[locale]/checkups/[id]` |
| `v14_MarkDone` | bottom sheet: date, result (نرمال / نیاز به پیگیری / منتظر جواب), optional attachment (photo / PDF — **stored on the device only**), note, computed next-due banner with «تغییر», «ثبت» | sheet id `checkup-mark-done` |
| `v14_History` | tabs همه / امسال / با پیوست, timeline of records (date, checkup, result chip, attachment chip), «خلاصه برای پزشک» PDF | `/[locale]/checkups/history` (optionally `?type=`) |
| `v14_SelfExam` | cycle-timed hero with day ring, 3 steps guide, findings chips, «این ماه انجام دادم», 12-month adherence line | `/[locale]/checkups/self-exam` (the guide of the `breast_self_exam` type) |

Flow: home card «همه» → Checkups; «راهنما» → SelfExam; «ثبت نوبت» → M3 AddAppointment
(`/reminders/appointment/new?kind=in_person&topic=…&title=…`); a row → CheckupDetail; «انجام دادم» → MarkDone
sheet → back to the detail with the new status; «همه» under سابقه → History filtered to that type.

## Backend decision

Same as M3: **Go only** (`backend-go/`), user API under **`/api/v1/checkups`**, admin API under
**`/api/admin/v1/checkup-types`**. New tables ship as goose migrations **plus** schema-only Laravel twins
(docs/go-migration/migrations.md); `make schema-diff` compares row counts, so the default catalog seed is in both.

## Data model

`checkup_types` (admin catalog; `user_id` set = the user's custom checkup):

| column | notes |
|---|---|
| `id`, `key` (unique, null for custom), `user_id` (nullable FK, cascade) | |
| `category` | `monthly` / `six_monthly` / `annual` / `multi_year` / `age_based` / `custom` |
| `title`, `subtitle`, `why` | bilingual JSON `{"fa":…,"en":…}` (same pattern as other admin content) |
| `performed_by` | `self` / `doctor` / `lab` / `dentist` |
| `icon`, `tone` | icon name from the frontend set; tone `rose/violet/amber/teal/green/neutral` |
| `interval_months`, `interval_months_max` | 1, 6, 12, 36, or a range (mammography 12–24) |
| `age_min`, `age_max` | eligibility from the profile birthday; outside → status `not_yet` with the start year |
| `cycle_day_from`, `cycle_day_to` | cycle-timed items (self-exam 7–10, Pap best 10–20) |
| `remind_lead_days` | e.g. 30 for multi-year, 3 for monthly |
| `prep_steps`, `guide_steps`, `finding_options` | JSON arrays of bilingual items (guide = title+body; findings = chips) |
| `hide_in_pregnancy`, `is_active`, `sort_order`, `source_note` | |

`checkup_records`: `id, user_id, checkup_type_id, done_on date, result (normal|follow_up|pending),
findings json null (self-exam chips), note text null, has_attachment bool, next_due_on date null (user override),
timestamps`; index `(user_id, checkup_type_id, done_on)`. **The attachment file never leaves the phone**
(IndexedDB, keyed by record id) — the server only stores `has_attachment`.

`user_checkup_settings`: `user_id, checkup_type_id, enabled, remind, unique(user_id, checkup_type_id)`.

Default catalog seed (content needs a medical review before production — see T-M4-01): breast self-exam (monthly,
cycle day 7–10, guide + findings), clinical breast exam (annual, doctor), Pap smear / HPV (every 3 years, age 21–65,
cycle day 10–20), full blood test (annual — CBC, thyroid, vitamin D, iron), dentist (6 months), mammography (from 40,
every 1–2 years).

Seeded by `backend-go/db/migrations/00003_checkups.sql` and its twin
`backend/database/migrations/2026_09_24_000001_create_checkup_tables.php` (identical values, idempotent on `key`;
`source_note` = "Needs medical review before production"):

| key | category | by | icon / tone | interval | age | cycle days | lead | hidden in pregnancy |
|---|---|---|---|---|---|---|---|---|
| `breast_self_exam` | monthly | self | `ribbon` / rose | 1 | – | 7–10 | 3 | yes |
| `clinical_breast_exam` | annual | doctor | `breast` / violet | 12 | – | – | 30 | no |
| `pap_smear` | multi_year | doctor | `flask` / violet | 36 | 21–65 | 10–20 | 30 | no |
| `blood_test` | annual | lab | `blood` / amber | 12 | – | – | 14 | no |
| `dentist` | six_monthly | dentist | `tooth` / teal | 6 | – | – | 14 | no |
| `mammography` | age_based | lab | `shieldCheck` / rose | 12–24 | 40+ | – | 30 | yes |

JSON shapes (every text is a bilingual object keyed by language code):
`prep_steps` `[{"fa":…,"en":…}]` · `guide_steps` `[{"title":{…},"body":{…}}]` ·
`finding_options` `[{"key":"none","exclusive":true,"label":{…}}, {"key":"lump","label":{…}}, …]` —
`checkup_records.findings` stores the chosen `key`s; `exclusive` = the «چیزی متفاوت نبود» chip that clears the others.
`remind_lead_days` defaults to 7, `tone` to `neutral`.

## Status engine (`internal/checkups/engine`, pure, clock-injected)

`engine.Evaluate(Input, clock)`; `checkups.EngineInputs(ctx, q, userID)` loads types/records/settings (3 queries);
birthday, pregnancy and the cycle prediction (`engine.CycleFromHistory` — the cycle engine's resolved current period
start + effective length) come from the caller. Package doc in `engine.go` is the authoritative rule list.

For each applicable type (active; not `hide_in_pregnancy` while pregnant; age ≤ `age_max` when the birthday is known):
`last = latest record.done_on` (ties → higher id); `next_due = that record.next_due_on ?? last + interval` (calendar
months, clamped to month end). A range interval (12–24) is due from the minimum and **overdue only after the maximum**.
A **monthly** cycle-timed type is due at the `[from,to]` window of the first predicted cycle starting after `last`
(never recorded → this cycle's window, or the next one once it has passed) and overdue after the window end; without
cycle data it falls back to last + 1 month. Longer cycle-timed types (Pap) keep their calendar date — the window is
advice for the reminder/label only. Status: `not_yet` (age below `age_min`; `next_due_on` = the day she reaches it,
for «از ۱۴۰۹ (۴۰ سالگی)») · `overdue` (today after the due-by date) · `due` (never recorded, or today ≥ next_due − lead)
· `soon` (within 60 days) · `up_to_date` · `disabled` (setting `enabled=false`; still listed, left out of the summary).
Section: `this_month` for a cycle-timed type that is overdue, or due with next_due inside the current **Jalali** month
(`engine.ToJalali` / `JalaliMonthEnd`), else `overdue`, else the category.
Summary over enabled items: `total`, `up_to_date` (= up_to_date + soon + **not_yet**, matching the artboard ring
«۴ از ۶» where the not-yet mammography counts), `due`, `overdue`.

## API contract (Go, auth:api, Accept-Language, standard envelope)

| Method | Path | Notes |
|---|---|---|
| GET | `/api/v1/checkups` | `{age, summary:{total,up_to_date,due,overdue}, items:[{id,key,title,subtitle,category,section,status,icon,tone,interval_label,timing_label,last_done_on,next_due_on,next_due_label,is_custom}]}` (`?filter=action|done`) |
| GET | `/api/v1/checkups/home` | `{summary, highlights:[max 2 items: due/overdue, cycle-timed first]}` — hidden when nothing applies |
| GET | `/api/v1/checkups/{id}` | item + `why`, `prep_steps`, `guide_steps`, `finding_options`, `records` (latest 2), `settings` |
| POST | `/api/v1/checkups/{id}/records` | `{done_on,result,findings?,note?,has_attachment,next_due_on?}` → 201 + recomputed item |
| PUT/DELETE | `/api/v1/checkups/records/{recordId}` | |
| GET | `/api/v1/checkups/records` | `?filter=all|this_year|with_attachment&type=` newest first, paginated |
| GET | `/api/v1/checkups/preview-next` | `?type=&done_on=` → next due + reminder label for the MarkDone banner |
| POST/PUT/DELETE | `/api/v1/checkups/custom[/{id}]` | user's custom type (title, interval, performed_by, note) |
| PUT | `/api/v1/checkups/{id}/settings` | `{enabled, remind}` |
| GET/POST/PUT/DELETE | `/api/admin/v1/checkup-types[/{id}]` | admin CRUD (`editor` and `super`), bilingual fields, validation |
| POST | `/api/admin/v1/checkup-types/reorder` | `{ids:[…]}` |
| GET | `/api/admin/v1/checkup-types/stats` | per type: users with records, records last 30 days, overdue users |

## Colors — light → dark (from the `nbd_` artboards)

Frontend rule (frontend/CLAUDE.md §10): tokens only, both themes, `lint:dark` green. On top of the M3 table
(docs/care-reminders/README.md):

| role | light | dark |
|---|---|---|
| status up-to-date (dot, bar, pill) | `#0F7B6C`, tint `#E7F8EF` | `#7FE0A8`, tint `#1F3A2E` |
| status due (bar/dot) | `#F5A623` | `#FFB86B` |
| due pill / overdue row tint | `#B45309` on `#FFF3DF` | `#FFC98A` on `#3A2F1E` |
| overdue (bar, pill) / self-exam row | `#C42D57` on `#FDE4EB` | `#FF8FA3` on `#3A2140` |
| soon pill | `#0C87A7` on `#E3F9FE` | `#6FE7D0` on `#1E3340` |
| body copy (why / steps) | `#4E4E59` | `#C9C0E6` |
| hero chip line / dashed upload border | `#DDD6FA` | `#3B2F63` (chips `rgba(255,255,255,.3)`) |
| white action button inside tinted row | `#FFFFFF` | `#221A3D` |

Big numbers and dates in Lalezar; Jalali dates for fa; all digits localized; ≥ 44 px targets.

## Rollout (T-M4-10) — staging 2026-09-26

**Routing / deploy:** no redeploy — the code was shipped to stage by T-M3-09 (`stage` @ 484a4ad, goose v5 with
`checkup_types`, `checkup_records`, `user_checkup_settings`). Staging is Go-only (`location /api/` → `stage-backend-go`),
so `/api/v1/checkups/*` and `/api/admin/v1/checkup-types*` need no vhost change. Go logs list domains
`checkups` and `admin_checkups`. Admin API is host-gated (`ADMIN_HOSTS=stage.ritmeapp.ir`). Production still needs a
`checkups` Go route line (and admin-web) when the user asks.

**API e2e** (run on the server against `stage-backend-go` on `ritme-edge`; user 09900000901, OTP read from
`otp_verifications`):
```
GET checkups 200 (summary total 6, due 5, up_to_date 1; 6 items) | GET checkups/home 200 (2 highlights)
GET checkups/1 200 status due, 0 records | GET preview-next 200
POST checkups/1/records 201 (has_attachment) → item status soon, next_due_on 2026-10-14
GET checkups/records?type=1 200 contains the record | PUT records/{id} 200 | DELETE records/{id} 200
PUT checkups/1/settings 200
POST custom 201 → PUT custom/{id} 200 → GET checkups/{id} 200 is_custom:true, new title → DELETE 200 → GET 404
leftovers for the user: 0 smoke records, 0 custom types
```
**Admin API e2e** (temporary `editor` admin inserted with a random throwaway password, session + CSRF cookies,
`Host: stage.ritmeapp.ir`; deleted afterwards):
```
POST auth/login 200 | GET checkup-types 200 (6) | POST checkup-types 201 (key tm410_smoke, fa+en)
PUT checkup-types/{id} 200 interval 24 | POST reorder 200 → sort_order 1 | PUT is_active:false 200
GET checkup-types/stats 200 | DELETE checkup-types/{id} 200 (no records) | reorder restored original order 200
cleanup: 0 tm410_smoke types, 0 smoke admins
```
Public proxy: `GET https://stage.ritmeapp.ir/api/v1/checkups` without credentials → 401 (the stage gate).

**Open (human):**
- Content sign-off of the seeded catalog copy (titles, subtitles, why, prep/guide steps, finding options, intervals,
  age ranges, cycle-day windows; fa + en) by the user or a named clinician, in admin → Checkup types.
- UI e2e in light + dark (needs the stage password): home card → list → detail → mark done with a local attachment →
  history → doctor PDF → self-exam guide; an admin edit of a type showing up in the app; screenshots into PROGRESS.
- Production rollout only when the user asks.

## Local e2e (T-M4-10) — 2026-09-27

Everything below ran locally on branch `stage`. backend-go ran on `:8020` against the docker test-stack MariaDB. It
used a scratch DB `ritme_m410` (goose up to v5, which includes `00003_checkups`), with `SMS_PROVIDER=log` and
`ADMIN_COOKIE_SECURE=false`. The Next dev server ran on `:3000` with
`NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1` passed as an env var. admin-web ran on `:3001` with
`ADMIN_API_PROXY_TARGET=http://127.0.0.1:8020`. The test user was `09120000001`, with the OTP read from the DB. Its profile
has birthday 1992-05-10 (age 34), `user_goal=non_ttc` and a 28-day cycle. Four periods were logged through
`POST /cycle/period` (06-20, 07-18, 08-15, 09-12), so today, 2026-09-27, is cycle day 16. A dentist record dated 2025-11-10
was added through the API so the overdue state shows. A temporary `super` admin `editor@local.test` was used. It existed
only in the scratch DB, which was dropped afterwards. No staging or production host was contacted.

### verify-all

| Check | Result |
|---|---|
| `go vet ./...` | ✔ pass |
| `go test ./...` | ✔ pass (63 packages ok) |
| `golangci-lint run` | ✔ 0 issues |
| `npm run typecheck` | ✔ pass |
| `npm run lint` | ✔ pass |
| `npm run fsd:lint` | ✔ No problems found |
| `npm run lint:styles` | ✔ style gate passed (457 files) |
| `npm run lint:dark` | ✔ passed (57 contrast pairs, 123 tokens; only the pre-existing light-mode ⚠ pairs) |
| `npm run test` | ✔ 67 files, 555 passed |
| Laravel (`pint`, `artisan test`) | skipped: `backend/` unchanged |

### UI flow (headless Chrome over CDP, 390×844 @2x, fa, light and dark)

Light used Pap smear (`/checkups/3`) and dark used the full blood test (`/checkups/4`). The results were the same in both themes.
The automated DOM checks passed on every screen: no horizontal overflow (no element outside the viewport, `scrollWidth`
≤ `innerWidth`), no raw i18n keys in the DOM text, and no console errors or exceptions. The one exception is the PDF step (bug 1).

| # | Step | Result |
|---|---|---|
| 1 | Cycle home → «چکاپ‌های دوره‌ای» card | ✔ ring `۱/۶` (dark run `۲/۶`), counts line, 2 highlight rows (self-exam «راهنما» rose; dark run: dentist overdue amber «ثبت نوبت»), disclaimer |
| 2 | Card «همه» → `/checkups` | ✔ «بر اساس سن ۳۴ سال», stacked status bar + legend, tabs, sections, info note, «افزودن چکاپ سفارشی»; the «نیاز به اقدام» tab sets `?filter=action` |
| 3 | Row → detail | ✔ hero (category chip, status pill, next due, «انجام دادم» / «ثبت نوبت»), چرا مهم است؟, قبل از رفتن, cycle hint, سابقه, disclaimer |
| 4 | «انجام دادم» → MarkDone sheet | ✔ date chip (today, Jalali), 3 result radios, photo/PDF buttons, privacy line, note, next-due banner (`۶ مهر ۱۴۰۸ · هر ۳ سال · ۳۰ روز قبل یادآوری می‌کنیم`) |
| 5 | Attachment via `DOM.setFileInputFiles` (`lab-report.jpg`) + «نیاز به پیگیری» + note → «ثبت» | ✔ thumbnail + file name shown; toast «ثبت شد»; detail flips to «به‌روز», next due shown, record has «گزارش پیوست شده»; the file is stored in IndexedDB `ritme-local-files` and the API only receives `has_attachment: true` |
| 6 | `/checkups/history` | ✔ timeline grouped by month, result chips, «پیوست» chip; the «با پیوست» tab filters to the attached record |
| 7 | «خلاصه برای پزشک» → PDF | ✔ `ritme-checkups.pdf` generated on the device and downloaded (59 KB light / 75 KB dark; it has title, date, records, notes, attachment line, disclaimer). ✘ footer is a raw key and the console logs an `IntlError` (bug 1) |
| 8 | `/checkups/self-exam` | ✔ cycle-day ring `۱۶ از ۲۸`, best-time box, 3 steps, findings chips, «این ماه انجام دادم», adherence line «۰ ماه از ۱۲ ماه» |
| 9 | `/checkups/custom/new` | ✔ name, interval chips, performed-by chips, last-done date, note, «ذخیره» |
| 10 | admin-web login → Checkup types list → edit `blood_test` fa title to «آزمایش خون کامل (ویرایش ادمین)» → «ذخیره تغییرات» | ✔ redirected to the list with the new title; the stats block reads users with records 3, records in the last 30 days 2, overdue users 1 |
| 11 | App `/checkups` and `/checkups/4` after the admin edit | ✔ both show the new title (no cache issue) |

### Screenshots (`screenshots/`)

App screens were captured in `-light` and `-dark`. Admin screens are light only.

| File | Shows |
|---|---|
| `home-card-*.png` | checkups card on the cycle home |
| `list-*.png`, `list-bottom-*.png`, `list-tab-action-*.png` | Checkups list: top, bottom (info note + add custom), «نیاز به اقدام» tab |
| `detail-*.png`, `detail-bottom-*.png` | CheckupDetail before marking done |
| `markdone-sheet-*.png` | MarkDone sheet, empty |
| `markdone-filled-*.png`, `markdone-filled-bottom-*.png` | sheet with local attachment, «نیاز به پیگیری», note |
| `detail-after-save-*.png` | detail after save: «به‌روز», next due, record with attachment |
| `history-*.png`, `history-with-attachment-*.png` | History: all and «با پیوست» |
| `history-pdf-*.png` | History right after the PDF trigger (it returned to idle, no error line) |
| `pdf-summary-page1-light.png` | page 1 of the generated PDF, rendered with `sips` (footer bug visible) |
| `self-exam-*.png`, `self-exam-bottom-*.png` | breast self-exam guide |
| `custom-form-*.png` | custom checkup form |
| `admin-login.png`, `admin-checkup-types-list.png` | admin-web login and catalog list + usage stats |
| `admin-checkup-type-form.png`, `admin-checkup-type-form-edited.png` | type form (preview, texts, steps, appearance, timing) before and after the title edit |
| `admin-checkup-type-saved.png`, `admin-checkup-types-list-after.png` | list after save with the new title |
| `app-list-after-admin-edit-light.png`, `app-detail-after-admin-edit-light.png` | the admin edit showing in the app |

Note: the admin form is a full-page capture, so its sticky save bar appears mid-page. That is a capture artifact, not a bug.

### Bugs found (1–4 fixed in T-M4-11)

1. **PDF footer is a raw i18n key.** Every page of the doctor summary prints `checkups.history.pdf.footer`, and the console
   logs `IntlError: FORMATTING_ERROR: The intl string context variable "page" was not provided`. The cause is
   `frontend/src/screens/checkup-history/ui/CheckupHistoryPage.tsx:82`, which calls `t('history.pdf.footer')` without values. `renderPdf`
   (`frontend/src/shared/lib/pdf/render.ts:141-143`) expects the template with literal `{page}`/`{pages}`, so the call
   needs `t.raw(…)`. The same pattern exists in `frontend/src/screens/pregnancy-calendar/ui/PregnancyCalendarPage.tsx:562`.
   `render.ts` also substitutes `String(p + 1)`, which gives Latin digits in the fa footer once the template works.
   To reproduce: open History, tap «خلاصه برای پزشک», then open the PDF (`pdf-summary-page1-light.png`).
2. **Checkups cards have no inner padding.** This is the same root cause as fertility bug 3: `.card` (`frontend/src/app/globals.css:477`)
   has no padding. Titles, body copy, the icon tile, pills and «همه» touch the card border in both themes. The affected sites are
   `frontend/src/screens/checkups/ui/CheckupsPage.tsx:57` (status card), `:102` (rows), `:218` (empty state);
   `frontend/src/screens/checkup-detail/ui/CheckupDetailPage.tsx:182,189,213` (why / prep / history);
   `frontend/src/screens/checkup-history/ui/CheckupHistoryPage.tsx:122` (record card), `:218` (summary card);
   `frontend/src/screens/checkup-self-exam/ui/SelfExamPage.tsx:121,145` (steps / findings). The custom form is fine
   because it uses `fld-card`. See `list-*.png`, `detail-*.png`, `history-*.png`, `self-exam-*.png`.
3. **Latin digits in fa.** The History header shows «2 مورد ثبت‌شده», from
   `frontend/src/screens/checkup-history/ui/CheckupHistoryPage.tsx:201` (`count: total`). The self-exam hero shows «19 روز دیگر», from
   `frontend/src/screens/checkup-self-exam/ui/SelfExamPage.tsx:67` (`count: days`). Both pass the raw count to ICU, while the
   PDF subtitle on line 61 uses `formatNumber`.
4. **Overdue section in the wrong place.** The list renders این ماه → سالانه → **عقب‌افتاده** → بر اساس سن, but the spec order
   (v14_Checkups, above) puts عقب‌افتاده second, right after این ماه. Sections follow the order in which the server first sends them
   (`frontend/src/screens/checkups/model/view.ts:48-56`), and the server sends the items in `sort_order` without ordering sections
   (`backend-go/internal/checkups/engine/engine.go:282-292` assigns them). Reproduce with any overdue item whose type sorts
   after an annual one (see `list-*.png`).
5. Minor, to confirm with design:
   - (a) The History «پیوست» chip sits outside and below the record card
     (`CheckupHistoryPage.tsx:137-147`, a sibling of the card button), while the artboard shows it inside the card.
   - (b) The detail hero always shows a `shield` icon (`CheckupDetailPage.tsx:82`) instead of the type's icon (the list shows a flask for
     Pap smear and blood test).
   - (c) admin-web mixes digits on the fa UI: the list and form use Latin («هر 12 ماه», «ثبت‌های 30 روز اخیر», «21 تا 65
     سال») while the stats values use Persian («۳», «۰»).
   - (d) The breast self-exam is «موعدش رسیده» while its row reads «بعدی: ۱۹ روز دیگر» (never done, next cycle window). The
     wording may confuse users.

### Catalog items that need content sign-off

These values come from the Go seed, `backend-go/db/migrations/00003_checkups.sql`. Every row carries `source_note` "Default catalog (T-M4-01). Needs medical
review before production." The fa and en title, subtitle, why, prep steps, guide steps and finding options all need review in
admin → چکاپ‌های دوره‌ای.

| key | fa / en title | category · by | interval | age | cycle days | remind lead | hidden in pregnancy | prep / guide / findings |
|---|---|---|---|---|---|---|---|---|
| `breast_self_exam` | خودآزمایی سینه / Breast self-exam | monthly · self | 1 month | – | 7–10 | 3 d | yes | 2 / 3 / 5 |
| `clinical_breast_exam` | معاینه بالینی سینه / Clinical breast exam | annual · doctor | 12 months | – | – | 30 d | no | 2 / – / – |
| `pap_smear` | پاپ‌اسمیر / HPV / Pap smear / HPV | multi_year · doctor | 36 months | 21–65 | 10–20 | 30 d | no | 3 / – / – |
| `blood_test` | آزمایش خون کامل / Full blood test (CBC، تیروئید، ویتامین D، آهن) | annual · lab | 12 months | – | – | 14 d | no | 3 / – / – |
| `dentist` | دندان‌پزشکی / Dentist | six_monthly · dentist | 6 months | – | – | 14 d | no | 2 / – / – |
| `mammography` | ماموگرافی / Mammography | age_based · lab | 12–24 months | from 40 | – | 30 d | yes | 3 / – / – |

### Still open (human)

- **Catalog content sign-off in admin**: the user or a named clinician signs off the table above (copy plus intervals, age
  ranges and cycle-day windows, fa and en).
- **Staging UI click-through**: the agent has no server access (ssh and curl to the staging host are denied). The same flow in light
  and dark on stage.ritmeapp.ir, with screenshots into PROGRESS, is still the user's to run.
- **Production**: only when the user asks.

## Staging UI e2e (T-M4-10) — 2026-09-28

Ran against `https://stage.ritmeapp.ir` (`stage` @ ea8a1eb, goose v6, Go-only), with no redeploy. The browser was local headless
Chrome over CDP (port 9242, private profile) at 390×844 @2x in fa, in light and dark. The gate `ritme_stage` cookie came from one page load
with a Basic `Authorization` header. The header was then dropped, so the app's Bearer calls ran on the cookie alone.
The login ran through the UI: `/fa/signup` → `09900000961` → `/fa/otp`, with the code read from the **stage** DB
`otp_verifications` (container `ritme-stage-mysql-1`). The profile was then set through the API: birthday 1992-05-10 (age 34),
`pregnancy_intention=avoiding`, a 28/5 cycle, and 4 periods through `POST /cycle/period` (06-21, 07-19, 08-16, 09-13), so today is cycle day 16.
A dentist record dated 2025-11-10 was added so an overdue item exists. Light used Pap smear (`/checkups/3`) and dark used the full blood test
(`/checkups/4`). The dark `detail-*` shots show clinical breast exam (`/checkups/2`), because the first capture caught the loading state.

**Automated checks passed on every app screen in both themes:** no horizontal overflow (`scrollWidth` = 390 and 0 elements outside the
viewport), no raw i18n keys, no Latin digits, no console errors or exceptions, and `data-theme` matched the run.

| # | Step | Result |
|---|---|---|
| 1 | Cycle home → «چکاپ‌های دوره‌ای» card | ✔ ring `۱/۶` (dark `۲/۶`), counts line, 2 highlight rows (self-exam and Pap «راهنما»; dark run: self-exam and dentist overdue «ثبت نوبت»), disclaimer. ✘ The `·` separator reads as `۰` (bug S1) |
| 2 | Card → `/checkups` | ✔ «بر اساس سن ۳۴ سال», status bar and legend, and cards with padding. **عقب‌افتاده now comes second, right after این ماه** (local bug 4 is fixed). «نیاز به اقدام» sets `?filter=action` |
| 3 | Row → detail | ✔ hero, چرا مهم است؟, قبل از رفتن, سابقه, disclaimer |
| 4 | «انجام دادم» → MarkDone | ✔ Jalali date chip `۶ مهر ۱۴۰۵`, result radios, next-due banner (`۷ مهر ۱۴۰۸ · هر ۳ سال · ۳۰ روز قبل…`) |
| 5 | `DOM.setFileInputFiles` (`lab-report.jpg`) + «نیاز به پیگیری» + note → «ثبت» | ✔ thumbnail and file name. `POST checkups/{id}/records` 201, then the detail shows «به‌روز», the next due date and «گزارش پیوست شده». IndexedDB `ritme-local-files` exists. The light toast «ثبت شد» was seen. In dark it had already gone by the 3.5 s check |
| 6 | `/checkups/history` | ✔ month timeline, result chips, «پیوست» chip; «با پیوست» filters (1 of 2 in light, 2 of 3 in dark); header digits are Persian («۲ مورد ثبت‌شده») |
| 7 | «خلاصه برای پزشک» → PDF | ✔ `ritme-checkups.pdf` (54 KB light / 69 KB dark) with no UI error or IntlError. The page-1 footer reads **«ریتمی · صفحه ۱ از ۱»** with Persian digits (local bug 1 is fixed) |
| 8 | `/checkups/self-exam` | ✔ ring `۱۶ از ۲۸`, «روز ۱۶ سیکل · ۱۹ روز دیگر» with Persian digits (local bug 3 is fixed), 3 steps, findings |
| 9 | `/checkups/custom/new` | ✔ name, interval chips, performed-by, last-done, note, «ذخیره» |
| 10 | admin-web `/panel/login` (stage admin) → چکاپ‌های دوره‌ای → edit `blood_test` fa title to «آزمایش خون کامل (ویرایش ادمین)» → «ذخیره تغییرات» | ✔ `PUT /api/admin/v1/checkup-types/4` 200 and the list shows the new title |
| 11 | App `/fa/checkups` and `/fa/checkups/4` | ✔ both show the new title straight away |
| 12 | Restore | ✔ the title was set back through admin (PUT 200). The row was then written back byte-for-byte from a pre-edit snapshot, because admin re-encodes the JSON columns as `\u` escapes and bumps `updated_at`. `diff` against the snapshot shows no changes |

**API statuses (all `/api/v1` and `/api/admin/v1` calls carried `X-Backend: go`):** send-otp 200, verify-otp 200, profile
200, cycle/period 200 ×4, checkups 200, checkups/home 200, checkups/{id} 200, preview-next 200, records POST 201 ×3,
checkups/records 200, admin auth/login 200, auth/me 200 (401 before login, as expected), checkup-types 200, options 200, stats 200,
checkup-types/4 GET 200 and PUT 200 ×2. There were no other 4xx or 5xx responses.

**verify:** the task's `verify:` passed (exit 0) with `STAGE_AUTH`. The request reaches Go (`X-Backend: go`), and Go answers 401 because a
Basic header is not a Bearer token.

**Screenshots:** `screenshots/stage-*.png` use the same names as the local run (`home-card`, `list`, `list-bottom`,
`list-tab-action`, `detail`, `detail-bottom`, `markdone-sheet`, `markdone-filled`, `markdone-filled-bottom`, `detail-after-save`,
`history`, `history-with-attachment`, `history-pdf`, `self-exam`, `self-exam-bottom`, `custom-form`, each `-light` and `-dark`),
plus `stage-pdf-summary-page1-light.png` and the admin set (`stage-admin-login`, `-checkup-types-list`, `-checkup-type-form`,
`-checkup-type-form-edited`, `-checkup-type-saved`, `-checkup-types-list-after`, `stage-app-list-after-admin-edit-light`,
`stage-app-detail-after-admin-edit-light`). All are 585 px wide and compressed with pngquant.

**Bugs / open on stage:**
- **S1: the middle-dot separator looks like a Persian zero.** In the checkups card counts line, «۲ مورد به‌روز · ۳ موعدش رسیده · ۱
  عقب‌افتاده» renders as «… ۳۰ موعدش رسیده ۱۰ عقب‌افتاده», because U+00B7 next to a Persian digit is indistinguishable from `۰`
  (`stage-home-card-*.png`). It comes from `frontend/messages/fa/checkups.json:10` (`separator`), which is used by
  `frontend/src/widgets/checkups-card/ui/CheckupsCard.tsx:133-136`. The same `·` is in `:63` (`lastNext`), `:106` (`nextDueMeta`, e.g.
  «هر سال · ۱۴ روز قبل…»), `:143` (PDF subtitle), `:149` (PDF footer) and `:157` (`dayWhen`). Suggestion: use «،» or «–» in fa,
  or add more spacing.
  **Fixed locally in T-M7-17 (not yet on stage):** the fa separator is now «، » (`separator`, `lastNext`, `nextDueMeta`,
  the PDF subtitle and `dayWhen`), e.g. «۱ مورد به‌روز، ۵ موعدش رسیده». The PDF footer «ریتمی · صفحه…» keeps `·` because
  no digit sits next to it.
- The minor items 5(a)–(d) from the local run are unchanged: the «پیوست» chip sits outside the history card, the detail hero always shows a
  `shield` icon, admin-web mixes Latin and Persian digits on the fa UI («هر 12 ماه», «ثبت‌های 30 روز اخیر» next to stats «۵»), and
  the self-exam shows «موعدش رسیده» with «بعدی: ۱۹ روز دیگر».
- Still open (human): catalog content sign-off; production only when asked.

**Cleanup (stage DB only):** user `09900000961` (id 7) was deleted with its 3 checkup records, 4 cycle histories, profile,
access token and OTP rows. That leaves 0 users for 961/962, 0 custom types and 0 records. Checkup type 4 is byte-identical to its
pre-edit state. No prod container, DB or config was read or changed.
