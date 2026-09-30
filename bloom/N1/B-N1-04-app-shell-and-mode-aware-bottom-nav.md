---
id: B-N1-04
title: App shell and mode-aware bottom nav (امروز · حالت · + · خدمات · من)
milestone: N1
type: frontend
status: todo
depends_on: [B-N1-03]
parallel_group: N1-D
touches: [frontend/src/widgets/bottom-nav,frontend/src/app/[locale],frontend/src/shared/config,frontend/CLAUDE.md,frontend/src/screens/services]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-04 — App shell and mode-aware bottom nav (امروز · حالت · + · خدمات · من)

## Why
The new nav adds «خدمات» and makes the second tab depend on the life-stage mode.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Home.dc.html` (+ `nbd_Cycle_Home`)
- `docs/design/night-bloom/g-me-settings/nbl_Me_Hub.dc.html` (+ `nbd_Me_Hub`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_Main.dc.html` (+ `nbd_v17_Main`)

## Scope
- Tabs: امروز · <mode tab> · FAB(+) · خدمات · من. Mode tab: cycle→تقویم, ttc→باروری, pregnancy→بارداری, postpartum→کودک, menopause→علائم, teen→تقویم. FAB opens the mode's log sheet.
- New `/services` route with a placeholder hub (real hub in B-N7-01) so the tab works now.
- Safe-area, active states and badge support per artboards.
- (B-N1-01) Spec in `docs/night-bloom/nav.md`: solid FAB (drop gradient/goo), glass pill geometry, nav shown only on tab roots and first-level hubs, hidden on back-header screens/forms/sheets; the `/cycle` tab leaves the nav.
- (B-N1-01) FAB opens `?sheet=log` (full `AppSheet`) for the mode; `/log` stays as a route. Postpartum «کودک» → `/children/[id]` for one child else `/children`; menopause «علائم» → `/analysis/symptoms` (placeholder until N3); male companion nav has no FAB/mode tab (N4).
- (B-N1-01) There is no `(app)` route group today — routes live under `frontend/src/app/[locale]/`; add the group or place `/services` there. Update `frontend/CLAUDE.md` §4.1 (routes vs sheets) to match the route map in `docs/night-bloom/routes.md`.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
