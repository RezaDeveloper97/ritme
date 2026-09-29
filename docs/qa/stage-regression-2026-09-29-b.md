# Stage regression 2026-09-29 — part B (care reminders, checkups, security fixes)

Target: `https://stage.ritmeapp.ir`, compose project `ritme-stage`, branch `stage` @ e3aea8d (goose v8). Production was not
touched. Tester: Claude (QA agent B). Method: headless Chrome 154 on CDP port 9302 with its own profile, viewport 390×844,
DPR 1.5 (585 px PNGs, full-page = viewport grown to the `.scroll` container height, pngquant), locale `fa`, light and
dark theme (`ritme_theme`). Gate: `ritme_stage` cookie minted by one Basic-auth page load (creds from
`/root/ritme-stage-credentials.txt` into a chmod-600 scratch file, never printed), then the header was dropped.
Login: `send-otp` → code read from `otp_verifications` in the `ritme_stage` DB → `verify-otp`.

Test users (all deleted afterwards): `09900001201` cycle (Pap overdue since 1404, blood test with follow-up + note,
folic acid, NT visit), `09900001202` pregnant week 11 (LMP 2026-07-14; folic acid, vitamin D, iron every other day
paused, online nutrition consult), `09900001203` attachments + logout, `09900001204` caps + throttle.

Design reference: `docs/care-reminders/screenshots/audit/*-design.png` and `docs/checkups/screenshots/audit/*-design.png`,
checked against the *Resolution* columns of both design audits and `docs/security/audit-m3-m7.md`.

## Summary

**All checklist items pass (39 ✔, 0 ✘). 4 new low-severity bugs + 3 informational notes, no high or medium.**

- 602 `/api` responses observed in the browser: **all `X-Backend: go`**. The only non-2xx responses were the expected
  ones: 1 × 422 `limit_reached` (the cap test) and 401 on `/api/admin/v1/auth/me` before the admin login.
- 96 document loads: all 200 and all carry an **enforced** `Content-Security-Policy` (no `…-Report-Only`).
  0 CSP violations, 0 console errors/warnings, 0 uncaught exceptions across every flow. (The capture channel was
  proven with a deliberate probe — a blocked `fetch` to example.com produced the expected `security` log entry; probe
  excluded from the counts.)
- 45 page checks: 0 horizontal overflow, 0 raw i18n keys, 0 Latin digits in fa text (the only hit was the test user's
  own name «QA B2»).

## Checklist

### Care reminders (fa, light + dark)

