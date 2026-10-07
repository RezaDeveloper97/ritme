---
id: B-N7-03
title: Booking with payment, cancellation rules and data-share consent
milestone: N7
type: backend
status: in_progress
depends_on: [B-N7-02,B-N2-05]
parallel_group: N7-C
touches: [backend-go/internal/telemed,backend-go/internal/payments,backend-go/internal/care,backend-go/api]
skills: [new-endpoint,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N7-03 — Booking with payment, cancellation rules and data-share consent

## Why
Review & book.

## Scope
- Hold slot → pay via gateway adapter (Plus discount) → confirm; free cancel/reschedule until 2h before; creates a care appointment + reminders; for-whom (self/child/other); scoped consent to share cycle summary, BBT/LH, assistant summaries with this doctor for this visit only, revocable.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Double-booking race test
- `verify` green
