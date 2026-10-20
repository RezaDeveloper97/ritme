---
id: T-M10-04
title: Message selection v6: slots, invariants, caps, tie-break and weighted variants
milestone: M10
type: backend
status: todo
depends_on: [T-M10-03, T-M9-26]
parallel_group: M10-B
touches: [backend-go/internal/messages/rules, backend-go/db/queries/messages]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test -race ./internal/messages/... && make test-int PKG=./internal/messages/...
---

# T-M10-04 — Message selection v6: slots, invariants, caps, tie-break and weighted variants
**Model:** opus. Pure logic with many boundaries — write the tests first.

## Why
`Select` returns one winner per placement (`rules.go:226-245`), so "today's advice × 3" and the alerts list are
impossible; caps ignore `dismissed` (`rules.go:262`); a "cycle" cap is a rolling 28 days, not the real cycle
(`rules.go:282-294`); ties break by id only; A/B is two variants; safety still passes audience and A/B; none of the
three fixed rules of SPEC §5.4.1 exist (requires_user_input, low confidence, hormonal contraception), nor the 24-hour
block after a safety message, nor push quiet hours (gap 03 rows 2b, 2f, 3e–3g, 4, 5a–5c). DECISIONS.md M-04, M-05,
M-06, M-07, M-08, M-09; D-162.

## Scope
1. Slot table in code (`rules.Slots`): name, capacity, allowed kinds (T-M10-02 constants), default caps, list or
   winner — urgent_modal 1 · status_banner 1 (no cap, shown while its condition holds) · smart_tip 1 (default 1 per
   cycle per message) · daily_advice 3 (no repeat of a message within 7 local days) · daily_challenge 1 · push (1 per
   local day and 3 per 7 local days across all push messages; quiet hours 23:00–08:00 local) · modal (1 per 7 local
   days across all modal messages) · alerts_inbox (list of all eligible) · member_alert and member_tip reserved for M11.
2. Pipeline `Select(slot, facts, history, now) → (winners, dropped[])`, every dropped candidate with one reason:
   kind not allowed → schedule (`starts_at/ends_at`) → hour windows (user-local clock, X-07) → conditions
   (all_of/any_of/exclude) → audience (hash; safety exempt, M-07) → invariants → caps → ordering → capacity → variant.
3. Invariants in code (SPEC §5.4.1, never admin data):
   - `requires_user_input` → only safety and engine;
   - `confidence = low` → no clinical message unless the rule has a condition on `confidence` (M-05);
   - `hormonal_contraception` → no message with topic `fertile_window` (M-06);
   - product and marketing are blocked for 24 h after any safety message was shown to the user;
   - push: nothing between 23:00 and 08:00 user-local (M-08, M-09).
4. Caps: every cap of the rule (day/week/cycle) and `cooldown_days`, counted from the last `shown` **or** `dismissed`;
   `cycle` = since the engine's `current_period_start` (M-04), rolling `effective_cycle_length` when there is no
   anchor; `week` = the last 7 user-local days. Slot default caps apply on top.
5. Ordering: safety first; then priority desc; then least recently shown (never shown first); then smaller id.
6. Variants: weighted, stable per user and message: `fnv32a(salt|user|rule) mod Σweights`. For a legacy a/b rule the
   assignment must equal today's `Assign` for every user (no user flips variant on deploy).
7. History: one grouped query per request (extend T-M9-13's `ListUserMessageEventStats`): last shown and last
   dismissed per rule, counts per rule and event in the largest cap window, last safety shown (any rule), shown counts
   per slot in the last 7 days. Never a query per candidate.
8. Reasons enum for the simulator: `kind_not_allowed`, `schedule`, `hours`, `conditions`, `exclude`, `audience`,
   `requires_user_input`, `low_confidence`, `contraception`, `post_safety_24h`, `quiet_hours`, `cap_day`, `cap_week`,
   `cap_cycle`, `cooldown`, `slot_cap`, `capacity`.

## Tests
- SPEC §6.1: safety beats a priority-999 engine message with a smaller id; requires_user_input hides clinical and
  product and keeps safety and engine; cooldown counted from dismissed; stable A/B.
- Boundaries: caps at N−1, N, N+1; cooldown at exactly `cooldown_days × 24h` − 1 s / + 0 s; a cycle cap across a real
  period start; quiet hours 22:59 allowed, 23:00 blocked, 07:59 blocked, 08:00 allowed — with the server clock in UTC
  and the user in Asia/Tehran (UTC 20:30 = 00:00 Tehran) and in Europe/Berlin; product at +23:59:59 after a safety
  message blocked, at +24:00:00 allowed.
- Selection: least-recently-shown tie-break; id tie-break; daily_advice returns 3 distinct messages none shown in the
  last 7 days; hormonal contraception drops `fertile_window` messages; low confidence hides clinical unless the rule
  tests `confidence`; a kind not allowed in the slot is dropped with its reason.
- Property tests (`testing/quick` or seeded loops; no new dependency): golden hash vectors; χ² uniformity over
  100 000 users within ±2 %; audience monotonic for any p1 < p2; variant shares within tolerance of the weights;
  legacy a/b assignment unchanged for 10 000 users.
- One history query per request (counting DB wrapper).

## Budgets
- `Select` for one slot over 200 parsed rules with prefetched facts and history ≤ 200 µs (T-M10-05 lowers it).

## Out of scope
- Compiling and caching rules (T-M10-05), the feed endpoint and event writes (T-M10-06), the inbox (T-M10-07).

## Acceptance
- Verify passes; PROGRESS.md cites D-162.
