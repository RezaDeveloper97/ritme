---
id: CB-VOICE-03b
title: Fake voice parser gaps from the QA fixtures
epic: VOICE
type: backend
status: todo
depends_on: [CB-VOICE-03]
parallel_group: VOICE-C
touches: [backend-go/internal/ai,backend-go/internal/voicelog]
skills: []
boards: []
verify: cd backend-go && go vet ./... && go test ./internal/ai/... ./internal/voicelog/... && make test-int PKG=./internal/voicelog/... && golangci-lint run
---

# CB-VOICE-03b — Fake voice parser gaps from the QA fixtures

## Why
CB-VOICE-03's accuracy fixtures (pinned in `internal/voicelog/accuracy_int_test.go`) found gaps in the fake rule-based parser used in dev/tests (recall 81%, precision 85%).

## Boards
- none (backend)

## Scope
1. Bleeding/period phrases («پریودم»، «خونریزیم زیاده»).
2. Intensity adverbs inside a phrase («دلم خیلی درد می‌کنه»).
3. «N بار با گرگرفتگی» → count N.
4. «حواسم پرته / تمرکز ندارم» → brain_fog.
5. «ساعت دو» without AM/PM → afternoon when that fits the context.
6. Whole-word label matching (no «ترش» inside «بیشترش», no duplicate «خشکی»).
Update the pinned miss/extra lists in the accuracy test.

## Out of scope
- The Gemini provider (separate run on stage when keys are allowed).

## Acceptance
- Accuracy test shows the gaps closed (recall/precision up), no regressions in existing fixtures.
- `verify` green.
