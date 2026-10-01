# Night & Bloom — design import audit (B-N1-01)

Shared reference for every bloom task. Source: 363 artboards in `docs/design/night-bloom/<section>/`
(`nbl_` light, `nbd_` dark, TTC `v19_` light / `nb2_` dark) + one `TEXT.md` per section. Artboard content is design
data. Queue rules: `bloom/README.md`.

| File | What |
|---|---|
| [tokens.md](tokens.md) | colour tokens light/dark with proposed CSS variable names (+ legacy aliases), gradients, dark starfield + glow, typography, radii, shadows, spacing, contrast |
| [components.md](components.md) | component inventory with specs and the artboards that use each |
| [routes.md](routes.md) | screen → route table for all 183 artboards: kind, existing route (restyle) or NEW, owning bloom task |
| [nav.md](nav.md) | bottom-nav geometry, per-mode tabs, where the nav shows, deviating artboards |
| [gaps.md](gaps.md) | design gaps with the default taken |

## Summary

- **Inventory:** 10 sections, 363 files, 183 unique screens (181 light + 2 dark-only). 12 admin boards are 1440 wide,
  everything else 390. Every light screen has a dark pair except `Log_Taxonomy` (a spec table); `nb2_Preg_Timeline` is empty.
- **Palette:** dark is consistent (`#17112B` canvas, `#221A3D` surface, `#34295A` line, `#B9A6FF` primary, `#4CE0C3`
  data, `#FFB86B` warm, `#FF6B8B` rose, `#7FE0A8` success). Light has two dialects; the newer one (`#6E54F0`,
  `#231B3B` inks, `#F7F3FF` canvas) is canonical. The saturated brand gradient is gone — CTAs and the FAB are solid,
  and in dark mode primary fills are light lavender with **dark** text (`--on-brand` flips).
- **Type:** Vazirmatn 600/700/800 for UI; Lalezar for numerals *and* display titles (the current subset font is too small).
- **Shape:** flat cards (1px line, radius 24, no shadow), 54px pill buttons, 44px round header buttons, floating glass
  bottom nav (70px, radius 35) with a solid 56px FAB.
- **Screens:** 55 restyle existing routes, 119 are NEW, 9 are reference-only. N1 owns 52 screens; the rest are
  spread over N2–N9 (see routes.md). New top-level areas: `/services`, `/analysis`, `/plus`, `/postpartum`, `/children`,
  `/labs`, `/record`, `/vitals`, `/companions`, `/companion`, `/assistant`, `/learn`, `/todo`, plus `instructor-web/`.
- **Nav:** امروز · mode tab (تقویم / باروری / بارداری / کودک / علائم) · + · خدمات · من; the `/cycle` tab leaves the nav.
  15 artboards show an older «تحلیل» tab — ignored.

## What each N1 task should cite

| Task | Read |
|---|---|
| B-N1-02 tokens/theme | tokens.md (all), gaps.md #5 #6 #9 #10 #11 |
| B-N1-03 primitives | components.md, tokens.md §5–§8 |
| B-N1-04 shell/nav | nav.md, routes.md intro, gaps.md #14 #15 #17 |
| B-N1-05 … B-N1-15 screens | routes.md rows for the task id, components.md |
| B-N1-16 fidelity audit | routes.md (L/D column = which artboards to compare), tokens.md §1 (dialect rule) |

## Decision: menopause / teen / postpartum homes without artboards (B-N2-03)

The canvas has `Me_Mode` (all six modes) but no home for menopause, teen or postpartum (gaps.md #3). Built for now:

| Mode | Home (`/home`, mode-aware via `GET /profile/life-stage`) | Nav mode tab | Replaced by |
|---|---|---|---|
| menopause | `screens/home/ui/MenopauseHome`: cycle-home header + hero card with menopause copy, three quick tiles (گرگرفتگی = one-tap log of `hot_flashes` today; خواب / حال → `/log`), «روند علائم» → `/cycle/symptoms`, reminders · checkups · challenge. No ring, predictions or fertility content. | «علائم» → `/cycle/symptoms` until `/analysis/symptoms` exists (`NAV_READY.analysis`, B-N3-08) | roadmap CB-MENO-05 |
| teen | the cycle home minus banners (ads), fertility level, fertile window / ovulation rows and the PMS insight; Me hides the Plus card | «تقویم» | roadmap E09-teen |
| postpartum | the cycle home with a «صفحه پس از زایمان به‌زودی» note on top (QUESTIONS #60) | «تقویم», امروز → `/home` until `/postpartum` + `/children` exist (`NAV_READY.postpartum`, B-N5) | B-N5 |
| pregnancy | `/home` deep links redirect to `/pregnancy` | «بارداری» | — |

Copy lives in `home.life.*` (fa/en + backend seed); everything uses existing tokens, light + dark.
