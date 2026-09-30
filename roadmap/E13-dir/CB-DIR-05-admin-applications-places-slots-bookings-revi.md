---
id: CB-DIR-05
title: Admin: applications, places, slots, bookings, reviews
epic: DIR
type: fullstack
status: todo
depends_on: [CB-DIR-03, CB-DIR-04, CB-CORE-04]
parallel_group: DIR-D
touches: [backend-go/internal/cityservices/admin, backend-go/internal/http/routes_admin_cityservices.go, backend-go/api/openapi.yaml, admin-web/src/screens/city-services, admin-web/src/app/(panel)/city-services, admin-web/src/widgets/shell/model/nav.ts, admin-web/messages]
skills: [new-endpoint]
boards: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/cityservices/... && make test-int PKG=./internal/cityservices/... && golangci-lint run && cd ../admin-web && npm run typecheck && npm run lint && npm run test && npm run build
---

# CB-DIR-05 — Admin: applications, places, slots, bookings, reviews

## Why
All marketplace operations in admin-web (DECISIONS).

## Scope
1. Application review (doc viewer, approve → place, reject + reason), place edit/publish/suspend, services + weekly slot generator, bookings queue (confirm/reject/complete/no-show), reviews & reports moderation. Extends bloom's city-services admin.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- application → place → booking confirmed → review, end to end.
- `verify` green.
