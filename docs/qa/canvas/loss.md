# LOSS — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png`. Screens: 390 px, fa (plus one en), headless Chrome over CDP
(`bloom/bin/shot.mjs --token`) against the worktree Go API :8122 + Next dev :3102. Test user **`09120005402`** (created
for CB-LOSS-02: onboarding name مریم / female / goal pregnancy, pregnancy onboarding LMP 2026-08-01, mode pregnancy; the
loss was recorded through the UI — early miscarriage, ~۹ مهر — then beta next test 2026-10-19 via the API, mood «غمگین»
via the UI, one earlier `pregnancy_losses` row (2025-05-01) inserted in `ritme_dev` so `recurrent_hint` shows; next step
«فقط پیگیری سیکل» saved through the UI → mode cycle). Colours are checked against the token map
(`docs/canvas-build/README.md` §4), not the board hex.

## CB-LOSS-02

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Loss_Start` | `/loss` | [start](loss/CB-LOSS-02/fa_loss.light.png) · [entry row](loss/CB-LOSS-02/fa_profile_mode.entry.light.png) | [start](loss/CB-LOSS-02/fa_loss.dark.png) · [entry row](loss/CB-LOSS-02/fa_profile_mode.entry.dark.png) | ✔ | Same order: back button only (no title), 24/900 «متأسفیم که این را تجربه کردی» + lead, card «چه اتفاقی افتاد؟» with the five `RadioCardGroup` cards (catalog `loss_types` titles; heart/period ×2, warning/bloom, dropLine/brand, info/neutral — no red), settings card «تاریخ تقریبی» (Jalali, opens a calendar sheet, «بدون تاریخ» clears), «محتوا و یادآورهای بارداری متوقف شود» switch, sticky glass «ادامه». No tab bar, banners or shop. Deliberate: the stop-content switch is a fixed ON (disabled, full opacity, sr «این گزینه همیشه روشن است») because CB-LOSS-01 always stops content; nothing is preselected and the date starts empty (every answer optional — «هر چیزی را که نمی‌خواهی جواب نده»; shot taken after tapping the first card); «به همدمم هم خبر بده» (default OFF) renders only when she has an active companion holding the pregnancy grant — the test user has none, so the shot has no companion row. Entry: the calm-exit row in `/profile/mode` (B-N2-03) now opens `/loss`; `/profile/mode/loss` redirects there. |
| `nbl_Loss_Care` | `/loss/care` | [care](loss/CB-LOSS-02/fa_loss_care.light.png) · [visit sheet](loss/CB-LOSS-02/fa_loss_care.visit-sheet.light.png) · [note sheet](loss/CB-LOSS-02/fa_loss_care.note-sheet.light.png) · [en](loss/CB-LOSS-02/en_loss_care.light.png) | [care](loss/CB-LOSS-02/fa_loss_care.dark.png) · [visit sheet](loss/CB-LOSS-02/fa_loss_care.visit-sheet.dark.png) · [note sheet](loss/CB-LOSS-02/fa_loss_care.note-sheet.dark.png) | ✔ | Same hierarchy: back header «مراقبت از خودت»; `UrgentCard` (`--danger-soft`) «این علائم را فوراً پیگیری کن» with the warning signs joined from catalog `loss_warning_signs` and a solid `--danger` «تماس با اورژانس ۱۱۵» `tel:` button; «پیگیری جسمی» list (bleeding «ثبت» link → sheet: open the log sheet / «خونریزی قطع شد»; beta «یادآور» pill → calendar sheet `beta_next_on` / «جواب منفی شد»; visit «نوبت بگیر» brand pill → date + time chips → `visit_at`, private care appointment); «حال دلت» card with the four mood chips (`POST /loss/moods`) + the not-your-fault line; private-note row → sheet (encrypted, fetched only while open, 503 `note_unavailable` shows a calm «الان در دسترس نیست» with no editor, 409 offers delete); crisis line with 1480 / 123 `tel:` pills; sticky «ادامه». Deliberate: «گفت‌وگو با مشاور» hidden until N7 doctors exist; the crisis text is a quiet `UrgentCard variant=note` with hotline pills (board: plain text); bleeding row shows today's logged flow or «امروز ثبت نشده»; visit row shows the catalog line until booked, then the booked day/time. In en the catalog titles keep their capitals inside the joined warning sentence (catalog copy). |
| `nbl_Loss_Next` | `/loss/next` | [next](loss/CB-LOSS-02/fa_loss_next.light.png) | [next](loss/CB-LOSS-02/fa_loss_next.dark.png) | ✔ | Same order: header «هر وقت آماده بودی», 24/900 «از اینجا به بعد» + lead, three `RadioCardGroup` cards from catalog `loss_next_steps` (drop/period, sprout/data, moon/brand), «سقط دوم یا سوم» `InfoNote` (only when `recurrent_hint`, i.e. ≥ 2 losses — shot with 2), sticky «ذخیره». «فقط پیگیری سیکل» preselected as on the board (her earlier answer wins). Save → `PUT /loss/next-step`, life-mode hint + full refetch, `/home` (verified: lands on the cycle home, mode cycle). |

## CB-LOSS-03 — epic QA + tone

Run 2026-10-04 against a worktree Go API :8181 (`APP_ENV=local`; once `APP_ENV=staging` without `PRIVATE_NOTE_KEY`)
+ Next dev :3108, 390 px, headless Chrome over CDP (`bloom/bin/shot.mjs --token`). Shots in `loss/CB-LOSS-03/`.
`ritme_dev` already had all seven `loss_*` catalog groups (5/5/3/3/4/4/3 rows) — nothing seeded.

**Test data (ritme_dev):** owner **`09120005403`** (مریم, female, goal pregnancy, pregnancy onboarding LMP 2026-08-01
→ week 10); companion **`09120005404`** (علی, link with `pregnancy: view` + `appointments: view`); companion
**`09120005405`** (رضا, link with `cycle: view` only). At the end the owner is in cycle mode with one loss row
(early miscarriage, no notice) and two private care appointments (beta 2026-10-10, visit 2026-10-18 10:30) left over
from the first loss (see P3).

### Journey

| # | Step | Result |
|---|---|---|
| 1 | Pregnant owner: `/home` → `/fa/pregnancy` (week 10 of 40) | ✔ [before](loss/CB-LOSS-03/fa_home.before.light.png) |
| 2 | `/profile/mode` → calm-exit row «بارداری‌ام ادامه پیدا نکرد» | ✔ opens `/loss` ([mode](loss/CB-LOSS-03/fa_profile_mode.before.light.png) → [loss](loss/CB-LOSS-03/fa_profile_mode.journey1.light.png)); «به همدمم هم خبر بده» shows (a companion holds the pregnancy grant), default OFF |
| 3 | `/loss`: pick «سقط در ۳ ماه اول», no date, notice OFF, «ادامه» | ✔ `POST /loss` 201 → `/loss/care` ([shot](loss/CB-LOSS-03/fa_loss.submitted.light.png)) |
| 4 | Pregnancy content gone (API, owner) | ✔ `/home` mode `cycle`, `/profile/life-stage` `cycle`, `/messages/mode` `cycle`; `/pregnancy/v2/today`, `/v2/weeks/10`, `/v2/calendar`, `/analysis/pregnancy` → 409 `pregnancy_not_active`; `/pregnancy/status` `is_active: false`; alerts summary all 0; `/pregnancy/content/10` 404; `/messages/daily?mode=pregnancy` answers exactly like `/messages/daily` (forced mode ignored; 400 «last period» because this user never logged a period) |
| 5 | Pregnancy screens after the loss (UI) | ✔ `/fa/pregnancy` and `/fa/pregnancy/weeks/10` show the «حالت بارداری روشن نیست» empty state (API 409) — [pregnancy](loss/CB-LOSS-03/fa_pregnancy.after.light.png) · [week](loss/CB-LOSS-03/fa_pregnancy_weeks_10.after.light.png). See P1 about its CTA |
| 6 | `/loss/care`: follow-ups (beta 10 Oct + visit 18 Oct 10:30 via `PUT /loss/followup`), mood «غمگین» (UI tap), note (API) | ✔ rows show «بعدی: ۱۸ مهر» / «۲۶ مهر · ساعت ۱۰:۳۰», mood chip selected, «یادداشتی داری · فقط برای خودت»; the note sheet decrypts and shows the text ([sheet](loss/CB-LOSS-03/fa_loss_care.note-sheet.light.png)) |
| 7 | Note without a key (`APP_ENV=staging`, no `PRIVATE_NOTE_KEY`) | ✔ `GET /loss/note` 503 `note_unavailable`; `GET /loss` still says `has_note: true`; the sheet shows only «یادداشت خصوصی الان در دسترس نیست. کمی بعد دوباره امتحان کن.» with no editor — [light](loss/CB-LOSS-03/fa_loss_care.note-503.light.png) · [dark](loss/CB-LOSS-03/fa_loss_care.note-503.dark.png). API back to local afterwards |
| 8 | Companion with `appointments: view` must not see the follow-ups | ✔ `/companions/links/2/sections/appointments` → `[]`; `/care/appointments/{4,5}?for_user_id=5` → 404; `/care/appointments?for_user_id=5` → `[]`; `GET /loss` as the companion → empty; companion home says «نوبت پیش رویی ثبت نشده» ([shot](loss/CB-LOSS-03/fa_home.companion-a.light.png)). Cycle-only companion → 403 `section_not_shared` for both sections |
| 9 | Companion notice only when opted in | ✔ loss with notice OFF → nothing in either inbox (`/home/notifications` empty, 0 `pregnancy_not_continuing` rows). Pregnant again → `POST /loss {notify_companion: true}` → علی (pregnancy grant) gets exactly one «مریم خبر داد که بارداری ادامه ندارد.» (→ `/companion`), رضا (cycle only) nothing; a same-day re-post sends no second one |
| 10 | `/loss/care` «ادامه» → `/loss/next` → «ذخیره» | ✔ lands on the cycle home: no pregnancy card; the beta reminder shows with the neutral title «آزمایش» ([next](loss/CB-LOSS-03/fa_loss_care.to-next.light.png) → [home](loss/CB-LOSS-03/fa_loss_next.saved.light.png)) |
| 11 | `DELETE /loss` | ✔ 200 «پاک شد.»; `GET /loss` empty; note gone (404 `loss_not_found`); the companion notice is removed from علی's inbox; mode stays `cycle`, pregnancy stays off (by design); a DELETE with nothing left → 404. Known (CB-LOSS-01 open item): the two private follow-up appointments stay in her care list (P3) |

### Fidelity (390 px)

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Loss_Start` | `/loss` | [fa](loss/CB-LOSS-03/fa_loss.light.png) · [en](loss/CB-LOSS-03/en_loss.light.png) | [fa](loss/CB-LOSS-03/fa_loss.dark.png) | ✔ | Same order and copy as the board, now including the companion row (as on the board). Selected card uses the brand tint (board: rose) — token map, CB-LOSS-02 decision. Fixed-ON stop switch and empty date as documented in CB-LOSS-02. |
| `nbl_Loss_Care` | `/loss/care` | [fa](loss/CB-LOSS-03/fa_loss_care.light.png) · [en](loss/CB-LOSS-03/en_loss_care.fixed.light.png) · [note](loss/CB-LOSS-03/fa_loss_care.note-sheet.light.png) · [503](loss/CB-LOSS-03/fa_loss_care.note-503.light.png) | [fa](loss/CB-LOSS-03/fa_loss_care.dark.png) · [note](loss/CB-LOSS-03/fa_loss_care.note-sheet.dark.png) · [503](loss/CB-LOSS-03/fa_loss_care.note-503.dark.png) | ✔ | **Fixed here:** in en the joined warning sentence kept every catalog title's capital («…2 hours in a row), A fever…, Severe pain…»). `warningSignParts` now lower-cases the leading capital of every part after the first and of the bracketed detail (acronyms like «hCG»/«IVF» and Persian untouched; unit test added). Counsellor row still hidden until N7 (documented). |
| `nbl_Loss_Next` | `/loss/next` | [fa](loss/CB-LOSS-03/fa_loss_next.light.png) · [en](loss/CB-LOSS-03/en_loss_next.light.png) | [fa](loss/CB-LOSS-03/fa_loss_next.dark.png) | ✔ | Same order; no recurrent note for this user (1 loss) — shown with 2 losses in CB-LOSS-02. Save → cycle home verified. |

