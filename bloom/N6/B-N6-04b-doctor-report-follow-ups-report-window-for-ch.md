---
id: B-N6-04b
title: Doctor report follow-ups — report window for checkups/labs, atomic link cap, share entry points
milestone: N6
type: fullstack
status: todo
depends_on: [B-N6-04]
parallel_group: N6-D2
touches: [backend-go/internal/healthrecord,backend-go/internal/sharelinks,frontend/src/screens/vitals-report,frontend/src/screens/lab-result]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test
---

# B-N6-04b — Doctor report follow-ups — report window for checkups/labs, atomic link cap, share entry points

## Why

## Scope
- Security audit of B-N6-04, L-2: the report window (`Options.From`) must also bound checkups (`done_on >= from`) and labs
  (`COALESCE(taken_on, created_at) >= from`) for the `share` audience / `/health-record/report`; keep the owner record
  unchanged (latest 5).
- L-3: active-link cap (10) count + insert in one transaction (`SELECT … FOR UPDATE` on the user row) so parallel POSTs
  cannot exceed it.
- Wire «اشتراک با پزشک» from the vitals report (B-N6-02) and the lab result (B-N6-07) to `/record/export` with the
  matching section preselected (query param), no cross-screen imports.
- PWA install banner hidden on `/shared/*` (public doctor view).

## Out of scope
- 

## Acceptance
- 
