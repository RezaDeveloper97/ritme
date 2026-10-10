# N7 design-fidelity audit (B-N7-09)

Every N7 screen next to its artboard (light + dark, 390 px) after B-N7-01 … B-N7-08:
`docs/design/night-bloom/d-doctor-assistant/nb{l,d}_v17_*` (Main, Doctors, DoctorProfile, ReviewBook, Booked, Chat) and
`nb{l,d}_v18_*` (AssistantHub, DeptIntro, AIChat, Summary, Handoff, History, Profile). The admin-web telemed and
assistant-department screens have no artboard of their own — they were judged against the N9 admin theme
(`h-admin/nb{l,d}_Admin_*`). Artboards rendered locally with `shot.mjs --files`.

**Method — real data, not mocks.** A fresh DB `ritme_n709` on the test MariaDB (goose via `RUN_MIGRATIONS`), the Go API
on :8031 (fake payment / AI / video providers) and Next dev on :3061 (`.next-n709`), admin-web on :3062. Seeded: two
doctors (a doctor and a midwife) with visit types and weekly availability, a doctor admin account linked to doctor 1,
four bookings (one failed, two confirmed paid through the fake gateway, one completed), a doctor chat with two doctor
lines, a patient line and an e-prescription item, two assistant conversations (a normal reply from the fake AI and an
emergency one that produced the 115 card), consents + assistant profile. DB, Redis keys, storage and dist dirs removed
at the end. Side-by-sides (board left · app right): `docs/qa/bloom/B-N7-09/side-by-side/`; admin shots `…/admin/`.

