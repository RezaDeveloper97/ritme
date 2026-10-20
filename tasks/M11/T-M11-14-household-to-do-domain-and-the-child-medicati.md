---
id: T-M11-14
title: Household to-do domain and the child medication dose log
milestone: M11
type: backend
status: todo
depends_on: [T-M11-11]
parallel_group: M11-C
touches: [backend-go/internal/todo, backend-go/db/queries/todo, backend-go/internal/babylog, backend-go/db/queries/babylog, backend-go/internal/family/household, backend-go/internal/http/routes_todo.go, backend-go/internal/http/routes_babylog.go, backend-go/internal/http/routes_family.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/todo/... ./internal/babylog/... ./internal/family/... && make test-int PKG="./internal/todo/... ./internal/babylog/... ./internal/family/..." && make contract ROUTES=todo,babylog
---

# T-M11-14 — Household to-do domain and the child medication dose log
**Model:** sonnet with main-session review (the authorization comes from T-M11-05 and T-M11-11).

## Why
SPEC §1.7 lists the household to-do list as a shared domain (write for partners) and medication in the child profile;
Fam_PPostpartum has a night shift. To-do is strictly personal and there is no child medication record (gap 05 §2.17,
§2.9). DECISIONS.md F-05, F-24, F-25, F-26; D-202.

## Scope
1. Household list (`todo_tasks.scope = household`, `user_id` = owner): the owner and members with household ≥ view
   read; write members create, edit, complete and delete directly (F-05) with actor columns, activity rows and the
   24 h undo of T-M11-11; `assigned_to` must be the owner or an active member of her circle.
2. `GET /api/v1/family/owners/{ownerId}/household`, `POST`, `PATCH`, `DELETE …/household/{id}`; the owner's own to-do
   endpoints include her household items with `scope`; personal lists stay private (foreign id → 404, unchanged).
3. One household list per owner; a member sees each owner's list labelled with her name (F-25).
4. Night shift (F-24): `POST …/household/night-shift` creates «شیفت شب» due tonight, assigned to the caller.
5. Child medication dose log (F-26): `GET`, `POST`, `PATCH`, `DELETE /api/v1/children/{id}/med-doses` for active
   write guardians; actor name in responses; activity rows and undo; no reminders, schedules or dosing advice.

## Tests
- `TestHousehold_MemberWriteDirectAndUndoable`, `TestHousehold_OffDenies`, `TestHousehold_ViewCannotWrite`,
  `TestTodo_PersonalListsStayPrivate`, `TestHousehold_AssignOnlyCircleMembers`, `TestNightShift_OneTap`.
- `TestChildMedDoses_GuardiansOnly` (carer limited → 403), `TestChildMedDoses_ActorAndUndo`.

## Out of scope
- Screens (T-M11-20, T-M11-21).

## Acceptance
- Verify passes; PROGRESS.md cites D-202.
