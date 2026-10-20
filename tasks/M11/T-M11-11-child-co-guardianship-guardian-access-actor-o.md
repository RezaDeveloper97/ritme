---
id: T-M11-11
title: Child co-guardianship: guardian access, actor on every write, 24-hour undo, two-guardian approvals, deletion transfer
milestone: M11
type: backend
status: todo
depends_on: [T-M11-05, T-M11-09]
parallel_group: M11-C
touches: [backend-go/internal/children, backend-go/internal/babylog, backend-go/internal/family/guardians, backend-go/internal/family/activity, backend-go/internal/profile/service.go, backend-go/internal/admin/users/users.go, backend-go/db/queries/children, backend-go/db/queries/babylog, backend-go/internal/http/routes_family.go, backend-go/internal/http/routes_children.go, backend-go/internal/http/routes_babylog.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist, docs/engine-v6/jobs.md]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/family/... ./internal/children/... ./internal/babylog/... && make test-int PKG="./internal/family/... ./internal/children/... ./internal/babylog/... ./internal/profile/... ./internal/admin/users/... ./internal/http/..." && make contract ROUTES=children,babylog
---

# T-M11-11 — Child co-guardianship: guardian access, actor on every write, 24-hour undo, two-guardian approvals, deletion transfer
**Model:** opus (data ownership and deletion).

## Why
SPEC §1.7–§1.8: both parents have write on a shared child, every write stores the actor and can be undone within
24 hours, deleting a child or removing a parent needs both parents, and the child's data stays with the other parent
when an owner deletes her account. Today only the owner writes, no actor is stored, overwrites are lost, and owner
deletion cascades away the child with every log (gap 05 §2.9, §2.11; FAM-H4, FAM-M6). DECISIONS.md F-14, F-15, F-17,
F-19; D-192, D-199.

## Scope
1. `children.Access/Editable` from `child_guardians`: active `write` guardians read and write; carers with `limited`
   write feeds, diapers and sleeps only; the `families/family_children` path is no longer read (tables kept).
2. Guardian invites: the owner invites a member of her circle, or T-M11-07's delivery suggestion does;
   `GET /api/v1/family/guardian-invites`, `POST …/{id}/accept|decline`.
3. Actor: `created_by/updated_by` on every child and baby-log write; responses carry the actor's display name for
   guardians.
4. Last write wins; every guardian write adds an activity row (subject child) with sealed before/after (T-M11-09
   seal) and `undo_deadline = +24 h`.
5. `POST /api/v1/family/activity/{id}/undo`: within 24 h; caller is the actor or an active write guardian; foreign id
   → 404; 409 `changed_since` when the record changed after that write; undoing a delete re-inserts the before-image.
6. Two-guardian approvals through proposals (`child_delete`, `guardian_remove`; approver = the other active write
   guardian; 7 days); a sole guardian acts alone; leaving yourself is unilateral unless you are the last guardian
   (F-15).
7. Account deletion (profile and admin paths), inside the deletion transaction: each child the user owns moves to the
   oldest remaining active write guardian (activity row + notice) or is deleted when none is left. The CASCADE FK
   stays (D-192).
8. Daily job `family.activity_retention`: sealed payloads purged after 30 days, rows after 12 months (F-19).
9. Document that feeds logged by a co-guardian still update the mother's `baby.feeds_count` (keyed on her
   postpartum profile, unchanged).

## Tests
- `TestChild_DeleteNeedsBothGuardians`, `TestChild_RemoveGuardianNeedsBoth`, `TestChild_SelfLeaveUnilateral`,
  `TestChild_LastGuardianCannotLeave`, `TestChild_ActorStoredAndLWWLogged`, `TestCarer_LimitedSubScopes`.
- `TestUndo_Within24hOnly`, `TestUndo_ChangedSince409`, `TestUndo_DeleteRestoresRow`, `TestUndo_IDOR`.
- `TestOwnerDeletion_ChildKeptForCoGuardian` (profile and admin paths), `TestOwnerDeletion_SoleGuardianDeletesChild`.
- `TestGuardianWrite_RecheckInTx`, `TestRetention_PurgesSealedAfter30Days`, `TestGuardianInvite_IDOR`.

## Out of scope
- Child medication log (T-M11-14); UI (T-M11-21).

## Acceptance
- Verify passes; PROGRESS.md cites D-192, D-199.
