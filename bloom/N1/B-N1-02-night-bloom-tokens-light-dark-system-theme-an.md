---
id: B-N1-02
title: Night & Bloom tokens, light/dark/system theme and gates
milestone: N1
type: frontend
status: done
depends_on: [B-N1-01]
parallel_group: N1-B
touches: [frontend/src/app/globals.css,frontend/src/shared/theme,frontend/src/app/[locale]/layout.tsx,frontend/CLAUDE.md,frontend/scripts,frontend/public/splash,frontend/src/app/fonts,frontend/src/app/manifest.ts,frontend/package.json]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-02 — Night & Bloom tokens, light/dark/system theme and gates

## Why
The whole app moves to the Night & Bloom palette in both themes; the current purple→rose tokens retire.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Home.dc.html` (+ `nbd_Cycle_Home`)

## Scope
- Replace the colour tokens in `globals.css` with the N1-01 token table for `[data-theme=light]` and `[data-theme=dark]`; keep token *names* stable where possible so screens restyle for free.
- Theme store: `light | dark | system` (default `system`, follows `prefers-color-scheme` live).
- Dark background: starfield + radial glow as a reusable layer (CSS only, no images), disabled under reduced motion if animated.
- Update `frontend/CLAUDE.md` §10.2/§10.3 (palette rules, theme-color meta) and the `lint:styles` / `lint:dark` scripts + baselines so the gates enforce the new palette.
- Regenerate iOS startup images / theme-color if they encode old colours.
- (B-N1-01) Canonical light palette is dialect A (`#6E54F0` primary, `#231B3B`/`#5E5873`/`#6A6480` inks, `#F7F3FF` canvas); implement the table + legacy aliases in `docs/night-bloom/tokens.md` §2–§3.
- (B-N1-01) Add `--on-brand` (white light / `#17112B` dark) for text on primary fills — primary fills are light lavender at night; update `THEME_STABLE` in `scripts/check-dark-mode.mjs`.
- (B-N1-01) Add radius, shadow, type-scale and spacing tokens (tokens.md §5–§8); solid-fill CTAs/FAB — retire the brand-gradient rule in `frontend/CLAUDE.md` §10.2 (gradients only as soft tints).
- (B-N1-01) Replace `src/app/fonts/Lalezar-Subset.woff2` with a full Arabic-script + Latin Lalezar subset (display titles use it, not only digits); `theme-color`/`manifest.ts` colours → `#F7F3FF` / `#17112B`.
- (B-N1-01) Fresh installs default to `system`; a stored `light`/`dark` preference is kept (see `bloom/QUESTIONS.md`).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- No hard-coded old palette hex left (`grep` for #7B61FF, #FF6FAE, #3DD6F3, #F2ECFF returns only history/docs)
- System theme follows the OS and switches live
- All gates green
- `verify` green
