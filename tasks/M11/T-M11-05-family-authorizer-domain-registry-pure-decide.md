---
id: T-M11-05
title: family.Authorizer: domain registry, pure Decide, batched snapshot, mode epoch, grant-change service
milestone: M11
type: backend
status: todo
depends_on: [T-M11-04, T-M9-36, T-M9-26, T-M10-23]
parallel_group: M11-C
touches: [backend-go/internal/family, backend-go/internal/companion, backend-go/internal/care/delegation.go, backend-go/internal/shop/checklist/handlers.go, backend-go/internal/http/routes_companion.go, backend-go/internal/http/routes_family.go, backend-go/internal/http/routes_shop_checklist.go, backend-go/internal/consent/catalog.go, backend-go/internal/consent/lang, backend-go/db/queries/companion, backend-go/db/migrations, backend-go/api/openapi.yaml, backend-go/contract/allowlist]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test -race ./internal/family/... ./internal/companion/... && make test-int PKG="./internal/family/... ./internal/companion/... ./internal/care/... ./internal/http/..." && make contract ROUTES=companion,companion-home,reminders,shop-checklist,teen && go test -run '^$' -bench BenchmarkDecide ./internal/family/...
---

# T-M11-05 — family.Authorizer: domain registry, pure Decide, batched snapshot, mode epoch, grant-change service
**Model:** opus (security core of the family feature). Write the `Decide` table first.

## Why
Access today is allow/deny spread over `companion.Service.access`, `Delegation.Authorize`, `children.Access` and the
view builders; there is no propose outcome, no expiry, no mode binding, no per-domain maximum and no single
never-share rule (gap 05 §2.3, FAM-M4, FAM-M6, FAM-L2). A member home costs about four queries per link and section
(§5.3). SPEC §1.8 defines one `check(actor, subject, domain, action) → allow | propose | deny`. DECISIONS.md F-01,
F-02, F-04, F-05, F-13, F-16, F-17, F-20, F-21, F-23; D-193, D-203, D-204.

## Scope
1. `internal/family/registry.go` from family.md: domains (cycle_phase, pregnancy, appointments, fetal_activity,
   vitals, meds, urgent_alert, household, ttc_fertility; children go through guardianship), max level, mode scope,
   roles allowed, role defaults, templates, carer time-box (F-13), male-owner domains (F-20); teen_parent keeps the
   legacy teen sections unchanged.
2. Pure `Decide(s Snapshot, subj Subject, d Domain, a Action, now time.Time) Decision{Outcome, Reason}` in this order:
   owner → child subject with guardian (limited sub-scopes for carers) → link missing / not active / owner blocked →
   expired → domain not allowed for the role or male owner → domain not available in the owner's mode, or grant epoch
   ≠ owner epoch (F-23: every owner-data domain except urgent_alert) → level below the action → `nudge` (quota decided
   by the caller) → `write` on an owner record → propose; `write` on household → allow. Reasons: no_link, expired,
   blocked, not_allowed, mode_inactive, level.
3. Snapshot loader: one statement per (owner, member) pair, one for every owner of a member, one for the guardians of
   a set of children; memoised per request. `SyncMode(owner)` runs lazily inside loading: it resolves the effective
   mode (engine state provider from T-M9-32; `enums.ResolveLifeMode` + pregnancy profile as fallback), bumps
   `mode_epoch` and sets `review_pending` when the mode changed — a missed hook can only deny, never leak.
4. Grant-change service `family.Grants.Apply(ctx, tx, change)` — the only way a grant changes (owner edit, invite,
   accept, re-approve, share-request approval): validates against the registry (422 `level_not_allowed`), stamps the
   current epoch, writes the activity row with `diff_json {domain: [old, new]}` (FAM-L5) and a `consent_events` row
   (code `family_sharing`, versioned text fa/en in the consent catalog, source `family`, scope `{link_id, member_id,
   role, domains, expires_at}`) through the T-M10-23 consent service, all in the caller's transaction.
5. `family.RecheckTx(tx, …)`: reads the link, grant (and guardian) rows `FOR SHARE` inside every write transaction, so
   a revoke committed meanwhile wins (CMP-L4).
6. Data migration (goose, data only — no twin): legacy grants → v6 per family.md (symptoms rows deleted with one owner
   notice each, F-03; none → off; cycle/pregnancy edit → view; meds/appointments edit → write; codes renamed);
   `mode_epoch = 1` for owner-data grants, except pregnancy grants of owners not pregnant now (NULL).
7. Rewire `companion.Service.Level/CanRead/CanWrite`, `Delegation.Authorize`, the teen guard and the shop-checklist
   share onto the Authorizer; `/companions/*` keeps its request and response shape through a legacy↔v6 mapping
   (allow-listed under D-204).
8. Read-audit coalescing in Redis (`SET ritme-go:family:ra:{owner}:{member}:{domain} NX EX 900`, DB check on a Redis
   error); writes, proposals, nudges and lifecycle events are always logged (F-16, D-203).
9. `GET /api/v1/family/me` (Go-only): links as owner (role, template, status, expiry, grants with their state) and
   as member (owner display name, role, effective domains only — no paused or inactive hint). Unknown or revoked link
   → 404; active link without the domain → 403 `family_domain_denied` (F-21). `Cache-Control: private, no-store`.

## Tests
- `TestDecide_Matrix`: table over domain × level × action × role × mode × epoch × expiry × blocked, every row with its
  reason; property test (`testing/quick`): a never-shared category is never reachable, `write` on an owner record is
  never `allow`, revoked/expired/blocked is always `deny`.
- `TestSnapshot_OneStatementPerPair`, `TestSnapshot_AllOwnersOneStatement` (counting DB wrapper).
- `TestSyncMode_SelfHealsToDeny`; `TestGrants_ApplyWritesDiffAndConsent`; `TestGrants_RejectsAboveMaxLevel`;
  `TestRecheckTx_RevokeWins`; `TestRevoke_DeniesImmediately`; `TestBlockedOwner_Denied`.
- Data migration on seeded legacy rows → expected v6 rows and owner notices.
- Existing companion, care, shop-checklist, teen and children tests green (`TestGrantMatrix` rewritten for v6).

## Budgets
- `Decide` ≤ 1 µs and 0 allocations; one authorization statement per member request; member section read p95 ≤ 40 ms
  locally.

## Out of scope
- Projections (T-M11-06), invites (T-M11-08), proposals (T-M11-09), nudges (T-M11-10), guardians (T-M11-11).

## Acceptance
- Verify passes; PROGRESS.md cites D-193, D-203, D-204.
