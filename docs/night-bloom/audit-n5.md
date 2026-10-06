# N5 design-fidelity audit (B-N5-10)

Every N5 screen side by side with its artboard (light + dark, 390 px, full page) after B-N5-04 … B-N5-08:
`docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nb{l,d}_v15_*|v16_*` and
`b1-cycle-log-analysis/nb{l,d}_Log_Feed|Log_Kick|Log_Contraction|An_Hub_Post` (rendered boards reused from
`docs/qa/bloom/B-N5-0{4..8}/artboards/`). Fresh app shots on ritme_dev (API :8020 at migration 33, restarted without
migrations; Next :3000). Personas: 09900005501 postpartum owner (children 1 نیلوفر / 2 کیان), 09900005502 spouse (child 1
shared, read-only), 09900000171 postpartum hub data, 09900000058 pregnant week 33, 09900000063 postpartum without
setup, plus 09120005403 (recent loss, canvas CB-LOSS) and 09120005404 (her partner, pregnancy grant «فقط دیدن»).

Data-driven differences (names «آوا/سام» → «نیلوفر/کیان», dates, counts, empty logs for 5501) are not deviations.
Deliberate defaults in `bloom/QUESTIONS.md` are **accepted** — #97 (EPDS-3 items, Q10 in the full check), #100 (mood
chips mapping, «زایمان کردم» from week 20, 6-week visit suggestion only), #102 (vaccine schedule, no percentile without
sex), #104 (5-1-1 rule), #105 (empty `tel:` until admin sets a number), #106 (hub computed client-side, Plus lock
UI-only). Boards draw a fake status bar and ↑ back arrow — accepted (QUESTIONS #30). Sticky footers on
`--surface-glass` — project-wide pattern, not listed per screen.

Severity: **high** = broken control / unreadable / misleading; **med** = visible departure from the board, a missing
or wrong control, the token rules; **low** = polish. Totals: **high 1 · med 5 · low 10** — fixed: high 1 + med 5
(incl. both canvas QA items); **10 low open**. Side-by-sides (board · before · after) for the fixed items only:
`docs/qa/bloom/B-N5-10/side-by-side/`; after-fix full pages in `docs/qa/bloom/B-N5-10/`.

## Postpartum — `/postpartum` (v15_Main), `/postpartum/setup`

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| P1 | Top bar, children strip with «+», today / week chips, «بهبودی» + duration display, delivery tag, «ثبت وضعیت امروز», mood card (4 tinted chips + note), bleeding / feeding / sleep tiles, upcoming visits with «افزودن مراجعه», «کی فوراً تماس بگیریم؟» with 115, week tip, postpartum nav: match in both themes. | — | — | match |
| P2 | Past day 42 the ring froze at «روز ۴۲ / از ۶ هفته» (seen at week 8 and week 15) — reads as if she were still on day 42. | med | `frontend/src/screens/postpartum/ui/PostpartumPage.tsx:242-254` | **Fixed** — after the puerperium the ring shows «هفته / ۶ / کامل شد» (new keys `home.ringWeeks`, `home.ringDone`); inside it unchanged. `side-by-side/postpartum-ring.png`. |
| P3 | Hero lochia note and the «شیردهی» tag show only while relevant (inside 6 weeks / feeds today); board draws both. | — | — | as designed (B-N5-04, QUESTIONS #100) |
| P4 | Setup (date calendar, delivery type, baby count stepper, privacy note, «فعال کن») and the setup-required home: no board — tokens, 44 px targets, both themes OK. | — | — | match (no board) |

## Recovery & mood — `/postpartum/recovery` (v15_Recovery), `/postpartum/mood?kind=short|full` (v15_MoodCheck)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| R1 | Lochia amount / colour, pain, breasts chips, save CTA: match. Pain location appears after a pain level is picked; the 6-week visit card only inside 6 weeks. | — | — | as designed (B-N5-04) |
| R2 | Feeds / sleep use the shared `NumberStepper` (− value + in a row) instead of the board's tinted input bar with small ± buttons. | low | `frontend/src/screens/postpartum-recovery/ui/PostpartumRecoveryPage.tsx:221,251` | Open — shared primitive; polish. |
| M1 | Mood check: privacy note card, numbered questions with radio rows, «دیدن نتیجه» / «الان نه», urgent safety screen with call buttons first: match. Items differ from the board. | — | — | accepted (QUESTIONS #97) |
| M2 | No step dots above the CTA (board draws 4 dots) — all items are on one page. | low | `frontend/src/screens/postpartum-mood/` | Open — polish. |

## Children — `/children` (v15_Children), `/children/new` + edit (v15_AddChild)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| C1 | Header «فرزندان من · n فرزند» with «+», child cards (avatar, name, sex pill, age, vaccine + growth chips), info note, «افزودن فرزند», «کودک» tab active; spouse list (shared card «از طرف …»): match in both themes. | — | — | match |
| C2 | «افزودن فرزند» label in brand violet (board: ink). | low | `frontend/src/screens/children/` | Open — polish. |
| C3 | Add child: photo disc (local only + privacy note), name, date picker, sex / delivery chips, weight / length / head, vaccine + reminder rows: match. Selected chip is a filled brand pill (board: outlined lavender); the two toggles are «فعال» status chips (always on). | low | `frontend/src/screens/child-add/` | Open — chip style is the shared `ChipGroup`; always-on reminders per B-N5-05. |

## Child home & sub-screens — `/children/[id]` (v16_ChildHome), growth, vaccines, milestones, learn

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| H1 | Profile header, age hero with avatar + chips, three measurement tiles with percentile chips, «ثبت اندازه جدید», next-vaccine card, 2×2 tiles, «این هفته {name}», «امروز» rows: match. | — | — | match |
| H2 | Spouse (read-only) saw «ثبت ›» on every empty «امروز» row — a write call-to-action she cannot use. | med | `frontend/src/features/baby-log/ui/BabyTodayList.tsx:19-39`, `frontend/src/screens/child-home/ui/ChildHomePage.tsx:362` | **Fixed** — `readOnly` prop (`!child.canEdit`): empty rows read «هنوز ثبت نشده» (`babyLog.today.none`); owners unchanged. `side-by-side/child-home-spouse.png`. |
| H3 | Dark: the board's hero sits on a violet gradient card and the girl avatar is light pink; app hero is on the canvas and the girl avatar uses the peach tone in dark. | low | `frontend/src/entities/child/ui/ChildAvatar.tsx` | Open — polish. |
| G1 | Growth: indicator tabs, latest value + «در بازه طبیعی», P3–P97 band + median + points, legend, history with percentiles, WHO note, CTA: match. | — | — | match |
| V1 | Vaccines: ring + «n نوبت کامل شده» + next, برنامه / کارت / یادداشت tabs, visit groups with checkboxes, «ثبت نوبت» / «تزریق شد», note: match. | — | — | match |
| V2 | Spouse sees «ثبت نوبت» (it creates an appointment in her own reminders, not on the child). | low | `frontend/src/screens/child-vaccines/ui/VaccinesPage.tsx:156` | Open — harmless; hide for read-only later if wanted. |
| S1 | Milestones: ticked items used brand violet checkboxes; the board ticks them in the warm tone of the progress ring. | med | `frontend/src/app/globals.css:10050-10055` | **Fixed** — scoped `.cms-item` checked box on `--warm` (tick `--surface`), both themes via tokens. `side-by-side/milestones.png`. |
| S2 | Month chips, ring «۵/۷», non-judgemental lead, play ideas, doctor note: match. | — | — | match |
| L1 | Learn: topic chips, weekly pick card, article rows, disclaimer: match. Health rows reuse the stethoscope icon (board: per-article icons). | low | `frontend/src/screens/child-learn/` | Open — icon comes from the catalog topic; polish. |

## Feeding — `/children/[id]/feeding` (Log_Feed)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| F1 | Header with info / ×, child picker, سینه / شیشه / پمپ tabs, L/R timer tiles (live tile tinted in data), «دفعه قبل» / «امروز» rows, sticky «پایان و ذخیره»; below the board: manual entry, baby sleep and diapers cards; spouse read-only banner: match in both themes. | — | — | match (sleep / diapers cards are B-N5-07 scope) |

## Postpartum analysis — `/analysis` for a postpartum user (An_Hub_Post)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| A1 | Week pill, EPDS chart with bands + note, lochia strip, feeding bars + L/R line, Plus-locked sleep, growth chips, weight trend, disclaimer: match in both themes (0171). Empty states for 5501 read well. | — | — | match |
| A2 | Feeding bars are one tone; the board darkens today's bar. | low | `frontend/src/screens/analysis-postpartum/` | Open — polish. |
| A3 | Postpartum nav has no «تحلیل» tab (board: امروز · تقویم · + · تحلیل · من); the hub is reached from «امروز»/services. | — | — | as designed (nav.md, B-N5-05 «کودک» tab) |

## Pregnancy tools — `/pregnancy/kicks` (Log_Kick), `/pregnancy/contractions` (Log_Contraction), «زایمان کردم»

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| T1 | A contraction left running (dev data: started 2 days earlier) ticked «۴۳:۲۴:۱۴» in the ring — the digits overflowed the circle, the table showed a 43-hour duration and the only way out was to end it as a real contraction. | high | `frontend/src/features/pregnancy-tools/model/timing.ts` (`isStaleContraction`), `frontend/src/features/pregnancy-tools/ui/ContractionTimer.tsx:43,88-104,128-134` | **Fixed** — a contraction running > 30 min or a session idle > 6 h counts as left open: the ring shows «شمارش قبلی باز مانده · — · اول آن را تمام کن» (disabled), a note says since when and to press «توقف و ذخیره», the running row shows «—». Unit test added. `side-by-side/contractions.png`. Backend still records the long duration on finish → follow-up: cap/auto-close stale contractions in `internal/pregnancy/tools` (outside this task's touches). |
| T2 | Contraction ring, averages, «۱ ساعت اخیر», start · duration · interval table, guidance, 5-1-1 card: match. | — | — | match |
| T3 | Kick counter: ring to 10, «یکی کم کن», start / elapsed tiles, reached card, history: match. A session left open for days shows elapsed «۴۳:۴۷:۵۱» (layout holds). | low | `frontend/src/features/pregnancy-tools/ui/KickCounter.tsx:130` | Open — needs the same backend auto-close as T1. |
| T4 | Pregnancy home week 33: «زایمان کردی؟» card with «زایمان کردم» (→ `/postpartum/setup`): match (no board). | — | — | match |

## Companion & spouse views

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| X1 | Companion home child card («فرزند» → shared child, age, next vaccine), spouse child list / home / growth / vaccines / feeding read-only (no write controls except V2): match in both themes. | — | — | match |
| X2 | **Canvas QA (CB-LOSS-03)** — shared rows said «بارداری · همین کارت بالا» (and «پریود و سیکل · همین کارت بالا») even when no card was drawn above (e.g. after a loss, pregnancy granted but no active pregnancy). | med | `frontend/src/screens/companion-home/ui/CompanionHomePage.tsx:219-232` | **Fixed** — «همین کارت بالا» only when `PartnerCard` renders that section (pregnancy active / cycle has data); otherwise «فعلاً چیزی برای نمایش نیست» (`shared.nothingYet`). `side-by-side/companion-hint.png`. |

## Canvas QA — `/pregnancy` after a loss

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| Q1 | Right after a pregnancy loss `/pregnancy` showed the big empty state «حالت بارداری روشن نیست» + «راه‌اندازی حالت بارداری» CTA. | med | `frontend/src/screens/pregnancy/ui/PregnancyPage.tsx:313-346` | **Fixed** — inside the loss-care window (`isLossCareOpen`, 60 days) the page shows one calm line «فعلاً اینجا چیزی برای نمایش نیست. هر وقت آماده بودی، ما همین‌جاییم.» on a brand-tinted disc, no CTA, plus the quiet «مراقبت از خودت» row (`LossCareReturn variant="row"`); `loss` namespace added to the `pregnancy` route scope. After the window the normal setup CTA returns. `side-by-side/pregnancy-after-loss.png`. |

## Cross-cutting

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| Z1 | Tokens only (no raw hex) in the touched slices; `lint:styles` / `lint:dark` green; touch targets ≥ 44 px; RTL logical properties. | — | — | match |
| Z2 | First pass showed Latin digits in B-N5-06 strings («5 مورد دیده شده», «ماه 3», «2 نوبت…», «3 ماهگی»). Cause: the dev API had been down and the web served a stale message bundle; after restarting the API the same screens render Persian digits (code passes `formatNumber`). | — | — | environment (no code change) |

## Open low items (not fixed)

N5: R2 recovery steppers · M2 mood step dots · C2 «افزودن فرزند» colour · C3 add-child chip style / toggles · H3 dark
child hero / girl avatar tone · V2 spouse «ثبت نوبت» · L1 learn icons · A2 today's feeding bar · T3 stale kick session
(+ backend auto-close follow-up for T1/T3).
Earlier milestones: see the open lists in `audit-n4.md` … `audit-n1.md`.

## Environment notes

- The Go API was down at the start of the window (last log the day before); restarted with `RUN_MIGRATIONS=false`
  (goose stays at 33) after each translation change. Next dev `.next-dev/prerender-manifest.json` was twice left with
  trailing bytes (two writers) → every page 500; repaired in place (the valid JSON prefix kept), no code involved.
