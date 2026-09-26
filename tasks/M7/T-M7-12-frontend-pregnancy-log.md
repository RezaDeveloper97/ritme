---
id: T-M7-12
title: Frontend — pregnancy «ثبت علائم» (Log) v2 with offline outbox
milestone: M7
type: frontend
status: done
depends_on: [T-M7-08, T-M7-03]
parallel_group: M7-C
touches: [frontend/src/screens/pregnancy-log, frontend/src/shared/lib/outbox, frontend/src/app/[locale]/pregnancy/log]
skills: [new-fsd-slice, pwa]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M7-12 — Frontend — pregnancy Log v2

## Scope
1. `/pregnancy/log?date=` from `Log.dc.html`: header (close, date · week, other-day picker), 5 mood faces
   (radiogroup), 9 symptom toggles with per-symptom severity rows, water stepper (0–15), weight with last entry,
   visit note, spotting info box, sticky save with «ذخیره شد» state. Existing v1 weekly/movement tabs stay reachable
   (weekly checkup from Today).
2. `shared/lib/outbox`: IndexedDB queue for the day PUT when offline, replayed on reconnect (idempotent); pending
   badge. If this proves unreliable, switch the offline sentence to honest copy and note it in PROGRESS.
3. Alerts returned by the save appear inline with a link to the Alerts screen.

## Acceptance
- Light/dark × fa/en screenshots; offline save → reconnect → persisted (manual test documented); build green.
