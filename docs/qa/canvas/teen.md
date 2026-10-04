# TEEN — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png` (`roadmap/bin/shot-board.sh`). Screens: 390 px, fa, headless Chrome
over CDP (`bloom/bin/shot.mjs --token`) against a worktree Go API (:8133) + Next dev (:3103).
Test user **`09120007702`** (created for CB-TEEN-02; onboarding name «نیلا», female, goal cycle → life stage teen via
`PUT /profile/life-stage`; teen profile 13_15 / not_yet; kit «۲ نوار بهداشتی» + «یک لباس زیر اضافه» ticked). Colours are
checked against the token map (`docs/canvas-build/README.md` §4), not the board hex.

## CB-TEEN-02

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Teen_Onb` | `/teen/onboarding` | [picked](teen/CB-TEEN-02/fa_teen_onboarding.picked.light.png) | [picked](teen/CB-TEEN-02/fa_teen_onboarding.picked.dark.png) | ✔ | Same order and copy: back button only (→ `/profile/mode`; never `/home`, which would bounce her back here), «سلام! کمی از خودت بگو» + privacy line, card «چند سالته؟» with three pill chips (۱۰ تا ۱۲ · ۱۳ تا ۱۵ · ۱۶ تا ۱۷), card «پریود شده‌ای؟» with three `RadioCardGroup` cards (هنوز نه sprout/data + «کمکت می‌کنیم آماده باشی», تازه شروع شده drop/period + «کمتر از یک سال», بیشتر از یک سال است calendar/brand), sticky glass footer «ادامه». No bottom nav (form). The selected age chip is the app's solid single-select chip (board: soft outline) and the selected radio card uses `--brand-soft` (board: teal tint) — the shared primitives' selected states. «ادامه» = `PUT /profile/life-stage {mode: teen}` (skipped when already teen) then `PUT /teen/profile`, then `/home` (verified end-to-end). Answers are prefilled from `GET /teen/profile` when she comes back. |
| `nbl_Teen_Home` | `/home` (mode teen) | [not yet](teen/CB-TEEN-02/fa_home.not-yet.light.png) · [kit tick + FAQ open](teen/CB-TEEN-02/fa_home.kit-faq.light.png) · [first year](teen/CB-TEEN-02/fa_home.first-year.light.png) · [16–17, not yet (caution)](teen/CB-TEEN-02/fa_home.talk.light.png) | [not yet](teen/CB-TEEN-02/fa_home.not-yet.dark.png) · [kit tick + FAQ open](teen/CB-TEEN-02/fa_home.kit-faq.dark.png) · [first year](teen/CB-TEEN-02/fa_home.first-year.dark.png) · [16–17, not yet (caution)](teen/CB-TEEN-02/fa_home.talk.dark.png) | ✔ | Same hierarchy: «ریتمی نوجوان / سلام نیلا» (Lalezar) + sprout disc; signs card «نشانه‌های نزدیک شدن اولین پریود» + body, the turquoise estimate bar (grows from the inline start) and «بر اساس جواب‌هایت: احتمالاً در ماه‌های آینده»; «کیف اضطراری مدرسه» checklist (`PUT /teen/kit/{code}`, optimistic, `--brand-fill` boxes); «طبیعی است؟» one card of disclosure rows (all 5 FAQ rows from the API; board shows 3); when-to-talk `InfoNote`; «همراهی مادر» row; bottom nav امروز · تقویم · + · خدمات · من. No banners, Plus trial offer, shop or fertility copy. Additions: «۲ از ۴» kit count and «کیفت آماده است» when all ticked; the second approach sign («جهش قد · …») under the first; the bar is drawn only for the two «before the first period» estimates (35% / 65%), none for «talk» (caution tint instead) or unknown codes; after the first period the estimate item is the whole card + «دیدن تقویم و ثبت پریود» (→ `/calendar`). Without onboarding answers the home redirects to `/teen/onboarding`. «همراهی مادر» links to `/companions` for now — bloom's invite wizard has no `parent` type; CB-TEEN-03 repoints it to `/teen/parent`. |

## CB-TEEN-03

Worktree Go API :8143 + Next dev :3104 (own CDP script). `ritme_dev` was re-created by another session mid-task, so the
teen **`09120007702`** was re-created with the same CB-TEEN-02 setup (نیلا, teen 13_15 / not_yet, two kit ticks). Parent
test user **`09120007703`** (new; onboarding name «مریم», female, goal cycle, no period data). They were linked through the
UI (teen «فرستادن دعوت» → parent «کد همراهی از فرزندت داری؟» on `/companions`) and the link was revoked again through
the UI at the end; the teen's note was cleared (`PUT /teen/parent-note {note: null}`).

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Teen_Parent` | `/teen/parent` (no link) | [all off](teen/CB-TEEN-03/parent-empty.light.png) · [two on](teen/CB-TEEN-03/parent-shared.light.png) · [invite sheet](teen/CB-TEEN-03/parent-invite-sheet.light.png) · [code sent](teen/CB-TEEN-03/parent-invite-sent.light.png) · [en](teen/CB-TEEN-03/en-parent.light.png) | [two on](teen/CB-TEEN-03/parent-shared.dark.png) · [invite sheet](teen/CB-TEEN-03/parent-invite-sheet.dark.png) | ✔ | Same order and copy: back header «همراهی مادر» (→ `/home`), «چه چیزی را با مادرت در میان بگذاری؟» + lead, one card with the three switch rows (زمان تقریبی پریود بعد + «فقط «این هفته» یا «هفته بعد»», یادآور کیف اضطراری, علائم و یادداشت‌ها), «مادرت این را می‌بیند» + dashed preview card (name + sentences), shield note, sticky glass «فرستادن دعوت». No bottom nav (flow). Deviation (on purpose, minors' privacy): every switch starts **off** (board shows two on); the preview then says «فعلاً چیزی به اشتراک گذاشته نشده؛ …». Additions: «علائم و یادداشت‌ها» caption «فقط یادداشتی که خودت می‌نویسی»; the CTA opens a half sheet (mother's phone — required, the code is bound to it — and an optional name) and then the one-time code sheet (bloom's `InviteCodeCard` + «کد برای 0912****703 پیامک شد»). |
| `nbl_Teen_Parent` | `/teen/parent` (linked) | [active + note](teen/CB-TEEN-03/parent-active-note.light.png) · [active](teen/CB-TEEN-03/parent-active.light.png) · [revoke sheet](teen/CB-TEEN-03/parent-revoke-sheet.light.png) · [after revoke](teen/CB-TEEN-03/parent-revoked.light.png) | [active](teen/CB-TEEN-03/parent-active.dark.png) · [revoke sheet](teen/CB-TEEN-03/parent-revoke-sheet.dark.png) | ✔ | Not on the board (it shows only the pre-invite state): a status card «مامان همراهت است · وصل» (pending: «دعوت فرستاده شد» + «کد تازه»), switches now write `PUT /companions/{id}/grants` at once (optimistic), the note editor (≤ 280, counter, «ذخیره یادداشت» → `PUT /teen/parent-note`) appears while «علائم و یادداشت‌ها» is on, and «قطع همراهی» / «لغو دعوت» behind bloom's confirm sheet (`DELETE /companions/{id}`). The preview is `parent_preview` masked by the live grants — the same component the mother's card uses. |
| — (no board; parent side) | `/home` (mother, cycle mode) | [teen card](teen/CB-TEEN-03/mother-home.light.png) | [teen card](teen/CB-TEEN-03/mother-home.dark.png) | ✔ | `widgets/linked-teen-card`: «همراهی فرزندت» + one read-only card per `GET /teen/linked` row — the teen's name, «فقط دیدنی» pill, the week bucket / kit / note sentences, «نیلا خودش انتخاب می‌کند چه چیزهایی را ببینی.». Renders nothing for non-parents. Also mounted on the menopause home and the companion (male) home. |
| — (no board; parent side) | `/companions` | [code typed](teen/CB-TEEN-03/parent-code-filled.light.png) · [linked](teen/CB-TEEN-03/parent-code-linked.light.png) | [code typed](teen/CB-TEEN-03/parent-code-filled.dark.png) | ✔ | «کد همراهی از فرزندت داری؟» card (bloom's `CompanionCodeField` + `POST /companions/accept`) under the owner's companion list — a woman had no place to enter a companion code before. |


## CB-TEEN-04 — epic QA

Worktree Go API :8191 + Next dev :3109, 390 px, fa (+ one en check). Board states via `bloom/bin/shot.mjs --token`, the
two-account journey through the Playwright MCP browser (same build). Screens in [`teen/CB-TEEN-04/`](teen/CB-TEEN-04/).

**Test data (local `ritme_dev`, created through the API/UI):** teen **`09120007704`** «آوا» (female, goal cycle, one
period 2026-09-20 / 5 d / 28 d cycle → teen 13_15 / under_1y through the UI; kit «۲ نوار بهداشتی» + «دستمال مرطوب»
ticked), mother **`09120007705`** «لیلا» (female, cycle; she also started a Plus trial during the commercial check),
partner **`09120007706`** «سام» (male; partner link #4 to آوا, accepted while she was still adult — left in place).
Parent link #5 was created and revoked through the UI; آوا's note was cleared by the revoke. One QA banner was inserted
and deleted again. CB-TEEN-02/03's **`09120007702`** «نیلا» (13_15 / not_yet) was reused for the «not yet» board state.

### Journey

| # | Step | Result |
|---|---|---|
| 1 | New account → `/profile/mode` → «نوجوان» card | ✔ `PUT /profile/life-stage {teen}`; [mode picked](teen/CB-TEEN-04/profile_mode.j1-teen-picked.light.png) |
| 2 | `/home` without answers → `/teen/onboarding` | ✔ redirect; [empty](teen/CB-TEEN-04/home.onb-empty.light.png) / [dark](teen/CB-TEEN-04/home.onb-empty.dark.png) |
| 3 | Onboarding: ۱۳ تا ۱۵ + «تازه شروع شده» → «ادامه» | ✔ `teen/profile` = 13_15 / under_1y, lands on `/home` (first-year estimate + calendar link) |
| 4 | Teen home: tick 2 kit items, open FAQ | ✔ persisted (`checked_count` 2, pads + wipes), «۲ از ۴», FAQ disclosure opens; no `/api` errors |
| 5 | `/teen/parent`: switches | ✔ all three **off** by default; preview «فعلاً چیزی به اشتراک گذاشته نشده…» |
| 6 | Switch all three on → invite sheet → mother's phone + «مامان» → send | ✔ code `8SPRMF`, «کد برای 0912****705 پیامک شد» |
| 7 | Mother: `/companions` → «کد همراهی از فرزندت داری؟» → code → «اتصال» | ✔ «به آوا وصل شدی.» |
| 8 | Mother's `/home` | ✔ «همراهی فرزندت» card: name, «فقط دیدنی», week bucket + kit + note sentences, footer ([light](teen/CB-TEEN-04/home.mother-all.light.png) / [dark](teen/CB-TEEN-04/home.mother-all.dark.png)) |
| 9 | Mother tries `/companions/links/5/sections/{cycle,teen_period_week,symptoms}`, `/companion/home` | ✔ 404 ×3, 403 `not_companion_account` |
| 10 | Teen switches «علائم و یادداشت‌ها» off | ✔ `teen_notes: none`, `note: null`; mother's card drops the note line at once ([after](teen/CB-TEEN-04/home.mother-notes-off.light.png)) |
| 11 | Teen «قطع همراهی» → confirm sheet | ✔ `GET /teen/linked` (mother) = `[]`; teen side back to the invite state |
| 12 | Bloom B-N4-08b guard: partner link accepted **before** the switch (grants cycle + symptoms) | ✔ after the switch `/companion/home` shows `grants` all `none`, `cycle`/`symptoms` null, phase `general`; `/sections/cycle` 403 `section_not_shared` |
| 13 | Commercial surfaces (teen vs mother, one active banner seeded) | ✔ teen: `/banners` empty in every slot (mother gets the banner), `plus/status.trial_offer` + `plus/trial.offer` + `home/cycle-overview.plus_trial_offer` null, `plus/trial/start` + `plus/checkout` **403** `teen_commercial_blocked`, `allows` all false; `/plus` → `/profile`; `/shop` 404; teen home has no banner/Plus/shop; `/services` has no shop/Plus rows |
| 14 | Period-week bucket never reveals lateness | ✔ **after fix** (below). Live, today Sun 10-04 (week 10-03…10-09), 28-day cycle, moving her one period back: predicted 10-18 → later · 10-06 → this_week · **10-02 / 10-01 (2–3 days late, previous week) → later** (was this_week) · 09-29 → later · 09-25 → later · 09-18 → next_week · 09-11 → this_week (rolled to 10-09; a week later it rolls again → later). |
| 15 | Fertility copy in teen mode | ~ teen home, onboarding, `/teen/*`, log sheet: none. **Calendar is not clean**: legend «پنجره باروری» and the day line «تخمک‌گذاری» ([screen](teen/CB-TEEN-04/calendar-teen-fertility.light.png)); `/messages/daily` and `/cycle/today` answer a teen with ovulation copy («در اوج انرژی و جذابیت هستی!») — the teen home does not render it, but the API serves it. Outside the teen files → **CB-TEEN-04b**. |

**Fix (backend, teen package):** `teen.WeekBucket` let the 3-day overdue grace carry a prediction across a week
boundary: a Thursday/Friday prediction read «این هفته» in its own week and again on the next Saturday–Monday, i.e.
«this week» two weeks in a row for a late period. The grace now applies only while the prediction is still in
today's week; otherwise the date rolls forward by the cycle length. `teen_test.go` gains the concrete case and an
exhaustive check (60 predictions × 90 days × 4 cycle lengths: never «this week» on a day and 7 days later).

### Fidelity (390 px, light + dark)

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Teen_Onb` | `/teen/onboarding` | [picked](teen/CB-TEEN-04/teen_onboarding.picked.light.png) | [picked](teen/CB-TEEN-04/teen_onboarding.picked.dark.png) | ✔ | Unchanged since CB-TEEN-02: same order and copy; selected chip / radio card use the shared primitives' selected states (documented deviation). |
| `nbl_Teen_Home` | `/home` (teen) | [not yet](teen/CB-TEEN-04/home.not-yet.light.png) · [first year + kit + FAQ](teen/CB-TEEN-04/home.kit-faq.light.png) | [not yet](teen/CB-TEEN-04/home.not-yet.dark.png) · [first year + kit + FAQ](teen/CB-TEEN-04/home.kit-faq.dark.png) | ✔ | Matches the board's hierarchy, bar, kit, FAQ (5 rows vs the board's 3), talk note and «همراهی مادر» row; no banners / Plus / shop. |
| `nbl_Teen_Parent` | `/teen/parent` | [default](teen/CB-TEEN-04/teen_parent.default.light.png) · [linked](teen/CB-TEEN-04/teen_parent.linked.light.png) · [en invite sheet](teen/CB-TEEN-04/en-invite-sheet.light.png) | [default](teen/CB-TEEN-04/teen_parent.default.dark.png) · [linked](teen/CB-TEEN-04/teen_parent.linked.dark.png) | ✔ | **Fixed drift:** the hairlines between the three switch rows bent up at both ends (the shared `.nb-row` 12 px radius on a `border-top`); `.tnp-share .nb-list-rows > * + * { border-radius: 0 }` in the CB-TEEN-03 CSS block — straight like the board. All-off default kept on purpose. |
| — (parent side) | `/home` (mother) | [card](teen/CB-TEEN-04/home.mother-all.light.png) | [card](teen/CB-TEEN-04/home.mother-all.dark.png) | ✔ | Read-only card as in CB-TEEN-03. |

### Copy (age-appropriate pass)

Frontend `teen.json` is calm, second-person, no shaming; the parent card never states lateness. Changed (fa + en,
Go copy in `resources/translations/*/teen.json`, i18n goldens fa/en/ar), to stop assuming a mother where it can be
avoided (the feature itself is «همراهی مادر» per the board, so titles keep «مادر»):

| Key | Before | After |
|---|---|---|
| `teen.home.talk` (sr-only fallback title) | کی با مادرت یا پزشک صحبت کنی · When to talk to your mother or a doctor | کی با یک بزرگ‌تر مورد اعتماد یا پزشک صحبت کنی · When to talk to a trusted adult or a doctor |
| `teen.parent.invite.lead` | شماره موبایل مادرت را بنویس. … · Enter your mom's mobile number. She'll get … | شماره موبایل مادرت (یا بزرگ‌تری که به او اعتماد داری) را بنویس. … · Enter your mom's mobile number (or another adult you trust). They'll get … |
| `teen.parent.invite.namePlaceholder` | مثلاً مامان · e.g. Mom | مثلاً مامان یا خاله · e.g. Mom or Auntie |

**Catalog rewrite proposals** (`00029_teen.sql`, all rows `needs_review` — for the content owner / clinical review, not
changed here):

| Row | Now | Proposal | Why |
|---|---|---|---|
| `teen_signs.when_to_talk` title + body | «… با مادرت یا پزشک …» / "… your mother or a doctor" | «… با یک بزرگ‌تر مورد اعتماد (مثل مادرت) یا پزشک …» / "… a trusted adult (like your mom) or a doctor" | not every teen has a mother to talk to |
| `teen_signs.estimate_talk` title | «بهتر است با مادرت یا پزشک صحبت کنی» | «خوب است با یک بزرگ‌تر مورد اعتماد یا پزشک صحبت کنی» | same |
| `teen_signs.estimate_talk` body | «بیشتر دخترها تا ۱۵ سالگی پریود می‌شوند. …» / "Most girls get their first period by 15. …" | «هر بدنی زمان خودش را دارد. یک سر زدن به پزشک کمک می‌کند خیالت راحت باشد که همه چیز روبه‌راه است.» | a 16–17-year-old reads "most girls already have" as "something is wrong with me"; keep it calm |
| `teen_faq.how_much_bleeding` body | «… به مادرت یا پزشک بگو» | «… به یک بزرگ‌تر مورد اعتماد یا پزشک بگو» | same as the first row |
| `teen_faq.period_pain` body | «… مسکن ساده با اجازه بزرگ‌ترها …» | keep, but flag: medication advice for minors — clinical sign-off needed | §11 «not medical advice» |
| `teen_signs.approaching_signs` body | «… بعد از شروع رشد سینه‌ها …» | keep (accurate, neutral) | — |

### Follow-up proposed: `CB-TEEN-04b` — teen mode outside the teen screens

1. `/calendar` in teen mode: hide the fertile-window / ovulation markers, the «پنجره باروری» legend chip and the
   day line phase «تخمک‌گذاری» (the home already does this via `hideFertility`).
2. Backend: `/messages/daily` and `/cycle/today` for a teen-mode account return no ovulation/fertile phase copy (or a
   teen-safe phase set); the teen home still fires `/messages/daily` (400 without period data) — drop that call in
   teen mode.
3. `/profile` in teen mode: the «همدم‌ها و خانواده» row shows the dormant **partner**'s name («سام») although it opens
   `/teen/parent`; show the parent link instead. The «کارها و خریدها / سفارش‌ها» (coming soon) rows are a commercial
   surface — hide in teen mode. The log sheet's «وزن و دمای پایه» could drop BBT for teens.
4. Product decision (CB-TEEN-01 TODO): a partner link from before the switch stays `active` (grants nothing while
   teen, but resumes when she leaves teen mode) — auto-revoke or pause on entering teen mode?
