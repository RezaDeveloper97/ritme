# N6 design-fidelity audit (B-N6-09)

Every N6 screen side by side with its artboard (light + dark, 390 px) after B-N6-01 … B-N6-08:
`docs/design/night-bloom/c-health-record/nb{l,d}_Vitals_*|Record_*|Lab_*`, `f-tools/nb{l,d}_Todo_*`,
`b1-cycle-log-analysis/nb{l,d}_An_Labs` and `g-me-settings/nbl_Me_Privacy` (privacy share-links row). Rendered boards
reused from `docs/qa/bloom/B-N6-0{2,3,4,7,8}/artboards/`; the missing ones (`nbd_Vitals_Add*`,
`nbd_Vitals_GlucoseReport`, `nb{l,d}_An_Labs`, `nbl_Me_Privacy`) rendered locally for the comparison only. Fresh app
shots on ritme_dev (API :8020 at goose 43, Next :3000, neither restarted). Persona 09900000004 (Plus, regular cycle:
52 vital readings, 3 ready labs). To-do items, shopping items and two share links (one revoked, for the 410 page) were
created through the API for the shots; the urgent alert sheet was opened by saving a 185/125 reading (deleted
afterwards); the processing screen by a temporary `extracting` lab row + far-future queued job (deleted afterwards).

Data-driven differences (names, dates, counts, which sections are empty, no period-supplies suggestion for 04) are not
deviations. Deliberate defaults in `bloom/QUESTIONS.md` are **accepted** — #109 (log-sheet values merged read-only),
#111 (lab routes, «به‌زودی» actions, server consent text, no «دفعه بعد نپرس»), #112 (HR report built like the BP report,
form defaults), #113 (record losses as a count, meds edit → `/reminders`), #114 (preview/PDF free, «یادداشت‌های روزانه»
→ «بارداری و زایمان»), #115 (× on the suggestion card, teen entry). Share buttons on vitals / lab / record headers and
the PWA banner on `/shared` belong to **B-N6-04b** — not listed. Boards draw a fake status bar and ↑ back arrow —
accepted (QUESTIONS #30). Sticky footers on `--surface-glass`, the bottom nav on `/analysis/*` sub-pages (project-wide
since B-N3) and `/labs` without a nav (`Lab_Intro` is listed as a deviating board in `nav.md`) — not listed per screen.

Severity: **high** = broken control / unreadable / misleading; **med** = visible departure from the board, a missing
or wrong control, the token / digit rules; **low** = polish. Totals: **high 0 · med 7 · low 17** — fixed: all 7 med;
**17 low open**; 1 environment note. Side-by-sides for the fixed items (board · before · after light [· after dark]):
`docs/qa/bloom/B-N6-09/side-by-side/`; after-fix pages in `docs/qa/bloom/B-N6-09/`.

Note on method: `shot.mjs` full-page captures grow the viewport, so on screens with a sticky footer the last block of a
scroller looks hidden behind the footer. Every such case was re-checked in a 390×844 viewport scrolled to the end; only
V1 was real.

