---
id: B-N2-06
title: Plus gating, trial offer engine and usage counters
milestone: N2
type: backend
status: done
depends_on: [B-N2-04]
parallel_group: N2-F
touches: [backend-go/internal/plus,backend-go/internal/http,backend-go/internal/home,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N2-06 — Plus gating, trial offer engine and usage counters

## Why
Trial banner shows a countdown, 50% offer and what the user used.

## Scope
- Middleware/helper to guard Plus endpoints; 402-style error envelope with `error_code=plus_required`.
- Trial offer: during the 7-day trial, discounted price (admin %), countdown end; `/home` exposes the banner payload.
- Usage counters per feature for the trial sheet (analysis views, lab analyses, assistant messages, PDF shares, visit discount).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Tests for gating, trial start/expiry, offer price
- `verify` green
