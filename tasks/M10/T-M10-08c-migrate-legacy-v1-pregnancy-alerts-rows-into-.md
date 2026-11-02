---
id: T-M10-08c
title: Migrate legacy v1 pregnancy_alerts rows into the inbox; feed modal for code-held locked rules
milestone: M10
type: backend
status: done
depends_on: [T-M10-08b]
parallel_group: M10-B
touches: [backend-go/db/migrations,backend-go/internal/pregnancy,backend-go/db/queries/pregnancy,backend-go/internal/messages/feed,backend-go/internal/messages/compiled,backend-go/contract/allowlist/pregnancy.yaml]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/pregnancy/... ./internal/messages/... && make test-int PKG="./internal/pregnancy/... ./internal/messages/... ./db/migrations/..." && make contract ROUTES=pregnancy,pregnancy-v2,messages
---

# T-M10-08c — Migrate legacy v1 pregnancy_alerts rows into the inbox; feed modal for code-held locked rules

**Model:** opus (health alerts, safety).

## Why
T-M10-08b report: goose 00113 moved only the v2 `pregnancy_alerts` rows (`alert_type 'v2:%'`) into `message_inbox`.
The v1 rows written before T-M10-08 (`symptom_based` …, by Laravel on prod and Go on beta) were not migrated, so the v1
`/pregnancy/alerts` list/summary/read/dismiss/read-all and the loss path still read and update the legacy table next to
the inbox (an id equal in both resolves to the inbox row; the legacy one can then only be cleared by read-all). Also the
feed's compiled snapshot does not contain a code-held locked rule whose DB row is missing/inactive/unapproved, so such
an alert is in the inbox and the lists but is never fanned out as the urgent modal.

## Scope
1. A data-only goose migration (no Laravel twin) that moves the legacy v1 `pregnancy_alerts` rows (not `v2:%`) into
   `message_inbox` with a `legacy_alert_id`, mapping alert_type/level to the preg.* rule (symptom_based bleeding /
   fluid / pain → preg.critical_symptom, BP → preg.bp_*, fetal movement → preg.fetal_movement, glucose → preg.gdm …;
   unmappable rows → archived), states as in 00113 (newest per (user, rule) within 30 days keeps its state; older
   resolved/archived), reversible Down, re-run safe; int test.
2. Drop the legacy reads/writes from the v1 alert handlers and the loss path once (1) is in; keep the v1 shape.
3. Feed fanOut: a locked preg.* rule held only in code is still fanned out as urgent_modal/status_banner (use the same
   code-held fallback as inboxalerts).
4. Update the D-165 allow-list entries accordingly.

## Out of scope
- Production data: the migration runs on stage/beta first; prod runs it at the Go cutover (T-M2-26, blocked).

## Acceptance
- After the migration no code path reads `pregnancy_alerts`; a pre-T-M10-08 v1 emergency row still shows as unread
  emergency in `/pregnancy/alerts/summary` (int test).
- Deactivating `preg.bp_severe` and raising a severe BP still produces the urgent modal on the feed (int test).
- Verify green.
