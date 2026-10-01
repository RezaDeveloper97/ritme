# N1 design-fidelity audit (B-N1-16)

Side-by-side of every N1 screen (light + dark, 390 px, full page) against its artboard, after B-N1-05 … B-N1-15.
Screens, artboard renders and 4-up comparisons (artboard light · app light · artboard dark · app dark):
`docs/qa/bloom/B-N1-16/` — `p04` regular cycle, `p06` near period, `p07` during period, `p12` TTC, `p15` pregnancy,
`p18` reminders/checkups, `p01` fresh user, `public` splash/intro, `artboards/`, `side-by-side/`.
Data-driven differences (dates, counts, copy coming from the API, empty states of the dev DB) are not deviations.
Deliberate defaults already recorded in `bloom/QUESTIONS.md` are listed as **accepted (QUESTIONS #n)** and not changed.

Severity: **high** = broken control / unreadable; **med** = visible departure from the board or the token rules;
**low** = polish. Totals: **high 1 · med 7 · low 9** found — fixed: high 1 / med 7 / low 2; **7 low open** (+ accepted / data rows).

Environment note: `ritme_dev` (rebuilt from the contract dump, QUESTIONS #38) had goose rows 1–13 but lacked the tables of
migrations 00002/00003/00005 (+ data of 00007/00008) → `/care/today`, `/checkups/home`, `/pregnancy/v2/*` returned 500.
Their Up sections were applied to `ritme_dev` before shooting (dev data only, no code change).

## Cross-cutting

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| X1 | Below-the-fold buttons in flex-column scrollers shrink to their text line (~23 px): «افزودن سیکل‌های قبلی» on `/cycle`, «خروج از حساب» on `/profile/account` (board: 54 px pill). Touch target broken. | high | `frontend/src/app/globals.css:508` | **Fixed** — `.scroll > * { flex-shrink: 0 }` for every screen scroller. Re-shot `p04/fa_cycle.*`, `p04/fa_profile_account.*`. |
| X2 | Bottom sheets painted `--page` (lavender / night canvas); every sheet board (Cycle_Phase, Cycle_EditPeriod, v13_AddChooser, v14_MarkDone, Log_Sheet_*) paints the panel `--surface`, and tokens.md lists `--surface` for sheets. | med | `globals.css:730` | **Fixed** — `.osheet` background `--surface`; cards inside keep their 1px line (as drawn). Re-shot phase, edit-period, log, reminders-add, mark-done. |
| X3 | Leftover from B-N1-02: labels on brand/bloom fills still `--on-accent` (white on lavender/peach at night): onboarding checks (conditions, intention, pregnancy-basis, name terms box), DayTasks check (`/log`), PeriodButton, DailyStatusCard, WeekStrip. TodayChallenge, IntroIllustration, DayLogPage were already switched by their screen tasks. | med | `ConditionsPage.tsx:78`, `IntentionPage.tsx:81`, `PregnancyBasisPage.tsx:108`, `NamePage.tsx:74`, `DayTasks.tsx:160`, `PeriodButton.tsx:59-71`, `DailyStatusCard.tsx:74-75`, `WeekStrip.tsx:16` | **Fixed** — `--on-brand` (+ `--brand-fill` where the fill was `--brand`). |
| X4 | Danger fills (`--danger` → `#FF8FA3` at night) still carry white `--on-accent`: delete-confirm button, track-pregnancy danger chip, mark-done error toast, alert badge. Not brand fills, so outside X3. | low | `globals.css` `.del-confirm`, `features/track-pregnancy/ui/controls.tsx:72`, `screens/checkup-mark-done/ui/MarkDoneToast.tsx:28` | Open — needs an `--on-danger` token (flip like `--on-brand`); follow-up. |
| X5 | Boards draw a 54 px fake status bar; ScreenHeader back is an arrow, boards draw a chevron. | — | — | accepted (B-N1-05 note; QUESTIONS #30) |
| X6 | Light TTC boards (`v19_`) use the older dialect (dark nav pill, `#7B61FF`). | — | — | accepted (QUESTIONS #1) |

## Cycle home — `/home` (Cycle_Home, _Near, _During; personas 04/06/07)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| H1 | Legacy «بر اساس سیکل فعلی‌ات» articles block off-system: no border, radius 12, 17/700 title, 40 px / r16 CTA. | med | `globals.css:1207-1217`, `screens/home/ui/HomePage.tsx:455` | **Fixed** — flat card (1px `--line`, `--r-card`), 15/800 title, 54 px pill CTA. |
| H2 | «یادآورهای امروز» and «چکاپ‌های دوره‌ای» cards inset 16 px narrower than the feed and carry a drop shadow (cards are flat). | med | `globals.css:5530-5532` | **Fixed** — `.trm-sec` gutter reset inside `.ch-feed`, `box-shadow: none`. |
| H3 | Header action is a gear → `/cycle/settings`; board draws a sun icon. | — | — | accepted (QUESTIONS #53) |
| H4 | Extra legacy blocks (banners, reminders, checkups, articles) between the board's cards. | — | — | accepted (QUESTIONS #40) |
| H5 | Banner slot shows a broken-image box with alt text when the banner file is missing (dev data has no uploads). | low | `widgets/banner-slideshow/ui/BannerSlideshow.tsx:137` | Open — hide a slide whose image fails (`onError`); env-only today. |
| H6 | «ثبت امروز» has no «N روز پشت‌سرهم» streak overline; challenge card «۰ از ۱». | — | — | data (streak 0) / accepted (QUESTIONS #40) |
| H7 | Fresh user (`p01`): ring area shows no dotted track behind «هنوز پریودی ثبت نشده». No board for this state. | low | `HomePage.tsx` ring block | Open — polish. |
| H8 | Near period: «امروز» instead of «۱ روز دیگر»; during: day 11. | — | — | data (persona dates) / QUESTIONS #39 |

## Cycle — `/calendar`, `/cycle`, `/cycle/symptoms`, `/cycle/settings`, sheets `phase`, edit-period

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| C1 | `/cycle` bars ran right→left (period at the right); Cycle_History draws day 1 / period at the **left**, like every chart board (TTC_BBT, TTC_Insights, v14 status bar) and the app's own BBT chart. | med | `globals.css:5795`, `screens/cycle/ui/CyclePage.tsx:206-213` | **Fixed** — `.cyh-bar` `direction: ltr`. |
| C2 | `/cycle/symptoms` typical-cycle axis and heat strips ran right→left («۱ روز» at the right); board: left. The ovulation label also mis-centred in RTL (`translateX(-50%)` with inline-start). | med | `globals.css:5795`, `CycleSymptomsPage.tsx:186-230` | **Fixed** — `.csp-typ-bar/.csp-axis/.csp-strip` `direction: ltr`. |
| C3 | Phase sheet: small «جزئیات فاز» header title duplicated the big phase title. | low | `app/sheets/registry.tsx:40` | **Fixed** — header title visually hidden (still names the dialog). |
| C4 | Phase sheet: close button sits in its own row above overline + title (board: same row). Info note filled, board outlined. | low | `screens/phase-details/ui` | Open — polish. |
| C5 | `/cycle` share/export header button dropped; SmartTip/WeekSummary/BMI cards gone. | — | — | accepted (QUESTIONS #46) |
| C6 | Phase sheet tab mapping and long DB copy instead of 3 bullets. | — | — | accepted (QUESTIONS #47) / data |
| C7 | Calendar: current month first; no ovulation legend in cycle mode; year view has no board. | — | — | accepted (QUESTIONS #37) |
| C8 | `/cycle/settings`, `/calendar` (month/legend/day card), edit-period sheet | — | — | match |

## TTC — `/home` (Main), `/calendar` (TTC_Calendar), `/fertility/log|bbt|insights` (persona 12)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| T1 | `/fertility/log` BBT value sat at the left of the field (input is `dir=ltr`, `text-align: start`); board: value at the right, unit + steppers at the left. | med | `globals.css:4313`, `FertilityLogPage.tsx:363` | **Fixed** — `[dir="rtl"] .ttc-bbt-input { text-align: right }`. |
| T2 | TTC calendar header: pencil (edit periods) at the end; board draws a filter icon. | low | `screens/calendar` header | Open — the filter has no function yet; keep the working edit action. |
| T3 | TTC ring shows «پریود بعدی» + «ویرایش پریود» after ovulation (board shows the pre-ovulation «تخمک‌گذاری تا N روز» + «ثبت امروز» state). | — | — | state-dependent, implemented (`HomePage.tsx:797-831`) |
| T4 | Amber/turquoise text uses `-deep` tokens in light. | — | — | accepted (QUESTIONS #11) |
| T5 | BBT/insights on dev data are empty (persona 12 BBT days lost); compared with `docs/qa/bloom/B-N1-13/` shots — chart, stats, tip, dots rows match. | — | — | data (QUESTIONS #38) |

## Pregnancy — `/pregnancy`, `/pregnancy/weeks`, `/calendar`, `/log`, `/alerts`, `/setup` (persona 15)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| P1 | Alerts header gear opened the inbox sheet instead of notification settings. | low | `screens/pregnancy-alerts/ui/PregnancyAlertsPage.tsx:65` | **Fixed** — → `/profile/notifications` (QUESTIONS #35). |
| P2 | Bottom nav still on `/pregnancy/calendar`. | — | — | accepted (QUESTIONS #17) |
| P3 | Month grid kept below care plan; week switcher only; no week carousel on Today; 4-step setup with the API's 3 bases / 5 moods. | — | — | accepted (QUESTIONS #24–#27) |
| P4 | Week screen «در انتظار بازبینی متخصص · منابع» footer card is low-contrast and loosely laid out vs the source note in the board. | low | `screens/pregnancy-week/ui/WeekSections.tsx` | Open — polish. |

## Reminders & checkups — `/reminders/*`, `/checkups/*`, sheets (persona 18)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| R1 | All screens + `reminders-add` / `checkup-mark-done` sheets match (after X2). Checkup history/detail empty on dev data. | — | — | match / data |
| R2 | Dark switch knob painted `--surface` on these screens only. | — | — | accepted (QUESTIONS #29) |

## Me — `/profile`, `/profile/account|appearance|notifications|privacy|support|about|legal`, `/services`

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| M1 | Me hub: the mode pill «● چرخه» sits inside the profile card; Me_Hub draws it as a full-width rose strip under the card. | low | `screens/profile/ui/ProfilePage.tsx` (me-id card) | Open — polish. |
| M2 | Account: family name / email / phone change / devices «به‌زودی»; extra «چرخه و سلامت» group. | — | — | accepted (QUESTIONS #19–#20) |
| M3 | Notifications: «متن خنثی» switch in the note card, inline quiet-hours pickers, static subtitles. | — | — | accepted (QUESTIONS #31–#33) |
| M4 | Privacy: «دسترسی دیگران» / backup rows «به‌زودی»; support FAQ groups and about/legal from admin info-sections (no phone row). | — | — | accepted (QUESTIONS #42–#43) |
| M5 | `/services` is the B-N1-04 placeholder. | — | — | accepted (QUESTIONS #18) |
| M6 | Latin test-user name renders in the fallback Latin face. | — | — | data |

## Onboarding entry — `/splash`, `/welcome` (Intro_1–5, Onb_Welcome)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| O1 | All five slides and the welcome card match in both themes. | — | — | match |
| O2 | Skip → slide 5; welcome-card placement; no AI disclaimer on the slides. | — | — | accepted (QUESTIONS #13–#15) |

## Open low items (not fixed)

X4 `--on-danger` token · H5 banner image fallback · H7 fresh-user ring track · C4 phase sheet header row / note style ·
T2 TTC calendar filter icon · P4 pregnancy-week source footer · M1 Me hub mode strip.
