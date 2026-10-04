---
id: B-N7-06
title: AI assistant clinic backend — departments, profile, chat, triage, summary, handoff
milestone: N7
type: backend
status: todo
depends_on: [B-N6-05,B-N7-03,B-N6-05b]
parallel_group: N7-F
touches: [backend-go/internal/assistant,backend-go/internal/ai,backend-go/db,backend-go/api]
skills: [new-endpoint,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N7-06 — AI assistant clinic backend — departments, profile, chat, triage, summary, handoff

## Why
«کلینیک دستیار ریتمی».

## Scope
- Departments (general, gynecology, sexual health, skin & beauty, nutrition, mental health) with admin-editable system prompts, guardrails and «what it doesn't do». Health profile shared per toggle. Streaming chat (SSE) via AI adapter, quick-reply chips, photo input, emergency keyword → 115 card, triage level (self-care / non-urgent / soon / emergency), summary document, handoff to a doctor with consent, history + delete. Free quota vs Plus unlimited.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Guardrail tests with the fake provider (emergency, diagnosis requests)
- `verify` green
