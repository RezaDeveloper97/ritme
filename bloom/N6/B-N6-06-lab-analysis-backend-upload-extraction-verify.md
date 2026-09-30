---
id: B-N6-06
title: Lab analysis backend — upload, extraction, verify, interpretation, trends
milestone: N6
type: backend
status: todo
depends_on: [B-N6-05]
parallel_group: N6-F
touches: [backend-go/internal/labs,backend-go/db,backend-go/api,backend-go/seeds]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N6-06 — Lab analysis backend — upload, extraction, verify, interpretation, trends

## Why
«تحلیل آزمایش».

## Scope
- Upload images/PDF (multi-page, size/type limits, encrypted storage, deletable), async job → markers with value, unit, sheet reference range, confidence; user verify/edit/add; interpretation (plain Persian, context: mode, cycle, meds), doctor questions; marker catalog (admin-editable explanations); trends across labs; feedback thumbs.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Job pipeline tested with the fake provider
- `verify` green
