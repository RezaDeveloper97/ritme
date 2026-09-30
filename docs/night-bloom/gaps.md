# Night & Bloom — design gaps and defaults (B-N1-01)

Each gap has the default taken so work can continue. Rows marked **Q** are also in `bloom/QUESTIONS.md` for the user.

| # | Gap | Default taken | Owner |
|---|---|---|---|
| 1 | `nb2_Preg_Timeline` (dark) is **empty**: only the title «بارداری · مسیر». The whole `nb2_/v19_ Preg_*` set is pregnancy v1. | Don't build v1. Pregnancy uses `PregFull_*` (light + dark). If a timeline view is needed, take `v19_Preg_Timeline` layout with dark tokens. `Preg_Onboard` is only a copy reference for the TTC → pregnancy prompt. | B-N1-14 |
| 2 | `nbd_User_Library` / `nbd_User_Player` are **dark-only** (in `a-start-plus/`), with an old nav. | Superseded by `Learn_Hub` and `Learn_Video` / `Learn_Audio` (both themes). Use them only as a dark cross-check. | B-N8-03 |
| 3 | **No menopause or teen home artboards.** | Menopause: cycle-home layout with menopause copy + symptom tiles (hot flash, sleep, mood), no fertility/predictions, mode tab «علائم» → `/analysis/symptoms`. Teen: simplified cycle home, tab «تقویم», no ads/shop/Plus upsell. Full versions come from `roadmap/E02-meno` (CB-MENO-05 replaces the minimal home) and `roadmap/E09-teen`. **Q** | B-N2-03 |
| 4 | **No doctor-side UI.** | Doctors answer chat / write prescriptions from admin-web under a `doctor` role (B-N7-08), styled with the admin Night & Bloom shell (B-N9-01). No separate doctor app. | B-N7-08 |
| 5 | Two light dialects (`#6E54F0`/`#231B3B` on 16 hub screens vs `#7B61FF`/`#2F2F35` on 165). | Dialect A canonical; fidelity audits treat B values as A. See [tokens.md §1](tokens.md). **Q** | B-N1-02 |
| 6 | The saturated purple→pink gradient (CTAs, FAB) is gone; `frontend/CLAUDE.md` §10.2 still mandates it. | Solid `--brand-fill` for CTAs and FAB; gradients only as soft tints. B-N1-02 rewrites §10.2. | B-N1-02 |
| 7 | **No entry point to `/analysis`** in any tab-root artboard (the older nav had a «تحلیل» tab; `An_Hub` marks the mode tab active). | Reach `/analysis` from home cards («در تحلیل ببین», predictions «جزئیات»), a header icon/segment on the mode-tab screen (calendar: ماه · سال · تحلیل), and a Me row. Matches `roadmap/E01-nav/CB-NAV-03` ("analysis next to its data inside the stage tab"). **Q** | B-N3-08 |
| 8 | Male companion home (`Hamdam_Home`) has **no nav drawn**. | امروز (`/companion`) · خدمات · من, no FAB, no mode tab. **Q** | B-N4-05 |
| 9 | Theme default: bloom decisions add "follow-system", but `shared/theme/store.ts` deliberately has no `system` and defaults to light. | Fresh installs default to `system`; a stored `light`/`dark` is kept. **Q** | B-N1-02 |
| 10 | `Lalezar-Subset.woff2` covers digits + «روز/امروز» only, but artboards set 159 Persian display titles in Lalezar. | Ship a full Arabic-script + Latin Lalezar subset (`font-display: swap`, preload only on screens that use it). | B-N1-02 |
| 11 | Danger and period share the rose ramp in the design (`#E8436F`/`#C42D57`), while §10.2 says "red = menstruation only". | Keep the design; distinguish danger by icon + copy + `--danger-soft` surface, never by a period marker shape. | B-N1-02 |
| 12 | `Log_Taxonomy` is light-only and is a spec table, not a screen. | Data spec for B-N3-01; no dark counterpart needed. | B-N3-01 |
| 13 | `LearnEntry_TabOption` (4 tabs, «آموزش» tab). | Rejected (decision «Learning entry: Option A»). Not built. | — |
| 14 | Older artboards link to placeholders (`Main.dc.html`, `#`) and show «تحلیل» in the nav. | Ignore links; nav per [nav.md](nav.md). | B-N1-04 |
| 15 | Postpartum mode tab «کودک» with 0 or several children. | 1 child → `/children/[id]`; 0 or ≥2 → `/children`. | B-N1-04 / B-N5-05 |
| 16 | `Onb_Cycle` and `Onb_Health` merge several current onboarding routes into one screen each. | New `/onboarding/cycle` and `/onboarding/health`; old routes redirect. | B-N2-02 |
| 17 | Log entry: design opens a full log **sheet** from the FAB; today the FAB routes to `/log`. | FAB → `?sheet=log` (full `AppSheet`); `/log` stays as the day-log route reached from home «ثبت امروز» until B-N3-03 makes it the same content. | B-N1-04 / B-N3-03 |
| 18 | Admin artboards are desktop 1440 wide; instructor artboards are mobile 390. | admin-web desktop-first (responsive down to tablet); instructor-web mobile-first, responsive up. | B-N9-01 / B-N8-05 |
| 19 | The 54px status bar (`۹:۴۱`) and sample dates (مهر ۱۴۰۵) are artboard mocks. | Never build/hard-code. | all |
