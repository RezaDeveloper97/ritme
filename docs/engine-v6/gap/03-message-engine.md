# Gap analysis 03 — Message engine, alerts inbox, challenges (Engine v6 §5.3–§5.6)

Scope: `docs/engine-v6/SPEC.md` §5.3 (771–801), §5.4 (802–823), §5.4.1 (824–853), §5.5 (854–863), §5.6 CTR (880), §1.8 member_alert/member_tip (199), §4.3 paywall (758), §6.1 message tests (916).
Code inspected: `backend-go/internal/messages/{rules,v2.go,source.go,pregnancyalerts,menomessages,conditionnudges}`, `internal/admin/messages/{rules.go,engine.go}`, `internal/challenges`, `db/migrations/00061_messages_v2.sql`, `db/queries/messages/rules.sql`, `internal/home/challenge.go`, admin-web `screens/message-rules`, frontend `entities/message`.
Read-only analysis; nothing in the repo was changed.

**The main finding:** the v2 rule engine (B-N9-05, D-86) is a solid pure core: bands, levels, approval, a stable FNV hash, ops, audience, caps, a simulator. But **nothing in the product uses it**:
- The frontend never calls `GET /messages/selected` or `POST /messages/events`. grep across `frontend/src` finds no match; `entities/message/api/queries.ts` only calls `/messages/mode` and `/messages/daily`.
- `/home` never calls it either. The home daily challenge is still the legacy crc32 pick in `internal/home/challenge.go:20-98`.
- Pregnancy alerts, menopause messages and condition nudges are three more hard-coded engines running beside it.

So M9 is mostly integration work: one fact set from the v1.1/v6 resolver, multi-slot selection, server-side event logging, an alerts_inbox store with a lifecycle, and moving the separate engines into rules. It is not a rewrite.

## 1. Summary table