### Tone review

Read: every key of `frontend/messages/{fa,en}/loss.json`, the Go response copy in `backend-go/internal/loss/lang/{fa,en}/loss.json`,
and the `loss_*` catalog seed in `backend-go/db/migrations/00028_loss.sql` (incl. the companion notice). Checked for: calm,
non-judgmental, no blame, no medical advice beyond «talk to your doctor», no celebratory words, no partner/mother
assumption. Overall the copy is calm and second-person; no celebratory words (toasts are plain «ثبت شد» / «Saved»);
the not-your-fault line is there; companions are «همدم» / «companion», never «همسر».

**Changed (frontend `loss.json` + Go copy `resources/translations/{fa,en}/loss.json` + i18n goldens fa/ar/en):**

| Key | Before | After | Why |
|---|---|---|---|
| `care.bleeding.sheetBody` (fa) | خونریزی هر روز را در ثبت روزانه بنویس تا قطع شود. … | تا وقتی خونریزی ادامه دارد، هر روز آن را در ثبت روزانه بنویس. وقتی قطع شد، اینجا علامت بزن. | «بنویس تا قطع شود» can read as "write it down so that it stops" |
| `care.bleeding.sheetBody` (en) | Log the bleeding each day in your daily log until it stops. … | While the bleeding goes on, log it each day in your daily log. When it has stopped, mark it here. | aligned with fa |
| `care.visit.sub` (fa / en) | حدود ۲ هفته بعد نوبت بگیر / Book one for about 2 weeks later | اگر پزشکت ویزیت پیگیری خواسته، اینجا ثبتش کن / If your doctor wants to see you again, add it here | a timing instruction is medical advice; timing belongs to her doctor |
| `care.moodSupport` (fa / en) | … بعد از سقط … / … after a pregnancy loss … | … بعد از این اتفاق … / … after this … | the clinical word isn't needed in the comfort line, and this way it fits every type (ectopic, chemical, «rather not say») |
| `next.recurrentFallback` (fa / en) | اگر سقط دوم یا سوم بود، … با پزشکت صحبت کن. / If this was a second or third loss, talk with your doctor … | اگر این اولین بار نبود، می‌توانی با پزشکت درباره آزمایش‌هایی که علت را بررسی می‌کنند صحبت کنی. / If this wasn't the first time, you can talk with your doctor about tests that look for a cause. | counting losses reads cold; an invitation, not an order |
| `start.stopSub` (en) | … or baby growth any more | … or the baby's growth any more | grammar |

