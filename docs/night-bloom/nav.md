# Night & Bloom — bottom nav spec per mode (B-N1-01)

Implemented by **B-N1-04** (`frontend/src/widgets/bottom-nav`). Source artboards: `Cycle_Home`, `Cycle_Calendar`,
`Me_Hub`, `v17_Main`, `PregFull_Main/Week`, `v15_Main`, `v16_ChildHome`, `v19_Main`, `v19_TTC_Calendar`.

## Geometry

- Floating pill: `position: fixed; bottom: 24px (+ safe-area-inset-bottom); inset-inline: 14px; height: 70px;
  padding: 0 10px; border-radius: 35px; background: var(--surface-glass); border: 1px solid var(--line);
  box-shadow: var(--shadow-float); backdrop-filter: blur(10px)`.
- Tab: `flex: 1`, 56px tall, column, gap 4, icon 22px stroke 1.8, label 11px.
  Inactive: `--text-2`, weight 600. Active: `--text-1`, weight 800, `aria-current="page"`, plus a 16×3px
  `--brand` bar (radius 2) at `bottom: 2px`.
- FAB (centre): 56px circle, `--brand-fill`, `--shadow-fab`, plus icon 24px stroke 2.4 in `--on-brand`, `aria-label="ثبت"`.
  Solid — the old gradient FAB and the goo animation are not in the design.
- DOM order = RTL visual order: امروز · mode tab · FAB · خدمات · من.
- Icons: home (امروز), calendar (تقویم), 2×2 grid (خدمات), person (من); mode tabs: heart (بارداری), baby/child (کودک),
  sparkle/drop (باروری) — copy the SVG paths from the artboards.

## Per mode

| Mode | امروز → | Mode tab (label → route) | FAB opens | خدمات | من |
|---|---|---|---|---|---|
| cycle | `/home` | تقویم → `/calendar` | `?sheet=log` (Log_Sheet_Cycle) | `/services` | `/profile` |
| ttc | `/home` (TTC tiles) | باروری → `/calendar` (TTC variant) | `?sheet=log` (cycle sheet, TTC tiles) | `/services` | `/profile` |
| pregnancy | `/pregnancy` | بارداری → `/pregnancy/weeks` | `?sheet=log` (Log_Sheet_Preg) | `/services` | `/profile` |
| postpartum | `/postpartum` | کودک → `/children/[id]` (1 child) or `/children` | `?sheet=log` (Log_Sheet_Post) | `/services` | `/profile` |
| menopause *(no artboard)* | `/home` (menopause copy) | علائم → `/analysis/symptoms` | `?sheet=log` (cycle sheet, menopause tiles) | `/services` | `/profile` |
| teen *(no artboard)* | `/home` (simplified) | تقویم → `/calendar` | `?sheet=log` | `/services` (no shop/ads) | `/profile` |
| male companion | `/companion` | — | — (no FAB) | `/services` | `/profile` |

Menopause/teen/companion rows are defaults (see [gaps.md](gaps.md), `bloom/QUESTIONS.md`); the roadmap queue
(`roadmap/E02-meno`, `E09-teen`, `CB-NAV-03`) later replaces the minimal versions.

## Where the nav is shown

Artboards show the nav only on **tab roots and first-level hubs**: home variants, calendar, `/cycle`-less mode tab
screens, `/services` hubs (`v17_Main`, `v18_AssistantHub`, `Vitals_Hub`), Me hubs (`Me_Hub`, `Learn_Hub`,
`Todo_Home`) and `/analysis` hubs. It is **hidden** on every screen with a ScreenHeader back button, onboarding,
Plus checkout, forms and sheets. Active-tab mapping: Services-hubs → خدمات; `Learn_Hub`, `Todo_Home` → من;
`/analysis` → the mode tab (as `An_Hub` shows «تقویم» active).

## Artboards whose nav deviates (ignore → use the table above)

| Deviation | Artboards | Rule |
|---|---|---|
| «تحلیل» as 4th tab instead of «خدمات» | `Cycle_Home_Near`, `Cycle_Home_During`, `An_Hub_TTC`, `An_Hub_Preg`, `An_Hub_Post`, `v19_Preg_Home`, `Lab_Intro`, `v13_Preg_Home`, `LearnEntry_HomeContinue`, `LearnEntry_HomeNew`, `Learn_Downloads`, `Learn_Profile`, `Todo_Empty`, `Todo_List`, `nbd_User_Library` | older generation with placeholder links (`Main.dc.html`, `#`); use خدمات |
| «تقویم» as mode tab outside cycle mode | `An_Hub_TTC/Preg/Post`, `v19_Preg_Home`, `v13_Preg_Home` | use the mode's tab |
| mode tab active on analysis | `An_Hub` («تقویم» active) | analysis lives under the mode tab |
| «باروری» on a checkups home | `v14_Main` | it is the TTC home with the checkups card — correct for ttc |
| 4 tabs, no FAB, «آموزش» tab | `LearnEntry_TabOption` | rejected option B (decision: Option A) — never build |
| own 4-tab nav داشبورد · محتوا · گروه‌ها · شاگردان, no FAB | `Ins_Dashboard`, `Ins_Content`, `Ins_Groups`, `Ins_Students` | instructor-web app nav (B-N8-05/06) |
| no nav drawn | `Hamdam_Home` | companion default above |

Today's nav (`nav-items.ts`: امروز · تقویم · + · چرخه(`/cycle`) · پروفایل, plus a pregnancy variant) loses the
`/cycle` tab: cycle history moves to `/cycle` reached from home («پیش‌بینی‌ها · جزئیات») and the calendar.