Data-driven differences (names, prices, counts, dates, doctor photos, the data-shaped consent defaults) are not
deviations. Deliberate defaults in `bloom/QUESTIONS.md` are accepted (#119 hub sections from admin catalog, #120/#125
booking rules, consent default off, #127 admin console). Boards draw a fake status bar, ↑ back arrow and mic button —
accepted. Next's dev badge (round «N», bottom-left) hides part of the composer/nav in the shots; it is not in the app.

Severity: **high** = broken / misleading; **med** = visible departure from the board, wrong or missing control;
**low** = polish. Totals: **high 0 · med 3 · low 13** — fixed: 2 med; **1 med + 13 low open**; 2 environment notes.

## Services hub — `/services` (v17_Main)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| S0 | Search, upcoming-booking card («ویزیت ویدیویی · …», date tile, «جزئیات»), care tiles, programs strip, mother & child map card, learning, shop, 115 card, nav: match in light and dark. | — | — | match |
| S1 | In a fresh DB the «دستیار سلامت» and «پزشک و ماما» tiles show «به‌زودی» because the `services_care` catalog rows have no `meta.href` yet (admin step in PROGRESS B-N7-05 / B-N7-07). | env | catalog `services_care` | Environment note — data, not code; set `assistant → /assistant`, `doctors → /services/doctors` in the catalog editor before release. |
| S2 | Tile glyphs differ slightly (folder vs document, pulse vs heart on vitals); programs strip lacks the «همه» link; shop section shows «ورود» vs «به‌زودی» by data. | low | `entities/service-hub` | Open — polish. |

## Doctors — list, profile, book, booked, bookings, chat

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| D0 | List (mode tabs, specialty chips, cards with rating, mode pills, first slot, price, ask-assistant card), profile (summary tiles, visit-type radio cards, day strip + slot grid, reviews, sticky CTA), review & book (doctor card, for-whom, reason chips, share toggles, price card, pay + cancel note), booked (check disc, actions, share toggles, manage), bookings list, return page: match in both themes. | — | — | match |
| D1 | **Chat bubbles on the wrong side.** The patient's purple bubbles were at the start edge (right in RTL) and the doctor's white ones at the end edge; the board is the reverse (mine at the left, doctor at the right) and the assistant chat of the same slice already follows the board. | med | `globals.css` `.dr-msg.is-mine/.is-theirs` | **Fixed** — swapped (`flex-end` / `flex-start`, logical so LTR flips too). `side-by-side/doctor-chat.{light,dark}.png` (before: `doctor-chat.before.light.png`). |
| D2 | **Chat header mirrored.** DOM order video → name → avatar → back put the back button on the far left and the video button on the right; the board has back (right), avatar, name + status, video (left). | med | `screens/doctor-chat/ui/DoctorChatPage.tsx:108` | **Fixed** — back, avatar, id, video. Same screenshots. |
| D3 | Prescription shows as a separate card under the thread; the board puts it as a card inside the doctor's bubble; patient-sent report PDF card and attach button absent (`attach` shown only with a file — composer has no attach). | low | `DoctorChatPage.tsx` | Open — layout choice; attachments accepted by API. |
| D4 | Profile: satisfaction tile is warm orange, board green; day strip has no disabled/greyed days; sticky bar label «یک وقت انتخاب کن» before a slot (board always shows a chosen slot). | low | `screens/doctor-profile` | Open — polish / state. |
| D5 | Doctor card avatar is a pink→lavender gradient disc; board uses a solid per-doctor colour. Rating sits above the mode pills on the left (board: under the avatar). | low | `entities/doctor/ui` | Open — polish. |
| D6 | Review & book: no reason pre-selected, note is a textarea (board: single-line field), share toggles default off (board: first two on). | low | `features/book-visit` | Open / accepted — consent default off is deliberate (QUESTIONS). |
| D7 | Booked page title is «تأیید شد» in a small heading; the board's «نوبتت ثبت شد» is a display title, and the «آماده‌شدن برای ویزیت» row is missing. | low | `screens/doctor-booked` | Open — polish; detail page doubles as a booking view. |
| D8 | Header of list / profile is larger than the board's (title row + back circle ~44 px vs 36). | low | shared `HeaderButton` | Open — project-wide. |
| D9 | `specialty.title` is null for a fresh DB (catalog group `telemed_specialties` not seeded); the card falls back to the headline, admin list shows the raw code («gynecologist»). | low | catalog | Environment note — seed the specialty labels in the catalog. |

## Assistant — hub, department, chat, summary, handoff, history, profile

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| A0 | Hub (hero ask field, quota/Plus card, profile card with chips, department tiles, recent chats with triage pill, 115 card, disclaimer), department intro (not-for list, start CTA), chat (user bubble at the left, assistant card at the right with sparkle, emergency 115 card, disclaimer, composer), summary, handoff, profile toggles: match in both themes. Emergency card (red tint, «تماس با ۱۱۵») verified from a real emergency message in both themes. | — | — | match |
| A1 | **History** is a flat list of assistant chats; the board has «همه / دستیار / پزشک» tabs and one timeline that merges assistant chats with doctor visits, plus the profile card and «خروجی PDF برای پزشک». | med | `screens/assistant-history` | **Open** — needs a merged feed (bookings + conversations) and a PDF export; both are new features rather than a styling fix. Delete one / delete all and the profile-sharing card already work. Proposed follow-up task. |
| A2 | Hub: no gradient greeting card with the user's name or mic button; the ask field is a plain card. | low | `screens/assistant-hub` | Open — polish. |
| A3 | Chat: no mic button in the composer; quick-reply chips and the numbered guidance list of the board are plain text (the fake AI returns neither). Triage level is not highlighted when the model sent no `[[triage]]` line. | low | `features/assistant-chat` | Open — data-dependent; chips render when the model sends them. |
| A4 | Summary: «آنچه گفتی» is the raw last answer instead of bullets; self-care checklist and profile chips appear only when the model supplies them. | low | `screens/assistant-summary` | Open — data-dependent. |
| A5 | Dark: the «این بخش چه کاری نمی‌کند» card on the department intro is an olive-brown warning mix; the board is a softer amber. | low | `screens/assistant-dept` | Open — token mix (same as audit-n6 V10). |
| A6 | Handoff lists bookings as outlined mode pills instead of the board's radio doctor cards; consent checkbox is a bare square. | low | `screens/assistant-handoff` | Open — polish. |

## admin-web — telemed and assistant departments (no board; Night & Bloom admin theme)

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| W0 | Doctors list (status tabs, search, table with mode pills / status chips, pager), doctor form (language tabs fa/en with «پیش‌فرض», photo drop zone, visit types, weekly schedule, time off), bookings & refund queue, reviews, departments editor (list + empty «یک بخش را انتخاب کن») and the nav group «پزشکان و تله‌مدیسین»: consistent with the theme in light and dark; RTL correct (select chevrons, breadcrumb). | — | — | match |
| W1 | Doctors table shows the specialty as the raw code when the catalog has no label (see D9). | low | `admin-web/src/screens/telemed` | Open — falls back to code on purpose. |
| W2 | The doctor form is one long page (≈6 000 px at 1440); sections could be collapsible. | low | `admin-web/src/screens/telemed` | Open — polish. |

## Other

- `frontend/src/features/app-lock/model/controller.test.ts` «after 10 wrong passcodes …» timed out at 5 s on a loaded
  machine (≈12 real PBKDF2 runs): the test now has a 30 s timeout — **fixed**.
- Dev tooling note: `next dev` appended `.next-n709/types/**` to `frontend/tsconfig.json`; removed again at the end.
