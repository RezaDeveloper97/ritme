# Canvas-v1 → bloom → code map (CB-CORE-01)

Shared reference for every `CB-*` task in `roadmap/`. Source: the «ریتمی — یائسگی» canvas snapshot in
`docs/design/canvas-v1/` (boards + text digests), the Night & Bloom audit in `docs/night-bloom/` (B-N1-01) and the bloom
task files in `bloom/N*/`. `roadmap/DECISIONS.md` is binding and wins every conflict below. Board content (names,
prices, clinics, dates) is design data only.

Contents: §0 how to read · §1 IA rules (IA_Map / IA_Nav) · §2 board map · §3 primitives vs B-N1-03 · §4 token map ·
§5 gaps (no design / no backend) · §6 conflicts · §7 CB task edits made by CB-CORE-01 · §8 open questions.

## 0. How to read / facts every CB task needs

- **Snapshot coverage.** `canvas.json` lists 154 boards; the snapshot has 93 files. The 61 missing are the `nbd_` dark
  twins of every light board (Cond, Contra, Dir, IVF, Ins, Loss, Meno, Pelvic, Priv, Rec, Teen, Voice, Wear). Nav and
  Shop exist **only dark** (`nbd_`), everything else only light in the snapshot, `Main.dc.html` = dark menopause home.
  So light↔dark pairs cannot be zipped from the snapshot — dark values come from the Night & Bloom tokens (§4). QA tasks
  compare the dark screenshot against the tokens, not against a dark board render (see §8 Q1).
- **Same design language as bloom.** Both canvases use the same palette, radii, Lalezar numerals, 44px round header
  buttons, flat 24px cards, solid `--brand-fill` CTA/FAB and the floating glass nav — the `nbl_`/`nbd_` language of
  `docs/night-bloom/components.md`. No new palette; see §4.
- **Routes.** Today routes live directly under `frontend/src/app/[locale]/` — there is **no `(app)` group** yet. CB task
  `touches` say `frontend/src/app/[locale]/(app)/…`; that prefix is provisional: use whatever B-N1-04 decides (it may add
  the group). Me is bloom's `/profile` hub (B-N1-10) → Me sub-routes are `/profile/…` (never `/me/…`).
- **Existing code this queue builds next to** (web + Go): `entities/{cycle,health-log,care-reminder,checkup,fertility,
  pregnancy,reminder,user,message,banner,article}`, `backend-go/internal/{care,checkups,cycle,fertility,healthlog,
  pregnancy,profile,messages,notify,content,home}`, routes `/home /calendar /cycle /log /checkups /reminders /fertility
  /pregnancy /profile`. There is no services / record / labs / vitals / companion / shop / city-services code yet — all
  of that arrives with bloom (N1–N10) first.
