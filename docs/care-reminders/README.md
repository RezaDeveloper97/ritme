# Care reminders (M3) — medications, doctor appointments, today's reminders

Source design: the Design canvas (claude.ai artifact `LQgpWxuwxq5oEuwmhCYsvf`), v13 series. Local copies of
every artboard, light (`nbl_`) and dark (`nbd_`), are in [`docs/design/reminders-v13/`](../design/reminders-v13/).
Open them in a browser or read the markup — the inline styles are the spec (sizes, radii, weights, copy).

| Artboard | What | Frontend route / slot |
|---|---|---|
| `v13_Preg_Home` → «یادآورهای امروز» card | today's doses + next appointment + "افزودن یادآور" | widget `today-reminders`, on **both** homes (cycle home, where `DayTasks` is hidden, and pregnancy home) |
| `v13_AddChooser` | bottom sheet: دارو یا مکمل / نوبت پزشک / مشاوره پزشکی | sheet id `reminders-add` in `app/sheets/registry.tsx` |
| `v13_Reminders` | tabs همه/داروها/نوبت‌ها, today's dose strip, medication list with switches, appointment list, CTA | `/[locale]/reminders` |
| `v13_AddMedication` | medication form | `/[locale]/reminders/medication/new`, edit: `/reminders/medication/[id]` |
| `v13_AddAppointment` | appointment form (kind, who, specialty, topic chips, note, date, time, place, remind-before, calendar, prep) | `/[locale]/reminders/appointment/new`, edit: `/reminders/appointment/[id]/edit` |
| `v13_AppointmentDetail` | hero, detail rows, reminder switch, prep checklist, add-to-calendar, cancel | `/[locale]/reminders/appointment/[id]` |

Flow: home card «افزودن یادآور» and the Reminders CTA open the **AddChooser sheet**; «دارو یا مکمل» → AddMedication;
«نوبت پزشک» → AddAppointment with kind `in_person`; «مشاوره پزشکی» → AddAppointment with kind `phone`
(`?kind=`). Home «همه» → Reminders. An appointment row → AppointmentDetail. Save → back to where the flow began.

## Backend decision

**Go only** (`backend-go/`), under a new route prefix **`/api/v1/care/`** so the strangler can send the whole
prefix to Go with one nginx `location` while Laravel still serves everything else. No Laravel controllers.

Schema rule (docs/go-migration/migrations.md): Laravel owns migrations until T-M2-27, so the one new table ships
as a goose migration **and** a schema-only Laravel twin migration in the same commit; `make schema-diff` stays green.
Everything else lives in the existing `reminders` table (`type` + `meta` JSON), so legacy `/reminders` clients
(DayTasks on the log page, the profile reminders sheet, home sections `doctor_reminder`/`medication_reminder`)
keep seeing these rows.

## Data model

`reminders` row, `type = medication`:

| column | value |
|---|---|
| `title` | medication name ("فولیک اسید") |
| `subtitle` | derived "dose unit" ("۴۰۰ میکروگرم") — for legacy readers |
| `recurrence` | `daily` (all 7 weekdays) or `weekly` (subset) |
| `recurrence_time` | first slot, for legacy readers |
| `starts_on` / `ends_on` | start date / end date (null = open-ended) |
| `is_active` | the list switch |
| `notes` | يادداشت |
| `meta` | `{"v":1,"dose":"400","unit":"mcg","form":"tablet","times":["08:00","20:00"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"pregnancy_end","notify":true}` |

