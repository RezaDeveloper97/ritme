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