| # | Spec item | Status | Evidence |
|---|---|---|---|
| 1a | message fields id/name/type/slot/priority/status | ⚠️ | `message_rules` 00061:15-47. Fields are renamed: `kind`=type, `placement`=slot, `title`=name. The status set is draft/pending_approval/approved plus a separate `is_active`; spec status `active` maps to is_active. There is no `marketing` kind (rules.go:27-32). |
| 1b | content{title[variant], body, cta_text, cta_deeplink, variables[]} | ⚠️ | Exactly two variants `a`/`b` (rules.go:101-116). Variables are a fixed list of 8 (validate.go:13). There is no per-message `variables[]` declaration. |
| 1c | rule{all_of, any_of, exclude} | ⚠️ | Modelled as AND-of-groups, each group and/or, plus exceptions (rules.go:91-95, 317-335). That is more expressive, but `any_of` = "≥1 group" has no direct form: it needs one `or` group. Fact names are free text and not validated (validate.go:155-158). |
| 1d | ops `= != in not_in >= <= between contains` | ✅ | rules.go:98, 397-449 (also gt/lt). Missing fact: only ne/nin match (rules.go:396). |
| 1e | frequency{max_per_cycle, max_per_week, cooldown_days} | ⚠️ | Only one cap (`cap_count` + `cap_period` day/week/cycle) per rule (00061:27-28). You cannot set both cycle and week. A "cycle" is a rolling `cycle_length`/28 days, not the real cycle (rules.go:282-294). The cooldown counts only `impression` and ignores `dismiss` (rules.go:262). |
| 1f | schedule{start_at,end_at,local_hours} | ⚠️ | Date-only `starts_on/ends_on` plus a single hours window, Tehran fixed (rules.go:297-313). No datetime, no multiple windows, no per-user timezone. |
| 1g | audience_percent, ab_variants[{key,weight}] | ⚠️ | Audience ✅ (rules.go:198-200). A/B is 2 variants with `ab_split` instead of N weighted keys (rules.go:187-195). |
| 1h | goal_event | ⚠️ | A free-text `goal` column (00061:35). The goal event is reported by the client only (D-86 "honest gaps"). It is never matched against domain events. |
| 1i | clinical_review{required, approved_by, approved_version} | ⚠️ | `approved_by/approved_at` exist; `approved_version` is missing (00061:39-40). Any edit voids approval (admin/messages/rules.go:243-277). But the edited head is then **not served at all** until re-approved, because `Served()` needs approved (rules.go:166-174). The last approved version is not kept live. |
| 1j | message_event{user_id,message_id,version,variant,slot,event,at}, events eligible/shown/clicked/dismissed/goal | ❌ | `message_events` has rule_id,user_id,variant,event,created_at (00061:60-70). It has **no version and no slot**. Event names are impression/click/dismiss/goal (v2.go:30). There is **no `eligible`**, and `shown` is only what the client posts. The server logs nothing (Selected has "no side effects", v2.go:79-119). |
| 2a | bands safety 1000 / engine 800-999 / clinical 100-799 / product 1-99 | ✅ | rules.go:54-64; validated validate.go:72-74. |
| 2b | safety: no caps/AB/audience | ⚠️ | No caps ✅ (rules.go:177, 258). But safety still passes through `InAudience` (rules.go:255) and `Assign` A/B (rules.go:187). The validator allows audience<100 and variants on safety (validate.go:83-90, 149-151). |
| 2c | only medical advisor edits & publishes safety | ⚠️ | Approve/reject need `CapClinical` (rules.go:306-341). **Create, update, toggle and delete only need the `messages` capability** (rules.go:37-50, 358-372). Any messages admin can unpublish or delete a safety message. |
| 2d | engine messages: status banners from cycle warnings, max one | ❌ | No `warnings` fact exists. The status_banner capacity of 1 is implicit (single winner), but nothing ties `engine` kind to resolver warnings. |
| 2e | clinical/content need approval; text change voids it | ⚠️ | See 1i. `content` is not a separate kind: content messages ride in `clinical` (needs approval) or `product`. |
| 2f | product/marketing: daily+weekly caps; never within 24 h after a safety message | ⚠️/❌ | A cap is required for product (validate.go:113-115), but only one period. The **24 h after safety** suppression is missing. |
| 2g | levels info/suggestion/follow_up/urgent + mandatory behaviours | ⚠️ | Levels exist (rules.go:19-24). seen/confidence/contact are validated (validate.go:92-112). Gaps: no "suggestion requires exactly one CTA", only label/target pairing; no "info not for safety"; no "suggestion only engine/content/product"; follow_up kind not restricted to engine/clinical; urgent kind restriction is done (validate.go:101). `add_to_visit_note` is not in the generic API. `dismissible` is computed but not enforced server-side (v2.go:132). |
| 3a | triggers (open, log save, push schedule, mode change) | ⚠️ | Only pull on request (`/messages/selected`). Pregnancy alerts evaluate on log save (pregnancyalerts/engine.go:283) and daily (List with evaluate). No push schedule exists (no push channel). |
| 3b | state from the cycle engine for the local target day | ❌ | Facts come from the **legacy** engine (`legacy.New(...).CalculateForDate`, source.go:137-154; v2.go:60-69), not `internal/cycle/resolver` (v1.1). Only `mode,user_goal,is_premium,life_mode,name,cycle_day,cycle_length,phase,subphase,days_to_period,is_pms_window,is_fertile_window,pregnancy_week` exist (v2.go:39-75). |
| 3c | candidates: active + slot + schedule + local_hours | ✅ | SQL `ListActiveMessageRules` (rules.sql:59-63) + InWindow. |
| 3d | evaluation with hash(user,message) mod 100 < audience | ✅ | rules.go:180-200 (FNV-32a with salt "aud"). |
| 3e | caps from last shown or dismissed | ⚠️ | Only impressions (rules.go:262, 271); dismiss is ignored. |
| 3f | selection: safety, priority desc, **least-recently shown**, smaller id | ⚠️ | priority desc + id asc only (rules.go:228-233). There is no least-recently-shown tie-break. "Safety first" holds only because 1000 is the top band. |
| 3g | stable variant over weights; log eligible + shown | ⚠️ | Stable 2-way ✅. Server logs nothing ❌. |
| 3h | allowed rule fields | ❌ | 13 of the ~28 spec fields are missing (see §2.3 table). |
| 4 | slots & capacities | ❌ | 9 placements (rules.go:42), names differ from the spec. **No `member_alert`, `member_tip`, `alerts_inbox`** (`alerts` exists but is a single winner). Capacity is always 1, so "today's recommendations = 3" and the inbox list are impossible. No per-slot default caps (no 7-day no-repeat, no 1 per cycle per message, no in-app modal 1/week). Push: no channel and no quiet hours 23–08. |
| 5a | requires_user_input → only safety + engine | ❌ | No fact and no fixed rule. Only an admin-authored exception in a test (rules_test.go:156). |
| 5b | clinical with confidence=low hidden unless allowed | ❌ | No confidence fact, no rule. |
| 5c | fertile-window messages removed under hormonal contraception | ❌ | No `hormonal_contraception` fact and no "fertile-window" tag on messages. |
| 5d | emergency texts bundled, not server-disableable | ⚠️ | Out of the rules engine. Safety rules in `message_rules` can be toggled off by any messages admin (2c). The bundled-text check belongs to the safety/emergency audit (not covered here). |
| 6 | alerts_inbox from the same engine | ❌ | Pregnancy alerts are a **separate engine** with hard-coded detectors (pregnancyalerts/rules.go:116-300) and a `pregnancy_alerts` table. Lifecycle is only ack and add_to_visit_note (engine.go:681-726). No `resolved`, no `archived`/30 d; the list is a time window instead (engine.go:627-650). Merge is "dedupe per rule+key" (engine.go:298-356), so bp_high/sugar_high/fetal create **one row per reading** rather than updating the same message_id. Ordering and fan-out to modal/banner are not from the engine. The other two engines (menomessages D-43, conditionnudges D-41) are computed per request with no inbox at all. |
| 7a | simulator: user + date → engine output, winner per slot, why others dropped | ✅ T-M10-11 (`POST /messages/simulate`) — was: | `POST /messages/rules/simulate` takes **synthetic facts only**. user_id only seeds hashes and **history = nil, so caps and cooldown never trigger** (admin/messages/engine.go:85-123). The real-user path `/preview` is per single rule (rules.go:594-640). Engine output (cycle_view) is not shown. Reasons ✅ (explain.go:9-19). |
| 7b | editor conflict check | ⚠️ | Heuristic only: shared fact name + overlapping dates (validate.go:244-279). It ignores hours, audience and value overlap, and does not compute the real winner. |
| 8a | challenge{start_rule on_join / on_phase_start…} | ⚠️ | start_rule is anytime / menstrual / fixed_date (challenges/program.go:16-20). There is no generic on_phase_start for any phase. |
| 8b | eligibility & contraindications as rules; contraindications win | ⚠️ | Include/exclude lists over life mode and main phase only (program.go:34-68); exclude wins ✅. Contraindications are not a rule: you cannot express "<6 weeks postpartum" or pregnancy week. |
| 8c | completion = days_done ≥ min_days in duration | ✅ | program.go:110-134, service.go:251-370. |
| 8d | stats: participants, completion rate, day-by-day drop-off | ✅/⚠️ | D-87: participants, completion rate, users per day. Drop-off is derivable but not computed as a rate. |
| 8e | suggested through the "today's challenge" slot via the engine | ❌ | Home uses the legacy pick (home/challenge.go). `daily_challenge` placement rules exist but nothing serves them. |
| 9 | performance | ⚠️ | No cache. Every `/messages/selected` reads all active rules from the DB, rebuilds facts (DetectMode, profile, life mode, name, histories + legacy engine), and runs **2 history queries per candidate** (Last + Count, rules.go:262/271, store.go:44-58). That is roughly 6 + 2N queries per slot, times 6+ slots if the home screen adopts it. Event inserts are synchronous (v2.go:192). `MessageRuleEventStats` groups by `created_at >= ?` with no leading index (rules.sql:77-79): a full scan for the admin list. No retention or pruning of `message_events`. |
| 10 | tests | ⚠️ | Pure tests cover safety-first, caps/cooldown, A/B stable+split, audience monotonic, groups, window, validate, conflicts, simulate (rules_test.go:48-288). Missing: requires_user_input gate, 24 h post-safety, least-recently tie-break, quiet hours across Tehran midnight, cap boundaries, hash distribution, multi-slot capacity, inbox lifecycle. |