## Vitals — `/vitals` (Vitals_Hub), `/vitals/{bp,glucose,heart-rate}/new` (Vitals_Add*), reports, urgent alert sheet

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| V0 | Hub header + record button, latest BP / HR / glucose cards with class pill and «when», weekly plan dots + «ویرایش برنامه», recent readings, the three coloured add buttons, nav «خدمات»; add forms (three value tiles / one big value with − +, class meaning card, condition chips, time, note, sticky «ذخیره»); BP / glucose reports (range + filter tabs, average, LTR chart with normal / target band, min · max · pulse tiles, distribution, morning vs night, night note, all readings with delete); HR report (no board); urgent alert sheet (danger card, call 115 first, «دوباره اندازه می‌گیرم»): match in both themes. | — | — | match |
| V1 | Hub scrolled to the end: the last «ثبت‌های اخیر» row and the card's bottom edge stayed under the sticky add bar. The hub's own 72px reserve lost to the nav reserve `.view:has(> .nbnav) > .scroll::after` (higher specificity), which leaves no room for the bar. | med | `frontend/src/app/globals.css:10368`, `:4464` | **Fixed** — B-N6-09 block reserves nav + bar + 12px on the hub (`.view.vt-hub-screen:has(> .nbnav) > .scroll::after`); card now ends above the bar. `side-by-side/vitals-hub-bottom.png`. |
| V2 | Glucose values in mixed units on one screen: a reading typed in mmol/L showed «۵٫۲ mmol/L» among mg/dL rows in the report list and the hub list, and the min / max tiles used each reading's own unit while the average used the report's. | med | `frontend/src/entities/vital/ui/useVitalFormat.ts:36-60`, `VitalBits.tsx:40-75`, `screens/vitals-report/ui/VitalsReportPage.tsx:125,410-411,487`, `screens/vitals-hub/ui/VitalsHubPage.tsx:77,201` | **Fixed** — `value / unit / title` and `ReadingRow` take an optional display unit (from the API's `mg_dl` / `mmol_l`); the report passes its unit to the list and tiles, the hub passes the latest glucose card's unit to «ثبت‌های اخیر». `side-by-side/glucose-units.png`. |
| V3 | Add BP «این عدد یعنی» scale mirrored in RTL (مرحله ۲ … طبیعی from left to right); the board draws it LTR (طبیعی on the left → مرحله ۲ on the right), like every other chart of the slice. | med | `frontend/src/features/add-vital/ui/AddVitalForm.tsx:261`, `globals.css:10479` | **Fixed** — `.vt-scale { direction: ltr }` (labels and marker follow). `side-by-side/bp-scale-digits.png`. |
| V4 | Morning vs night with no reading in one slot printed «ثبتی نیست» in the 26px display number font (HR report, «صبح»), reading like a value. | med | `frontend/src/screens/vitals-report/ui/VitalsReportPage.tsx:229-250` | **Fixed** — empty slot is a muted 14px line (`.vt-mvn-empty`). `side-by-side/hr-morning-night.png`. |
| V5 | Typing on a Latin keyboard left «185» / «125» in Latin digits in the big value fields (the field only re-localised on − / +). | med | `frontend/src/features/add-vital/ui/ValueInput.tsx:101` | **Fixed** — on blur the field shows the parsed value in the locale's digits («۱۸۵»). `side-by-side/bp-scale-digits.png`. |
| V6 | Class pills are filled tints; the board outlines them (1.5px border, light fill) on the hub, reports and rows. | low | `frontend/src/entities/vital/ui/VitalBits.tsx:20-33` | Open — shared `StatusPill` look; polish. |
| V7 | Night / caution note is the yellow warn tint; the board uses a soft warm gradient card (also `Lab_Verify`'s low-confidence chip). | low | `frontend/src/screens/vitals-report/ui/VitalsReportPage.tsx:245` | Open — shared `InfoNote` warning tone. |
| V8 | Glucose / HR add: − + sit at the card edges; the board keeps them tight around the number. | low | `frontend/src/features/add-vital/ui/ValueInput.tsx` (`is-row`) | Open — polish. |
| V9 | HR icon is a plain heart; the board uses a heart with a pulse line. | low | `frontend/src/entities/vital/ui/VitalBits.tsx` (`VITAL_LOOK.hr`) | Open — no heart-pulse glyph in `Icon` yet. |
| V10 | Dark: the «بالا» (elevated) segment of the distribution bar / BP scale is a brown mix; the board uses a lighter amber. | low | `frontend/src/app/globals.css:10482` | Open — token mix; polish. |
| V11 | Selected condition chips are filled brand pills; the board outlines them in lavender (same as audit-n5 C3). | low | shared `ChipGroup` | Open — cross-cutting. |

## Health record — `/record` (Record_Summary) + edit sheets

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| R0 | Header, person card, basics tiles, conditions chips, meds, allergies, 6-month cycle summary, 30-day vitals, pregnancies, checkups & labs, disclaimer, «ساخت گزارش برای پزشک»; edit sheets (blood type chips, height / weight rows, condition chips, allergy add + «ندارم» toggle, manual pregnancies): match in both themes (sheets have no board). | — | — | match |
| R1 | Person card: gradient disc with an ink initial; the board draws a light pink disc with a pink ring and pink initial. Without a profile name the card repeats «پرونده سلامت من» as the name. | low | `frontend/src/screens/health-record/` | Open — polish; name comes from the profile. |
| R2 | Basics sheet: «ویرایش وزن» uses the shopping-bag icon. | low | `frontend/src/screens/health-record/` (basics sheet) | Open — needs a scale glyph. |

## Doctor report — `/record/export` (Record_Export), `/record/export/preview` (Record_Preview), `/{locale}/shared/report/{token}`, privacy links

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| E0 | Preview card, range chips (incl. «سفارشی»), section toggles, patient question, privacy note, «دانلود PDF» + «اشتراک لینک»; preview page (report sheet with summary tiles, sections, vitals table, disclaimer, «ارسال»); public report (read-only header, «تا … معتبر است», sheet, «دانلود PDF») and the revoked / 410 state (lock disc, «این لینک دیگر معتبر نیست»): match in both themes (shared page has no board). | — | — | match |
| E1 | Export toggle rows have no dividers; «اشتراک لینک» is brand text with a share glyph (board: ink text, upload glyph). | low | `frontend/src/screens/record-export/` | Open — polish. |
| E2 | Preview / shared sheet use 14px table text; the board's sheet is page-scale (~11px), so ranges wrap («۱۲۴/۸۰ (۱۱۵/۷۳ – / ۱۳۶/۹۰)»). | low | `frontend/src/screens/record-export/` | Open — the PDF itself is laid out separately. |
| P1 | Privacy «پزشکانی که گزارش دیده‌اند» said «هنوز لینک گزارشی نساخته‌ای» + «به‌زودی» while an active link was listed right below (board: «۱ لینک فعال · ۴ روز مانده» with a chevron). | med | `frontend/src/screens/privacy/ui/PrivacyPage.tsx:48-75,339` | **Fixed** — `DoctorsRow` reads the share links: «فعال تا ۲۱ مهر» (latest active expiry) and opens the links list; once all ended it shows the latest state («لغو شده» / «منقضی شده»); without links it keeps the empty line and opens `/record/export`. Existing `recordExport.links` keys, no new copy. `side-by-side/privacy-doctors.png`. |
| P2 | An active link row wraps «ساخته‌شده ۱۴ مهر ۱۴۰۵» onto two lines next to its two pills. | low | `frontend/src/screens/privacy/ui/ShareLinksSection.tsx:55-75` | Open — polish. |

## Labs — `/labs` (Lab_Intro), consent sheet (Lab_Consent), `/labs/new` (Lab_Upload), processing, verify, result, marker

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| L0 | Intro hero + topic chips, previous analyses with state pills, disclaimer, sticky upload CTA (Plus badge when locked); consent sheet (consented and first-time states); upload (camera / gallery / PDF, pages grid, type chips, date / fasting, tips, disabled «تحلیل کن»); processing ring + 4 steps + «برو، خبرت می‌کنیم» and the failed state; verify (low-confidence chip, value boxes, edited row in warm, «افزودن شاخص جاافتاده»); result (score ring, attention / normal groups with LTR range bars, doctor questions, «به‌زودی» actions, feedback, disclaimer, edit / delete); marker (value card + range bar, trend chart, what / factors / when-to-see-doctor): match in both themes. | — | — | match |
| L1 | Intro list header has no «همه» link (every analysis is listed). | low | `frontend/src/screens/lab-intro/` | Open — polish. |
| L2 | Consent sheet title is the small sheet title; the board shows a display-size «قبل از شروع» with the shield. | low | `frontend/src/screens/lab-consent/` | Open — shared sheet header. |
| L3 | Upload: no lab type preselected (board preselects «خون»). | low | `frontend/src/screens/lab-upload/` | Open — deliberate explicit choice; polish. |
| L4 | Marker trend chart labels only the band edges (۱۵۰ / ۱۵); the board also labels ۱۰۰ / ۵۰. | low | `frontend/src/screens/lab-marker/` | Open — polish. |

## Lab trends — `/analysis/labs` (An_Labs)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| A1 | Each marker was a stacked block (name / value / pill line, a full-width 56px chart, then the server sentence — for 3 of 5 markers the same «روند وقتی نمایش داده می‌شود…» repeated); the board is one compact row per marker: name + «مرجع ۱۵–۱۵۰» · sparkline · value + state pill. | med | `frontend/src/screens/analysis-labs/ui/AnalysisLabsPage.tsx:86-125`, `globals.css` B-N6-09 block | **Fixed** — grid row (name + `labs.result.reference` · 120px sparkline with non-scaling stroke · value in the state tone + pill); the per-row sentence dropped (the 2-lab rule stays in the page note, the direction sentence on the marker page). New route scope `analysisLabs` (= analysis + `labs`) in `message-scopes.ts`. `side-by-side/analysis-labs.png`. |

## To-do — `/todo` (Todo_Home), `/todo/lists/[id]` (Todo_List), `?sheet=todo-add` (Todo_Add), empty (Todo_Empty)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| T0 | Hub header + progress ring, category chips with dots, period-supplies suggestion (B-N6-08 shots), today / tomorrow / later groups with category-tinted ticks, «کار جدید», nav «من»; shopping list (header + count, items, «افزودن قلم», done items); add sheet (title, note, date / time / category / reminder chips, round submit); empty states (fresh, filtered): match in both themes. `Todo_Empty`'s board overlaps its CTA with the disc — app follows the intent. | — | — | match |
| T1 | Ring read «1/5» in Latin digits. Cause: the running API serves a stale message bundle — `/languages/fa/messages` returns `todo.progressShort = "{done}/{total}"` while `frontend/messages/fa/todo.json` and the Go translation copy say `"{done, number}/{total, number}"` (also 7 missing `vitals.weekdays.*` keys). The API binary predates the B-N6-08 copy change; after the next `dev-up.sh api` / deploy the ring renders «۱/۵». | — | `frontend/src/screens/todo/ui/TodoPage.tsx:106` | environment (no code change; API not restarted — shared server) |
| T2 | Shopping list: done items sit in a collapsible «انجام‌شده» card and every row shows a × (board: flat list, strike-through, no ×). | low | `frontend/src/screens/todo-list/` | Open — B-N6-08 choice; polish. |
| T3 | Add sheet: large gap between the title field and «یادداشت». | low | `frontend/src/screens/todo-add/` | Open — polish. |

## Cross-cutting

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| Z1 | Tokens only in the touched files; `lint:styles` / `lint:dark` green; touch targets ≥ 44px; RTL logical properties (the two LTR blocks — BP scale, values — are numeric scales, as on the boards). | — | — | match |

## Open low items (not fixed)

N6: V6 pill outline · V7 warn note tone · V8 glucose / HR stepper spacing · V9 heart-pulse icon · V10 dark elevated
segment · V11 chip style · R1 person avatar / name fallback · R2 weight icon · E1 export toggles / share button · E2
preview table scale · P2 link row wrap · L1 «همه» link · L2 consent title · L3 lab type preselect · L4 marker axis labels
· T2 shopping done card / × · T3 add-sheet gap.
Earlier milestones: see the open lists in `audit-n5.md` … `audit-n1.md`.

## Follow-ups (outside this task)

- Restart the local API (and redeploy stage) so the served message bundle matches the translation files (T1).
- Share buttons on vitals / lab reports and the PWA banner on `/shared` → B-N6-04b (already queued).
- No backend change was needed; nothing in `internal/{healthrecord,sharelinks,pregnancy}` touched.