| # | Item | Result | Evidence |
|---|---|---|---|
| C1 | Home «یادآورهای امروز» card — pregnancy home (doses, ✓ circle, «افزودن یادآور») | ✔ | `care-home-card-pregnancy-{light,dark}` |
| C2 | Home card — cycle home, with the next appointment row («سونوگرافی NT، دکتر احمدی», «۲ روز دیگر، ساعت ۱۰:۳۰، …» clamped to one line, «۱ روز قبل یادآوری» pill) | ✔ | `care-home-card-cycle-{light,dark}` |
| C3 | AddChooser sheet: 3 cards + disclaimer, opened from the home card (`?sheet=reminders-add`) | ✔ | `care-add-chooser-{light,dark}` |
| C4 | Chooser «مشاوره پزشکی» → `/reminders/appointment/new?kind=phone`, place relabelled «شماره تماس» | ✔ | `care-appointment-new-phone-light` |
| C5 | /reminders tabs همه / داروها / نوبت‌ها (URL `?tab=`), sort active-first then by slot (L-1), «یک روز در میان» (L-3), meta «۱۸:۰۰، آنلاین، خانم مهدوی» | ✔ | `care-reminders-{all,meds,appts}-{light,dark}` |
| C6 | Dose strip: 174 px cards, one-line names, scrolls; tapping a dose → `POST …/intakes` 200, progress «۱ از ۴ مصرف شد» | ✔ | `care-reminders-dose-taken-light` |
| C7 | Add medication: no card headers, «شکل دارو» label, count chips, «ساعت نوبت‌ها» slot rows «۸:۰۰»/«۲۰:۰۰», stepper, «هر روز» caption, start/duration rows, solid CTA (H-3, H-4, M-1…M-4, L-9, L-10) | ✔ | `care-medication-new{,-filled}-{light,dark}` |
| C8 | Medication dose typed as Persian «۱۰۰۰» is shown in Persian and stored as ASCII `"1000"` (M-12); save → `POST /care/medications` 201 → back to /reminders | ✔ | API read-back |
| C9 | Edit medication: dose shows «۴۰۰», save → `PUT /care/medications/14` 200 | ✔ | `care-medication-edit-{light,dark}` |
| C10 | Add appointment: separate kind cards (H-1), 3 card groups (M-8), filled fields with icons, date/time two-field grid (M-7), calendar + wheel pickers, calendar switch default on (L-9); save → 201 → detail | ✔ | `care-appointment-new{,-filled}-{light,dark}`, `care-appointment-timepicker-*` |
| C11 | Edit appointment: values prefilled, change remind-before → `PUT /care/appointments/19` 200 | ✔ | `care-appointment-edit-{light,dark}` |
| C12 | Appointment detail: tinted hero, kind + week pills, 72 px date tile, «چهارشنبه · ساعت» eyebrow, big «۱۰:۰۰», «در خصوص», prep checklist in one card with square boxes, one-row actions (H-2, M-9…M-11) | ✔ | `care-appointment-detail{,-online}-{light,dark}` |
| C13 | Cancel flow: rose «لغو نوبت», confirm sheet «این نوبت لغو شود؟» with rose «بله، لغو شود»; confirm → `POST …/17/cancel` 200 → `/reminders?tab=appointments`, cancelled visit gone from the list and home | ✔ | `care-cancel-confirm-{light,dark}`, `care-appointment-cancelled-light` |
| C14 | Legacy reminders sheet (Profile → `?sheet=reminders`) lists medications and appointments, toggles and delete icons | ✔ (see B-1…B-3) | `care-legacy-reminders-sheet-{light,dark}` |

### Checkups (fa, light + dark)

| # | Item | Result | Evidence |
|---|---|---|---|
| K1 | Home card «چکاپ‌های دوره‌ای»: brand ring «۲/۶», self-exam first with «راهنما», Pap overdue on amber with «عقب‌افتاده از فروردین ۱۴۰۴» + «ثبت نوبت» (A1…A5) | ✔ | `care-home-card-cycle-{light,dark}` (lower half) |
| K2 | List: sections «این ماه» → **«عقب‌افتاده» (Pap) second, as in the artboard** → سالانه → هر ۶ ماه → بر اساس سن; one card per section, outlined status pills with icons, month-year dates, wrapped lines, filter icon (B1…B12) | ✔ | `ck-list-{light,dark}` |
| K3 | Detail (Pap, overdue): lavender hero, subtitle chip + icon, «فروردین ۱۴۰۴», «حدود ۱۷ ماه از موعد گذشته، آخرین بار فروردین ۱۴۰۱», history row with dot + outlined «نرمال» pill, plain disclaimer (C1…C8) | ✔ | `ck-detail-{light,dark}` |
| K4 | MarkDone is a content-height sheet (700 of 844 px, `half`), result cards with icons and green «نرمال», dashed attachment buttons, filled note, green next-due banner «موعد بعدی: ۸ مهر ۱۴۰۸» (D1…D6) | ✔ | `ck-markdone-{light,dark}` |
| K5 | History: one timeline card, note shown («نیاز به پیگیری، ویتامین D پایین»), «پیوست» chip, «خلاصه برای پزشک» at the bottom (F1…F4) | ✔ | `ck-history-{light,dark}`, `ck-history-with-attachments-light` |
| K6 | PDF export: `ritme-checkups.pdf` (1 page), Persian content and **Persian footer «ریتمی · صفحه ۱ از ۱»** | ✔ | `ck-history-pdf-page1-light` |
| K7 | Self-exam: «۳ مرحله، حدود ۵ دقیقه» (E1), tinted hero with best-time headline + day ring, subtitle «هر ماه، روز ۷ تا ۱۰ سیکل», bell toggle, «چیزی متفاوت نبود» preselected (E2…E5) | ✔ | `ck-self-exam-{light,dark}` |
| K8 | Custom checkup form: fill «معاینه چشم», ۲ سال, note → `POST /checkups/custom` 201, listed | ✔ | `ck-custom-new-{light,dark}`, `ck-custom-filled-light`, `ck-list-with-custom-light` |
| K9 | Admin `/panel` checkup-type edit (dentist subtitle + « (QA-B)») → `PUT /api/admin/v1/checkup-types/5` 200 → visible in the app list and detail, light and dark | ✔ | `admin-checkup-type-edit-{form,saved}`, `ck-{list,detail}-after-admin-edit-{light,dark}` |
| K10 | Restored: reverted through the panel, then the row was put back **byte-for-byte** from a pre-edit snapshot (see N-3); `diff` of `SELECT * … WHERE id=5` before/after = identical; app shows «معاینه و جرم‌گیری» | ✔ | `admin-checkup-type-restore-*` |

