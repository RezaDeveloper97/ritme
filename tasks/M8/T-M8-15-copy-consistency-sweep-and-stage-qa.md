---
id: T-M8-15
title: M8 wrap-up — cross-namespace consistency sweep, visual QA, stage
milestone: M8
type: release
status: todo
depends_on: [T-M8-04,T-M8-05,T-M8-06,T-M8-07,T-M8-08,T-M8-09,T-M8-10,T-M8-11,T-M8-12,T-M8-13,T-M8-14]
parallel_group: M8-H
touches: [frontend/messages/fa,backend-go/resources/translations/fa,docs/copy-fa/QA.md]
skills: [persian-writing:persian-writing,verify-all,deploy-stage]
verify: node frontend/scripts/copy-fa/check.mjs --all && cd frontend && npm run typecheck && npm run lint && npm run test && cd ../backend-go && go test ./internal/i18n/...
---

# T-M8-15 — M8 wrap-up — cross-namespace consistency sweep, visual QA, stage
**Model:** main session (opus) for the sweep and QA; haiku subagent allowed only for the mechanical
`check.mjs --all` re-runs and grep counts.

## Why
Eleven batches were written by different subagents; terms and tone drift across namespaces.

## Scope
1. `check.mjs --all` → 0 errors, and warnings ≤ the justified list in PROGRESS.md.
2. Cross-namespace consistency: for every GLOSSARY.md term grep the rejected variants across all fa files; the
   same UI action has one wording everywhere (ذخیره / ثبت، انصراف / لغو، بعدی / ادامه …) — fix outliers.
3. Visual QA in fa, light + dark, mobile width, via the headless screenshot flow (see memory "Headless UI
   verification"): welcome, auth, onboarding, home (cycle + pregnancy), calendar, log sheet, profile, one screen
   from each batch. Look for overflowed buttons/chips/tabs and mid-word wraps. Record in `docs/copy-fa/QA.md`
   with screenshots paths.
4. `verify-all`. Then ask the user before `deploy-stage` (outward action); after deploy, spot-check 5 screens on
   stage. Note in PROGRESS.md that production still runs Laravel until T-M2-26, so prod copy needs
   `php artisan translations:import` there or waits for the Go cutover.

## Acceptance
- 0 checker errors, consistency fixes listed, QA.md with screenshots, verify-all green, stage deployed (if
  user approved) and spot-checked.
