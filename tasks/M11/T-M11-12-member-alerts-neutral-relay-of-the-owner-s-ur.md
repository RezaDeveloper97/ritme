---
id: T-M11-12
title: Member alerts: neutral relay of the owner's urgent alerts to members with the switch
milestone: M11
type: backend
status: blocked
depends_on: [T-M11-05, T-M10-08, T-M9-12]
parallel_group: M11-C
touches: [backend-go/internal/family/alerts, backend-go/internal/messages/inbox, backend-go/internal/messages/rules/rules.go, backend-go/internal/http/routes_family.go, backend-go/db/queries/companion, backend-go/api/openapi.yaml, docs/engine-v6/jobs.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test -race ./internal/family/... ./internal/messages/... && make test-int PKG="./internal/family/... ./internal/messages/inbox/..."
---

# T-M11-12 — Member alerts: neutral relay of the owner's urgent alerts to members with the switch
**Model:** opus (safety relay and privacy).

## Why
SPEC §1.7–§1.8: urgent items of the owner's alerts inbox («پیگیری زودتر», type safety) are copied to the `member_alert`
slot of members for whom she turned the urgent switch on — only a title, the time, "call her" and 115, no medical
detail. Nothing exists; the alert title itself is medical detail (gap 05 §2.13). DECISIONS.md F-09, F-10, F-23, M-11,
M-16; D-200.

## Scope
1. Hook: when the T-M10-07 inbox writer makes a safety row `active` at level urgent (new, or re-activated per M-16),
   it enqueues `family.member_alert {owner_id, inbox_id, hit}` after commit (queue v2, unique key
   `{inbox_id}:{hit}`, retries). Members' requests never evaluate her rules.
2. Worker: her active links whose `urgent_alert` grant passes check() (stays on across mode changes, F-23) →
   `family_member_alerts` rows (unique per member, alert, hit) and one in-app notification each («پیامی از خانواده در
   ریتمی», `link_id` set). With `urgent_sms`: the discreet SMS template, one per alert per member, at most 3 per member
   per day, through the existing SMS gate (F-09).
3. Member API: `GET /api/v1/family/alerts` → `{id, owner_display_name, title (fixed neutral text per level), at,
   actions: [call_owner, call_115], help_key}` and `POST /api/v1/family/alerts/{id}/seen`; the owner's mobile appears
   only for switch holders; never the rule title, body, evidence or level name.
4. Purge by `link_id` on revoke and on switch-off; deletion cascades.
5. Replace the M10 reservation with the `member_alert` slot constant.

## Tests
- `TestMemberAlert_OnlyWithSwitchAndNeutral` (JSON scan: no rule title, body, evidence or clinical term),
  `TestMemberAlert_NonUrgentNotRelayed`, `TestMemberAlert_Idempotent` (job retried → one row),
  `TestMemberAlert_ReactivatedUrgentRelayedAgain`, `TestMemberAlert_PurgedOnRevokeAndSwitchOff`,
  `TestMemberAlert_SMSOnlyWithSwitchAndCapped`, `TestMemberAlert_OwnerMobileOnlyForSwitchHolders`,
  `TestMemberAlert_IDOR`.

## Budgets
- From inbox commit to member rows ≤ 5 s p95 with the worker running (measured on stage in T-M11-23).

## Out of scope
- Member alert screen (T-M11-20); real push.

## Acceptance
- Verify passes; PROGRESS.md cites D-200.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.
