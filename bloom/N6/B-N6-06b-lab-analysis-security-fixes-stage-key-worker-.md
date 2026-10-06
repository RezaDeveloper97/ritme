---
id: B-N6-06b
title: Lab analysis security fixes (stage key, worker panics/leases, shutdown, caps, prompt data)
milestone: N6
type: backend
status: done
depends_on: [B-N6-06]
parallel_group: N6-F2
touches: [backend-go/internal/labs,backend-go/db/queries/labs,backend-go/internal/platform/config,docker-compose.stage.yml,docker-compose.prod.yml,docker-compose.contract.yml,.env.stage.example,backend-go/internal/ai]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N6-06b — Lab analysis security fixes (stage key, worker panics/leases, shutdown, caps, prompt data)

## Why
Security review of B-N6-06 (2a176fb8): H1 + M1 + M2 must be fixed before lab analysis ships.

## Scope
- H1 stage dev key / key not wired; M1 worker panic + unbounded reclaim; M2 shutdown leaves labs busy; M3 daily cap bypass via delete; L1 verify TOCTOU + cap; L2 prompt data fencing; L3 claim token; L4 upload buffers + global semaphore; L5 double refund; L6 strict PDF sniff; L8 sweep safety. Details in the review report (session log) and B-N6-06 PROGRESS.

## Out of scope
- 

## Acceptance
- Each finding fixed with tests
- `verify` green
