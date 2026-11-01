---
id: T-M10-14b
title: Feed alert_id on fanned-out inbox items, urgent modal above sheets, banner action layout
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-14,T-M10-07]
parallel_group: M10-W
touches: [backend-go/internal/messages/feed,frontend/src/entities/message,frontend/src/screens/home,frontend/src/app/globals.css,docs/engine-v6/qa/T-M10-14b,backend-go/api/openapi.yaml]
skills: []
verify: cd backend-go && go vet ./internal/messages/... && go test ./internal/messages/... && make test-int PKG=./internal/messages/... && make contract ROUTES=messages && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M10-14b — Feed alert_id on fanned-out inbox items, urgent modal above sheets, banner action layout
**Model:** sonnet (UI) + opus review of the feed change.

## Why
Open items from T-M10-14: a fanned-out inbox item in `urgent_modal` / `status_banner` carries the rule id as `id`, but
`POST /alerts/{id}/ack` needs the inbox row id, so the client looks the row up in `GET /alerts` (extra request); the
urgent modal (`z-dialog` 60) can sit under an open sheet backdrop; the banner's two actions stack vertically.

## Scope
1. Feed: add `alert_id` (inbox row id) to items fanned out from the inbox (`feed/inbox.go`, `json.go`); OpenAPI.
   Warm feed stays 3 queries + auth.
2. Frontend: use `alert_id` for ack; keep the `GET /alerts` lookup only as fallback when the field is absent.
3. Urgent modal renders above any open sheet (z-index token in `globals.css`), still not closable before ack.
4. Status banner CTA + close sit on one row (RTL, light + dark).

## Acceptance
- Verify passes; ack makes one request; screenshots light + dark of the urgent modal over an open sheet in
  `docs/engine-v6/qa/T-M10-14b/`.
