---
id: B-N2-04
title: Subscription domain — plans, trials, subscriptions, discounts, entitlements
milestone: N2
type: backend
status: done
depends_on: [B-N2-01]
parallel_group: N2-D
touches: [backend-go/db,backend-go/internal/plus,backend-go/internal/http,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N2-04 — Subscription domain — plans, trials, subscriptions, discounts, entitlements

## Why
Ritme Plus is the revenue model; core tracking stays free forever (intro slide 5).

## Scope
- Tables: plans (duration months, price, per-month display, badge, active), subscriptions (status, period, auto-renew), trials (7 days, once per user), discount codes (percent/amount, limits, expiry), invoices (VAT rate from config), receipts.
- Entitlement service: `plus.deep_analysis`, `plus.voice_log`, `plus.lab_ai` (10/month), `plus.assistant_unlimited`, `plus.pdf_share`, `plus.visit_discount`; free users get quotas where designed.
- Endpoints `/api/v1/plus/{plans,status,trial/start,checkout,verify,cancel,restore,history,usage}`; OpenAPI; tests with the test clock.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Entitlements resolved in one place and unit-tested
- verify green
- `verify` green
