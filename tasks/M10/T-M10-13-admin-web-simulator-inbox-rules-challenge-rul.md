---
id: T-M10-13
title: Admin-web simulator, inbox rules, challenge rules and CTR per version
milestone: M10
type: frontend
status: todo
depends_on: [T-M10-12, T-M10-11, T-M10-10, T-M10-08]
parallel_group: M10-W
touches: [admin-web/src/screens/message-rules, admin-web/src/screens/challenges, admin-web/src/screens/pregnancy-alert-rules, admin-web/src/widgets/shell/model/nav.ts, admin-web/messages/fa.json, admin-web/messages/en.json]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M10-13 — Admin-web simulator, inbox rules, challenge rules and CTR per version
**Model:** sonnet for the UI; the main session reviews the health-access prompt.

## Why
Admins need to answer "why didn't user X see message Y today", see conflicts while editing, manage the alert rules
that replaced the pregnancy alert engine, build challenge contraindications and read CTR per version (gap 03 ADM-02,
ADM-03; SPEC §5.4.1, §5.5, §5.6).

## Scope
1. Simulator screen: pick a user (id or RT code), date and time; when the API answers `health_access_required`, show
   the existing reason + ticket prompt; show the engine summary, each slot's winner and variant, and every dropped
   message with its reason in plain fa/en words.
2. Conflict panel inside the editor (T-M10-11 API): overlapping rules, the overlap and the winner.
3. Inbox rules: a filter for slot `alerts_inbox` and `rule_key` prefixes (`preg.`, `meno.`, `cond.`), a locked badge.
   The old `pregnancy-alert-rules` screen is removed from the nav and redirects to this filtered list.
4. Challenge editor: eligibility and contraindications with the shared condition builder, start rule (`on_join`,
   `on_phase_start` + phase, fixed date), day-by-day drop-off chart.
5. Rules list: CTR per version and per variant.

## Tests
- Vitest: simulator view-model (reasons → labels), conflict panel mapping, challenge payload round trip, redirect of
  the old screen.
- Headless screenshots light and dark of the simulator and the challenge editor.

## Out of scope
- Backend changes.

## Acceptance
- Verify passes; screenshots saved under `docs/engine-v6/qa/admin/`.
