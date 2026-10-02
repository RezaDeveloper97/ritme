# N4 design-fidelity audit (B-N4-09)

Side-by-side of every N4 screen (light + dark, 390 px, full page) against its artboard
(`docs/design/night-bloom/companion-family/nb{l,d}_Hamdam_*`, `a-start-plus/nb{l,d}_Onb_Partner*`,
`g-me-settings/nb{l,d}_Me_Hub|Me_Privacy`), after B-N4-04 … B-N4-07. Personas: p04 owner (Plus) with an active spouse
(persona 12) plus pending partner invites; 0990…551 male linked to p04 (cycle view, meds edit, appointments view);
0990…552 male without a link. Admin-web: `qa@ritme.local`.

**Source of the app shots.** The local Go API (:8020) was down for the whole audit window and may not be restarted
until the orchestrator's migrations land, so no fresh app shots could be taken. The comparison uses the full-page
shots the screen tasks committed — `docs/qa/bloom/B-N4-04/` (list, wizard type/access/children/invite/code/done,
detail, pending detail, empty list, Me hub, privacy, `en_` LTR), `B-N4-05/` (onboarding partner + linked, companion
home, no-link home, `/companion/links`, male Me hub), `B-N4-06/` (record-for sheet + medication form from the
companion, home «افزودن … برای»), `B-N4-07/` (admin tips + links). `git log` shows no change to
`entities/companion`, `features/invite-companion`, `screens/companion-*`, `screens/onboarding-flow/ui/Partner*` or the
`cmp-`/`cmh-`/`rcf-` CSS since those commits (c1f1b09, 479f72d, c3a848a), so the shots are the current UI. Artboards
were rendered to a scratch folder only (not committed). The one med item has a before-fix composite
(artboard light · app light · artboard dark · app dark): `docs/qa/bloom/B-N4-09/side-by-side/companion-home.before-fix.png`.