- Status legend in §2: **B-full** = the bloom task builds this board's screen (CB only restyles/extends) · **B-part** =
  bloom builds a base / entry / data the CB task extends · **exist** = shipped code (M3/M4/M5/M7) reused · **NEW** =
  nothing exists or is planned · **drop** = not built (DECISIONS #5).

## 1. IA rules from IA_Map / IA_Nav (rendered to `docs/qa/canvas/boards/IA_*.png`)

| Rule (board) | Bloom today | Owner |
|---|---|---|
| 5 slots: امروز · تب مرحله · + · خدمات · من (was امروز·تقویم·+·تحلیل·من) | Same (B-N1-04, `docs/night-bloom/nav.md`) | B-N1-04 ✔ |
| 7 modes: cycle→تقویم, ttc→باروری, **IVF (sub-mode)→درمان**, pregnancy→بارداری, postpartum→کودک, menopause→علائم, teen→تقویم | 6 modes; IVF only an `ivf_iui` flag (no own tab); menopause tab → `/analysis/symptoms` | IVF wiring → CB-IVF-02; menopause target → CB-MENO-08 |
| Mode set from Me › حالت اپ **or the mode chip on the Today header** | Mode screen B-N2-03 ✔; no header chip | CB-NAV-03 |
| Today: status hero, «برای امروز» (reminders, tasks, companion msg), advice, **customisable «دسترسی سریع»**, continue course/program, **mode chip + global search in header** | B-N1-06 hero/predictions ✔, B-N8-03 continue card, B-N6-08 tasks, B-N4 companion; no quick access, no search | CB-NAV-02 (search), CB-NAV-03 (chip, quick access, «برای امروز») |
| Stage tab: analysis lives **inside** it (cycle: تقویم · تحلیل · تاریخچه) | Matches bloom gap #7 (calendar segment ماه·سال·تحلیل, `/cycle` history off-nav) | B-N1-07/B-N3-08, check in CB-NAV-03 |
| + always opens the mode's log sheet (manual / voice tabs, mode tiles, med/appointment/task, log for child, kicks/contractions in pregnancy) | B-N1-04 FAB → `?sheet=log`; B-N3-03/05/06 | «برای بعد» + child rows checked in CB-NAV-03 |
| Services: care (assistant, doctors, record, labs, vitals, insurance, checkups) · programs · mother & child · learning · **shop last in its own frame** + 115 card | B-N7-01 hub has the same sections (+ B-N1-04 placeholder) | Order/frame check CB-NAV-03 |
| Me: account, family, courses/tasks/**bookings/orders**, devices/backup, privacy/lock, notifications, appearance, support — no health tool reachable only from Me | B-N1-10 groups identical | Bookings/orders rows: CB-DIR-09, CB-SHOP-08 |
| Global everywhere: search (shop excluded), notifications from Today/Services header, 115 card in Services + every alert | B-N1-03 UrgentCard, B-N7-01 115 card | CB-NAV-01/02 |
| Full-screen, no tab bar: onboarding/login/mode pick, log sheet, voice, forms, booking/cart/checkout, loss path, lock + companion panel | nav.md "hidden on back-header screens/forms/sheets" | every CB frontend task (acceptance line) |
| Teen: Services without shop; doctor & labs only with a parent | Teen: no ads/shop/Plus upsell (gap #3) | CB-NAV-03 (+ CB-TEEN-01 flag) |
| Re-tap active tab → tab root | not specified | CB-NAV-03 |
| Cross-links: class booking → reminder + «برای امروز»; layette order → checklist «دارم»; heavy pain → pain diary + doctor; lab sheet → record → insurance form prefill; bookings & orders collected in Me | — | CB-DIR-03, CB-SHOP-03, CB-COND-06, CB-REC-01/CB-INS-01, CB-DIR-09/CB-SHOP-08 |

## 2. Board map (93 snapshot boards)

Route = proposed canvas route (locale prefix implied). "Bloom" = task that builds it fully/partly. "CB" = every CB task
whose `boards:` lists it (QA tasks included). Conflicts refer to §6.

### NAV / IA

| Board | Screen | Route | Bloom | Existing | Status | CB tasks | Note |
|---|---|---|---|---|---|---|---|
| `IA_Map` | IA map | — | B-N1-04, B-N2-03, B-N1-10, B-N7-01 | bottom-nav widget | B-part | CORE-01, NAV-03, COND-06 | spec board |
| `IA_Nav` | nav rules, 7 modes | — | B-N1-04 (nav.md) | `widgets/bottom-nav` | B-part | CORE-01, NAV-03 | IVF tab missing in bloom (C3) |
| `nbd_Nav_Today` | Today (mode chip, search, quick access) | `/home` | B-N1-06, B-N8-03, B-N6-08, B-N4-03 | `/home`, `screens/home` | B-part | NAV-02, NAV-03, NAV-04 | chip/search/quick access NEW |
| `nbd_Nav_Stage` | stage tab: تقویم·تحلیل·تاریخچه | `/calendar` (+`/analysis`, `/cycle`) | B-N1-07, B-N3-08, B-N1-08 | `/calendar`, `/cycle` | B-full | NAV-03, NAV-04 | day card «افزودن ثبت» → sheet |
| `nbd_Nav_Plus` | + sheet | `?sheet=log` | B-N3-03, B-N3-02, B-N3-05, B-N3-06 | `/log` | B-part | NAV-03, NAV-04 | «برای بعد», «ثبت برای آوا» rows |
| `nbd_Nav_Services` | Services tab | `/services` | B-N7-01 (B-N1-04 placeholder) | — | B-full | NAV-03, NAV-04 | shop last, separate frame |
| `nbd_Nav_Me` | Me tab | `/profile` | B-N1-10 (+ rows N2-07, N4, N5, N6-08, N8-03, N10-07) | `/profile` | B-full | NAV-03, NAV-04 | bookings/orders rows = CB (C6) |
| `nbd_Nav_Mode` | life-stage picker | `/profile/mode` | B-N2-03 (Me_Mode) | — | B-full | NAV-03, NAV-04 | same content incl. IVF toggle, contraception, loss note |
| `nbd_Nav_Search` | global search | `/search` | — | — | NEW | NAV-01, NAV-02, NAV-04 | shop excluded |

### MENO (menopause)

| Board | Screen | Route | Bloom | Existing | Status | CB tasks | Note |
|---|---|---|---|---|---|---|---|
| `Main` (dark) | menopause home | `/home` (mode=menopause) | B-N2-03 minimal home | `screens/home` | B-part | MENO-01, 02, 05, 12 | replaces bloom's minimal home |
| `nbl_Meno_Home` | menopause home | `/home` | B-N2-03 | `screens/home` | B-part | MENO-02, 05, 13 | nav: علائم → score (C3) |
| `nbl_Meno_Stage` | stage picker | `/menopause/stage` | B-N2-02 `Onb_Meno`, B-N2-01 columns | `entities/user` | B-part | MENO-01, 02, 05, 13 | reuse profile columns (C4) |
| `nbl_Meno_Log` | daily symptoms | log sheet preset + `/menopause/log` | B-N3-01, B-N3-03, B-N3-05 | `internal/healthlog` | B-part | CORE-02, MENO-01, 06, 13 | SeverityScale |
| `nbl_Meno_HotFlash` | hot-flash timer | `/menopause/hot-flash` | — | — | NEW | MENO-01, 02, 07, 13 | CountdownRing |
| `nbl_Meno_Alert` | bleeding alert | `/menopause/alert` | B-N1-03 UrgentCard | — | NEW | MENO-01, 09, 13 | doctor CTA → M3 form until N7 |
| `nbl_Meno_Score` | monthly score (stage tab) | `/menopause/score` | B-N3-07/09 patterns | — | NEW | MENO-01, 02, 08, 13 | Plus? (C8) |
| `nbl_Meno_Checkups` | checkups | `/checkups?audience=menopause` | B-N1-15 restyle | M4 `/checkups`, `internal/checkups` | exist | MENO-01, 09, 13 | audience filter NEW |
| `nbl_Meno_Treatment` | HRT & care | `/menopause/treatment` | — | M3 `care-reminder` (meds) | NEW | MENO-01, 03, 10, 13 | WeekDots, ProgressBar |
| `nbl_Meno_Report` | doctor report | `/menopause/report` | B-N6-04 builder/PDF/share | — | B-part | MENO-03, 11, 13 | share Plus-gated in bloom (C8) |

### IVF, LOSS

| Board | Screen | Route | Bloom | Existing | Status | CB tasks | Note |
|---|---|---|---|---|---|---|---|
| `nbl_IVF_Home` | IVF today | `/ivf` (Today in IVF) | B-N2-03 `ivf_iui` toggle only | M5 fertility (ttc) | NEW | CORE-02, IVF-01, 02, 06 | StepTimeline; own nav (C3) |
| `nbl_IVF_Meds` | injections (tab «درمان») | `/ivf/meds` | — | M3 care reminders | NEW | IVF-01, 03, 06 | site rotation, inventory |
| `nbl_IVF_Scan` | scan log | `/ivf/scan` | B-N1-03 NumberStepper, charts | — | NEW | IVF-01, 04, 06 | |
| `nbl_IVF_TWW` | two-week wait | `/ivf/tww` | — | M7 pregnancy setup | NEW | IVF-01, 05, 06 | negative → `/loss` |
| `nbl_Loss_Start` | calm start | `/loss` | B-N2-03 calm exit option | M7 `internal/pregnancy` | B-part | LOSS-01, 02, 03 | entry = bloom's option |
| `nbl_Loss_Care` | care & support | `/loss/care` | B-N7 counsellor (later) | M3 reminders | NEW | LOSS-01, 02, 03 | counsellor row hidden until N7 |
| `nbl_Loss_Next` | next step | `/loss/next` | B-N2-03 mode switch | `entities/user` | NEW | LOSS-01, 02, 03 | |

### COND, CONTRA, PELV

| Board | Screen | Route | Bloom | Existing | Status | CB tasks | Note |
|---|---|---|---|---|---|---|---|
| `nbl_Cond_Hub` | programs hub | `/programs` | B-N7-01 program tiles (admin content) | — | B-part | COND-01, 02, 07 | PCOS minimal (DECISIONS #8) |
| `nbl_Cond_Endo` | pain diary | `/programs/pain` | B-N3-01 pain, B-N3-03 body map | `internal/healthlog` | B-part | CORE-02, COND-01, 03, 07 | NumericScale 0–10 |
| `nbl_Cond_PMDD` | PMDD daily | `/programs/pmdd` | B-N3-01 mood (partial) | — | NEW | CORE-02, COND-01, 04, 07 | NumericScale 1–6, hotlines |
| `nbl_Cond_Bleed` | PBAC chart | `/programs/bleeding` | B-N3-01 bleeding (partial) | — | NEW | COND-01, 05, 07 | |
| `nbl_Contra_Setup` | method | `/contraception/setup` | B-N2-03 «track contraception» flag | — | B-part | CONTRA-01, 02, 04 | one flag (C5) |
| `nbl_Contra_Pill` | pill pack | `/contraception` | B-N1-09 pill reminder pref | M3 medication reminders | B-part | CONTRA-01, 02, 04 | one reminder (C5) |
| `nbl_Contra_Missed` | missed pill | `/contraception/missed` | B-N1-03 UrgentCard | — | NEW | CONTRA-01, 03, 04 | |
| `nbl_Contra_Other` | IUD/injection/implant | `/contraception/other` | — | M3 reminders | NEW | CONTRA-01, 03, 04 | |
| `nbl_Pelvic_Plan` | 8-week plan + bladder diary | `/programs/pelvic` | B-N7-01 tile only | — | NEW | PELV-01, 02, 03 | WeekDots |
| `nbl_Pelvic_Kegel` | Kegel trainer | `/programs/pelvic/kegel` | — | — | NEW | CORE-02, PELV-01, 02, 03 | CountdownRing, full-screen |

### PRIV, TEEN, dropped

| Board | Screen | Route | Bloom | Existing | Status | CB tasks | Note |
|---|---|---|---|---|---|---|---|
| `nbl_Priv_Settings` | privacy & lock | `/profile/privacy` | B-N1-12 (lock, blur, export, delete), B-N1-11 neutral text | `screens/profile*` | B-full | PRIV-01 | icon row dropped (C2) |
| `nbl_Priv_Lock` | lock screen | overlay | B-N1-12 | — | B-full | PRIV-01 | + emergency-card link |
| `nbl_Priv_Icon` | disguised icon | — | — | — | drop | — | DECISIONS #5 |
| `nbl_Teen_Onb` | teen onboarding | `/teen/onboarding` | B-N2-01/02 (teen mode) | — | NEW | TEEN-01, 02, 04 | |
| `nbl_Teen_Home` | teen Today | `/home` (mode=teen) | B-N2-03 minimal teen home | `screens/home` | B-part | TEEN-01, 02, 04 | no shop/ads |
| `nbl_Teen_Parent` | mother sharing | `/teen/parent` | B-N4-01/02/04 companion | — | B-part | TEEN-01, 03, 04 | new `parent` type |
| `nbl_Wear_Connect` | wearables sources | — | B-N10-07 «به‌زودی» row | — | drop | — | DECISIONS #5 |
| `nbl_Wear_Data` | wearable data | — | — | — | drop | — | DECISIONS #5 |

### REC, INS, VOICE

| Board | Screen | Route | Bloom | Existing | Status | CB tasks | Note |
|---|---|---|---|---|---|---|---|
| `nbl_Rec_Home` | record summary | `/record` | B-N6-03 Record_Summary | — | B-part | REC-01, 04, 06 | canvas grid + bloom sections |
| `nbl_Rec_Timeline` | documents timeline | `/record/timeline` | B-N6-06 labs (in timeline) | — | NEW | REC-01, 04, 06 | |
| `nbl_Rec_Doc` | document detail | `/record/documents/[id]` | B-N6-05/06 extraction pattern | — | NEW | REC-01, 02, 04, 06 | Plus = AI only (DECISIONS #11) |
| `nbl_Rec_Share` | who sees my record | `/record/share` | B-N6-04 7-day link, B-N1-12 «دسترسی دیگران», B-N4 | — | B-part | REC-03, 05, 06 | no insurer access (C7) |
| `nbl_Rec_Emergency` | emergency card | `/record/emergency` | — | — | NEW | PRIV-01, REC-03, 05, 06 | shown on lock screen |
| `nbl_Ins_Home` | my insurance | `/insurance` | B-N10-06 info page | — | B-part | INS-01, 03, 07 | self-tracking (DECISIONS #12) |
| `nbl_Ins_Coverage` | coverage & caps | `/insurance/coverage` | — | — | NEW | INS-01, 03, 07 | ProgressBar |
| `nbl_Ins_Health` | health questionnaire | `/insurance/questionnaire` | B-N6-03 record data | — | NEW | INS-01, 05, 07 | export PDF only |
| `nbl_Ins_Status` | request status | `/insurance/status` | — | — | NEW | INS-01, 05, 07 | StepTimeline, user-advanced |
| `nbl_Ins_Claims` | claims list | `/insurance/claims` | — | — | NEW | INS-01, 04, 07 | members from B-N4/B-N5 |
| `nbl_Ins_ClaimNew` | new claim | `/insurance/claims/new` | — | — | NEW | INS-01, 04, 07 | docs from record |
| `nbl_Ins_ClaimDetail` | claim detail | `/insurance/claims/[id]` | — | — | NEW | INS-01, 04, 07 | StepTimeline |
| `nbl_Ins_Centers` | contracted centres | `/insurance/centers` | — | — | NEW | CORE-06, INS-01, 06, 07 | Neshan map |
| `nbl_Voice_Entry` | voice entry | log sheet «با صدا» tab | B-N3-05 (`Log_Day`) | — | B-part | VOICE-02, 03 | Plus (DECISIONS #11) |
| `nbl_Voice_Record` | recording + live text | same | B-N3-05 | — | B-part | VOICE-02, 03 | audio never stored |
| `nbl_Voice_Review` | review & edit | same | B-N3-05 «این‌ها را فهمیدیم» | — | B-part | VOICE-01, 02, 03 | quote per item, ambiguity chooser |
| `nbl_Voice_Saved` | saved | same | B-N3-05 | — | B-part | VOICE-01, 02, 03 | 21:00 reminder opt-in |

### DIR (mother & child) — app + desktop web

| Board | Screen | Route | Bloom | Existing | Status | CB tasks | Note |
|---|---|---|---|---|---|---|---|
| `nbl_Dir_Home` | services home | `/services/mother-child` | B-N10-05 listings | — | B-part | DIR-02, 06, 13 | |
| `nbl_Dir_Map` | map | `/services/mother-child/map` | — | — | NEW | CORE-06, DIR-02, 07, 13 | list fallback w/o key |
| `nbl_Dir_List` | list & filter | `/services/mother-child/list` | B-N10-05 list | — | B-part | DIR-02, 07, 13 | |
| `nbl_Dir_Place` | place page | `/services/mother-child/place/[id]` | B-N10-05 detail | — | B-part | DIR-01, 02, 08, 13 | «پیام» hidden (C9) |
| `nbl_Dir_Book` | pick a time | `/services/mother-child/book` | B-N10-05 request | — | B-part | DIR-01, 03, 09, 13 | request, no payment (C1) |
| `nbl_Dir_Booked` | booking sent | `…/booked/[id]` | — | — | NEW | DIR-03, 09, 13 | «پرداخت شد» → pay at place |
| `nbl_Dir_MyBookings` | my bookings | `/profile/bookings` | B-N10-05 my bookings | — | B-part | DIR-03, 09, 13 | supersedes bloom list |
| `nbl_Dir_Join` | join: intro | `/services/mother-child/join` | — | — | NEW | DIR-04, 10, 13 | terms text TBD (Q6) |
| `nbl_Dir_JoinForm` | join: place & photos | same (step 2) | — | — | NEW | DIR-01, 04, 10, 13 | |
| `nbl_Dir_JoinDocs` | join: services & docs | same (step 4) | — | — | NEW | DIR-01, 04, 10, 13 | «online» = request (C1) |
| `nbl_Dir_JoinDone` | join: under review | same (done) | — | — | NEW | DIR-04, 10, 13 | StepTimeline |
| `W_Dir_Search` | web search + map | `(web)` `/mother-child` | — | — | NEW | DIR-11, 13 | web shell NEW |
| `W_Dir_Place` | web place + booking | `(web)` | — | — | NEW | DIR-11, 13 | |
| `W_Dir_Booked` | web booked | `(web)` | — | — | NEW | DIR-11, 13 | store badges (Q5) |
| `W_Dir_Business` | for businesses | `(web)` `/business` | — | — | NEW | DIR-12, 13 | |
| `W_Dir_Join` | web join wizard | `(web)` `/business/join` | — | — | NEW | DIR-12, 13 | |
| `W_Dir_JoinDone` | web join done | `(web)` | — | — | NEW | DIR-12, 13 | |

### SHOP — app (dark-only boards) + desktop web

| Board | Screen | Route | Bloom | Existing | Status | CB tasks | Note |
|---|---|---|---|---|---|---|---|
| `nbd_Shop_Home` | baby shop home | `/services/shop` | B-N10-03 entry + categories | — | B-part | SHOP-02, 06, 12 | link-out replaced (C1) |
| `nbd_Shop_Beauty` | beauty home | `/services/shop/beauty` | B-N10-03 categories | — | B-part | SHOP-02, 04, 06, 12 | supply reminder off by default |
| `nbd_Shop_List` | list & filter | `/services/shop/list` | B-N10-03 products | — | B-part | SHOP-02, 07, 12 | |
| `nbd_Shop_Product` | baby product | `/services/shop/product/[id]` | — | — | NEW | SHOP-01, 02, 07, 12 | Gallery, rating |
| `nbd_Shop_ProductBeauty` | beauty product | same | — | — | NEW | SHOP-01, 02, 07, 12 | IRC, ingredients |
| `nbd_Shop_Checklist` | layette checklist | `/services/shop/checklist` | — | M7 due date | NEW | SHOP-01, 04, 09, 12 | |
| `nbd_Shop_Cart` | cart | `/services/shop/cart` | — | — | NEW | SHOP-01, 03, 08, 12 | |
| `nbd_Shop_Checkout` | checkout | `/services/shop/checkout` | — | — | NEW | SHOP-01, 03, 08, 12 | COD only, gateway row hidden (C1) |
| `nbd_Shop_Order` | order placed | `/services/shop/orders/[id]` | — | — | NEW | SHOP-01, 03, 08, 12 | «رسید پرداخت» copy → COD |
| `W_Shop_Home` | web shop home | `(web)` `/shop` | — | — | NEW | SHOP-10, 12 | |
| `W_Shop_List` | web list | `(web)` | — | — | NEW | SHOP-10, 12 | |
| `W_Shop_Product` | web product | `(web)` | — | — | NEW | SHOP-10, 12 | |
| `W_Shop_Cart` | web cart | `(web)` | — | — | NEW | SHOP-11, 12 | |
| `W_Shop_Checkout` | web checkout | `(web)` | — | — | NEW | SHOP-11, 12 | COD only |
| `W_Shop_Done` | web done | `(web)` | — | — | NEW | SHOP-11, 12 | store badges (Q5) |

**Coverage check:** every snapshot board is listed by ≥1 CB task except the three dropped ones (`nbl_Priv_Icon`,
`nbl_Wear_Connect`, `nbl_Wear_Data`). Counts (93): B-full 6 · B-part 31 · exist 1 · NEW 52 · drop 3.

## 3. Primitive gaps vs B-N1-03

B-N1-03 ships: ScreenHeader, HubHeader, DateStrip, Card, HeroCard, SectionTitle, PillChip, SegmentedTabs, NumberStepper,
StatusPill, ListRow, Accordion, BottomSheet (= AppSheet), Primary/SecondaryButton, TileButton, IconCircle, Switch,
PlusLock, InfoNote, UrgentCard, ProgressSteps, ProgressRing, Line/BarChart, Avatar, Skeleton, EmptyState, background
layer. FAB/BottomNav = B-N1-04.

| Need (boards) | Verdict | Where |
|---|---|---|
| **SeverityScale** — ندارم/خفیف/متوسط/شدید rows (Meno_Log ×13), 4-level hot-flash (Meno_HotFlash) | missing | CB-CORE-02 |
| **NumericScale** 0–10 (Cond_Endo) / 1–6 ×6 (Cond_PMDD) with end labels | missing | CB-CORE-02 |
| **StepTimeline** vertical done/current/todo + date/deadline (IVF_Home 6 stages, Ins_Status, Ins_ClaimDetail, Dir_JoinDone, Shop_Order, W_Shop_Done) | missing (ProgressSteps = header bars only) | CB-CORE-02 |
| **CountdownRing** elapsed / countdown (Meno_HotFlash 01:42, Pelvic_Kegel hold 3/5 s, IVF_TWW days to beta) | partial — wrap B-N1-03 ProgressRing + timer hook | CB-CORE-02 |
| **DangerNote** (Meno_Alert, Loss_Care 115, Cond_PMDD/Loss hotlines 1480/123, IVF_TWW OHSS, Contra_Missed) | covered — extend UrgentCard with optional actions / no-CTA variant | CB-CORE-02 (extension, not new) |
| **WeekDots** ش…ج adherence (Meno_Treatment, Pelvic_Plan) | missing (DateStrip is a 42×54 picker) | CB-CORE-02 |
| **ProgressBar** labelled «X از Y» (Ins_Home/Coverage caps, Meno_Treatment 90/150, Shop_Checklist groups, Pelvic streak) | missing | CB-CORE-02 |
| **RadioCard** title + description (Meno_Stage, Loss_Start/Next, Teen_Onb, Nav_Mode, Dir_JoinDocs booking mode) | missing in B-N1-03 list (bloom builds Me_Mode/Onb_Goal cards in screens) | CB-CORE-02 unless B-N2-02/03 export one |
| **SearchField** (Nav_Search, Dir_List/Map, Shop_List, Ins_Centers, Ins_Claims) | not in B-N1-03 list | CB-CORE-02 if B-N1-03 lacks it |
| **Checkbox / TaskRow** (Teen kit, Shop_Checklist states, Dir_Join terms) | in components.md, not in B-N1-03 scope | CB-CORE-02 if B-N1-03 lacks it |
| Text / textarea / amount fields (Meno_Report questions, Ins_ClaimNew, Dir_Join) | not in B-N1-03 list (existing `LocaleNumberField`, `CalendarPicker`) | reuse existing; ask B-N1-10 output first |
| SlotGrid (time chips with «پر شد / ۲ جا») (Dir_Book, Shop_Checkout) | PillChip + disabled + caption variant | feature-level in CB-DIR-09 / CB-SHOP-08 |
| Compact qty stepper (Shop_Cart) | NumberStepper compact variant | CB-SHOP-08 |
| Band meter (score band بدون علامت…شدید) (Meno_Score) | chart-level | CB-MENO-08 on bloom charts |
| Chart bands / two series (Cond_PMDD period + last-10-days bands, IVF_Scan growth) | partial — bloom Line/BarChart (+ `widgets/bbt-chart` band) | CB-COND-04, CB-IVF-04 |
| Rating summary + sub-scores, reviews (Dir_Place, Shop_Product) | not in bloom shared/ui (B-N7-05 doctor reviews may build one) | reuse B-N7-05's if extracted, else CB-DIR-08 → shared later |
| Image gallery / carousel (Dir_Place, Shop_Product, W_*) | missing | CB-DIR-08 / CB-SHOP-07 (marketplace-local) |
| File / camera picker (Rec upload, Ins_ClaimNew, Dir_JoinForm/Docs) | bloom B-N6-07 `Lab_Upload` builds one in a screen | promote in CB-REC-04 |
| Body map (Cond_Endo) | B-N3-03 pain panel | reuse in CB-COND-03 |
| PIN keypad (Priv_Lock) | B-N1-12 | reuse |
| QR code (Rec_Share) | missing | CB-REC-05 (lib) |
| Map | missing | CB-CORE-06 (Neshan) |
| Desktop web shell header/footer (W_*) | missing | CB-DIR-11 |
| Pill-pack grid (Contra_Pill) | feature-specific | CB-CONTRA-02 |

## 4. Token map — board hex → Night & Bloom tokens

Board hex values are not the spec (DECISIONS #6); this table only tells a builder which token a board colour means.
Canonical names/values: `docs/night-bloom/tokens.md` (B-N1-02). The canvas uses exactly bloom's dialect A light and the
single dark dialect — **no new palette**. Counts = occurrences in the snapshot.

### 4.1 Light boards (`nbl_*`, `W_*`, IA) → token (light value / dark value)

| Board hex (count) | Token | Light | Dark | Use in boards |
|---|---|---|---|---|
| `#231B3B` (925) | `--text-1` | `#231B3B` | `#F6F1FF` | titles, body |
| `#5E5873` (900) | `--text-2` | `#5E5873` | `#B8AED6` | secondary text |
| `#E7E1F4` (868) | `--line` (bars: `--track`) | `#E7E1F4` | `#34295A` | card borders, rails |
| `#FFFFFF` (467) | `--surface` | `#FFFFFF` | `#221A3D` | cards; white on fills = `--on-brand` / `--on-accent` |
| `#6E54F0` (443) | `--brand` / `--brand-fill` | `#6E54F0` | `#B9A6FF` | CTA, FAB, selected chips (text on it = `--on-brand`, flips) |
| `#F7F3FF` (193) | `--page` | `#F7F3FF` | `#17112B` | canvas |
| `#0A7390` (147) | `--data-deep` | `#0A7390` | `#6FE7D0` | turquoise text/icons, «طبیعی» |
| `#5B41E6` (136) / `#5238D4` (web, 12) | `--brand-strong` | `#5B41E6` | `#D6CBFF` | links, «همه», hover/pressed |
| `#B82A52` (76) | `--danger` in alerts/danger cards; `--period-deep` for period data | `#C42D57` / `#B82A52` | `#FF8FA3` / `#FF6B8B` | danger icons, bullets (gap #11: distinguish by icon + copy) |
| `#D9447F` (72) | `--bloom-deep` | `#D9447F` | `#FFD5B8` | mother/child, companion, rose accents |
| `#ECE6FF` (60) | `--brand-soft` | `#ECE6FF` | `rgba(185,166,255,.16)` | tint chips, icon circles |
| `#0C7064` (19) | `--success` | `#0F7B6C` | `#7FE0A8` | done / verified / «پرداخت شد» |
| `#8A5A00` (13 + web 51) | `--warm-deep` | `#B45309` | `#FFC98A` | stars, «نزدیک», warnings as text |
| `#F1EDFB` (4) | `--surface-3` | `#F1EDFB` | `rgba(255,255,255,.05)` | inset tiles |
| `#E0A040` (IA, 2) | `--warm` | `#F5A623` | `#FFB86B` | TTC mode icon |
| `rgba(255,255,255,.94)` / `.92` (web) | `--surface-glass` | same | `rgba(34,26,61,.92)` | bottom nav, sticky bars |
| `rgba(110,84,240,.8)` (35) | `--shadow-cta` / `--shadow-fab` | per tokens §7 | per tokens §7 | CTA/FAB shadow |
| `rgba(0,0,0,.4–.6)` | `--shadow-float` / `--scrim` | per tokens | per tokens | nav shadow, sheet scrim |
| `#6E54F033→#6E54F000` radial (62) | `--bg-glow` | tokens §4 | tokens §4 | top glow on every screen |
| `#6E54F0` + alpha `22/1C/2E/33` | `--brand-soft` (or `color-mix(var(--brand) 13–20%)`) | | | selected/tint fills |
| `#0A7390` + alpha `22–66` | `--data-soft` (fills) / `--data` (borders) | `#E3F9FE` | `#1E3340` | data tints |
| `#B82A52` + alpha `22` | `--danger-soft` / `--period-soft` | `#FDE4EB` | `#5A2A3D` / `#3A2140` | danger card bg |
| `#0C7064` + alpha `1F/22` | `--success-soft` | `#E7F8EF` | `#1F3A2E` | verified badge bg |
| `#D9447F` + alpha `22–88` | no `--bloom-soft` token — use `color-mix(in srgb, var(--bloom) 14%, transparent)` / `--avatar-grad` | | | mother & child cards, avatars |
| `rgba(255,255,255,.25/.7)` | `color-mix` of `--on-brand` | | | text/rings on filled heroes |

### 4.2 Dark boards (`nbd_Nav_*`, `nbd_Shop_*`, `Main`, IA dark panel) → token

| Board hex (count) | Token | Dark value | Light value |
|---|---|---|---|
| `#B8AED6` (352) | `--text-2` | `#B8AED6` | `#5E5873` |
| `#F6F1FF` (342) | `--text-1` | `#F6F1FF` | `#231B3B` |
| `#34295A` (252) | `--line` | `#34295A` | `#E7E1F4` |
| `#B9A6FF` (230) | `--brand` / `--brand-fill` | `#B9A6FF` | `#6E54F0` |
| `#221A3D` (148) | `--surface` | `#221A3D` | `#FFFFFF` |
| `#17112B` (84) | `--page`; also text on lavender fills = `--on-brand` | `#17112B` | `#F7F3FF` / `#FFFFFF` |
| `#FFD5B8` (49) | `--bloom` / `--bloom-deep` | `#FFD5B8` | `#FF6FAE` / `#D9447F` |
| `#8C82AD` (36) | `--text-3` | `#8C82AD` | `#6A6480` |
| `#4CE0C3` (35) | `--data` | `#4CE0C3` | `#0FA3C9` (text: `--data-deep`) |
| `#FFB86B` (30) | `--warm` | `#FFB86B` | `#F5A623` (text: `--warm-deep`) |
| `#FF6B8B` (29) | `--period` / `--rose` (danger text → `--danger` `#FF8FA3`) | `#FF6B8B` | `#E8436F` / `#B82A52` |
| `#D6CBFF` (16) | `--brand-strong` / `--brand-ink` | `#D6CBFF` | `#5B41E6` / `#4428B8` |
| `#7FE0A8` (14) | `--success` | `#7FE0A8` | `#0F7B6C` |
| `rgba(255,255,255,.07)` (16) | `--surface-2` | same | `#F4F0FF` |
| `rgba(34,26,61,.92)` | `--surface-glass` | same | `rgba(255,255,255,.94)` |
| `rgba(185,166,255,.9)` | `--shadow-cta` / `--shadow-fab` | per tokens §7 | per tokens §7 |
| `#B9A6FF22/33`, `#4CE0C322`, `#FF6B8B22`, `#7FE0A822`, `#FFB86B33` | `--brand-soft`, `--data-soft`, `--period-soft`/`--danger-soft`, `--success-soft`, `--warm-soft` | tokens | tokens |
| `#1E1736` (2) | illustration band bg → `--surface-4` | `#2A2150` | `#F1EDFB` |
| `rgba(10,6,24,.55)` | `--scrim` | `rgba(0,0,0,.6)` | `rgba(0,0,0,.35)` |

### 4.3 Colours that are data, not tokens

- `nbl_Priv_Icon` app-icon swatches `#2F7D6B #B7791F #4A4A55 #2B6CB0 #8A4F7D` — board dropped.
- Product colour swatches / image placeholders `#F6EFE6 #D9D4F2 #CDEBE4 #F4D6DE #E9E9EE` (Shop product/list, W_Shop) —
  real values come from product variants (admin data); placeholders use `--surface-3`.
- `#3A2F66` border on the dark "get the app" band of `W_Dir_Booked`/`W_Shop_Done` → `--line-strong` (dark).

## 5. Gaps — no design and/or no backend, and who covers them

| Gap (where it appears) | Covered by | Until then / default |
|---|---|---|
| Health assistant (Services tile, Search «از دستیار سلامت بپرس») | B-N7-06/07 (N7) | tile/search group hidden |
| Doctors & midwives, booking (Services, Search, Meno_Alert «نوبت پزشک زنان», COND nudges, Loss counsellor) | B-N7-02..05 (N7) | appointment reminder form (M3 `/reminders/appointment/new`); counsellor row hidden |
| Courses / learning (Today «ادامه بده», Services «آموزش», Search, Me «دوره‌های من») | B-N8-03 (N8) | hidden |
| Plus rows (Me «ریتمی پلاس»), paywall | B-N2-04..08 (N2) | DECISIONS #11: only AI features Plus for canvas features |
| Companion (Today «از همدم», IVF companion reminders, Loss tell-companion, Rec_Share family, Teen parent) | B-N4-01..06 (N4) | rows hidden without a linked companion |
| Children (Dir child chips, Ins members, «ثبت برای آوا») | B-N5-02/05 (N5) | hidden without children; Dir «add child» → B-N5 form |
| Tasks / todo («۴ کار امروز», + «کار جدید») | B-N6-08 (N6) | hidden |
| Vitals (BP in Meno_Home/Checkups/Report, «علائم حیاتی» tile) | B-N6-01/02 (N6) | «ثبت کن» → hidden or checkups record |
| Labs («تحلیل آزمایش», «افزودن نتیجه آزمایش», record timeline labs) | B-N6-06/07 (N6) | hidden |
| Analysis / patterns (Nav_Stage «تحلیل کامل», Meno_Score patterns, Search «در تحلیل») | B-N3-07..09 (N3) | — |
| Services hub itself | B-N7-01 (N7); B-N1-04 placeholder (N1) | placeholder |
| Postpartum / child tab «کودک», feeding/diapers | B-N5 (N5) | — (DECISIONS #8 minimal) |
| Wearables («ساعت و گجت‌ها» row) | B-N10-07 «به‌زودی» (N10) | dropped from this queue (DECISIONS #5) |
| **Today «دسترسی سریع»** (Nav_Today, IA_Map) | **no bloom task** | added to CB-NAV-03 (§7); persistence Q3 |
| **Today mode chip** | no bloom task | added to CB-NAV-03 |
| **Global search** | no bloom task | CB-NAV-01/02 |
| Messaging a place/seller («پیام به مجموعه», «پیام به فروشنده») | none (DECISIONS #14: no business panel) | hidden, or → support |
| Directory/shop online payment («پرداخت و رزرو», «درگاه بانکی», «پرداخت شد») | B-N2-05 gateway exists but DECISIONS #13 forbids it here | requests + COD copy |
| Insurer-side workflow (Ins_Status «پزشک معتمد», Rec_Share insurer access) | none (DECISIONS #12) | user-advanced statuses, PDF export |
| Business cooperation terms «[مدل کارمزد یا اشتراک؛ هنوز تعیین نشده]» (Dir_Join, W_Dir_Business) | none | admin-editable text, Q6 |
| Store badges (Bazaar/Myket/Google Play/iOS) on W_Dir_Booked/W_Shop_Done | none (DECISIONS #4 no native work) | hidden unless links exist, Q5 |
| PCOS program detail | none (DECISIONS #8) | enrolment card only (CB-COND-01) |
| Dark twins of 61 light boards | exist in the canvas, not in the snapshot | tokens §4, Q1 |

## 6. Conflicts (board / bloom vs DECISIONS)

| # | Conflict | Resolution (DECISIONS) | Where enforced |
|---|---|---|---|
| C1 | Shop: bloom B-N10-03 = partner **link-out**; boards show bank gateway + COD; Dir boards show «پرداخت و رزرو» | #13, #15: internal shop, **COD only**; bloom's entry/categories kept, link-out purchase path superseded; directory bookings are requests paid at the place | CB-SHOP-01/03/08, CB-DIR-01/09 (edited) |
| C2 | Priv boards: disguised icon (`Priv_Icon`), «مخفی در صفحه برنامه‌های اخیر», wearables | #5: dropped (native only); recents row = bloom's web hide-preview blur (B-N1-12) | CB-PRIV-01 (edited) |
| C3 | IA: IVF sub-mode with tab «درمان»; menopause tab «علائم» → score. Bloom nav.md: no IVF row; menopause → `/analysis/symptoms` | #2 build on bloom, canvas IA wins for canvas modes (#3 IA first) | CB-IVF-02, CB-MENO-08 (edited), CB-NAV-03 checks |
| C4 | Menopause profile: bloom B-N2-01 already stores stage / last period / surgical / HRT; CB-MENO-01 planned `menopause_profiles` | #2 reuse, never rebuild | CB-MENO-01 (edited) |
| C5 | Contraception: bloom has a «track contraception» flag (B-N2-03) and a cycle-settings pill reminder (B-N1-09) | #2 reuse → one flag, one reminder | CB-CONTRA-01 (edited) |
| C6 | Me «سفارش‌ها»/«رزروها»: bloom B-N10-03 partner orders row, B-N10-05 my bookings; CB used `/me/...` routes | #15 internal shop; Me = bloom `/profile` | CB-SHOP-08, CB-DIR-09 (edited) |
| C7 | Rec_Share / Ins_Status show insurer access and insurer review | #12: self-tracking only, nothing sent to insurers | CB-REC-05 (edited), CB-INS-01 (already) |
| C8 | Bloom Plus-gates its report share link (B-N6-04), analysis detail (B-N3-07) and pattern sections; canvas Meno_Report/Meno_Score/COND reuse them | #11: canvas features free, only AI Plus — **but the gated pieces are bloom's** | open question Q2 |
| C9 | Dir_JoinDocs «رزرو آنلاین … خودکار تأیید می‌شود»; «پیام به مجموعه» | #13/#14: every booking is a request confirmed by the Ritme team; no business messaging | CB-DIR-01 (edited) |
| C10 | Discreet notifications: bloom B-N1-11 already has «متن خنثی» neutral push | #2 reuse the preference | CB-PRIV-01 (edited) |
| C11 | W_* boards show app-store badges | #4 no native work; the WebView shell exists outside this queue | CB-DIR-11 ("only if links exist"), Q5 |
| C12 | Loss entry: bloom B-N2-03 builds a calm exit option in pregnancy mode | #2 reuse as the entry to `/loss` | CB-LOSS-02 (edited) |

## 7. CB task edits made by CB-CORE-01

| Task | Change | Why |
|---|---|---|
| CB-CORE-02 | Scope: final primitive list (SeverityScale, NumericScale, StepTimeline, CountdownRing over ProgressRing, WeekDots, ProgressBar, RadioCard, SearchField/Checkbox only if missing; DangerNote = UrgentCard extension). Boards + Meno_HotFlash, Meno_Stage, Meno_Treatment, Ins_Coverage, Ins_ClaimDetail | §3 |
| CB-NAV-03 | Scope 3–5: Today mode chip, «برای امروز», «دسترسی سریع»; + sheet «برای بعد» / child rows; teen Services doctor/lab rule; IVF/menopause wiring moved out. Touches + `screens/home`, `widgets/quick-access` | §1, §5 |
| CB-MENO-01 | Reuse bloom B-N2-01 profile columns instead of `menopause_profiles`; deps + B-N2-01 | C4 |
| CB-MENO-08 | Re-point menopause tab «علائم» to the score screen; touches + `widgets/bottom-nav`; deps + B-N1-04 | C3 |
| CB-IVF-02 | IVF sub-mode nav (امروز → IVF home, «درمان» → meds); touches + `widgets/bottom-nav`, `screens/home`; deps + B-N1-04 | C3 |
| CB-LOSS-02 | Entry = bloom's calm exit option + IVF negative result | C12 |
| CB-CONTRA-01 | One flag / one pill reminder with bloom B-N1-09/B-N2-03; deps + B-N1-09 | C5 |
| CB-PRIV-01 | Reuse B-N1-11 neutral-text pref; icon row dropped, recents row = bloom blur | C2, C10 |
| CB-REC-05 | Insurer rows informational only, no insurer access history | C7 |
| CB-DIR-01 | `online` booking mode is still a request; no messaging | C9 |
| CB-DIR-09 | Route `/profile/bookings`; supersedes bloom B-N10-05 list, lists bloom doctor bookings | C6 |
| CB-SHOP-08 | Route `/profile/orders`; replaces bloom B-N10-03 partner-orders row; no gateway row | C1, C6 |

`bash roadmap/bin/next.sh --check` → `ok` after the edits.

## 8. Open questions (also in `roadmap/PROGRESS.md`)

1. **Dark boards:** snapshot 61 `nbd_` twins from the canvas into `docs/design/canvas-v1/boards/` so QA can compare dark
   screens against a render? (Default: compare dark against tokens only.)
2. **Plus vs bloom gates (C8):** menopause doctor-report share link (B-N6-04 Plus-gated share), analysis-engine
   patterns (B-N3-07 Plus sections) used by Meno_Score / COND — free for canvas modes or keep bloom's gate?
3. **Quick access persistence:** store Today shortcuts locally (default) or add a small server preference?
4. **Bloom N10 scoping (B-N10-01):** tell the bloom queue to drop the shop link-out purchase path and the partner
   «سفارش‌ها» row (DECISIONS #15) so B-N10-03 builds only entry/categories? CB-SHOP-01 depends on B-N10-03.
5. **Store badges** on desktop web (W_Dir_Booked, W_Shop_Done): link the existing Android WebView shell listing, or hide?
6. **Directory cooperation terms** (commission / subscription «هنوز تعیین نشده»): leave as admin-editable placeholder?
7. **NAV-03 timing:** CB-NAV-03 depends on B-N7-01 (Services hub, N7) and B-N3-03, so IA alignment lands late despite
   DECISIONS #3. Split it (shell rules now, Services order after N7)? Not changed here.
