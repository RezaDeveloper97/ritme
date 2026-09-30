---
id: B-N3-05
title: Voice logging (Plus) — record, transcribe, parse, review
milestone: N3
type: fullstack
status: todo
depends_on: [B-N3-03,B-N2-06]
parallel_group: N3-E
touches: [frontend/src/features/voice-log,frontend/src/screens/log,backend-go/internal/voicelog,backend-go/internal/ai,backend-go/api]
skills: [new-endpoint,verify-all,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-05 — Voice logging (Plus) — record, transcribe, parse, review

## Why
«بگو امروز چطوری».

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Day.dc.html` (+ `nbd_Log_Day`)

## Scope
- Browser MediaRecorder → upload → STT + NLU via the AI adapter (fake provider returns fixtures; LLM provider maps free Persian text to taxonomy items) → «این‌ها را فهمیدیم» review chips → tap opens the manual section → save.
- Audio deleted right after transcription (verified by test); Plus-gated with quota.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- Works end-to-end with the fake provider
- `verify` green