Most of these keys are fallbacks — the screen shows the **catalog** copy when it exists, so the same wording is proposed
for the seed below (no migration changed). Unchanged on purpose: «متأسفیم که این را تجربه کردی», the type names (board
copy), the 115 / 1480 / 123 lines (urgent by design), «اولین پریود معمولاً ۴ تا ۶ هفته بعد می‌آید» (informational,
already `needs_review`). The Go response messages (`internal/loss/lang`) read fine — unchanged.

**Proposed catalog rewrites (`catalog_items`, admin-editable — not changed here):**

| group / code | Field | Now | Proposed |
|---|---|---|---|
| `loss_support/companion_notice` | `meta.someone.en` | Your partner | Someone close to you (no partner assumption) |
| `loss_support/companion_notice` | `meta.someone.fa` | همراهت | یکی از نزدیکانت |
| `loss_support/mood_support` | body | …بعد از سقط… / …after a pregnancy loss… | …بعد از این اتفاق… / …after this… |
| `loss_support/recurrent_hint` | title + body | اگر سقط دوم یا سوم بود … / If this was a second or third loss … | title «اگر این اولین بار نبود» / "If this wasn't the first time"; body = the new `next.recurrentFallback` |
| `loss_followups/visit` | body | حدود ۲ هفته بعد نوبت بگیر / Book one for about 2 weeks later | اگر پزشکت ویزیت پیگیری خواسته، اینجا ثبتش کن / If your doctor wants to see you again, add it here |
| `loss_next_steps/cycle` | body | اولین پریود معمولاً ۴ تا ۶ هفته بعد می‌آید | keep, pending the clinical reviewer (medical statement) |

