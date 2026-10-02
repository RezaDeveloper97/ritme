# N3 design-fidelity audit (B-N3-13)

Side-by-side of every N3 screen (light + dark, 390 px, full page) against its artboard
(`docs/design/night-bloom/b1-cycle-log-analysis/`), after B-N3-03 … B-N3-12. App shots: `docs/qa/bloom/B-N3-13/` —
`p04/` regular cycle, Plus (sheet, panels, voice, `/log`, `/log/customize`, cycle hub, every `/analysis/*` detail,
monthly, plus `en_*` LTR checks), `p06/` free (hub, correlations lock, symptoms, monthly, voice lock), `p12/` TTC
(hub, fertility, sheet), `p121/` pregnancy 0990…121 (`/pregnancy/log`, pregnancy sheet, hub, weight), `p063/`
postpartum 0990…63 (sheet, hub), `p081/` menopause 0990…81 (hub, symptoms, sheet), `p082/` teen 0990…82 (hub,
correlations, sheet). Side-by-side composites (artboard light · app light · artboard dark · app dark) are kept only for
the med items: `side-by-side/`. The rendered artboards used for comparison are the ones already committed by the screen
tasks (`docs/qa/bloom/B-N3-03 … B-N3-12/artboards/`); none were re-rendered into this folder.

