---
id: B-N7-07b
title: Assistant history — tabs all/assistant/doctor, combined timeline with visits, PDF export
milestone: N7
type: fullstack
status: todo
depends_on: [B-N7-07,B-N7-05]
parallel_group: N7-G2
touches: [frontend/src/screens/assistant-history,frontend/src/entities/assistant,backend-go/internal/assistant]
skills: [new-endpoint]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# B-N7-07b — Assistant history — tabs all/assistant/doctor, combined timeline with visits, PDF export

## Why

## Scope
- From the N7 audit (audit-n7.md, med open): the assistant history artboard has tabs «همه / دستیار / پزشک», a combined
  timeline of assistant conversations and telemed visits, and a PDF export (on-device, reuse `shared/lib/pdf`). Add an
  owner-only combined feed endpoint (or compose client-side from existing endpoints if enough) and the screen.

## Out of scope
- 

## Acceptance
- 
