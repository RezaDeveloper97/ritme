---
id: T-M10-15
title: Frontend generic alerts inbox
milestone: M10
type: frontend
status: done
depends_on: [T-M10-08, T-M10-09, T-M10-14]
parallel_group: M10-W
touches: [frontend/steiger.config.ts, frontend/src/shared/i18n/bundled.ts, frontend/src/global.d.ts, docs/engine-v6/qa/T-M10-15, frontend/src/screens/alerts, frontend/src/entities/alert, frontend/src/screens/pregnancy-alerts, frontend/src/app/[locale]/alerts, frontend/src/app/[locale]/pregnancy/alerts, frontend/src/app/message-scopes.ts, frontend/src/screens/home/ui/HomePage.tsx, frontend/messages/fa/alerts.json, frontend/messages/en/alerts.json, backend-go/resources/translations/fa/alerts.json, backend-go/resources/translations/en/alerts.json]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- alerts
---

# T-M10-15 — Frontend generic alerts inbox
**Model:** sonnet for the UI; the main session reviews health copy and the urgent-alert behaviour.

## Why
SPEC §5.4.1: one «هشدارها و پیام‌ها» screen for every mode, from the inbox API (T-M10-07), with acknowledge, add to
visit note, urgent alerts that cannot be closed before acknowledgement, and history. Today only pregnancy has an alerts
screen (`screens/pregnancy-alerts`).

## Scope
1. `entities/alert`: list (open/history, cursor), ack, visit-note, dismiss; boundary parser; query keys.
2. `screens/alerts` (route `app/[locale]/alerts`): sections by level (urgent → follow_up → suggestion → info), each card
   with «چی دیدیم» and «چقدر مطمئنیم» (evidence), «دیدم، ممنون», «افزودن به یادداشت ویزیت», a call button on urgent
   cards; urgent cards have no close action until acknowledged; a history tab. Reuse the pregnancy alerts design.
3. `app/[locale]/pregnancy/alerts` redirects to `/alerts` (keep the URL working).
4. Entry point: a bell with the open/urgent count on the home header (the feed's `alerts` counts), shown only when
   there is at least one open item and the module is on.
5. Copy in a new `alerts` namespace (fa/en), registered in `message-scopes.ts`; sync to the backend translations.

## Tests
- Parser and view-model tests (ordering, evidence rendering, urgent close rule).
- Redirect test for the old route.
- Visual QA light and dark: urgent + follow_up + info + history, saved under `docs/engine-v6/qa/`.

## Out of scope
- Family member alerts (M11).

## Acceptance
- Verify passes; screenshots reviewed by the main session.