Other cross-cutting gaps:
- The CTR KPI (§5.6) is computed per rule over 7 days (admin rules.go:406). The spec wants it **per version**, which needs `message_events.version`. There is no automatic "CTR drop > 30 % vs 7-day mean" alert (§5.1).
- §4.3: there is no paywall in any message path. Nothing prevents a `product` rule from linking to a paywall CTA on alerts, though product is blocked on `alerts`/`urgent_modal` (validate.go:77). That is low risk; add a validator rule that a CTA on safety/urgent can't target `/plus`.

## 2. Findings (detail)

### 2.1 Data model (§5.3)
- **Versioning.** `RuleUpdate` bumps the version, snapshots it into `message_rule_versions` and sets status to draft whatever changed (admin rules.go:243-277). This satisfies "any text change voids approval", and is stricter: a priority-only edit also voids it. **Side effect:** an approved, live clinical message disappears for every user the moment someone fixes a typo. It stays gone until re-approved, because `Served()` requires approved and there is no `approved_version` to fall back on. Recommendation: store `approved_version` and serve the snapshot of that version while the head is in draft.
- **Approval covers the whole head row.** Ops changes such as audience or schedule void approval too. The spec only requires that for text. See ambiguity Q2.
- **Events.** A `message_event` needs version and slot for CTR per version (§5.6). The name mapping impression→shown, click→clicked, dismiss→dismissed is cosmetic and can be aliased.

### 2.2 Bands, levels and roles (§5.3 tables)
- Validation lives only in `Input.Validate`, so nothing re-checks a row inserted by a seeder.
- Missing level × kind constraints:
  - info: not for safety.
  - suggestion: engine/content/product only, exactly one CTA.
  - follow_up: engine/clinical only.
- Role gap: safety create, edit, toggle and delete should require `CapClinical` (spec: "only the medical advisor edits and publishes"). Today only approve and reject do (rules.go:306-341).
- `marketing` is not a kind. The spec groups product/marketing in band 1–99 but treats the in-app modal (marketing, 1/week) and alerts_inbox ("marketing not allowed") differently. A `marketing` kind (same band) or a `channel` tag is needed.

### 2.3 Fact set (§5.3 allowed fields) — what exists today

| Spec field | Exists? | Source today |
|---|---|---|
| mode | ✅ `mode`, plus `life_mode` | message manager DetectMode (legacy) v2.go:41-53 |
| main_phase | ⚠️ `phase` (legacy enum) | legacy engine v2.go:64 |
| subphase | ✅ | legacy v2.go:65 |
| cycle_day | ✅ | legacy v2.go:62 |
| days_to_period | ⚠️ derived `cycle_length - day + 1`, not the resolver's prediction | v2.go:66 |
| days_late | ❌ | — (resolver has it) |
| confidence | ❌ | — (resolver status.go:31) |
| data_quality | ❌ | — (resolver status.go:33) |
| warnings | ❌ | — (resolver status.go:76) |
| requires_user_input | ❌ | — (resolver status.go:35) |
| cycle_regularity | ❌ | — |
| suggest_pregnancy_test | ❌ | — |
| hormonal_contraception | ❌ | — (contraception module exists, not wired) |
| pregnancy week / trimester | ⚠️ week only, computed from LMP (ignores the M7 dating basis) | v2.go:71-75 |
| postpartum week | ❌ | — |
| last vitals category | ❌ | — |
| symptom_history | ❌ | — |
| plan | ⚠️ `is_premium` bool | v2.go:50 |
| days_since_signup, days_since_last_log, logs_30d | ❌ | — |
| segment | ❌ (admin user segments not wired) | — |
| age | ❌ | — |
| push_opt_in | ❌ (no push) | — |

