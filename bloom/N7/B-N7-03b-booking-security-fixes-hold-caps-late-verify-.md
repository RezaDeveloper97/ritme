---
id: B-N7-03b
title: Booking security fixes — hold caps, late-verify slot recheck, refund ledger, gateway outside lock, consent start bound
milestone: N7
type: backend
status: done
depends_on: [B-N7-03]
parallel_group: N7-C2
touches: [backend-go/internal/telemed,backend-go/internal/payments,backend-go/db,backend-go/api]
skills: [security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N7-03b — Booking security fixes — hold caps, late-verify slot recheck, refund ledger, gateway outside lock, consent start bound

## Why

## Scope
Security audit of B-N7-03 (telemed booking):
- M1 hold spam: ≤1 active hold per user (close older), ≤2 upcoming per doctor / ≤5 total, per-user hold throttle
  (10/h); 409/429 with error_code.
- M2 late verify on expired hold: re-run `slotOffered` + `starts_at > now` before confirm, else record-paid + refund.
- M3 Zarinpal refunds → `refund_pending`: Error log with booking/reference, admin-listable query + index; refund outbox
  row; document that paid telemed must stay off in production until B-N7-08 lists refunds.
- L1 gateway Verify before taking the doctor lock; refunds after commit from an outbox (done once).
- L2 `AuthorizeShare` start bound (from 24 h before the visit). L3 doctor still listed/active check.
- L4 reschedule cap 2. L5 declined verify on cancelled hold → terminal payment_status. L6 `Complete` locks doctor first.

## Out of scope
- 

## Acceptance
- 