Data-driven differences (dates, counts, values, empty or «not enough data» states of the dev DB, unselected default
state) are not deviations. Deliberate defaults already in `bloom/QUESTIONS.md` are listed as **accepted (QUESTIONS #n)**
and not changed. Boards draw a 54 px fake status bar and an ↑ back arrow (export artefact) — accepted (QUESTIONS #30).

Severity: **high** = broken control / unreadable; **med** = visible departure from the board, a missing control or the
token rules; **low** = polish. Totals (N3 screens): **high 0 · med 2 · low 6** found — fixed: med 2; **6 low open**.
No N1/N2 open low was trivial enough to fold in (list at the end, unchanged).

## Log sheet v2 — `?sheet=log` (Log_Sheet_Cycle, Log_Sheet_Preg, Log_Sheet_Post), `/log` (Log_Day)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| L1 | The sheet header had only the × — the customise gear the board draws opposite it (and the `/log` page header has) was a blank spacer, so the sheet gave no way to `/log/customize`. Same on the TTC, pregnancy (`/pregnancy?sheet=log`) and postpartum sheets. | med | `frontend/src/features/log-day/ui/LogDay.tsx:477-490`, `frontend/src/app/globals.css:7512` | **Fixed** — `LogDaySheetTitle` renders the title and the gear (`CustomizeButton`, 44 px) in one row; CSS `globals.css:8218-8220` drops the spacer padding so the title stays centred between × and gear. Before: `side-by-side/log-sheet.before-fix.png`; after: `side-by-side/log-sheet.png`, `p063/fa_home_sheet_log.*`. |
| L2 | Date strip, manual/voice tabs, quick tiles, accordion, summary footer, save pill: match in both themes. A search field above «همه موارد» is extra (Log_Taxonomy / panel boards draw it). | — | — | match |
| L3 | Pregnancy/postpartum sheets keep the date strip + manual/voice tabs and the «ثبت روزانه بارداری» row; postpartum tile labels «پریود»/«حال» (board «خونریزی»/«حال روحی»); kicks/contractions/baby rows «به‌زودی». | — | — | accepted (QUESTIONS #90, #84) |
| L4 | Pregnancy/postpartum footer pill reads «ذخیره»; Log_Sheet_Preg / _Post draw «ثبت کامل روز». | low | `frontend/src/features/log-day/ui/SummaryFooter.tsx` | Open — copy polish; the shared footer is the same save for every mode. |
| L5 | `/log` page: Cycle_Log (N1 board) is superseded by Log_Sheet_Cycle — the page renders the v2 sheet content with gear + ×. | — | — | as designed (B-N3-03) |
| L6 | Voice tab (Log_Day): idle state shown here; recording / review / denied / locked states were compared in `docs/qa/bloom/B-N3-05/` (code unchanged since). Plus badge on the tab only for free users (`p06/fa_log.voice.*`). | — | — | match |

## Detail panels — bleeding, pain + body map, measurements (Log_Bleeding, Log_Pain, Log_Measure)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| D1 | Body map: 34 px pins sat too close — the «شکم» label touched the «لگن» pin and the wide «یک‌طرفه پایین» label ran into the pelvis pin (board: 28 px pins, labels clear of each other). | med | `frontend/src/app/globals.css:7755-7772` | **Fixed** — 30 px pins, chest/abdomen/pelvis/ovary re-spaced to the board's proportions (`globals.css:8222-8227`); targets stay 48 px; stitches/back/leg/joints unchanged. Before: `side-by-side/log-pain.before-fix.png`; after: `side-by-side/log-pain.png`. |
| D2 | Panel headers have no gear (boards draw one). | low | `frontend/src/features/log-day/ui/panels/DetailPanel.tsx:35-40` | Open — leaving a panel for `/log/customize` would drop the unsaved draft (QUESTIONS #84); the gear is on the sheet underneath. |
| D3 | Bleeding (intensity, colour, clots, smell, warning), measure (steppers, LH / pregnancy test) and pain intensity + relief cards: match in both themes. Panels rise over the sheet (sheet visible above), boards draw them full height. | — | — | match |

## Log customisation — `/log/customize` (Log_Customize)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| C1 | Rows (handle, icon, label, pin, switch), «ثبت سریع: ۸ از ۸» counter, custom-item row and add button: match in both themes (and LTR). | — | — | match |

## Analysis hub — `/analysis` (An_Hub, An_Hub_TTC, An_Hub_Preg; cycle / teen / menopause / TTC / pregnancy / postpartum)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| H1 | Cycle hub (Plus `p04`, free `p06`): range tabs, chips, top finding, 8 cards, Plus badges / blurred locks: match. | — | — | match |
| H2 | Teen hides locked Plus cards; postpartum sees the cycle hub (An_Hub_Post waits for B-N5-07); menopause/teen history shows period days only. | — | — | accepted (QUESTIONS #83) |
| H3 | TTC hub: trying ring, BBT, LH, Plus-locked timing/mucus/luteal, regularity: match. Board's bottom nav shows an «تحلیل» tab; the app keeps the mode nav (باروری / خدمات). | — | — | match / accepted (nav.md, QUESTIONS #93) |
| H4 | Pregnancy hub: weight gain, BP with ACOG threshold, glucose (Plus), kicks, trimester symptoms, visits: match. The current trimester tile carries a brand ring (board: all three plain). | low | `frontend/src/screens/analysis-pregnancy/ui/AnalysisPregnancyHub.tsx:224` | Open — kept as an orientation cue; polish. |

## Analysis details — `/analysis/{cycle,period,symptoms,correlations,body,labs,fertility,pregnancy-weight}`, `/analysis/monthly/[ym]`

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| A1 | Cycle, period, symptoms, correlations, body, fertility, pregnancy weight: stat tiles, charts (LTR time axes), footnotes, range tabs, locks: match in both themes. | — | — | match (QUESTIONS #56 LTR axes) |
| A2 | Every detail board draws a share/export button at the header's end; the app header has only back. | low | `frontend/src/widgets/charts/ui/ReportFrame.tsx:76` | Open — no share flow yet (doctor PDF / share link is B-N6-04); same decision as monthly (QUESTIONS #88). |
| A3 | Phase bar adds coloured legend dots before each phase name (board: text only). | low | `frontend/src/widgets/charts/ui/PhaseBar.tsx` | Open — helps colour-blind reading; polish. |
| A4 | Pregnancy-weight IOM table rows ≈ 54 px vs board ≈ 45 px. | low | `frontend/src/app/globals.css:8204-8206` | Open — polish. |
| A5 | Monthly: month stepper, headline, metrics table with coloured deltas, top symptoms, suggestion: match; PDF button «به‌زودی» / Plus lock, share omitted. | — | — | accepted (QUESTIONS #88) |
| A6 | Labs: empty state until B-N6-06 (board shows five markers). | — | — | accepted (QUESTIONS #88) |
| A7 | Correlations for teen: lock card «ارتباط‌ها در ریتمی پلاس» without an upsell button. | — | — | as designed (B-N3-09: teen no upsell) |
| A8 | Moods still listed in «علائم هم‌زمان با پریود» and top chips. | — | — | accepted (QUESTIONS #86) |

## Cross-cutting

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| X1 | Tokens only, light + dark on every N3 screen; LTR (`en_*`) checked for hub, cycle detail, monthly, sheet, customise — no mirroring defects. | — | — | match |
| X2 | Fake status bar / ↑ back arrow on boards. | — | — | accepted (QUESTIONS #30) |

## Open low items (not fixed)

N3: L4 pregnancy/postpartum footer copy · D2 panel gear · H4 current-trimester ring · A2 share button on details ·
A3 phase legend dots · A4 IOM table row height.
N2 (still open from audit-n2): O5 menopause yes/no chip size · P2 manage row height · P3 paywall trial line style ·
A2 admin date filters.
N1 (still open from audit-n1): H7 fresh-user ring track · C4 phase sheet header row / note style · T2 TTC calendar
filter icon · P4 pregnancy-week source footer · M1 Me hub mode strip.

## Environment notes

- Shots taken against the shared local stack (Go API :8020 on `ritme_dev`, Next dev :3000) while B-N4-02 and a canvas
  session were running; no restarts were needed. Tokens were fetched once per persona (OTP rate limit 5/min).
- The menopause sheet preset (`MenopausePreset`, canvas CB-MENO-06) is outside N3 and was not audited here.