Conclusion: the facts come from the **legacy** engine. The v1.1 resolver (`internal/cycle/resolver`) is already used by challenges (challenges/service.go:70-95) and carries most of the missing engine fields. The M9 fact builder should call the v6 resolver once per request and expose `StatusAPI` fields 1:1.

### 2.4 Slots (§5.4, §1.8)
Mapping today → spec:

| Today | Spec |
|---|---|
| urgent_modal | urgent modal |
| status_banner | home status banner |
| smart_tip | smart tip |
| daily_advice | today's recommendations (cap 3) |
| daily_challenge | today's challenge |
| push | push |
| modal | in-app modal |
| alerts | alerts_inbox |
| sms | (not in spec) |
| missing | member_alert |
| missing | member_tip |

`Select` returns one choice (rules.go:226-245), so capacity 3 and list slots need `SelectN` / `SelectAll`. There are no per-slot type restrictions (spec table "allowed type") beyond product being blocked on urgent placements.

### 2.5 Fixed rules (§5.4.1)
None of the three fixed rules (requires_user_input, low confidence, hormonal contraception) exist as engine invariants. They must be hard-coded in `Select`, not left to admin-authored exceptions, and they need a message tag such as `topic: fertile_window` or a `mentions_fertile_window` bool.

### 2.6 alerts_inbox (§5.4.1)
- Pregnancy alerts are a separate engine with their own detectors, store (`pregnancy_alerts`, `alert_type v2:<rule>`), `message_contents` group texts and act API. The spec says explicitly that this must not be so (line 825).
- Detectors are still valuable: they compute evidence (`what_we_saw`/basis vars). The right move is to turn each detector into a **fact producer** (e.g. `preg.vomiting_days_7`, `preg.bp_last_category`, `preg.fetal_low`, `preg.contractions_511`) plus an `evidence` builder. Each alert becomes a `message_rules` row in slot alerts_inbox, and a generic `message_inbox` table holds the lifecycle.
- Missing pieces:
  - the `resolved` state (re-evaluate; the condition no longer holds)
  - `archived` after 30 days (a sweep)
  - urgent not closable until acknowledged, enforced server-side
  - merge on message_id (update the count and last_at)
  - safety fan-out to urgent_modal and engine fan-out to status_banner
- Menopause messages (D-43) and condition nudges (D-41) are also stand-alone engines, but read-only. They should become rules over new facts (`meno.bleeding_flag`, `meno.score_delta`, `cond.heavy_pain_days`…).

### 2.7 Admin
- The simulator has no real-user mode and runs with nil history, so it cannot answer "why didn't user X see Y today". Preview does that, but for one rule only. Neither shows the cycle-engine output.
- The conflict check is a heuristic. A better check: same slot, overlapping date ranges and hours, and value ranges that intersect for shared numeric facts and enum sets. It should report the winner in the overlap, and say "safety always wins".

### 2.8 Challenges (§5.5)
The model is close.
- **Gaps:**
  - `start_rule on_phase_start(phase)` exists only for menstrual. `on_join` = anytime ✅.
  - eligibility and contraindications should reuse the rule engine's `[]Group` over the same fact set, with contraindications evaluated first and always winning. Today it is mode and phase lists only, so "<6 weeks postpartum" cannot be expressed.
  - suggestion through the `daily_challenge` slot: the home pick is legacy.
  - day-by-day drop-off rate.
- Uses the v1.1 resolver for phase ✅ (service.go:70-95).

### 2.9 Performance
Per `GET /messages/selected` (one slot):
- 1 query for the active rules
- facts: DetectMode (a few queries), profile, life mode, user name, cycle histories, plus the legacy engine
- 2 history queries per candidate (Last and Count)

If the home screen calls 6 slots, that is about 6×(6 + 2N) queries, with N candidates per slot. Event writes are a synchronous INSERT on the request path. `message_events` grows unbounded: there is no retention, and the stats query cannot use an index on `created_at` alone.

### 2.10 Tests — `go test ./internal/messages/... ./internal/challenges/...`
```
ok  	github.com/ritme/backend-go/internal/messages	(cached)
ok  	github.com/ritme/backend-go/internal/messages/conditionnudges	(cached)
ok  	github.com/ritme/backend-go/internal/messages/content	(cached)
ok  	github.com/ritme/backend-go/internal/messages/manager	(cached)
ok  	github.com/ritme/backend-go/internal/messages/menomessages	(cached)
ok  	github.com/ritme/backend-go/internal/messages/pregnancyalerts	(cached)
ok  	github.com/ritme/backend-go/internal/messages/rules	(cached)
?   	github.com/ritme/backend-go/internal/messages/store	[no test files]
ok  	github.com/ritme/backend-go/internal/challenges	(cached)
?   	github.com/ritme/backend-go/internal/challenges/store	[no test files]
```
All green. The admin int tests (`internal/admin/messages/rules_int_test.go`) and route tests (`internal/http/routes_messages_v2_int_test.go`) were not run here.