The companion notice «{name} خبر داد که بارداری ادامه ندارد.» / "{name} let you know that the pregnancy is not
continuing." is calm and shares nothing else — ✔ (en shows the owner's Persian name as stored, e.g. «مریم let you know…»).

### Follow-up proposals (outside the loss files)

- **P1 (bloom pregnancy screen):** after a loss `/pregnancy` shows «حالت بارداری روشن نیست» with a large «راه‌اندازی حالت
  بارداری» button — an abrupt prompt right after a loss. Suggest: with a recent loss (`GET /loss`), redirect `/pregnancy*`
  to `/home`, or show a calm line without the CTA. Bloom-owned — for the user to decide.
- **P2 (bloom companion home):** with `pregnancy: view` and pregnancy off, the companion's shared list still says
  «بارداری · همین کارت بالا» but no card is above. Suggest hiding the row (or a neutral line) when `pregnancy.is_active` is false.
- **P3 (open since CB-LOSS-01/02):** `DELETE /loss` leaves the private beta/visit appointments in her care list; no way
  back into `/loss/care` after leaving it. Candidate `CB-LOSS-03b` if wanted in this epic.
- **P4:** apply the catalog rewrites above (admin edit or an `UPDATE` seed migration) — content owner + clinical review.

No ✘ left.