### Security fixes (audit M3–M7, T-M7-19/21)

| # | Item | Result | Evidence |
|---|---|---|---|
| S1 | Attach a PNG (Pap) and a PDF (blood test) in MarkDone → both records 201; IndexedDB `ritme-local-files`/`files` = 2 rows (`image/png`, `application/pdf`) | ✔ | `sec-png-attached-light` |
| S1b | Log out (Profile → «خروج از حساب») → `POST /auth/logout` 200 → /fa/signup; **IndexedDB `ritme-local-files` = 0 rows**; localStorage went from `ritme_theme, ritme_token, ritme-install-dismissed, ritme_home_ttc` to `ritme_theme, ritme-install-dismissed` (device prefs only); sessionStorage empty; `ritme_auth` cookie gone | ✔ | console log |
| S2 | `.svg` rejected: photo input `accept="image/jpeg,image/png,image/webp,image/heic,image/heif"`; an SVG forced into the photo input **and** into the PDF input both show «فقط عکس یا فایل PDF.» and nothing is picked | ✔ | `sec-svg-rejected-light` |
| S3a | Checkups card «ثبت نوبت» → `/fa/reminders/appointment/new?kind=in_person&prefill=<random id>`; title «پاپ‌اسمیر / HPV» prefilled from `sessionStorage.ritme_handoff`; no title/topic/date in the URL or in any RSC request URL | ✔ | `sec-prefill-checkups-light` |
| S3b | Pregnancy care plan «رزرو» (NT) → `…?kind=in_person&return_to=/pregnancy/calendar&prefill=<id>`; form prefilled with title «سونوگرافی NT و غربالگری اول», topic سونوگرافی, date «۷ مهر» (handoff carried `careItemKey: nt_scan`); survives a reload; no `care_item_key`/topic/date in any URL | ✔ | `sec-pregnancy-calendar-light`, `sec-prefill-careplan{,-nt}-light` |
| S4 | Enforced CSP on every document (`default-src 'self'; script-src 'self' 'unsafe-inline'; … connect-src 'self' https://stage.ritmeapp.ir; object-src 'none'; frame-ancestors 'self'`), 0 violations in all flows | ✔ | network log |
| S5 | Caps: 100 medications seeded via API; the 101st from the UI → `422 {"success":false,…,"errors":{"limit":[…]},"error_code":"limit_reached"}` and the form shows the server's message «به سقف تعداد یادآورهای دارو رسیده‌ای. برای افزودن مورد تازه، یکی از یادآورهایی را که دیگر لازم نداری حذف کن.» | ✔ | `sec-cap-101st-{light,dark}` |
| S6 | Write throttle: 65 back-to-back `POST /care/medications` → 60 × 201 then 429 on the 61st (`Retry-After: 17`, `X-RateLimit-Limit: 60`) | ✔ | seed script output |

### Cross-cutting