**Missing tests:**
- §6.1:
  - safety beats a priority-999 engine message even if the engine message has a lower id
  - requires_user_input=true hides clinical and product but keeps safety and engine
  - cooldown counted from **dismissed** as well as shown
  - stable A/B over N weighted variants
- property (rapid / testing/quick):
  - the hash is stable across process restarts (golden vectors for FNV)
  - the hash is roughly uniform: χ² over 100k users per bucket, ±2 %
  - the audience is monotonic for any p1<p2
  - the variant distribution matches the weights within tolerance
- boundaries:
  - cap N-1, N, N+1
  - cooldown at exactly `cooldown_days×24h` ± 1s
  - cap period "cycle" across a real period start
- quiet hours 23–08 Tehran:
  - 22:59 allowed, 23:00 blocked, 07:59 blocked, 08:00 allowed
  - the case where the server clock is UTC and the Tehran date differs (UTC 20:30 = 00:00 Tehran)
- 24 h after safety:
  - product blocked at +23h59m, allowed at +24h
- selection:
  - least-recently-shown tie-break
  - smaller id final tie-break
- capacity:
  - daily_advice returns 3 distinct
  - no repeat within 7 days
- inbox:
  - merge on the same message_id
  - resolved when the condition drops
  - archive at 30 d
  - urgent not closable before ack
  - safety duplicated to urgent_modal
- hormonal contraception removes fertile-window-tagged messages
- confidence=low hides clinical unless the rule explicitly references `confidence`
- the simulator equals the live path with real history (golden: simulate(user, date) winner == selected)

## 3. Spec ambiguities needing a user decision (recommended default)

| # | Question | Recommended default |
|---|---|---|
| Q1 | Keep the current naming (`kind`, `placement`, `impression`…) or rename to the spec's (`type`, `slot`, `shown`…)? | Keep the DB columns. Accept spec names as API aliases and map them in the admin UI. Avoids a migration churn with the Laravel twin. |
| Q2 | Does a non-text edit (priority, audience, schedule) void clinical approval? | Only content, rule and level/kind changes void it. Audience, schedule and caps changes do not. The version still bumps. |
| Q3 | While a clinical message's new version awaits approval, show the last approved version or nothing? | Serve the last `approved_version` snapshot (no gap for users). |
| Q4 | Does `max_per_cycle` mean the real cycle (since the last period start) or a rolling window? | The real cycle from the resolver's `cycle_start`. Fall back to a rolling `cycle_length` when unknown. |
| Q5 | "clinical with confidence=low hidden unless its condition explicitly allows": how does a condition allow it? | Allowed iff the rule has any condition on the `confidence` fact. |
| Q6 | How is a "fertile-window message" identified? | A new boolean `topics` tag (`fertile_window`) on the message, required by the validator when the text or conditions mention fertile/ovulation facts. |
| Q7 | Is safety exempt from audience and A/B entirely, or only from caps? | Exempt from all three. The validator forces audience=100 and single-variant for safety. |
| Q8 | Per-user timezone, or Tehran everywhere? | Tehran (civildate) now. Add `users.timezone` later (non-goal for M9). |
| Q9 | Push: no push channel exists. Build web push in M9 or only model the slot? | Model the slot, caps and quiet hours in the engine, and deliver to the in-app `user_notifications` inbox. Real web push is a separate milestone. |
| Q10 | Do the existing pregnancy-alert detectors move into rules, or stay as fact producers? | Detectors become fact and evidence producers. Thresholds move into rule conditions authored and approved in admin. Seed rules equal today's behaviour. The `pregnancy_alerts` table is migrated into the inbox. |
| Q11 | member_alert copies the owner's urgent safety alert: does the member see the owner's text? | No. Use a neutral fixed text ("پیامی از خانواده") plus a deeplink, per §1.8. Copy only message_id and level. |
| Q12 | Event `eligible` volume: log every eligible candidate, or only the winner? | Log eligible only for the slot's evaluated top-K (≤5), sampled at 10 % for product. Always log shown, clicked, dismissed and goal. |
| Q13 | Retention of message_events? | Raw rows 180 days. Daily rollup table (rule, version, variant, slot, event, day, count) kept indefinitely. |

## 4. Architecture & performance recommendation

1. **Compiled rule cache in memory, invalidated through Redis pub/sub.**
   - Load all active rules (with their approved snapshots) once into `map[slot][]CompiledRule`. Each rule is pre-sorted, with conditions pre-parsed (values coerced to float64 / sets).
   - Serve from an `atomic.Pointer`. On admin publish, toggle, approve or delete, bump `message_rules_rev` in Redis and `PUBLISH messages:rules:changed`. Every API instance reloads on the message.
   - Add a 60 s TTL safety net in case a pub/sub message is lost.
   - Justification: rules change rarely (admin), but reads happen on every home render × 6 slots. This removes ~6 queries per home render and parse cost per request. Pub/sub keeps beta, stage and multiple replicas consistent without restarts. Redis is already in the stack (ratelimit uses `d.Cache`).