Data-driven differences (names such as «Contract regular» / Latin initials of the dev personas, dates, counts, two
pending invites in the dev DB, a single article) are not deviations. Deliberate defaults in `bloom/QUESTIONS.md` are
listed as **accepted (QUESTIONS #n)** — #91–#96 all as taken. Boards draw a 54 px fake status bar and an ↑ back arrow
— accepted (QUESTIONS #30). Sticky form footers sit on `--surface-glass` (project-wide pattern for sticky CTAs; boards
draw the CTA on the canvas) — accepted, not listed per screen.

Severity: **high** = broken control / unreadable; **med** = visible departure from the board, a missing control or the
token rules; **low** = polish. Totals (N4 screens): **high 0 · med 1 · low 7** found — fixed: med 1 + low 1 (trivial);
**6 low open**.

## Owner — `/companions` (Hamdam_List), `/companions/[id]` (no board)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| O1 | Header, heading + lead, companion card (bubble, name, «همسر · فعال», access chips: edit in data tone, view in brand), «خانواده شما» strip with self/companion bubbles and note, «افزودن همدم» CTA: match in both themes; LTR (`en_companions`) mirrors correctly. Pending invites add a clock line «دعوت تا ۲۴ ساعت دیگر معتبر است» (board shows only an active spouse). | — | — | match |
| O2 | Family strip has no child bubble and the note reads «فرزندان مشترک بعد از راه‌اندازی بخش فرزندان…» (board: «آوا» + shared-child note). | — | — | accepted — children wait for B-N5-02 (B-N4-04 scope, QUESTIONS #91) |
| O3 | Empty list: users icon, «هنوز همدمی نداری», CTA. Detail page (hero, grants editor reusing the Access segmented control, shared children «به‌زودی», pending invite + «ساختن کد تازه», recent activity, «حذف همدم»): no board — consistent with the wizard cards and tokens in both themes. | — | — | match (no board) |

## Owner wizard — `/companions/new` (Hamdam_Type, _Access, _Children, _Invite, _Done)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| W1 | Type: header «افزودن همدم · مرحله ۱ از ۳», 3-step bar, two relation cards, chosen card tinted in bloom (not violet) as on the board: match. | — | — | match |
| W2 | Access: five rows × three-way segmented control, info note, «ادامه»: match. Default is «نبیند» everywhere (board draws a filled example). | — | — | accepted (B-N4-04 default none, QUESTIONS #91) |
| W3 | Children (spouse only): «بخش فرزندان به‌زودی می‌آید» card + disabled dashed «افزودن فرزند · به‌زودی» instead of the child checklist; partner skips the step as on the board. | — | — | accepted (until B-N5-02) |
| W4 | Invite: name, phone (LTR digits), «یا» divider, code card (six slots; empty dashed slots + «فقط کد بساز» before a code exists, then code + «کپی کد» / «فرستادن» + «یک‌بارمصرف · تا ۲۴ ساعت معتبر» + show-once warning), summary with «ویرایش», «ارسال دعوت»: match in both themes. Codes are created on demand, never prefilled (board shows one). | — | — | match (security: code lives only in component state) |
| W5 | Summary lists every level row and prints «هیچ» for an empty one (board omits nothing because its example fills every level). | low | `frontend/src/features/invite-companion/ui/InviteCompanionFlow.tsx:349-366` | Open — copy polish; «هیچ» keeps the summary explicit. |
| W6 | Done: teal check badge, «دعوت برای {name} فرستاده شد» / «کد دعوت آماده است», spouse lead + large family strip, note, «برگشت به همدم‌ها»: match. | — | — | match |

## Owner — Me hub row, privacy «دسترسی دیگران» (Me_Hub, Me_Privacy)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| P1 | Me hub «همدم‌ها و خانواده» row with companion names under «من و خانواده»; «فرزندان» «به‌زودی»: match (board subtitle «علی · مادر» is placeholder data). | — | — | match |
| P2 | Privacy adds a «مدیریت همدم‌ها» row (pending-invite count) under the per-companion «ریتمی همراه» rows; the board has only the companion row. | low | `frontend/src/screens/privacy/ui/PrivacyPage.tsx:205-260` | Open — keeps a path to `/companions` when every companion is pending; polish. |

## Male — onboarding partner code + linked (Onb_Partner, Onb_PartnerLinked)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| M1 | Partner step: «کد همدم» title, lead, six code boxes, hint, «با این کد چه چیزی می‌بینی؟» list, «شریکم هنوز ریتمی ندارد» row, privacy note, «اتصال» / «بعداً وصل می‌شوم», «رد کردن»: match in both themes. | — | — | match |
| M2 | The invite row icon is «share» (board: a chain link). | low | `frontend/src/screens/onboarding-flow/ui/PartnerStep.tsx:96`, `frontend/src/screens/companion-home/ui/CodeEntry.tsx:87` | Open — the row opens the Web Share sheet, so «share» describes the action; polish. |
| M3 | Linked: the companion's own avatar showed a generic person icon; the board draws the ♂ symbol in the data tone. | low | `frontend/src/screens/onboarding-flow/ui/PartnerLinkedStep.tsx:77` | **Fixed** (trivial) — `Icon name="male"` (already in `shared/ui/Icon.tsx`). |
| M4 | Linked: pair avatars, «به {name} وصل شدی», lead, «امروز {name}» card with phase pill and phase note, «ورود به پنل همدم»: match. | — | — | match |

## Male — companion home `/companion` (Hamdam_Home), `/companion/links`, Me hub

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| H1 | The phase pill («فاز لوتئال») and the access pills on the shared rows («ویرایش» / «فقط دیدن») were filled tinted 22 px `StatusPill`s with tone-coloured text; the board draws taller outlined pills (tone border, `text-1` ink) — visible on the hero and every shared row in both themes. | med | `frontend/src/screens/companion-home/ui/CompanionHomePage.tsx:96,159`, `frontend/src/app/globals.css:8634-8640` | **Fixed** — scoped `.nb-pill.cmh-phase-pill, .nb-pill.cmh-level`: 30 px, 12 px padding, transparent fill, 1.5 px `--nb-tone` 50 % border, `--text-1` text (tokens only, both themes via the tone vars). Before: `side-by-side/companion-home.before-fix.png`; after-shot pending (API down, see Environment). |
| H2 | Header «همدم {name}» / «سلام {me}», hero (day x of y, phase bar period→follicular→fertile→luteal with marker, legend, «پریود بعدی حدود …», phase note), shared list with «همین کارت بالا» and the not-shared line, «امروز چه کار کنی؟» tips with «انجام دادم», «برای خواندن» rail with «همه», «فرزند» card: match. | — | — | match |
| H3 | Phase bar is a template (period 5 d, fertile 5 d, ovulation −14); «انجام دادم» is local; the medication tip has no «یادآوری» button; child card is a «پروفایل کودک به‌زودی» placeholder (board: «آوا» + vaccine). | — | — | accepted (QUESTIONS #96; children B-N5-02) |
| H4 | Companion bottom nav امروز · خدمات · من (board draws no nav). Partner chips appear above the hero with more than one link. | — | — | as designed (B-N4-05, nav.md) |
| H5 | «افزودن دارو برای {name}» brand-soft pill under a shared section with edit (B-N4-06) — not on the board, which only offers «ویرایش». | — | — | as designed (B-N4-06 entry point) |
| H6 | No-link home (code entry card, «اتصال», invite row) and `/companion/links` (code entry, «به این افراد وصل هستی» rows with «خروج», «رفتن به پنل همدم»): no board — tokens, 44 px targets and both themes OK. Male Me hub hides Plus, life stage and cycle settings, adds «کد همدم». | — | — | match (no board) |

## Record-for — medication / appointment forms from the companion panel (Hamdam_RecordFor)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| R1 | «ثبت برای» row with «تغییر» at the top of the form; sheet with «خودم» / partner radio cards (bubbles, sub-lines), info note, «ثبت برای {name}» CTA: match in both themes. | — | — | match |
| R2 | Sheet title uses the shared `AppSheet` title (17 px bold) with × close; the board draws a 24 px display heading and no ×. A tall empty gap sits between the note and the CTA (sheet min-height). | low | `frontend/src/entities/companion/ui/RecordForSheet.tsx:77-122` | Open — shared sheet chrome; polish. |
| R3 | The picker opens on the full form (dose, form, times, weekdays) rather than the board's compact name/time/repeat form. | — | — | as designed (existing medication form, B-N4-06) |

## Admin-web — «نکته‌های همدم», «اتصال‌های همدم» (no board)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| A1 | Tips: phase tabs, locale tabs, phase note, ≤3 tips with reorder/delete, source badges, light + dark preview, save; Links: status counts by type, filters, masked owner/companion, grants count only: consistent with the admin design system in both themes. | — | — | match (no board; admin-web outside this task's `touches`) |

## Cross-cutting

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| X1 | Tokens only (no raw hex, no `--on-accent` on brand fills) in `entities/companion`, `features/invite-companion`, `screens/companion-*`; touch targets ≥ 44 px (chips are labels, not buttons); RTL via logical properties; LTR `en_companions` checked. | — | — | match |
| X2 | Avatars use the first letter of the profile name, so Latin dev names render Latin initials («C»). | — | — | data |
| X3 | Fake status bar / ↑ back arrow on boards. | — | — | accepted (QUESTIONS #30) |

## Open low items (not fixed)

N4: W5 summary «هیچ» rows · P2 privacy «مدیریت همدم‌ها» row · M2 invite row icon · R2 record-for sheet title / gap.
N3 (still open from audit-n3): L4 pregnancy/postpartum footer copy · D2 panel gear · H4 current-trimester ring ·
A2 share button on details · A3 phase legend dots · A4 IOM table row height.
N2 (still open from audit-n2): O5 menopause yes/no chip size · P2 manage row height · P3 paywall trial line style ·
A2 admin date filters.
N1 (still open from audit-n1): H7 fresh-user ring track · C4 phase sheet header row / note style · T2 TTC calendar
filter icon · P4 pregnancy-week source footer · M1 Me hub mode strip.

## Environment notes

- Go API :8020 down during the audit (waiting on the orchestrator's migrations 00028/00029 before 00030); Next dev
  :3000 up; admin-web :3001 down. No restarts were done. After-fix shots for H1 / M3 (`/companion`,
  `/onboarding/partner/linked` as 0990…551, light + dark) still to be taken once the API is back.
