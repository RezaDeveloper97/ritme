# N2 design-fidelity audit (B-N2-10)

Side-by-side of every N2 screen (light + dark, 390 px, full page) against its artboard, after B-N2-02 … B-N2-09, plus the
leftovers from B-N1-16 and the N1 stage smoke. Screens and 4-up comparisons (artboard light · app light · artboard dark ·
app dark): `docs/qa/bloom/B-N2-10/` — `onboarding/` (fresh user 0990…96), `mode/` (cycle 0990…91, pregnancy 0990…93 for
the loss exit), `menopause/` (0990…81), `teen/` (0990…82), `plus/` (free 0990…06, subscriber 0990…91), `plus/trial/`
(trial started for 0990…07), `admin/` (admin-web, qa@ritme.local), `side-by-side/`. Rendered artboards are not committed —
regenerate with `node bloom/bin/shot.mjs --files docs/design/night-bloom/a-start-plus/nb?_{Onb,Prem}_*.dc.html
docs/design/night-bloom/g-me-settings/nb?_Me_Mode.dc.html`.

Data-driven differences (dates, prices, counts, unselected default state, empty dev data) are not deviations. Deliberate
defaults already in `bloom/QUESTIONS.md` are listed as **accepted (QUESTIONS #n)** and not changed. The dark artboards
paint unselected radios and calendar cells as opaque white discs (`nbd_Onb_Meno`, `nbd_Onb_Cycle`) and draw the back
arrow as ↑ — export artefacts, not design intent.

Severity: **high** = broken control / unreadable; **med** = visible departure from the board, a mode decision or the
token rules; **low** = polish. Totals (N2 screens): **high 0 · med 2 · low 6** found — fixed: med 2 / low 2; **4 low open**.
N1 leftovers: X4 `--on-danger` and H5 banner fallback **fixed**; the other 5 N1 lows stay open (listed at the end).

Shots reused (state cannot be reached by URL): `/otp` needs a pending mobile and `/onboarding/setting-up` a finished
flow → compared with `docs/qa/bloom/B-N2-02/fa_otp.*`, `fa_onboarding_setting-up.*`; the trial sheet is local widget
state (no `?sheet=` route) → `docs/qa/bloom/B-N2-08/trial-sheet.*`. Code for all three is unchanged since those tasks.

## Leftovers (B-N1-16, N1 stage smoke)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| L1 | (audit-n1 X4) White `--on-accent` / `--on-brand` labels on `--danger` fills — `#FF8FA3` at night, white text unreadable: delete confirm, track-pregnancy danger chip, mark-done error toast, calendar error toast, reminders danger button + badge, pregnancy-tile alert badge. | low | `frontend/src/app/globals.css:85,306` | **Fixed** — new `--on-danger` token (white → `#17112B`, flips like `--on-brand`) in both theme blocks; `lint:dark` pair `--on-danger`/`--danger` ≥ 4.5 (`frontend/scripts/check-dark-mode.mjs:309`). Applied in `globals.css:1439,4578,4602,5173,6158`, `features/track-pregnancy/ui/controls.tsx:72`, `screens/checkup-mark-done/ui/MarkDoneToast.tsx:28`. |
| L2 | (audit-n1 H5) Banner slot shows a broken-image box with alt text when the file is missing (still visible on dev data: «بنر بالای خانه» / «بنر میانی»). | low | `frontend/src/widgets/banner-slideshow/ui/BannerSlideshow.tsx:33-37,146` | **Fixed** — a slide whose image fails (`onError`) is dropped; a slot with no loadable slide renders nothing. |
| L3 | (n1-stage B-1) `/profile/notifications` PMS row always said «روز ۲۴ سیکل» (static copy) while `/cycle/settings` shows the API `cycle_day`. | low | `frontend/src/screens/notification-settings/ui/NotificationSettingsPage.tsx:55-65`, `…/api/settings.ts:83` | **Fixed** — reads the PMS `cycle_day` from `GET /profile/cycle-settings` (key under the sibling's `cycle-settings` prefix, so a save there repaints it) and reuses `me.cycleSettings.reminders.items.pms.sub/subUnknown`; no message or backend change. Re-shot: «روز ۲۷ سیکل» on both screens (0990…91). |

## Onboarding v2 — `/signup`, `/otp`, `/onboarding/*` (Onb_Phone … Onb_Ready)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| O1 | Name, gender, goal, cycle, pregnancy basis, menopause, conditions, health, partner stub, ready: layout, progress dashes, cards, chips, steppers, sticky footer + privacy line match in both themes. | — | — | match (shots show the unselected / disabled-CTA state; boards show a filled-in state) |
| O2 | OTP has 4 boxes (board 5) and the copy says «کد ۴ رقمی». | — | — | accepted (QUESTIONS #73) |
| O3 | Phone: consent ticks not stored; «قوانین»/«حریم خصوصی» plain text. | — | — | accepted (QUESTIONS #74) |
| O4 | Pregnancy basis «تاریخ زایمان» → LMP − 280; ready card dates from this cycle. | — | — | accepted (QUESTIONS #75) |
| O5 | Menopause yes/no chips are the `PillChip` primitive (≈56 px wide, board 44 px circles) so the question wraps one word earlier. | low | `frontend/src/screens/onboarding-flow/ui/MenopauseStep.tsx:38-43` | Open — primitive sizing, polish. |
| O6 | Ready: status value «پیگیری چرخه» vs board «پیگیری سیکل». | — | — | copy from the goal label (data) |

## Mode switcher — `/profile/mode`, `/profile/mode/loss` (Me_Mode)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| M1 | Six radio cards with tab/log chips, IVF/IUI and contraception switches, info note: match in both themes. | — | — | match |
| M2 | Loss exit has no artboard; calm card «کنارت هستیم» with bundled copy. | — | — | accepted (QUESTIONS #68–#69) |

## Menopause and teen homes — `/home` (no artboards)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| H1 | Teen home still painted fertile-window (amber) and ovulation (turquoise) dots on the cycle ring and the week strip, although the teen decision is «no fertility read-outs» (docs/night-bloom/README.md, gaps.md #3; rows/level were already hidden). | med | `frontend/src/screens/home/ui/HomePage.tsx:780-786` | **Fixed** — in teen mode `fertile`/`ovulation` markers resolve to none for the ring and the week strip. Before: `side-by-side/teen.before-fix.png`; after: `teen/fa_home.*`. |
| H2 | Teen predictions card keeps the «دوره PMS» row; Me hub hides the Plus card. | — | — | as designed (README decision: only fertility rows + PMS insight removed) |
| H3 | Menopause: layout per the B-N2-03 decision; challenge card «روز ۱۸ چرخه»; «علائم» tab → `/cycle/symptoms`. | — | — | accepted (QUESTIONS #4, #70) |
| H4 | No trial banner for teen (UI only). | — | — | accepted (QUESTIONS #72) |

## Plus — `/plus`, `/plus/plans`, `/plus/checkout`, `/plus/success`, `/plus/manage` (Prem_*)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| P1 | Manage: the «روز مانده» ring sat at the end of the status card (left in RTL); Prem_Manage draws it at the start (right) with the plan text after it. | med | `frontend/src/app/globals.css:6805` | **Fixed** — `.plus-status-card > .plus-ring { order: -1 }`. Before: `side-by-side/manage.before-fix.png`; after: `plus/fa_plus_manage.*`. |
| P2 | Manage list rows are 64 px (`min-height`), the board's rows ≈ 53 px, so the card is taller. | low | `frontend/src/app/globals.css:6809` | Open — polish (64 px kept deliberately by B-N2-07 for the switch row). |
| P3 | Paywall: «۷ روز رایگان برای اولین اشتراک» is a bold brand link (it starts the trial); the board draws a muted `--text-2` caption. | low | `frontend/src/screens/plus-paywall/ui/PaywallPage.tsx` (trial line) | Open — it is the only trial entry on the paywall; restyling it as plain text would hide that it is tappable. |
| P4 | Paywall testimonial removed; checkout hides Bazaar/Myket; success drops «رسید به پیامکت فرستاده شد»; plans bullet reads «۷ روز رایگان برای اولین اشتراک». | — | — | accepted (QUESTIONS #77) |
| P5 | Success: «گزارش برای پزشک» / «دستیار سلامت» rows carry «به‌زودی» (features not built). | — | — | data (N6/N7) |
| P6 | Plans, checkout (bank gateway selected, VAT, total), success: match in both themes. | — | — | match |

## Trial banner and sheet — `/home` (Prem_TrialHome, Prem_TrialSheet)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| T1 | Banner above the nav (crown, countdown boxes, «۵۰٪ تخفیف») matches; amber text uses `--warm-deep` in light. | — | — | match / accepted (QUESTIONS #11) |
| T2 | Sheet «مقایسه کامل رایگان و پلاس» link painted `--brand-strong` (near-white `#D6CBFF` at night); board: brand lavender. | low | `frontend/src/app/globals.css:6912` | **Fixed** — `--brand` (`#6E54F0` / `#B9A6FF`). |
| T3 | Banner reads `/plus/status`; voice-log usage row hidden; «تخفیف ویزیت» row `danger` tone. | — | — | accepted (QUESTIONS #78) |

## Admin billing — admin-web `/plus/subscriptions|payments|plans|discount-codes|settings` (no artboards)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| A1 | Pages follow the current admin shell (sidebar group «اشتراک‌ها و پرداخت», cards, tables, status pills) in both themes; masked mobiles. | — | — | match (shell) / accepted (QUESTIONS #76) |
| A2 | Date-range filters on subscriptions/payments are native `type=date` inputs showing «mm/dd/yyyy» Gregorian placeholders in the Persian panel. | low | `admin-web/src/app/(panel)/plus/subscriptions`, `…/payments` (filter bar) | Open — admin-only; a Jalali picker belongs to the admin v2 theme (B-N9-01). |

## Cross-cutting (as N1)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| X1 | Boards draw a 54 px fake status bar; ScreenHeader back is an arrow. | — | — | accepted (QUESTIONS #30) |

## Open low items (not fixed)

N2: O5 menopause yes/no chip size · P2 manage row height · P3 paywall trial line style · A2 admin date filters.
N1 (still open from audit-n1): H7 fresh-user ring track · C4 phase sheet header row / note style · T2 TTC calendar
filter icon · P4 pregnancy-week source footer · M1 Me hub mode strip.

## Environment notes

- The local API was restarted by the concurrent canvas session during the run (goose 00020 applied); admin-web dev must
  proxy to `http://127.0.0.1:8020` — `localhost` resolves to `::1` and the Go API listens on IPv4 only (ECONNREFUSED →
  admin login 500 in `shot.mjs --admin`).
- A 7-day Plus trial was started for dev user 0990…07 to render the banner (dev data only).