2. **One fact build per request, shared by all slots.**
   - Add `GET /api/v1/messages/feed?slots=…` (or embed it in `/home`), which builds `Facts` once from the v6 resolver plus module producers and evaluates every slot.
   - Fact producers are lazy, keyed by the fact names the compiled rules reference. The union is known at compile time, so pregnancy and vitals queries never run when no active rule needs them.
3. **Per-user history snapshot instead of 2N queries.**
   - One query: `SELECT rule_id, event, MAX(at), COUNT(*) … WHERE user_id=? AND at >= now-max_window GROUP BY rule_id,event`. The window is the largest cap window (≤ 90 d).
   - Keep the result in a small in-memory `History`, plus `last_safety_shown_at` and `last_shown_at` per rule for the tie-break.
   - Optional: Redis counters `msg:cap:{uid}:{rule}:{period}` with TTL for push and product caps if load grows. Start with SQL; the index `(user_id, created_at)` is enough.
4. **Server-side event logging, async and batched.**
   - `shown` is logged by the server when a slot result is returned (spec step 7). The client still posts clicked, dismissed and goal (plus an `impression` alias for compatibility).
   - Events go into a buffered channel and are flushed every 1 s or 500 rows with a multi-row INSERT, in an in-process worker started through `auth.OnLifecycle`. The existing pattern is media `SweepLoop`.
   - Drain on shutdown. The cap-relevant `shown` write must not be lost silently, so fall back to a synchronous insert when the buffer is full.
   - Justification: removes a write from the request's critical path. Caps tolerate ~1 s lag; within the same request the shown rule is already known.
5. **Inbox as state, not log.** Add a `message_inbox` table:
   - columns `(user_id, message_id, version, slot='alerts_inbox', level, state, count, first_at, last_at, evidence_json, acked_at, archived_at)`
   - key `UNIQUE(user_id, message_id)` gives the merge
   - a daily sweep (OnLifecycle ticker) re-evaluates active inbox items to `resolved` and archives anything older than 30 d.
6. **Event table indexes and retention.**
   - Indexes: add `version`, `slot`; `(user_id, created_at)`; `(created_at, rule_id)` for stats.
   - A daily rollup into `message_event_daily` and pruning of raw rows older than 180 d, in the same sweep.
   - CTR per version is read from the rollup.
7. **Engine invariants in code, not data.** These checks run in `Select` before admin rules and are unit-tested:
   - safety first
   - requires_user_input gate
   - low-confidence gate
   - hormonal-contraception gate
   - 24 h post-safety product block
   - push quiet hours

## 5. Proposed task breakdown (backend first; admin-web separate; no Android, no prod cutover)

**M9-MSG-01 — Message model v6 migration + versioned approval**
- Scope:
  - goose migration: `message_rules` gains `approved_version`, `topics` (JSON list), `kind` adds `marketing`, an `any_of` group semantics flag (or `rule_json` v6 shape), `cap_week`, `cap_cycle` (multi-cap), `ab_variants` JSON `[{key,weight,content}]` (back-compat a/b)
  - `message_events` adds `version`, `slot`, and an `eligible`/`shown` event set
  - new indexes; Laravel twin
  - `Served` uses `approved_version` snapshot content when the head is not approved
  - approval-void policy per Q2
  - validator: level×kind matrix, safety audience=100 and no variants, suggestion exactly one CTA, follow_up/urgent evidence; fact names must belong to the registered fact vocabulary
  - safety create, update, toggle and delete require CapClinical
- touches: `db/migrations/0006x_messages_v6.sql`, `backend/database/migrations/*` (twin), `db/queries/messages/rules.sql`, `internal/messages/rules/{rules,validate,store}.go`, `internal/admin/messages/rules.go`
- deps: none
- tests: unit tests for the validator matrix; int test: edit an approved clinical rule → users still get the approved snapshot; approve → new text; non-clinical admin gets 403 on a safety toggle.

**M9-MSG-02 — Fact set v6 (resolver-backed, lazy producers)**
- Scope:
  - `internal/messages/facts` package: registry of fact name → producer, type and description (feeds the admin dropdown)
  - engine facts from the v6/v1.1 resolver `StatusAPI` (mode, main_phase, subphase, cycle_day, days_to_period, days_late, confidence, data_quality, warnings, requires_user_input, cycle_regularity, suggest_pregnancy_test)
  - hormonal_contraception from the contraception module
  - pregnancy week and trimester from the M7 dating
  - postpartum week; last vitals category; symptom_history; plan; days_since_signup; days_since_last_log; logs_30d; segment; age; push_opt_in (false until push exists)
  - lazy evaluation driven by the referenced-fact set; drop the legacy engine from `BuildFacts`
- touches: `internal/messages/facts/*`, `internal/messages/v2.go`, `internal/messages/source.go`, the module service read-only interfaces (cycle/resolver, contraception, pregnancy/v2, vitals, healthlog)
- deps: the Engine v6 resolver task (from report 01/02); MSG-01
- tests: table tests per producer; an int test where a user with an ambiguous period → `requires_user_input=true` and `warnings` present; a test that no producer runs when no rule references its fact (counting fake).

