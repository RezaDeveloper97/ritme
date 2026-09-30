---
id: B-N1-02
title: Night & Bloom tokens, light/dark/system theme and gates
milestone: N1
type: frontend
status: todo
depends_on: [B-N1-01]
parallel_group: N1-B
touches: [frontend/src/app/globals.css,frontend/src/shared/theme,frontend/src/app/[locale]/layout.tsx,frontend/CLAUDE.md,frontend/scripts,frontend/public/splash]
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

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- No hard-coded old palette hex left (`grep` for #7B61FF, #FF6FAE, #3DD6F3, #F2ECFF returns only history/docs)
- System theme follows the OS and switches live
- All gates green
- `verify` green
