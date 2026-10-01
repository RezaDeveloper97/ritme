---
id: B-N2-01
title: Profile & onboarding schema v2 and extended life-stage modes
milestone: N2
type: backend
status: done
depends_on: [B-N1-17]
parallel_group: N2-A
touches: [backend-go/db,backend-go/internal/profile,backend-go/internal/enums,backend-go/internal/auth,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N2-01 — Profile & onboarding schema v2 and extended life-stage modes

## Why
New onboarding asks gender, goal, conditions, menopause stage; modes grow from 3 to 6.

## Scope
- Columns/tables: gender (female/male), goal, chronic conditions[], gynecologic conditions[], ongoing med/contraception[], menopause stage (peri/meno/post/unsure) + approx last period + surgical + HRT, optional birth date/height/weight.
- Mode enum: cycle, ttc, pregnancy, postpartum, menopause, teen (+ `ivf_iui` flag, `track_contraception` flag). Engine and message-engine accept the new modes (menopause/teen fall back to safe defaults).
- Onboarding endpoints accept each step idempotently; OpenAPI + tests; fa/en enum labels in translation seed.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Existing users migrate with no data change (cycle/pregnancy stay)
- verify green
- `verify` green
