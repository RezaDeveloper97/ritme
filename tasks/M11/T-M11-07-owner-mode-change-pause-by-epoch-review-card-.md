---
id: T-M11-07
title: Owner mode change: pause by epoch, review card, re-approve, child profile suggestion at delivery
milestone: M11
type: backend
status: blocked
depends_on: [T-M11-06, T-M11-11]
parallel_group: M11-C
touches: [backend-go/internal/family/review, backend-go/internal/family/sync.go, backend-go/internal/companion, backend-go/internal/profile, backend-go/internal/postpartum/service.go, backend-go/internal/pregnancy, backend-go/internal/loss/service.go, backend-go/internal/teen, backend-go/internal/menopause, backend-go/internal/http/routes_family.go, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/family/... && make test-int PKG="./internal/family/... ./internal/postpartum/... ./internal/pregnancy/... ./internal/loss/... ./internal/profile/..."
---

# T-M11-07 — Owner mode change: pause by epoch, review card, re-approve, child profile suggestion at delivery
**Model:** opus.

## Why
SPEC §1.7–§1.8: on an owner mode change the previous mode's access is deactivated (not deleted), the owner gets a
"set up the new access" card, the member sees only children and household until she approves, and a delivery
suggests a child profile with shared parents. T-M9-02 resets pregnancy grants to none (a hotfix that deletes the
level); the lazy epoch of T-M11-05 makes a missed hook safe, but nothing tells the owner (gap 05 §2.10, FAM-H3).
DECISIONS.md F-23; D-195.

## Scope
1. Call `family.SyncMode(ctx, ownerID)` after commit of every mode write: profile life mode and goal (TTC), pregnancy
   start/end (v1 and v2), postpartum start/end, loss, teen, menopause. `companion.ResetModeScopedGrants` (T-M9-02)
   becomes a wrapper that only syncs (grants are paused by epoch, never rewritten).
2. On an epoch bump: `review_pending = 1` and one owner notice per epoch («حالت تو عوض شد؛ دسترسی همراه‌هایت را مرور
   کن»).
3. `GET /api/v1/family/review` (owner): per member, paused domains with their previous level, domains suggested for
   the new mode (template defaults within max level and role) and a one-tap `keep_as_before` set (previous levels of
   domains still available in the new mode).
4. `POST /api/v1/family/members/{linkId}/grants/reapprove {domains}` → `Grants.Apply` at the current epoch (consent +
   activity); `review_pending` clears when no member has paused domains.
5. Delivery: when postpartum starts and the owner has an active partner or other-parent link, the review payload
   carries `suggest_child_profile {member_ids}`; `POST /api/v1/family/children/suggestion {name?, birth_date,
   member_ids}` creates the child through the children service and invites the members as guardians (T-M11-11).
6. During review the member sees children, household and urgent alerts only (follows from the epochs; asserted).

## Tests
- `TestModeChange_PregnancyDomainsPausedAfterDelivery`: postpartum start → deny `mode_inactive`, grant row kept, the
  member projection hides it, the review lists it; re-approve → allow.
- `TestLoss_ThenNewPregnancy_StillDenied`; `TestMissedHook_SelfHealsToDeny`; `TestUrgentAlert_StaysOnAcrossModeChange`.
- `TestReview_KeepAsBeforeSkipsUnavailableDomains` (pregnancy is not restored in postpartum).
- `TestReapprove_WritesConsentAndActivity`; `TestReview_OneOwnerNoticePerEpoch`;
  `TestDeliverySuggestion_CreatesChildWithGuardianInvites`; IDOR on reapprove (foreign link → 404).

## Out of scope
- Owner UI (T-M11-17).

## Acceptance
- Verify passes; PROGRESS.md cites D-195.

## Note from the T-M11-05 security review (2026-10-28)
- The lazy epoch does not catch an A→B→A mode round trip with no read in between (e.g. pregnancy → end → new
  pregnancy, cycle → ttc → cycle): the epoch never moves, so F-23 pause and `review_pending` are skipped for
  `fetal_activity`, `vitals`, `meds`, `appointments`. The SyncMode hooks here must run on every mode write (in the
  same transaction where possible); add `TestModeRoundTrip_StillPaused`.
- L5: `internal/loss/service.go` notice recipients come from `ListForOwner`; filter them with `Decide(pregnancy, read)`.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.
