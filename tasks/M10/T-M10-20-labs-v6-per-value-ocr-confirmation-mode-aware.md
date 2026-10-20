---
id: T-M10-20
title: Labs v6: per-value OCR confirmation, mode-aware ranges, HbA1c, safe AI summary
milestone: M10
type: fullstack
status: todo
depends_on: [T-M10-16, T-M10-17, T-M10-18]
parallel_group: M10-C
touches: [backend-go/internal/labs, backend-go/internal/ai/pii.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist, frontend/src/screens/lab-verify, frontend/src/screens/lab-result, frontend/src/screens/lab-marker, frontend/src/entities/lab, frontend/messages/fa/labs.json, frontend/messages/en/labs.json, backend-go/resources/translations/fa/labs.json, backend-go/resources/translations/en/labs.json]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/labs/... ./internal/ai/... && make test-int PKG=./internal/labs/... && make contract ROUTES=labs && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:dark && npm run test && npm run copy:check -- labs
---

# T-M10-20 — Labs v6: per-value OCR confirmation, mode-aware ranges, HbA1c, safe AI summary
**Model:** opus (clinical ranges and AI safety). The main session reviews ranges, the word list and the copy.

## Why
Gap 04 rows 3.3–3.6, §2 S2 items 8 and 11:
- low OCR confidence is 0.7 (`labs.go:89`) and confirmation is all-or-nothing per report; SPEC §3.3: 0.9, and a value
  below it is excluded from rules and AI until the user confirms **that value**;
- fallback ranges ignore pregnancy and inflammation: Hb 12–15.5 for everyone, ferritin 15–150, vitamin D 30–100
  (contradicts IOM 2011: < 12 deficient, ≥ 20 sufficient), no pregnancy TSH (`00034_labs.sql:123-173`);
  `lab.ferritin.low` exists but nothing reads it;
- an extra ±10 % "borderline" class (`status.go:28,41-58`) the spec does not have (H-07);
- HbA1c has no ADA bands (SPEC §3.2);
- the AI summary is free text with prompt-only guardrails and no diagnostic-word filter (`interpret.go:234-245,361`).
DECISIONS.md H-07, H-08; D-171.

## Scope
1. Confidence: `lab.ocr.min_confidence`; values below it are amber on the verify screen and need a tap to confirm
   (`POST /api/v1/labs/{id}/markers/{marker_id}/confirm` → `confirmed_at`); the report can be saved with unconfirmed
   values, which are excluded from status rules, trends and the AI payload until confirmed; editing a value confirms
   it.
2. Fallback ranges (only when the sheet has no range; label «محدوده عمومی»), from the registry on the lab date:
   - Hb: not pregnant < 12 low; pregnant < 11 low, < 10.5 in trimester 2 (T-M10-18's function on `taken_on`);
   - ferritin < 15 low; with inflammation (CRP above its range in the same report) < 70 low;
   - vitamin D < 12 deficient, 12–19.9 «کمتر از حد کافی», ≥ 20 sufficient, plus a one-line note on the Endocrine
     Society 2024 guideline;
   - TSH in pregnancy: upper 4.0;
   - HbA1c: < 5.7 normal, 5.7–6.4 and ≥ 6.5 as ranges «که معمولاً پزشک بررسی بیشتری می‌خواهد» — never a diagnosis.
3. Borderline: `lab.borderline.enabled` 1 = today's class; 0 = no class, only a visual "near the edge" hint.
4. AI summary: the model must return JSON `{summary, out_of_range: [{code, explanation}], doctor_questions: []}`
   (validated); categories always come from the rules; the prompt forbids disease names, doses and changing
   categories; an output filter (fa/en diagnostic-word list + dose patterns such as mg, واحد, قرص) discards the AI
   text and falls back to the rules summary on any hit or invalid JSON (`labs_ai_filtered_total` metric). The word
   lists live in one file with a test.
5. Frontend: per-value confirm on lab-verify (amber chips, confirm all), lab-result renders the JSON sections and the
   «محدوده عمومی» label; fa/en, light/dark.

## Tests
- Hb 11.9 / 12.0 not pregnant; pregnant T1 10.9 / 11.0; T2 10.4 / 10.5.
- Ferritin 14 / 15; with inflammation 69 / 70.
- Vitamin D 11.9 deficient, 12 and 19.9 insufficient, 20 sufficient.
- TSH pregnant 4.0 in range, 4.1 out; HbA1c 5.6 / 5.7 / 6.4 / 6.5.
- Confidence 0.89 → excluded until confirmed; 0.9 → included; confirm → included in the next interpretation.
- Borderline: enabled 1 → today's class; 0 → no class.
- AI: «دیابت داری» or "take 500 mg" in the summary → rules summary returned; invalid JSON → rules summary.
- Frontend vitest for the confirm flow and the JSON renderer.

## Out of scope
- Image masking of names (keep the honest consent text, gap 04 row 3.7).

## Acceptance
- Verify passes; PROGRESS.md cites D-171 and notes H-07 as advisor-pending.
