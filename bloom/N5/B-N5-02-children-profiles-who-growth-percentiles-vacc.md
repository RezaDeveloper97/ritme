---
id: B-N5-02
title: Children — profiles, WHO growth percentiles, vaccines, milestones
milestone: N5
type: backend
status: todo
depends_on: [B-N5-01,B-N4-01]
parallel_group: N5-B
touches: [backend-go/db,backend-go/internal/children,backend-go/internal/companion,backend-go/api,backend-go/seeds]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N5-02 — Children — profiles, WHO growth percentiles, vaccines, milestones

## Why
Child profile (v16) and family sharing.

## Scope
- children (owner/family, name, dob, sex, birth weight/length, delivery type; photo stays on device), measurements with WHO LMS percentile computation (seed WHO tables for weight/length/head 0–5y), Iran national immunisation schedule (seeded, admin-editable) + per-child dose status + reminders 3 days before, milestone catalog by age + per-child checks, age-based learn tips (article tags). Shared children visible to spouse via family.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Percentile golden tests against WHO reference values
- `verify` green
