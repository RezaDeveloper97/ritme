---
id: T-M10-14
title: Frontend home reads the message feed
milestone: M10
type: frontend
status: done
depends_on: [T-M10-07, T-M10-10, T-M9-47]
parallel_group: M10-W
touches: [frontend/src/entities/message, frontend/src/screens/home, frontend/src/widgets/smart-tip, frontend/src/widgets/today-challenge, frontend/src/widgets/for-today, frontend/messages/fa/home.json, frontend/messages/en/home.json, backend-go/resources/translations/fa/home.json, backend-go/resources/translations/en/home.json, backend-go/internal/i18n/testdata, backend-go/contract/golden/public, docs/engine-v6/qa/T-M10-14]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- home
---

# T-M10-14 — Frontend home reads the message feed
**Model:** sonnet for the UI; the main session writes or reviews every health text (`docs/copy-fa/STYLE.md`,
GLOSSARY, warm «تو»).

## Why
The home renders the status banner, smart tip, today's advice, today's challenge and the urgent modal from separate
sources (legacy `/messages/daily`, client rules, the legacy challenge pick). SPEC §5.4 makes them slots of one engine,
served by `GET /messages/feed` (T-M10-06). Production still runs Laravel, which has no feed, so the old path must keep
working. DECISIONS.md M-14, M-15; D-168.

## Scope
1. `entities/message`: `fetchFeed(slots)` with a boundary parser (unknown slots and fields ignored), `postMessageEvent`
   (clicked/dismissed/goal), query keys; the feed query joins the cycle-write invalidation map (T-M9-41) and refreshes
   after a dismiss or an alert acknowledgement.
2. Home:
   - status banner: copy and CTA from `status_banner` when present, else the built-in copy of T-M9-42 (M-14);
   - smart tip from `smart_tip`; «توصیه‌های امروز» (up to 3) from `daily_advice`; today's challenge from
     `daily_challenge` (same widget design; legacy pick when empty);
   - urgent modal from `urgent_modal`: a call button (115 or the doctor), cannot be closed until «دیدم» calls
     `POST /alerts/{id}/ack`;
   - in-app modal from `modal` (dismissible).
3. Events: `clicked` on a CTA, `dismissed` on close; never `shown` (the server logs it).
4. Fallback: feed 404 (Laravel), network error or offline → exactly today's behaviour (built-in copy, legacy
   endpoints). The home stops calling `/messages/daily` only when the feed answered.
5. Module kill-switches respected through `entities/app-module`.
6. Copy keys in `messages/{fa,en}/home.json`; sync with `npm run copy:sync -- home`; copy the en file to
   `backend-go/resources/translations/en/`.

## Tests
- Parser tests (full feed, empty slots, unknown slot).
- Home model: banner source precedence; feed 404 → legacy path; offline → legacy path.
- Urgent modal: cannot close before ack; ack posts and closes.
- Event posting on CTA and close.
- Visual QA light and dark: feed with every slot filled, and feed 404 (local Go and a forced 404), saved under
  `docs/engine-v6/qa/`.

## Out of scope
- The alerts inbox screen (T-M10-15).

## Acceptance
- Verify passes; screenshots reviewed by the main session; PROGRESS.md cites D-168.
