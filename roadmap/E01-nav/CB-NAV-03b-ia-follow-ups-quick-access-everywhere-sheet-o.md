---
id: CB-NAV-03b
title: IA follow-ups: quick access everywhere, sheet order, board copy
epic: NAV
type: frontend
status: done
depends_on: [CB-NAV-03]
parallel_group: NAV-C
touches: [frontend/src/widgets/quick-access,frontend/src/widgets/log-later,frontend/src/screens/postpartum,frontend/src/screens/ivf,frontend/src/features/log-day]
skills: [new-fsd-slice]
boards: [nbd_Nav_Today.dc.html,nbd_Nav_Plus.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test
---

# CB-NAV-03b — IA follow-ups: quick access everywhere, sheet order, board copy

## Why
CB-NAV-03 aligned the shell with the canvas IA rules but kept scope tight; these items were left over.

## Boards
- `nbd_Nav_Today.dc.html`, `nbd_Nav_Plus.dc.html` (snapshot in `docs/design/canvas-v1/boards/`)

## Scope
1. «دسترسی سریع» + «برای امروز» on the postpartum (`/postpartum`) and IVF (`/ivf`) homes (CB-NAV-03 put them on cycle/TTC, menopause, and pregnancy only; the teen home has neither by design).
2. Quick-access shortcuts whose screens don't exist yet: set `href` in `widgets/quick-access/model/shortcuts.ts` for «پزشک» (doctor directory, B-N7 / CB-DIR front end), «دفترچه درد» (condition program pain diary, CB-COND-03+) and «دوره‌ها» (courses, B-N8 front end) when they ship. The doctor tile stays hidden for teens (IA_Nav).
3. + sheet order: the board puts «برای بعد» right under the quick tiles; CB-NAV-03 appends it after the full category list (minimal change to bloom's LogDay). Move it if the product agrees.
4. Companion suggestion row in «برای امروز» («… پیشنهاد داد · از همدم») once bloom N4 exposes companion suggestions to the owner.
5. Quick-access persistence on the server (README §8 Q3) if the user chooses that over on-device storage.
6. Dead keys after CB-NAV-03: `postpartum.home.modeLabel` (now the mode chip).

## Out of scope
- Android/native; anything owned by CB-DIR / CB-COND / B-N8.

## Acceptance
- Screens match `nbd_Nav_Today` / `nbd_Nav_Plus` in light + dark; rows in `docs/qa/canvas/nav.md`.
- `verify` green.