| # | Item | Result |
|---|---|---|
| X1 | Every `/api` response `X-Backend: go` (602/602) | ✔ |
| X2 | No unexpected 4xx/5xx | ✔ |
| X3 | No console errors / exceptions / CSP violations | ✔ |
| X4 | No raw i18n keys | ✔ |
| X5 | No Latin digits in fa UI text | ✔ (legacy sheet uses Persian digits but see B-2) |
| X6 | No horizontal overflow at 390 px | ✔ |
| X7 | Light and dark: every screen above captured in both, tokens switch (dark canvas, soft cards, turquoise/rose roles) | ✔ |

## Bugs

| ID | Sev | Where | Bug | Repro |
|---|---|---|---|---|
| B-1 | low | `backend-go/internal/reminder/handlers.go:343-346` (`is_active` applied to any row); `frontend/src/screens/profile-reminders/ui/RemindersSheet.tsx:142-158` | A **cancelled** appointment stays in the legacy reminders sheet as an ordinary switched-off row, and switching it on succeeds: `PUT /api/v1/reminders/:id {"is_active":true}` → 200, leaving `status:"cancelled"` with `is_active:true`. The new care screens ignore it (they skip cancelled), so the row is just inconsistent, and whether a reminder notification would fire for it depends on the notifier. | Cancel an appointment in /reminders → Profile → یادآورها → the cancelled visit is listed with a switch → turn it on → `GET /care/appointments/:id` shows `status:"cancelled", is_active:true`. (Reverted on the test data.) |
| B-2 | low | `frontend/src/screens/profile-reminders/ui/RemindersSheet.tsx:137` (join `' · '`), `:125-127` (`recurrenceTime` as stored) | fa meta line of the legacy sheet still joins with « · » right next to digits («دکتر احمدی · زنان و زایمان · ۱۵ مهر ۱۴۰۵», «۴۰۰ میکروگرم · هر روز ساعت ۰۸:۰۰»), which T-M7-17 replaced with «،» because «·» reads as a Persian zero; the time is zero-padded «۰۸:۰۰» while the new screens say «۸:۰۰ صبح». | Profile → یادآورها. See `care-legacy-reminders-sheet-light.png`. |
| B-3 | low | `backend-go/internal/care/medication.go:42-46` (`Recurrence()` → `weekly` for any weekday subset) → legacy label `messages/fa/reminders.json` `recurrence.weekly` | A medication taken 4 days a week («یک روز در میان» on /reminders) appears in the legacy sheet as «هفتگی ساعت ۱۳:۰۰» (weekly). | Create a medication on ش/د/چ/ج → Profile → یادآورها. |
| B-4 | low | `frontend/src/screens/checkup-detail/ui/CheckupDetailPage.tsx:27,119-123` (`BOOK_HREF` without a handoff) | «ثبت نوبت» on the checkup **detail** opens an empty appointment form, while the same action on the home checkups card prefills the checkup title through `?prefill=` (T-M7-19). No privacy problem (nothing in the URL), just an inconsistent prefill. | /fa/checkups/3 → «ثبت نوبت» → «توضیح کوتاه» is empty; compare home card → «ثبت نوبت». |

### Resolutions (T-M2-35, local only — not yet on stage)

- **B-1** Resolution: `PUT /reminders/{id}` and `PUT /care/appointments/{id}` answer 422 on `is_active` when a cancelled appointment's reminder is switched on (deviation **D-29**, integration test `TestAppointment_CancelledReminderStaysOff`); the legacy sheet shows the row as «لغو شده» with the switch off and disabled.
- **B-2** Resolution: the legacy sheet joins with the locale's `reminders.listSeparator` («، » in fa), rebuilds appointments' «پزشک، تخصص» from the care row instead of the stored « · » subtitle, and shows hours without a leading zero («ساعت ۸:۰۰»).
- **B-3** Resolution: medication rows take their schedule from the care row's weekdays («یک روز در میان» / «N روز در هفته» / «هر روز»), same rule as /reminders; the legacy `recurrence` column is unchanged (only daily/weekly exist there).
- **B-4** Resolution: the checkup detail's «ثبت نوبت» navigates with the `?prefill=` handoff (`{title}`), like the home card; local check: the form opens with «پاپ‌اسمیر / HPV».
- **N-1** Resolution: a new medication defaults to «تا پایان بارداری» for a pregnant user (also once the mode loads after mount, unless a duration was picked).
- **N-2** Resolution (partial): the medication form and the fertility Log show a localized «wait a moment» message on 429. The Go 429 body is still the framework's English «Too Many Attempts.»: localizing it needs `internal/http/write_throttle.go` / `internal/platform/ratelimit`, outside T-M2-35's paths; the appointment and checkup forms (also outside) still show their generic error.

