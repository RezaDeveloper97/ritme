---
id: B-N9-07
title: Challenges v2 — eligibility rules, daily tasks, rewards
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-02]
parallel_group: N9-G
touches: [backend-go/internal/challenges,backend-go/internal/admin,backend-go/api,admin-web/src,frontend/src/widgets/today-challenge]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N9-07 — Challenges v2 — eligibility rules, daily tasks, rewards

## Why
Challenge editor in the design.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Challenges.dc.html` (+ `nbd_Admin_Challenges`)

## Scope
- Duration, start rule, reminder time, eligibility (mode/phase include/exclude), per-day tasks, reward (badge + Plus days), completion rule; participants/completion stats; user-side badge + Plus-days grant.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Eligibility tests
- `verify` green
