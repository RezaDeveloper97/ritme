---
id: T-M11-18
title: Frontend member acceptance: code preview, accept or decline, mutual answer, pending phone invites
milestone: M11
type: frontend
status: blocked
depends_on: [T-M11-08]
parallel_group: M11-W
touches: [frontend/src/screens/companion-home, frontend/src/features/accept-invite, frontend/src/entities/family, frontend/src/app/[locale]/companion, frontend/src/app/message-scopes.ts, frontend/messages/fa/companion-home.json, frontend/messages/en/companion-home.json, backend-go/resources/translations/fa/companion-home.json, backend-go/resources/translations/en/companion-home.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- companion-home
---

# T-M11-18 — Frontend member acceptance: code preview, accept or decline, mutual answer, pending phone invites
**Model:** sonnet.

## Why
SPEC §1.8: the member enters a code, sees the role and the shared domains (read-only), may answer a mutual request
for her own meds and appointments, and accepts or declines; a phone invite appears after signup without a code. Today
entering the code activates the link at once (`CompanionLinksPage`). DECISIONS.md F-11, F-12.

## Scope
1. `features/accept-invite`: code → preview card (owner first name, role, domains in plain words) → optional mutual
   answer (levels for her own meds and appointments) → accept or decline; generic refusal copy for any bad code.
2. Pending phone invites: a card on the companion screens and on first open after signup (`/family/invites/pending`).
3. `awaiting_owner` state: «منتظر تأیید … هستیم».
4. Fallback to today's code flow on 404 (Laravel). Light and dark; fa/en.

## Tests
- Vitest: preview mapping, mutual payload, decline, refusal copy, the 404 fallback; headless screenshots light and
  dark under `docs/engine-v6/qa/m11/`.

## Out of scope
- Member home (T-M11-20).

## Acceptance
- Verify passes; screenshots reviewed by the main session.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.
