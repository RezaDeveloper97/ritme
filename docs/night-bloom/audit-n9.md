# N9 design-fidelity audit — admin-web (B-N9-12)

The 12 admin artboards `docs/design/night-bloom/h-admin/nb{l,d}_Admin_*` (1440 px) against admin-web after B-N9-01 … B-N9-11,
light + dark, with **real data**: an own Go API (`ritme_n912` DB on the test MariaDB, migrations from the API start, port 8912),
6 admins (super, clinical support, two medical advisors, analyst, product manager), 70 users across five life modes with
logs, cycles, analytics events and Plus subscriptions, the metrics job (snapshot of the 70 users), 8 message rules (draft /
pending medical approval / approved / active, one A/B, all four levels and kinds), 2 challenges (one v2 program with tasks,
badge and Plus reward), one OKR with a live and a manual key result, one pending clinical-config proposal. admin-web dev
on :3912 (own `NEXT_DIST_DIR`). Screens: `docs/qa/bloom/B-N9-12/screens/`, boards: `…/art/`, side-by-sides for fixed
items: `…/side-by-side/`.

Data-driven differences (names, counts, seeded numbers, the few KPI tabs the API provides, empty alerts) are not deviations.
Boards are mock-ups: they draw the 7-item flat nav, a notification bell, a «نرگس کریمی» footer card; admin-web has the
full grouped nav, the theme / language switch and the user chip — accepted (superset, B-N9-01). Screens that are editors
in admin-web but a master/detail on the board (challenges, message editor, roles) are judged on content, not layout.
⌘K: the palette opens from the top bar field and with the keyboard shortcut, arrow keys move, Enter navigates, Esc closes;
the field shows the «⌘K» hint like the board. Users directory without a health grant: no health field anywhere (list and
overview tab show the «داده سلامت محدود» card with «درخواست دسترسی موقت») — matches the board's rule.

Severity: **high** = broken / misleading / unreadable; **med** = visible departure from the board, wrong language or digits,
a missing board section; **low** = polish. Totals: **high 0 · med 9 · low 11** — fixed: 8 med; **1 med open (backend / policy)**;
**11 low open**.

