---
id: T-M10-08b
title: Pregnancy alerts off the legacy table: v1 list/summary/unread badge, healthlog flow alert, loss dismissal, feed preg facts
milestone: M10
type: backend
status: done
depends_on: [T-M10-08]
parallel_group: M10-B
touches: [backend-go/internal/pregnancy,backend-go/internal/pregnancy/v2,backend-go/db/queries/pregnancy,backend-go/internal/healthlog,backend-go/internal/loss,backend-go/internal/messages/source.go,backend-go/internal/messages/feed,backend-go/contract/allowlist/pregnancy.yaml,backend-go/api/openapi.yaml,backend-go/internal/http/routes_messages.go,backend-go/internal/messages/source_pregnancy_int_test.go]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/pregnancy/... ./internal/healthlog/... ./internal/loss/... ./internal/messages/... && make test-int PKG="./internal/pregnancy/... ./internal/healthlog/... ./internal/loss/... ./internal/messages/..." && make contract ROUTES=pregnancy,pregnancy-v2,pregnancy-tools,healthlog,loss,messages
---

# T-M10-08b — Pregnancy alerts off the legacy table: v1 list/summary/unread badge, healthlog flow alert, loss dismissal, feed preg facts

**Model:** opus (health alerts, safety).

## Why
T-M10-08 moved every pregnancy alert into `message_inbox` (preg.* rules) and stopped writing `pregnancy_alerts`, but
several readers still use the legacy table (T-M10-08 report, items 5–9). The frontend's `/pregnancy/alerts` page
(`entities/pregnancy/api/queries.ts` → `GET /pregnancy/alerts`, `/pregnancy/alerts/summary`, read/dismiss/read-all)
and the v2 today `unread_alerts` badge therefore miss every new alert: **ship this before any stage/beta deploy of
T-M10-08.**

## Scope
1. v1 `GET /pregnancy/alerts`, `GET /pregnancy/alerts/summary`, `POST /pregnancy/alerts/{id}/read|dismiss`,
   `POST /pregnancy/alerts/read-all`: serve them from the inbox rows with `rule_key LIKE 'preg.%'` (v1 shape and key
   order kept; read = acknowledged? / dismiss = archived — follow D-165 and the inboxalerts state mapping), so
   `has_emergency` / `latest_emergency` / `has_unread` reflect the engine alerts. Remove the TEMPORARY summary
   entries from `contract/allowlist/pregnancy.yaml`.
2. pregnancy/v2 `GetV2TodayExtras` `unread_alerts`: count active `preg.*` inbox rows.
3. healthlog `raisePregnancyFlowAlert` (`alerts.ForLoggedFlow`): raise through the inbox (preg.critical_symptom
   evaluation) instead of writing `pregnancy_alerts`; then delete what is left unused in `pregnancy/alerts`.
4. loss: when a pregnancy loss is recorded, archive/dismiss the user's open `preg.*` inbox rows.
5. Feed: `messages.StoreSource` gets the pregnancy-alert input (`inboxalerts.Input`) so the feed's BuildFacts sees the
   preg.* facts; fanOut renders the urgent_modal / status_banner from the stored evidence (preg copy).
6. Optional: call `inboxalerts.Install(d.DB, d.Logger)` once in routes_messages.go so the backend no longer depends on
   the pregnancy-tools group registering.

## Out of scope
- Frontend move to `/alerts` (T-M10-15). Seeding 00113 into the contract DB.

## Acceptance
- A critical symptom save shows up in `GET /pregnancy/alerts` and makes `/alerts/summary` `has_emergency: true`
  (int test); read/dismiss/read-all change the inbox row state; no code path writes `pregnancy_alerts`.
- The v2 today badge counts the inbox rows; a loss archives open preg.* rows; feed facts include preg.*.
- Contract green without the TEMPORARY allowlist entries; verify green.