## CB-LOSS-03b — care re-entry + follow-up cleanup on erase

Run 2026-10-04 against a worktree Go API :8202 (`APP_ENV=local`) + Next dev :3112, 390 px, headless Chrome over CDP
(`bloom/bin/shot.mjs --token`). Shots in `loss/CB-LOSS-03b/`. Test user **`09120005403`** (CB-LOSS-03; cycle mode,
one loss recorded today).

**Default window (documented in `entities/loss/model/care-return.ts`):** the re-entry shows for
`LOSS_CARE_WINDOW_DAYS` = **60 days** from the Tehran day she recorded the loss (`created_at`, else the approximate
`occurred_on`), and ends earlier when she erases the record (`DELETE /loss`). Home row: dismissible per loss
(«پنهان کردن» keeps only the loss row id in `localStorage['ritme_care_return_hidden']`, neutral key; a newer loss
shows it again). `/profile/mode` row: not dismissible — the way back after hiding the home row. Copy never names
the loss («مراقبت از خودت · پیگیری‌ها و حال دلت، هر وقت خواستی»), no celebratory tone, no red. Companions never
see it (they don't load her home/profile; `GET /loss` answers only for her).

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Loss_Care` (entry) | `/home` (cycle/TTC) | [row](loss/CB-LOSS-03b/fa_home.light.png) · [hidden](loss/CB-LOSS-03b/fa_home.hidden.light.png) · [en](loss/CB-LOSS-03b/en_home.light.png) · [erased](loss/CB-LOSS-03b/fa_home.erased.light.png) | [row](loss/CB-LOSS-03b/fa_home.dark.png) · [hidden, after reload](loss/CB-LOSS-03b/fa_home.hidden-persisted.dark.png) · [en](loss/CB-LOSS-03b/en_home.dark.png) · [erased](loss/CB-LOSS-03b/fa_home.erased.dark.png) | ✔ | No board for the entry (the canvas has no way back) — styled as a quiet flat card (surface + 1px line, brand heart disc, chevron, 44px × hide). First item of the feed, under the phase card; not for teen. After «پنهان کردن» one calm status line points to «من ← مرحله زندگی»; the row stays hidden after reload. Gone after `DELETE /loss`. |
| `nbl_Loss_Care` (entry) | `/profile/mode` | [row](loss/CB-LOSS-03b/fa_profile_mode.light.png) · [erased](loss/CB-LOSS-03b/fa_profile_mode.erased.light.png) | [row](loss/CB-LOSS-03b/fa_profile_mode.dark.png) · [erased](loss/CB-LOSS-03b/fa_profile_mode.erased.dark.png) | ✔ | A `ListRow` card between the mode cards and the contraception card, same shape as the contraception card; → `/loss/care`. |

**Erase (backend):** `DELETE /loss` now also deletes the private beta + visit care appointments linked to that loss
(past or upcoming), in the same transaction. Live: follow-ups set (lab 2026-10-11, checkup 2026-10-20 11:00 → care ids
39, 40) → `DELETE /loss` 200 «پاک شد.» → `/care/appointments` no longer lists 39/40. Ids 4/5 (lab 10 Oct, checkup
18 Oct) are orphans from losses erased **before** this fix (CB-LOSS-03 P3) and stay — no backfill. Int test
`TestLoss_EraseAndConcurrency` covers newest-only removal, the earlier loss keeping its visit, and her own
appointments staying. Contract `loss` 50/50 unchanged (the DELETE response shape didn't change). Test user left with
one fresh loss (re-recorded after the erase check).