`form` ∈ `tablet|capsule|syrup|injection|drops`; `unit` free text or one of `mg|mcg|ml|iu|drop`;
`weekdays` Saturday-based `0..6` (Saturday = 0, project week start); `times` 1–4 sorted `HH:MM`;
`duration` ∈ `ongoing|until_date|pregnancy_end` (`until_date` ⇒ `ends_on` required; `pregnancy_end` ⇒ server
resolves `ends_on` from the active pregnancy's due date, else behaves as `ongoing`).

`reminders` row, `type = appointment`:

| column | value |
|---|---|
| `title` | short description ("سونوگرافی NT هفته ۱۱"), falls back to the topic label |
| `subtitle` | "who · specialty" ("دکتر احمدی · زنان و زایمان") |
| `scheduled_at` | appointment datetime (Tehran wall-clock) |
| `recurrence` | `none` |
| `is_active` | reminder switch (notification on/off) |
| `notes` | prep free text from the form |
| `meta` | `{"v":1,"kind":"in_person","with":"دکتر احمدی","specialty":"زنان و زایمان","topic":"ultrasound","location":"…","remind_before":"1d","add_to_calendar":true,"prep":[{"id":"p1","text":"دفترچه بیمه","done":false}],"status":"scheduled"}` |

`kind` ∈ `in_person|phone|online`; `topic` ∈ `ultrasound|checkup|lab|consult|vaccine|other`;
`remind_before` ∈ `1h|3h|1d|2d`; `status` ∈ `scheduled|cancelled`. Cancel sets `status=cancelled` and
`is_active=false` (row is kept for history).

New table **`reminder_intakes`** (one row = one dose taken):

```
id bigint PK, user_id FK users cascade, reminder_id FK reminders cascade,
intake_date date, slot char(5) 'HH:MM', taken_at datetime, created_at, updated_at,
UNIQUE (reminder_id, intake_date, slot), INDEX (user_id, intake_date)
```

## API contract (`/api/v1/care`, auth:api, Accept-Language fa/en, standard `{success,data,message}` envelope)

| Method | Path | Notes |
|---|---|---|
| GET | `/care/medications` | the user's medication reminders (`?active=1`) |
| POST | `/care/medications` | 201; validates the meta shape above (422 fa/en) |
| GET/PUT/DELETE | `/care/medications/{id}` | PUT = partial (`is_active` for the switch) |
| POST | `/care/medications/{id}/intakes` | body `{date:"Y-m-d", slot:"HH:MM"}` → mark taken (idempotent) |
| DELETE | `/care/medications/{id}/intakes` | same body → untick |
| GET | `/care/appointments` | `?scope=upcoming|past|all` (default upcoming, cancelled excluded from upcoming) |
| POST | `/care/appointments` | 201 |
| GET/PUT/DELETE | `/care/appointments/{id}` | |
| POST | `/care/appointments/{id}/cancel` | |
| PATCH | `/care/appointments/{id}/prep/{itemId}` | `{done:bool}` |
| GET | `/care/today` | `?date=Y-m-d` (default today, Tehran) — see below |
| GET | `/care/enums` | localized labels for forms, units, kinds, topics, remind_before |

### Medication resource (T-M3-01, implemented)

Request bodies are flat (no `meta` wrapper): `title`, `dose`, `unit`, `form`, `times`, `weekdays`, `amount`, `duration`,
`starts_on`, `ends_on`, `notify`, `notes`, `is_active`. POST requires `title`, `form`, `times`; defaults: `weekdays` all
seven, `amount` 1, `duration` ongoing, `notify` true, `is_active` true, `starts_on` today (Tehran). `dose`/`unit` are
optional (a numeric dose is stored as its string). `ends_on` is read only for `until_date` (required, ≥ `starts_on`).
PUT merges the sent keys over the stored medication and validates the result as a whole. `times`/`weekdays` are
de-duplicated and sorted. 422 is the controller shape `{success:false, message, errors}` in the request language;
404 `{success:false, message}` for another user's id, a non-medication reminder or a malformed id.

`data` of `GET /care/medications[/{id}]`, POST (201) and PUT:

```json
{"id": 12, "type": "medication", "title": "فولیک اسید", "subtitle": "۴۰۰ میکروگرم", "notes": null,
 "form": "tablet", "dose": "400", "unit": "mcg", "times": ["08:00", "20:00"], "weekdays": [0,1,2,3,4,5,6],
 "amount": 1, "duration": "pregnancy_end", "notify": true, "starts_on": "2026-09-23", "ends_on": "2027-03-08",
 "is_active": true, "recurrence": "daily", "recurrence_time": "08:00",
 "created_at": "2026-09-23T06:30:00.000000Z", "updated_at": "2026-09-23T06:30:00.000000Z"}
```

Intakes: `POST /care/medications/{id}/intakes` `{date, slot}` → 200 `data: {reminder_id, date, slot, taken: true,
taken_at}`; `slot` must be one of `times`, `date` inside the weekday/start/end window and not after today (422 on
`slot`/`date` otherwise). `DELETE …/intakes?date=Y-m-d&slot=HH:MM` (query string or JSON body) → `taken: false,
taken_at: null`; both are idempotent. Legacy rows created via `POST /reminders` (no meta) are listed with a derived
meta (one slot at `recurrence_time`, all weekdays). `GET /care/enums` → `{forms, units, durations, kinds, topics,
remind_before}`, each `[{value, label}]`.

`GET /care/today` → `data`:

```json
{
  "date": "2026-09-23",
  "doses": [{"reminder_id": 12, "title": "فولیک اسید ۴۰۰ میکروگرم", "form": "tablet", "slot": "08:00", "taken": true}],
  "taken_count": 1, "total": 2,
  "next_appointment": {"id": 40, "kind": "in_person", "title": "سونوگرافی NT", "with": "دکتر احمدی",
    "scheduled_at": "2026-09-30 10:30:00", "days_until": 8, "location": "مطب", "remind_before": "1d"}
}
```

Doses = active medications whose weekday/start/end window covers `date`, one per slot, sorted by slot.
`next_appointment` = the earliest `scheduled` appointment at or after now (null if none).
Serialisation follows the Go conventions of the reminders package (civildate, Tehran wall-clock, ids as numbers).

## Colors — light → dark (from the `nbd_` artboards)

Frontend rule (frontend/CLAUDE.md §10): **no hex in components**; map each design color to an existing token in
`globals.css` (both `:root` and `[data-theme="dark"]`), add a token pair only when none fits, and keep
`npm run lint:dark` + `lint:styles` green. Reference values the design uses:

| role | light | dark |
|---|---|---|
| page | `#F7F3FF` | `#17112B` |
| card surface | `#FFFFFF` | `#221A3D` |
| field / chip / soft surface | `#F1EDFB` | `#2A2150` (or `rgba(255,255,255,.06–.07)`) |
| line | `#E7E1F4` | `#34295A` |
| text | `#2F2F35` | `#F6F1FF` |
| muted text | `#6F6F78` | `#8C82AD` |
| secondary text | `#5A5A64` | `#B8AED6` |
| brand (button fill, switch on) | `#7B61FF` | `#B9A6FF` (text on it `#17112B`) |
| brand link/icon | `#5B41E6` | `#7C5CFF` / `#B9A6FF` |
| brand soft (icon tile, selected chip) | `#ECE6FF` | `#2E2458` |
| success (taken check) | `#0F7B6C` on `#E7F8EF` | `#7FE0A8` on `#1F3A2E` |
| data/teal (capsule, online) | `#0C87A7` on `#E3F9FE` | `#6FE7D0` on `#1E3340` |
| amber (appointment date tile) | `#B45309` on `#FFF3DF` | `#FFC98A` on `#3A2F1E` |
| rose (iron, cancel) | `#C42D57` on `#FDE4EB`, cancel border `#F9C3D0` | `#FF8FA3` on `#3A2140`, border `#6B3A4D` |
| sheet scrim | `rgba(47,47,53,.42)` | `rgba(0,0,0,.6)` |
| card shadow | `0 18px 34px -20px rgba(93,71,214,.35)` | `0 18px 34px -20px rgba(0,0,0,.7)` |

Numbers use Lalezar (`font-family: Lalezar`) for big digits (date tiles, time buttons, amount); text is Vazirmatn.
All digits localized (fa → Persian digits); dates shown in the locale calendar (Jalali for fa).
Touch targets ≥ 44 px; switches are real `role="switch"` buttons; the sheet is the app's existing sheet primitive.

## Rollout (T-M3-09) — staging 2026-09-26

**Routing:** staging is Go-only since T-M2-28 — `location /api/` in `deploy/vhost-stage.inc` already proxies every
`/api/v1/*` path (incl. `/api/v1/care/*`) to `stage-backend-go`, so no `deploy/go-routes.inc` change was needed on
stage. Production still needs a `care` route line when the user asks (Go-only feature; prod is Laravel until T-M2-26).

**Verify-all (local, pre-deploy):** backend-go `go vet` / `go test` / `golangci-lint` (0 issues) ✔; frontend
typecheck, lint, fsd:lint, lint:styles, lint:dark, test (551/551), build ✔; admin-web typecheck, lint, fsd:lint,
test (77/77), build ✔; Laravel skipped (backend/ unchanged).

**Deploy:** `./deploy-stage.sh` (branch `stage` @ 484a4ad). The first full run lost its SSH session mid-build (server
load ~9 with three parallel Next/Go builds); re-run as `SKIP_SYNC=1 SERVICES=<svc> ./deploy-stage.sh` for
backend-go, admin-web, frontend — all verification assertions ok.

**Migrations:** backend-go log `{"msg":"migrations","action":"goose_managed","applied":[2,3,4,5],"version":5}`.
Tables present: `reminders`, `reminder_intakes`, `checkup_types`, `checkup_records`, `user_checkup_settings`,
`fertility_logs`, `pregnancy_*` v2 tables (alerts, care_items, daily_extras, fetal_movements, symptom_logs, …).

**API e2e** (smoke user 09900000901, token via OTP read from `otp_verifications`, run on the server inside the
`ritme-edge` network against `stage-backend-go`):
```
enums 200 | POST medications 201 | care/today 200 (contains med) | home 200 (contains med)
POST medications/{id}/intakes 200 → today taken:true, taken_count 1
POST appointments 201 | GET appointments/{id} 200 | POST appointments/{id}/cancel 200 status "cancelled"
cleanup: DELETE medication 200, DELETE appointment 200
```
Through the shared proxy: `GET https://stage.ritmeapp.ir/api/v1/care/enums` (no bearer) → 401 `x-backend: go`.

**Open (human):** light/dark screenshots of /reminders, medication form, appointment detail and the home card were
not taken — the browser recipe needs the staging Basic-auth password, which the agent was not permitted to read.
Click through the same checklist on a phone in both themes.

## Interval schedule and dose-time notifications (2026-11-10)

The medication form follows a reference flow: start date, frequency («هر N روز» / «هر N ساعت»), one or more doses
(amount + unit, time, before / after a meal). The meta stays `v: 1` and only gains two keys, so every reader of
`times` / `weekdays` / `amount` keeps working:

```json
{"v":1,"form":"tablet","times":["08:00","20:00"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"ongoing","notify":true,
 "frequency":{"every":1,"unit":"day"},
 "doses":[{"time":"08:00","amount":1,"unit":"pill","meal":"after"},{"time":"20:00","amount":2,"unit":"pill","meal":null}]}
```

- `frequency.unit = day`: every 1–30 days counted from `starts_on` (`Covers`). `hour`: every 1, 2, 3, 4, 6, 8 or 12 h
  (divisors of 24, so the slots are the same every day), around the clock from each dose's `time`; on the start date
  the chain begins at the first dose time (`SlotsOn`).
- `times` is derived (`TimesOf`), `weekdays` is all seven, `form` comes from the first dose's unit when not sent.
- Request: `frequency` + `doses` (1–6, distinct times, amount 1–99, unit `pill|capsule|drop|ml|spoon|puff|injection`,
  meal `before|after|null`). A PUT with `times` and no `doses` (an older client) turns the medication back into a
  weekday schedule. Resource and member projection return `frequency` (null for a weekday schedule) and `doses`.
- `/care/today` doses carry `amount`, `unit`, `unit_label`, `meal`.

Notifications at the dose time (`internal/care/push.go`), both channels gated by `notifications.Decide` / `Render`
(category «داروها», quiet hours → the dose is dropped, not announced late; «متن خنثی» → neutral text), ticked doses
skipped:

| Channel | How |
|---|---|
| Web Push (browser, installed PWA) | `GET /push/config` (VAPID public key), `POST/DELETE /push/subscriptions`; `Pusher.Loop` every 30 s pushes doses due in the last 10 min, each claimed once in `reminder_pushes` (goose 00121). Needs `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` (`api vapid-keys`). Endpoints are limited to known push services (no SSRF). |
| Android shell (WebView has no Push API) | The page hands `GET /care/upcoming?hours=168` to `window.RitmeNative.scheduleReminders(json)`; the shell arms local alarms (`android-shell/…/Reminders.kt`), re-armed on boot. The device schedule is as fresh as the last sync. |

## Reminders by SMS for Ritme Plus (2026-11-11)

Owner's decision: Plus subscribers (and running trials) also get their reminders by SMS, automatically. The reminder
itself stays free (in-app, Web Push, shell alarm) — SPEC §4.3 still holds; only the SMS copy is the Plus extra
(entitlement `plus.sms_reminders`). `internal/care` still never imports `internal/plus`: the sender is
`internal/remindersms`, wired in `routes_care.go` with `plus.Service.PlusUsers` as its audience.

- What is texted: every due dose of an active medication with `notify` on (not yet ticked), and every active,
  scheduled appointment at `scheduled_at − remind_before`.
- Gate, per SMS: the user's `sms_reminders` switch (notification settings, default on) → the category switch
  («داروها» / «نوبت پزشک») and quiet hours (`notifications.Decide`; a reminder in quiet hours is dropped) → the
  per-day cap (`REMINDER_SMS_DAILY_CAP`, default 8, counted from the user's local midnight) plus a rolling 24 h ceiling
  of twice the cap (the stored zone comes from `X-Timezone`, so the local day alone could be reopened by zone hops) → «متن خنثی»: the neutral gateway template, and no SMS at all
  when `KAVENEGAR_TEMPLATE_REMINDER_NEUTRAL` is unset.
- Dedupe: `reminder_sms` (goose 00122 + Laravel twin), one row per reminder occurrence, claimed with INSERT IGNORE.
- Off by default: `REMINDER_SMS_PROVIDER=none`. `gateway` uses Kavenegar verify/lookup with the OTP credentials and
  three templates (see `.env.example`); `fake` only logs.
- Web: the «یادآور پیامکی» card on the notification-settings screen (switch for Plus, a Plus link for free users)
  and a row in the paywall's feature table.

## Web Push for the whole inbox + appointment alarms in the shell (2026-11-11)

One VAPID backend, one `push_subscriptions` row per browser — no per-vendor code. What goes out on it now:

- **Medication doses** — unchanged (`Pusher.Tick`).
- **Every in-app notification** (`user_notifications`, the bell: companion / family notices, an opened course, the
  message engine's push slot, postpartum visits) — `Pusher.TickNotices` (`internal/care/push_notices.go`). The
  writers know nothing about push: the sender polls rows with `pushed_at IS NULL` (goose 00123 + Laravel twin; rows
  that existed at migration time are marked handled) and claims each row before sending. A row is pushed once when
  it is unread, at most 12 h old, the user has a subscription and `notifications.DecideNotice` allows it: types
  `companion` / `family` follow the «همراه» switch, `learning` the «آموزش» switch, other types have no switch. Inside
  quiet hours the row waits for the end of the window (it is not dropped). «متن خنثی» (on by default) replaces the
  text with «اعلان تازه» (`RenderNotice`); a tap opens `/<lang><action_url>` (or `/<lang>/home`).
- **Android shell** — event-driven inbox rows cannot be armed ahead as local alarms, so the shell still gets only
  what is predictable: `GET /care/upcoming` now also lists appointment reminders (`kind: appointment`, the visit −
  `remind_before`, gated by «نوبت پزشک»). The bridge payload is unchanged, so the current APK arms them as is; the
  channel label «یادآورها» ships with the next APK.

Install paths: Android = the shell (local alarms) or any browser with the Push API; iPhone = the PWA added to the
home screen (the device row on the notification-settings screen says «نیاز به نصب» in a plain Safari tab).
Appointments are not sent by Web Push yet.

### Retry rules (T-M12-22, T-M12-22b)

Both senders (`Tick` for doses, `TickNotices` for inbox rows) claim an item in the database before sending it, so
the API and the worker never announce it twice. What happens after the send (`deliver`, `internal/care/push.go`;
rules in `internal/care/push_retry.go`):

| Outcome | What the sender does |
|---|---|
| At least one browser accepted it | Delivered. The other browsers are not tried again (the one that has it must not get it twice). |
| No browser accepted it, at least one failed for a passing reason (timeout / network error, 429, 5xx) | The claim is **released in the database**: `reminder_pushes.sent_at` / `user_notifications.pushed_at` becomes a marker, `2000-01-01 00:00:00` (Tehran) + the number of failed tries in seconds. Any process — the same one, the other one, or one started after a restart or deploy — takes it again on a later tick with a compare-and-set on that exact marker, so only one of them retries it. |
| 404 / 410 | The subscription is deleted. Not retried. |
| Any other 4xx, or no subscription at all | Logged (no text). Not retried. |

- At most `PushAttempts` = **3 tries in all**, across processes; after the third passing failure the item stays
  claimed and silent (no retry storm).
- Retries happen only inside the lookback window: a dose up to `PushLookback` (10 min) after its time, an inbox row
  up to `NoticeLookback` (12 h) after it was written. A released dose past its window is simply never read again; a
  released inbox row past its window (or already read) is claimed and left silent. A released inbox row inside quiet
  hours waits for the end of the window without spending a try, like a new one.
- No schema change: the marker lives in the existing claim columns. The hourly prune of `reminder_pushes` keeps a
  released row until its dose day is two days past, so a prune never resets the try count of a live dose.
