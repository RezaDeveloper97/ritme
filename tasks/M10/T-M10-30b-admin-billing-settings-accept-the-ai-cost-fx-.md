---
id: T-M10-30b
title: Admin billing settings accept the AI cost FX rate (ai_cost_usd_irr_rate)
milestone: M10
type: backend
status: done
depends_on: [T-M10-30]
parallel_group: M10-D
touches: [backend-go/internal/admin/billing,docs/go-migration/admin-api.md]
skills: []
verify: cd backend-go && go vet ./internal/admin/billing/... && go test ./internal/admin/billing/... && make test-int PKG=./internal/admin/billing/...
---

# T-M10-30b — Admin billing settings accept the AI cost FX rate
**Model:** sonnet.

## Why
T-M10-30's AI cost share (AI cost / Plus revenue) reads `plus_settings.ai_cost_usd_irr_rate` (IRR per USD), but
`PUT /api/admin/v1/plus/settings` only accepts known keys, so the KPI stays null.

## Scope
1. Accept `ai_cost_usd_irr_rate` (positive number, finance capability, audited like the other billing settings).
2. Return it in `GET /plus/settings`; document it in `docs/go-migration/admin-api.md`.

## Acceptance
- Verify passes; setting the rate makes the next KPI snapshot's `ai_cost.share` non-null (int test).
