---
id: T-M9-42
title: Today screen on today_ui: ring actions, one banner, lateness tiers, disclaimer
milestone: M9
type: frontend
status: todo
depends_on: [T-M9-40, T-M9-32]
parallel_group: M9-C
touches: [frontend/src/screens/home, frontend/messages/fa/home.json, frontend/messages/en/home.json, backend-go/resources/translations/fa/home.json, backend-go/resources/translations/en/home.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- home
---

# T-M9-42 — Today screen on today_ui: ring actions, one banner, lateness tiers, disclaimer
**Model:** sonnet for the UI; the main session writes/reviews every health text (`docs/copy-fa/STYLE.md`,
GLOSSARY, warm «تو»).

## Why
§2.43 is implemented in the browser with different thresholds: the start prompt at ≤ 2 days with no +14 limit, the end
prompt from day 1 with no limit, "end missing" from a client rule, no overdue or last-period banner, no single-banner
priority, the same "a few days late is normal" text at 8+ and 15+ days, and a legacy-phase fallback when `main_phase`
is `unknown` (gap 02c D.6, D.9–D.11). E-17 makes the engine emit `today_ui`.

## Scope
1. `screens/home/model/today-ui.ts`: when `cycle_view.today_ui` exists, ring action, banner and test button come
   from it; otherwise (Laravel payload) the current `hero-state.ts` rules apply unchanged. «هنوز نه» / «هنوز ادامه دارد»
   dismissals stay local for the day (existing `prompt-dismiss.ts`).
2. `TodayBanner.tsx` (one at a time): `end_missing` → opens the period end editor; `overdue` → text by tier and a
   «ثبت تست بارداری» button when `show_pregnancy_test_button` (opens the log sheet on the measurement panel with
   `pregnancy_test`; add the deep link if it does not exist); `ask_last_period` → opens the existing `PeriodDateEditor`.
3. Lateness copy tiers: 1–7 calm, 8–14 overdue banner, > 14 ask for the last period — no "usually normal" text at 8+.
4. With a v6 payload: no legacy-phase fallback for `unknown` (neutral ring state); no rolled-forward period row while
   `days_late > 0` (`slotFor`).
5. Fixed disclaimer under the predictions card: «پیش‌بینی روش پیشگیری نیست» (fa/en).
6. Ring copy limits (label ≤ 14, unit ≤ 10, subtitle ≤ 18 characters, fa and en) enforced by a test over the message
   files.
7. Copy keys in `messages/{fa,en}/home.json`; sync with `npm run copy:sync -- home` and copy the en file to
   `backend-go/resources/translations/en/`.

## Tests
- Mapper table (SPEC §6.1, E-17): late 7/8/15; open period at EPL−2, EPL−1, hard cap, hard cap + 1; start window at
   next−4/−3/+14/+15; several warnings → one banner in table order; `unknown` main phase; Laravel payload → identical to
   today's `hero-state` tests (kept).
- Banner component tests (which action each button triggers).
- Copy-length test.
- Visual QA: light and dark screenshots of the four banner states and the ring (headless CDP + test-OTP token, local Go
  with `CYCLE_ENGINE=v6`), saved under `docs/engine-v6/qa/`.

## Out of scope
- Prediction card details (T-M9-43), contraception/limits copy (T-M9-44), the skip action (T-M9-47).

## Acceptance
- Verify passes; screenshots reviewed by the main session.