**M9-MSG-03 — Selection pipeline v6 (multi-slot, invariants, caps, tie-break)**
- Scope:
  - `SelectSlot(slot, cap)` returns up to the capacity (urgent_modal 1, status_banner 1, smart_tip 1, daily_advice 3, daily_challenge 1, push 1, modal 1, alerts_inbox all, member_alert, member_tip)
  - per-slot allowed kinds and default caps: smart_tip no default cap (dropped 2026-11-24: a matching tip shows every day; the rule's own caps and cooldown still apply); daily_advice no repeat within 7 d; modal 1/week (marketing); push 1/day and 3/week, quiet 23–08 Tehran
  - fixed invariants: requires_user_input; low confidence; contraception × `topics:fertile_window`; product/marketing blocked for 24 h after any safety shown
  - caps from shown **or** dismissed; real-cycle period per Q4; multi-cap
  - ordering: safety, priority desc, least-recently-shown, id asc
  - N-way weighted variant hash
  - single per-user history snapshot query
  - new slots `alerts_inbox` (rename of `alerts`, alias kept), `member_alert`, `member_tip`
- touches: `internal/messages/rules/{rules,explain}.go`, `internal/messages/rules/history.go` (new), `db/queries/messages/rules.sql`
- deps: MSG-01, MSG-02
- tests: every §6.1 message test; property tests (hash golden vectors, χ² uniformity, audience monotonic, weight distribution); cap and cooldown boundaries; quiet hours across Tehran midnight with a UTC clock; 24 h post-safety boundary; capacity 3 with no 7-day repeat; `Simulate` == `Select` property over random rules and facts.

**M9-MSG-04 — Rule cache + Redis pub/sub invalidation**
- Scope:
  - compiled-rule snapshot behind `atomic.Pointer`, reloaded on `messages:rules:changed` and on a 60 s TTL
  - admin mutations publish after commit
  - a metric/log line with the cache revision
- touches: `internal/messages/rules/cache.go` (new), `internal/admin/messages/rules.go`, `internal/http/routes_messages.go`, platform redis wiring
- deps: MSG-01
- tests: unit test that the cache reload swaps atomically; int test where an admin toggle is visible on the user endpoint immediately (pub/sub) and after TTL with pub/sub disabled; race test with `-race`.

**M9-MSG-05 — Feed endpoint + server-side event logging (async batch) + retention**
- Scope:
  - `GET /api/v1/messages/feed?slots=` (one fact build, all slots) and `GET /messages/selected` kept as a wrapper
  - the server logs `shown`, plus sampled `eligible` per Q12
  - `POST /messages/events` accepts clicked, dismissed and goal (alias impression → shown, ignored when the server already logged it today)
  - buffered batch writer started via `auth.OnLifecycle`
  - daily rollup `message_event_daily` and a 180-day prune sweep
  - CTR per version from the rollup
  - OpenAPI + contract case
- touches: `internal/messages/v2.go`, `internal/messages/events/*` (new), `db/migrations` (rollup), `db/queries/messages/*.sql`, `api/openapi.yaml`, `internal/http/routes_messages.go`
- deps: MSG-03, MSG-04
- tests: int: feed returns per-slot winners and writes shown rows after flush; flush on shutdown; full-buffer fallback is synchronous; prune keeps the rollup; benchmark of the feed with 200 rules < 15 ms p95 excluding the resolver.

**M9-MSG-06 — alerts_inbox store + lifecycle (generic)**
- Scope:
  - `message_inbox` table: UNIQUE(user, message), merge count and last_at, evidence what/basis required at level ≥ follow_up (no row without it)
  - states active → acknowledged | added_to_visit_note | resolved → archived after 30 d
  - urgent not closable before ack, enforced server-side
  - ordering urgent > follow_up > suggestion > info, then newest
  - safety also returned in `urgent_modal`, engine in `status_banner`
  - generic user API `GET /api/v1/alerts`, `POST /api/v1/alerts/{id}/actions/{ack|add_to_visit_note}`
  - evaluation triggers: feed/app open, log save hook, and a daily sweep ticker (resolve + archive)
- touches: `db/migrations`, `internal/messages/inbox/*` (new), `internal/http/routes_messages.go`, `api/openapi.yaml`
- deps: MSG-03, MSG-05
- tests: merge on the same message_id; resolved when the condition drops; archive at day 30 (fake clock); urgent dismiss → 409 until ack; IDOR (foreign id → 404); evidence-missing rule never creates a row.

**M9-MSG-07 — Migrate pregnancy alerts into the engine (fact producers + seeded rules)**
- Scope:
  - pregnancyalerts detectors become fact and evidence producers (`preg.*`)
  - seed `message_rules` rows (type and level per the §5.4.1 table: vomiting streak, weight missing, week entered, next visit, fetal movement, BP 140/90 and 160/110, sugar high, contractions 5-1-1, critical symptom), approved in the seed with `[needs clinical review]`
  - migrate `pregnancy_alerts` v2 rows into `message_inbox`
  - `/pregnancy/v2/alerts*` becomes a thin adapter over the inbox (same JSON shape, so contract goldens stay)
  - new deviation entry superseding D-21(c)
- touches: `internal/messages/pregnancyalerts/*`, `internal/messages/facts/pregnancy.go`, `db/migrations` (data move), `db/seeds`, `docs/go-migration/deviations.md`
- deps: MSG-06
- tests: the existing `pregnancyalerts` int tests stay green through the adapter; a parity table test where the old Detect() hits equal the new rule matches over fixtures; the fetal-movement safety alert appears in both urgent_modal and the inbox.

**M9-MSG-08 — Fold menopause messages + condition nudges into rules (optional, after 07)**
- Scope: `menomessages` and `conditionnudges` signals become facts; seeded rules in slot `alerts_inbox` / `smart_tip`; the old GET endpoints become adapters.
- touches: `internal/messages/{menomessages,conditionnudges}`, `internal/messages/facts/*`, seeds
- deps: MSG-06
- tests: the existing int tests stay green via the adapters; parity fixtures.

**M9-MSG-09 — Family slots member_alert / member_tip**
- Scope:
  - when an owner's inbox item is safety + urgent and the member's switch is on, write a member_alert copy (message_id and level only, neutral text per §1.8)
  - member_tip rules may only read owner `mode` and `pregnancy_week` facts; the validator refuses "never"-class facts for member_tip
- touches: `internal/messages/inbox`, `internal/companion/*` (read grants), `internal/messages/rules/validate.go`
- deps: MSG-06, the family task from the family gap report
- tests: alert copied only with the switch on; no details leak (JSON snapshot); a member_tip rule using `symptom_history` → 422.

**M9-MSG-10 — Challenges through the engine**
- Scope:
  - `challenge_programs.eligibility` and a new `contraindications` column as `[]Group` over the fact registry (contraindications evaluated first and always win)
  - `start_rule` gains `on_phase_start{phase}`; `on_join` = anytime
  - the `daily_challenge` slot candidate set = eligible, unjoined v2 challenges (a challenge message rule references `challenge_id`)
  - the home legacy pick stays as a fallback when no v2 candidate exists
  - day-by-day drop-off rate in stats
- touches: `internal/challenges/*`, `db/migrations`, `internal/messages/rules`, `internal/home/challenge.go`
- deps: MSG-02, MSG-03
- tests: pregnancy + intense workout refused even when eligibility includes pregnancy; postpartum week 5 refused and week 6 allowed; on_phase_start(luteal) joinable only on the first luteal day; drop-off math.

**M9-MSG-11 — Simulator v6 + conflict check v2 (backend)**
- Scope:
  - `POST /messages/rules/simulate {user_id?, date?, facts?}`: with a user, build real facts and **real history** at that date (audited); return the engine output (cycle_view summary) plus the winner per slot plus a per-rule reason, including invariant reasons (`requires_user_input`, `low_confidence`, `contraception`, `post_safety_24h`, `quiet_hours`, `capacity`)
  - conflict v2: interval and set intersection on shared facts, hours overlap, audience; report the winner in the overlap
- touches: `internal/admin/messages/{engine,rules}.go`, `internal/messages/rules/{explain,validate}.go`
- deps: MSG-03
- tests: simulate(user, date) winner == feed winner for the same clock; a cap reason appears with history; conflict v2 negative case (disjoint cycle_day ranges → no conflict).

**M9-MSG-12 — KPI: CTR per version + CTR-drop alert**
- Scope: CTR per version and variant from the rollup; a daily check where CTR drops > 30 % vs the 7-day mean raises an admin dashboard alert (§5.1).
- touches: `internal/admin/messages`, `internal/admin/dashboard`, the rollup queries
- deps: MSG-05
- tests: a rollup fixture where the drop is detected at 31 % and not at 29 %.

**Admin-web tasks (separate)**
- **M9-ADM-01 — Rule editor v6.** Fact dropdown from the registry endpoint (typed value inputs); all_of / any_of / exclude UI; N-variant weights; multi-cap; topics tag; approval banner showing "live = approved v{n}, draft v{m}"; safety editing disabled without the clinical role. touches: `admin-web/src/screens/message-rules/*`, `messages/{fa,en}.json`. deps: MSG-01, MSG-02. tests: vitest for form ↔ payload round-trip; light and dark.
- **M9-ADM-02 — Simulator v6 screen.** Pick a user and a date; show the engine output, the winner per slot and the dropped reasons; conflict v2 panel in the editor. deps: MSG-11.
- **M9-ADM-03 — Alerts inbox and challenges admin.** Inbox rules list filtered by slot; challenge contraindication builder; drop-off chart; CTR-per-version column. deps: MSG-06, MSG-10, MSG-12.

**Frontend (web) — needed for any of this to reach users, listed for completeness**
- **M9-WEB-01 — Consume `/messages/feed`.** Home status banner, smart tip, today's recommendations ×3, today's challenge, and the urgent modal driven by the feed; post clicked/dismissed/goal events.
- **M9-WEB-02 — Generic alerts inbox screen.** Replace the pregnancy-only alerts screen and keep its design.

Suggested order: MSG-01 → MSG-02 → MSG-03 → MSG-04 → MSG-05 → MSG-06 → MSG-07 → MSG-11 → MSG-10 → MSG-09 → MSG-08 → MSG-12. ADM-01 can start after MSG-02; the WEB tasks after MSG-05 and MSG-06.
