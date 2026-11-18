---
id: T-M12-19c
title: make sqlc is broken by migration 00128 (JSON_ARRAY syntax)
milestone: M12
type: backend
status: done
depends_on: []
parallel_group: M12-TEST
touches: [backend-go/db/migrations/00128_drop_alert_visit_note_action.sql,backend-go/sqlc.yaml,backend-go/internal]
skills: []
verify: cd backend-go && CGO_ENABLED=0 make sqlc && git diff --exit-code -- internal/*/store && CGO_ENABLED=0 go build ./...
---

# T-M12-19c — make sqlc is broken by migration 00128 (JSON_ARRAY syntax)

**Model:** sonnet. Found in T-M12-19 (2026-11-18).

## Why
`make sqlc` fails for the whole repo: sqlc cannot parse `00128_drop_alert_visit_note_action.sql:20`
(`ORDER BY jt.n), JSON_ARRAY())`, committed in 7dd38d6a). Every task that adds a query has to hand-copy sqlc output.

## Scope
1. Make sqlc skip the data-only statement (sqlc annotations / move the JSON rewrite into a Go migration / split the
   statement so the schema parser ignores it) without changing what the migration does on MariaDB. Never edit an
   applied migration's effect.
2. Regenerate all stores; the diff must be empty except for ordering fixes sqlc would have made anyway (explain).

## Acceptance
- `make sqlc` green; regenerated stores identical (or explained); `go build ./...` green.
