---
id: CB-LOSS-03b
title: Loss care re-entry and follow-up cleanup
epic: LOSS
type: frontend
status: todo
depends_on: [CB-LOSS-03]
parallel_group: LOSS-D
touches: [frontend/src/screens/loss-care,frontend/src/screens/home,frontend/src/screens/profile,frontend/src/entities/loss,frontend/messages,backend-go/internal/loss,backend-go/contract,backend-go/resources/translations,backend-go/internal/i18n/testdata,docs/qa/canvas]
skills: [new-fsd-slice]
boards: [nbl_Loss_Care.dc.html]
verify: cd backend-go && go vet ./... && go test ./internal/loss/... && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-LOSS-03b — Loss care re-entry and follow-up cleanup

## Why
CB-LOSS-03 QA: after leaving `/loss/care` there is no way back to it, and `DELETE /loss` leaves the private beta/visit follow-up appointments in her care list.

## Boards
- `nbl_Loss_Care.dc.html`

## Scope
1. A calm, low-key re-entry to `/loss/care` while a recent loss exists (e.g. a quiet row on the cycle home or in /profile → mode), never celebratory, dismissible.
2. `DELETE /loss` also removes the private follow-up appointments it created (backend), with int test.

## Out of scope
- bloom's pregnancy screen CTA after a loss and companion-home pregnancy row (reported to the bloom session as P1/P2).

## Acceptance
- Re-entry visible only to her, only while a loss exists; erase leaves nothing loss-related in care.
- `verify` green.