| # | Screen | Deviation | Sev | file | Resolution |
|---|---|---|---|---|---|
| S1 | Shell, ⌘K | Palette, nav, theme and language switch match in both themes; keyboard run OK. | — | — | match |
| O1 | Overview | «کاربران بر اساس حالت» bars were all turquoise; the board colours each mode (cycle rose, TTC amber, pregnancy violet, postpartum turquoise …). | med | `screens/dashboard/ui/OverviewPanels.tsx` | **Fixed** — per-mode token colour map. |
| O2 | Overview | OKR card shows a one-line summary + link; the board draws each key result with a progress bar and %. | low | `OverviewPanels.tsx` | Open — needs per-KR data in `/overview`. |
| O3 | Overview | KPI tiles lack the board's delta chips for MAU / stickiness / Plus (the snapshot has deltas only for DAU, signups); engine card has no confidence-distribution bar (suppressed under k≥10 by design). | low | — | Note — data / privacy rule. |
| U1 | Users | Board columns «حالت» (mode chip), «کیفیت داده», «هشدار موتور», KPI tiles above the table and 7 filter chips; the directory has status, plan, risk, signup, activity, 30-day logs. Mode, data quality and engine warning need an engine snapshot table (QUESTIONS #130). | med | `screens/users` (API) | **Open — for the orchestrator**: backend column set; frontend follows. |
| U2 | Users | Filters are labelled selects in a row below a search field instead of the board's pill dropdowns. | low | `screens/users/ui` | Open — polish. |
| U3 | User overview | Header prints the user's real name; the board header and the directory show only the pseudonymous «RT-…» id. The directory hides names, so the detail contradicts it. | med | `screens/users/ui`, API `users/:id/overview` | **Open — policy/backend**: decide whether a name is shown at all; not changed here. |
| U4 | User overview | Summary tab carries identity + health card only; the board's account / engine output / cycle chart / shown messages are on the other tabs (engine, events, messages). | low | — | Accepted — tabbed. |
| R1 | Roles / audit / health-access | Matrix with ●/○/— marks, role editor, audit log, health-access queue — match (editor is a superset). | — | — | match |
| M1 | Messages list | Level column was plain text and the four level / band cards were plain lists; the board uses coloured level chips (info turquoise, suggestion violet, worth-following amber, follow-soon rose) and tinted cards. | med | `screens/message-rules/ui/RulesListScreen.tsx`, `app/globals.css` (`.tone-card`) | **Fixed** — level badges + tinted level / band cards, light + dark. |
| M2 | Message editor | Version history printed `11-10-2026` (Latin ISO) in a Persian page. | med | `screens/message-rules/ui/RuleEditorScreen.tsx` | **Fixed** — `formatDate` (Jalali, Persian digits). |
| M3 | Message editor | Level is a `<select>` (board: four coloured chips with a dot); date inputs show Latin `2026-10-01` placeholders (they are ISO text inputs). | low | `RuleEditorScreen.tsx` | Open — polish. |
| E1 | Engine | Pipeline was a plain 2×4 list; the board has numbered step cards joined by arrows. | med | `screens/message-rules/ui/EngineScreen.tsx`, `globals.css` (`.step-chip`) | **Fixed** — numbered cards with RTL arrows. |
| E2 | Engine | Board's «قواعد ثابت» checklist (safety first, stable A/B hash, event log, versioning) was missing. | med | `EngineScreen.tsx`, `messages/{fa,en}.json` (`messageEngine.fixed*`) | **Fixed** — four verified rules (the two the engine does not implement yet are not claimed). |
| E3 | Engine | Board shows «جایگاه‌ها و ظرفیت» as a table beside the simulator; admin-web lists live rules per placement below it, with cap text. Simulator date is a native `datetime-local` (Latin, LTR). | low | `EngineScreen.tsx` | Open — layout polish. |
| C1 | Challenges | Run column printed `7 روز` with a Latin 7 next to Persian digits; category prints the raw key («nutrition»). | med | `screens/challenges/ui/ChallengesScreen.tsx` | **Fixed** (digits). Category label: low, open. |
| C2 | Challenges | Board is a card list + side editor; admin-web is table + full form (tasks, program, reward). | low | — | Accepted — functional editor. |
| K1 | KPI | The footers were English sentences from the API («Counts start with the first product analytics event …») in the Persian UI, punctuation flipped by RTL. Same for the OKR footer. | med | `screens/kpi/ui/KpiScreen.tsx`, `screens/okr/ui/OkrScreen.tsx`, `messages/{fa,en}.json` | **Fixed** — copy lives in the message files. |
| K2 | KPI | Funnel is horizontal bars; the board draws vertical gradient bars. Fewer tabs than the board (no revenue / telemedicine). Cohort / retention tab matches. | low | `KpiScreen.tsx` | Open — polish; tabs follow API data. |
| Q1 | OKR | Matches in structure; actions are text links (the board has none); period «1405-Q3» stays Latin (user-entered). | low | — | Open — polish. |
| F1 | Safety | The review queue listed every pending catalogue item (~110 rows with two stacked buttons each) in one unbounded page — the table dwarfed the six signal tiles; the board shows a short list. | med | `screens/safety/ui/SafetyScreen.tsx` | **Fixed** — ten rows per page + pager (resets with the source filter). |
| F2 | Safety | Review item sources print raw keys (`child_age_notes`) from the catalogue. | low | — | Open — data labels. |
| G1 | Clinical config | Value column shows field names in English («extra_days ۳ · min_days ۸», «before ۵ · after ۱»); proposals card and tabs match. | low | `screens/clinical-config/ui` | Open — needs per-key field labels. |
| X1 | B-N7-08 / B-N8-08 / N10 screens | Telemedicine, learning, shop, city services, partner lab: same table / panel chrome, badges and tabs in both themes. | — | — | match |

## Environment notes

- A Go API on :8912 against `ritme_n912`; admin-web dev on :3912, own `NEXT_DIST_DIR=.next-n912`; Redis DB 9 prefix `n912:`.
- Both the DB and the dist dir were removed at the end; the private GOCACHE was deleted.