## Notes (no action required / design calls)

- **N-1** Medication form default «مدت مصرف» is «بدون تاریخ پایان» even for a pregnant user; the artboard shows
  «تا پایان بارداری». The default isn't listed in the audit, so it may be deliberate.
- **N-2** The 429 body is Laravel-shaped `{"message":"Too Many Attempts."}` (English, no `success`/`error_code`), unlike the
  422 caps. Consistent with the auth throttles; the UI message for a 429 on a care form was not exercised.
- **N-3** Saving a checkup type in admin-web re-encodes every translation JSON column with `\uXXXX` escapes
  (`title`, `subtitle`, `why`, `prep_steps`). Semantically identical (PHP `json_encode` does the same), but a panel
  "undo" is not byte-identical, so K10 restored the row from the snapshot by SQL (including `updated_at`).
- Known/accepted items re-observed and left alone: 24 h «۱۸:۰۰» on appointment rows vs 12 h on medication rows (L-4),
  mammography «هنوز زود است» (B13), plain «×» on sheets (L-8/D7), the pregnancy home's card placement (L-6).

## Cleanup

- Test users 09900001201–1204: 1201/1202/1204 deleted with `DELETE /api/v1/account`; 1203 (logged out, token revoked)
  deleted in the stage DB exactly as `DeleteAccount` does (refresh tokens → access tokens → user, FK cascades). Their
  `otp_verifications` rows removed. Verified 0 users, reminders, checkup records, custom checkup types and tokens left.
- `checkup_types` id 5 restored byte-identical (K10).
- Scratch gate/admin creds, tokens, cookie jar and the Chrome profile deleted; Chrome on :9302 stopped.
- No code changed, nothing committed, no deploy.
- Note: one `rm` of an empty scratch download folder was blocked by a Claude Code safety check (relative glob after
  `cd`); it wasn't needed and was not retried.

## Screenshots (`docs/qa/screenshots/b/`, 585 px, pngquant)

Care: `care-home-card-pregnancy-{light,dark}`, `care-home-card-cycle-{light,dark}`, `care-add-chooser-{light,dark}`,
`care-reminders-{all,meds,appts}-{light,dark}`, `care-reminders-dose-taken-light`, `care-medication-new-{light,dark}`,
`care-medication-new-filled-{light,dark}`, `care-medication-edit-{light,dark}`, `care-appointment-new-{light,dark}`,
`care-appointment-new-filled-{light,dark}`, `care-appointment-timepicker-{light,dark}`, `care-appointment-new-phone-light`,
`care-appointment-edit-{light,dark}`, `care-appointment-detail-{light,dark}`, `care-appointment-detail-online-{light,dark}`,
`care-cancel-confirm-{light,dark}`, `care-appointment-cancelled-light`, `care-legacy-reminders-sheet-{light,dark}`.

Checkups: `ck-list-{light,dark}`, `ck-detail-{light,dark}`, `ck-markdone-{light,dark}`, `ck-self-exam-{light,dark}`,
`ck-history-{light,dark}`, `ck-history-with-attachments-light`, `ck-history-pdf-sheet-light` (history right after the
export tap), `ck-history-pdf-page1-light` (rendered PDF page), `ck-custom-new-{light,dark}`, `ck-custom-filled-light`,
`ck-list-with-custom-light`, `ck-list-after-admin-edit-{light,dark}`, `ck-detail-after-admin-edit-{light,dark}`,
`admin-checkup-type-{edit,restore}-{form,saved}`.

Security: `sec-svg-rejected-light`, `sec-png-attached-light`, `sec-prefill-checkups-light`, `sec-pregnancy-calendar-light`,
`sec-prefill-careplan-light`, `sec-prefill-careplan-nt-light`, `sec-cap-101st-{light,dark}`.
