# Gap analysis 05 — Ritme companion & family access (Engine v6 §1.7, §1.8, §4.2 family row, §6.1 family tests, member_alert / member_tip)

Read-only, evidence-based. Source of truth: `docs/engine-v6/SPEC.md` §1.7 (lines 101–155), §1.8 (156–201), §4.2 row (734),
§5.4/§5.4.1 alerts_inbox (814–853), §5.6 family KPIs (881), §6.1 family tests (917). Code: `backend-go/` at the current
`stage` working tree (2026-10-21). All paths below are relative to the repo root unless they start with `internal/` or
`db/` (then relative to `backend-go/`).

Test run requested by the brief (unit tests only — `TEST_DB_DSN` is unset in this shell, so every `*_int_test.go`
SKIPs; verified with `-v`: `companion_int_test.go:69: testdb: TEST_DB_DSN is not set`):

```
$ cd backend-go && go test ./internal/companion/... ./internal/partners/... ./internal/children/... ./internal/sharelinks/... 2>&1 | tail -30
ok  	github.com/ritme/backend-go/internal/companion	0.519s
ok  	github.com/ritme/backend-go/internal/companion/home	0.952s
ok  	github.com/ritme/backend-go/internal/companion/shared	1.378s
?   	github.com/ritme/backend-go/internal/companion/store	[no test files]
ok  	github.com/ritme/backend-go/internal/partners	1.575s
?   	github.com/ritme/backend-go/internal/partners/store	[no test files]
ok  	github.com/ritme/backend-go/internal/children	(cached)
ok  	github.com/ritme/backend-go/internal/children/growth	(cached)
?   	github.com/ritme/backend-go/internal/children/store	[no test files]
ok  	github.com/ritme/backend-go/internal/sharelinks	1.618s
?   	github.com/ritme/backend-go/internal/sharelinks/store	[no test files]
```

Note on scope: `internal/partners` is the **home-lab booking partner** (`internal/partners/partners.go:1-12`), and
`internal/sharelinks` is the **doctor-report 7-day link** (`internal/sharelinks/sharelinks.go:1-19`). Neither is part of
the family model. The spec's `share_link` naming collides with the existing `health_share_links` table and the
`sharelinks` package. Pick a non-colliding name for the family link (for example keep `companions`, or use
`family_links`).

---

## 1. Summary table

Legend: ✅ present and spec-conformant · ⚠️ partial or conflicting with v6 · ❌ absent.

| # | Spec item (v6) | Status | Evidence (file:line) |
|---|---|---|---|
| 1 | `family_member {owner_id, member_user_id, role, status, expires_at}` | ⚠️ | `companions` table `db/migrations/00026_companions.sql:35-53` (owner_id, companion_user_id, type, status invited/active/revoked). Roles are only `partner\|spouse\|parent(teen)` (`internal/companion/companion.go:33-39`). There is **no `expires_at`** on the link, only on the invite. |
| 2 | `family_grant {member_id, domain, level off/view/nudge/write, updated_at}` one row per domain | ⚠️ | `companion_grants` `00026_companions.sql:78-88` has one row per (companion, section) with `updated_at`. Levels are `none/view/edit` (`companion.go:117-122`), with **no `nudge`**. `edit` means a *direct* write. The sections are the 5 legacy ones (`companion.go:57-67`), not the 9 v6 domains. |
| 3 | Per-domain max level, role defaults, quick-start templates | ❌ | `Section.MaxLevel` only separates teen sections from the rest (`companion.go:104-112`), so `edit` is grantable on cycle/pregnancy/symptoms. There are no role defaults and no templates (the grants come from the request, `handlers.go:193-196`). |
| 4 | "Never shared" list (notes, mood/EPDS, sex, labs, chats, full history, fertility) | ⚠️ | **Mood is shared** through `symptoms` (`internal/companion/shared/shared.go:45`). Free-text **notes** are shared in the meds and appointments views (`internal/care/medication.go:153`, `internal/care/appointment.go:217,228,232`). Labs, chats, EPDS and sex are not reachable (§2.4). |
| 5 | `child_profile {id, guardians[{user_id, level}]}` | ❌ | There is a single `children.owner_id` (`00031_children.sql:37-51`). The spouse sees children **read-only** through `families/family_children` (`internal/children/service.go:54-58,84,136-145`). There is no guardian table. |
| 6 | `family_activity {actor_id, subject_id, domain, action, payload_ref, created_at, undone_at}` | ⚠️ | `companion_audit_logs` (`00026_companions.sql:118-133`) stores owner, actor, link, section, action and time. It has no subject id (child), no payload_ref and no undone_at. Reads are coalesced to 1 per 15 min (`internal/companion/audit.go:16`). |
| 7 | `check(actor, subject, domain, action) → allow \| propose \| deny` | ⚠️ | `Service.access/Level/CanRead/CanWrite` (`internal/companion/service.go:639-682`) and `Delegation.Authorize` (`internal/companion/delegation.go:35-56`) return only allow or deny. Nothing handles **propose**, **nudge quotas**, **expiry** or **mode-inactive** grants. |
| 8 | `share_link` directional; mutual = 2 independent links | ⚠️ | Each `companions` row is directional, and `Revoke` touches only that row (`service.go:463-507`), so reverse links are structurally possible. A spouse link creates its own `families` row (`service.go:251-261`), so a mutual spouse pair gets two families. There is no `relation/template` field. |
| 9 | `share_request` (ask for more access; nothing applies until owner approves) | ❌ | — |
| 10 | Invite: 6 chars, single-use, 24 h | ✅ | `internal/companion/code.go:13-33` (32-symbol alphabet, HMAC at rest `:64-69`), TTL `service.go:24`, one-time + lock after 5 `service.go:391-433`. Throttles `internal/http/routes_companion.go:20-31,60-76`, global breaker `internal/http/companion_guards.go:28-38`. |
| 11 | Member previews role + domains before accepting; can decline | ❌ | `POST /companions/accept` activates on code entry (`internal/companion/handlers.go:410-456`, `service.go:434-446`). There is no preview and no decline. |
| 12 | Phone invite auto-binds on signup | ❌ | A phone-bound invite still needs the code (`service.go:401-409`). A signup-hook mechanism exists (`internal/auth/handlers.go:80-110`, used by learning `internal/http/routes_auth.go:21`) but companion does not register one. |
| 13 | Proposals: pending → approved/rejected; expire after 7 days | ❌ | Members with `edit` write **directly** into the owner's records (`internal/care/delegation.go:16-23`). |
| 14 | A med proposal never activates a reminder without owner approval | ❌ | `StoreMedication` with `for_user_id` inserts an active medication reminder in the owner's list (`internal/care/handlers.go:98-138`). `UpdateMedication` lets a member pause or resume her reminder (`:143-187`, `care/delegation.go:20-21`). |
| 15 | Nudges: 60 min after due, 1 per dose/appointment, 3/day/member, quiet 23–08, fixed neutral copy | ❌ | Nothing exists. The building blocks do: dose slots + intake log (`internal/care/today.go:31-60`, `db/migrations/00002_reminder_intakes.sql:10`) and a quiet-hours helper (`internal/notifications/policy.go:33-72`, user-configured, not fixed). |
| 16 | Undo within 24 h in "member activity" | ❌ | — |
| 17 | Child data: both parents `write`, every write stores the actor, last write wins | ❌ | No actor columns (`00031_children.sql`, `00033_baby_logs.sql:40-60`). Spouse writes get 403 (`internal/babylog/handlers.go:54,95`). |
| 18 | Deleting a child or removing a parent needs **both** parents | ❌ | The owner deletes unilaterally (`internal/children/service.go:287-293`, `handlers.go:290`). |
| 19 | Pregnancy domains only in mode=pregnancy; auto-deactivated (not deleted) on mode change; owner prompted | ⚠️ | Views are mode-aware at read time (`internal/companion/shared/mode.go:20-57`). But (a) grants are never deactivated, (b) **postpartum counts as pregnancy** (`mode.go:37-39`), (c) the grant survives a loss and **a new pregnancy shows immediately** (`shared.go:189-212`, `internal/loss/service.go:263-279`), (d) there is no owner prompt. |
| 20 | At delivery: suggest creating a child profile with shared parents | ❌ | — |
| 21 | Urgent alert relay → member slot `member_alert` (title, time, call owner/115; no medical details) | ❌ | Placements have no member slot (`internal/messages/rules/rules.go:42-45`). Urgent pregnancy alerts are persisted per owner (`internal/messages/pregnancyalerts/engine.go:388-433`, levels `rules.go:25`), so a hook point exists. |
| 22 | `member_tip` slot ruled on owner's mode/week, never on "never" data | ⚠️ | Phase tips come from `message_contents` group `companion_tip` (`internal/companion/home/tips.go:18-72`), not from the message engine. There are no pregnancy-week or postpartum member tips. |
| 23 | Member notifications (phase/week change, appointment tomorrow, missed dose, urgent), neutral on lock screen | ❌ | Only **owner**-inbox notices exist (`internal/companion/links.go:93-162`). The `notifications.Companion` category exists but no sender uses it (`internal/notifications/prefs.go:44`). There is **no push channel at all** in Go, the frontend or the android-shell (inbox `user_notifications` only). |
| 24 | Revoke/off takes effect immediately; member never sees the reason | ✅ | The grant is re-read from the DB on every request (`db/queries/companion/companion.sql:109-116`). Revoke deletes grants (`service.go:495`). The member just stops seeing the link (404) and gets no notice. Covered by `TestRevoke_RemovesAccess`. |
| 25 | Member can leave at any time | ✅ | `DELETE /companions/links/:id` (`handlers.go:485-503`). |
| 26 | Owner account deletion voids every grant instantly | ✅ | FK cascade `00026_companions.sql:51-52` inside the deletion transaction (`internal/profile/service.go:138-176`). Loss notices are purged (`:161`). |
| 27 | Child data remains for the other parent after owner deletion | ❌ | `children.owner_id … ON DELETE CASCADE` (`00031_children.sql:51`) deletes the child and every measurement, dose and baby log. |
| 28 | Every access/nudge/proposal/write logged | ⚠️ | Reads are coalesced to 15 min (`audit.go:13-34`, a deliberate CMP-L1 fix). `grants_changed` has no diff (`service.go:535`). Nudges and proposals don't exist yet. |
| 29 | Consent at every invite and every access change, scoped by domain+level (§4.2) | ❌ | The consent catalog has no family code (`internal/consent/catalog.go:16-42`). `user_consents` is UNIQUE (user, consent) (`db/migrations/00012_privacy_support.sql:30`) and cannot hold a per-member scope. |
| 30 | Admin user detail "Family" tab: counts, roles, invite statuses; no grant contents | ⚠️ | User summary has only an active-companion count via raw SQL (`internal/admin/users/v2.go:280-281`). The global masked links list exposes `grants_count` (a count, OK) (`internal/admin/companions/links.go:71-96`). There is no per-user family tab. |
| 31 | Family KPIs (§5.6: invite acceptance, members/circle, proposal approval, dose-after-nudge) | ❌ | Only a distinct-companion count (`internal/metrics/metrics.go:263-266`). |
| 32 | Owner's Plus features available to view-level members for shared data | ❌ | `sharelinks.go:18-19` "no companion access". Plus status is per user (`internal/plus/service.go:169`). |
| 33 | Household to-do list domain | ❌ | Todo is strictly per user (`internal/todo/model.go:7-8`). |
| 34 | Weight/BP/glucose domain; fetal movements & contractions summary domain | ❌ | The data exists (vitals, `pregnancy_kick_sessions`/`pregnancy_contraction_sessions` `00033_baby_logs.sql:16-24`) but no member projection. |
| 35 | TTC fertility for spouse behind a separate switch | ❌ | — |
| 36 | Member home (Fam_PHome) for **any** member | ⚠️ | `GET /companion/home` is male-only (`internal/companion/home/home.go:84-90`, `account.go:39-50`), so female members/carers get 403. |
| 37 | Men own their meds/appointments and share with the model | ⚠️ | Structurally possible (`CreateInvite` has no gender check), but nothing limits which domains a male owner may grant. |
| 38 | §6.1 family tests | ⚠️ | Partial: grant matrix, revoke, expiry, single-use, IDOR. None for proposals, nudges, mode change, child co-guardians or urgent alert (§2.18). |

Count: ✅ 5 · ⚠️ 14 · ❌ 19.

---

## 2. Detailed findings

### 2.1 Data model mapping

| Spec | Existing | Gap |
|---|---|---|
| `family_member.owner_id / member_user_id` | `companions.owner_id / companion_user_id` (NULL until accepted) `00026:35-53` | ✅ Reusable. Index `companions_owner_id_status_index (owner_id,status)`; `GetAccess` filters on owner + companion_user + status. A composite `(owner_id, companion_user_id, status)` index would serve check() better (§5). |
| `role` | `companions.type` ∈ partner/spouse/parent (`companion.go:33-39`). `parent` is the **teen** parent card (CB-TEEN-01), not "other parent of the child". | Add v6 roles `partner`(spouse), `other_parent`, `carer`, `friend`. Keep `parent` (teen) as a special legacy role outside the v6 matrix. |
| `status invited\|active\|revoked` | Same values (`companion.go:47-52`). Revoked rows are kept. | ✅. v6 "expired/declined → removed": currently a pending link survives an expired invite and is renewable (`service.go:282-330`). There is no decline. |
| `expires_at` | Only `companion_invites.expires_at` (`00026:62`) | ❌ Link expiry is needed for carers ("time-boxed default, e.g. until 6 weeks postpartum"). |
| `family_grant` | `companion_grants(companion_id, section, level, created_at, updated_at)`, UNIQUE (companion_id, section) `00026:78-88` | Levels lack `nudge`. Neither the code nor the data has per-domain max levels. There is no *active/mode-epoch* field to deactivate without deleting. |
| `child_profile.guardians[]` | `children.owner_id` + `families(owner, spouse_user, companion_id)` + `family_children(family_id, child_id)` (`00026:90-116`, `00031:97`) | Children are tied to the spouse **link** (deleted on revoke, `service.go:498`). v6 decouples child guardianship from the partner link: the other parent may not be the current partner. |
| `family_activity` | `companion_audit_logs` + child reads written into the same table with section `children` (`db/queries/children/children.sql:55-58`) | Missing subject (child id), payload_ref, undone_at/undone_by, proposal ref, and the per-domain diff on grant changes. |
| `share_link {relation, template, …}` | = `companions` row | No `template`. `relation` = type. |
| `share_request` | — | ❌ |
| invite `{code 6, single-use, 24h} or {phone → auto-bind}` | `companion_invites` (code_hash, phone, attempts, expires_at, used_at, revoked_at) `00026:55-76` | The code half is ✅. The phone auto-bind is ❌ (§2.6). |

### 2.2 Domain mapping (v6 domains ↔ current sections)

| v6 domain (max · partner default · carer default) | Current section | Gap / required change |
|---|---|---|
| Overall cycle phase (view · view · off) | `cycle` (view/edit) | The view returns **more than phase**: `cycle_day, cycle_length, days_to_period, days_late (≤14), predicted_next_period_start, confidence` (`shared.go:116-137`). `edit` is grantable. v6 says the friend role is "phase only" and full history is never shared → decision A1. |
| Pregnancy week & EDD (view · view · view), pregnancy mode only | `pregnancy` (view/edit) | Shows `current_week, trimester, gestational_age, estimated_due_date` while `pregnancy_profiles.pregnancy_mode=1` (`shared.go:189-212`). The grant is **not bound to the mode/episode** (§2.10). It also unlocks the layette checklist with `due_date/weeks_left` (`internal/http/routes_shop_checklist.go:88-94`, `internal/shop/checklist/handlers.go:180-185`). |
| Owner visits/appointments (write-with-approval · write · view) | `appointments` | `edit` = **direct** create/update (`internal/care/appointment_handlers.go:94-172`). The view leaks `notes`, `topic`, `result_note` (`care/appointment.go:217-232`). A delegated update cannot change `prep` (`appointment_handlers.go:151-153`), but `prep` is still returned. |
| Fetal movements & contractions summary (view · view · off), pregnancy mode only | — | ❌ New projection over `pregnancy_kick_sessions`, `pregnancy_contraction_sessions`, `pregnancy_fetal_movements` (`00033_baby_logs.sql:16-24`). |
| Owner weight/BP/glucose (view · off · off) | — | ❌ New projection over vitals. |
| Owner meds & supplements (nudge, add only with approval · nudge · off) | `meds` | `edit` = direct create, edit, pause and resume (`care/handlers.go:98-187`). The view leaks `notes` (`care/medication.go:153`). |
| Urgent alert (receive title + contact · on · off) | — | ❌ (§2.13). |
| Child profile: vaccines, growth, feeding, sleep, meds (write · write as co-parent · limited write feeding/diaper/sleep) | spouse family share, read-only | ❌ No guardians, carer sub-scopes or child **medications** (no child-med table: `grep child_id` on migrations hits only children/baby logs/telemed/directory/insurance). |
| Household to-do (write · write · off) | — | ❌ |
| *(dropped)* | `symptoms`: 7 days of pain, **mood**, symptoms, sleep, appetite/energy, skin & hair (`shared.go:41-45,140-161`) | **Has no v6 domain, and mood is in the never-share list** → remove (decision A3). |
| TTC fertility (partner only, separate switch, ttc mode only) | — | ❌ (decision A2). |
| Teen parent card (CB-TEEN-01) | `teen_period_week/teen_kit/teen_notes` (`companion.go:69-81`) | Not in v6. Keep as is (decision A17). |

### 2.3 check() vs current authorization

Current flow: `access()` (`service.go:645-670`) is an allow/deny decision:

1. The owner gets `edit`.
2. Otherwise `GetAccess(owner, viewer, section)` reads `companions ⋈ companion_grants` where status = active (`db/queries/companion/companion.sql:109-116`).
3. It drops grants the link type may not hold.
4. It runs the teen suspension (`internal/companion/teen_guard.go:29-36`).
5. It clamps teen sections to view.

`Delegation.Authorize` (`delegation.go:35-56`) wraps this and writes the audit row before data is touched (fail closed). What is missing for v6:

- **Expiry** (no link expiry).
- **Mode-inactive** (no epoch/active flag; only read-time view masking).
- **nudge** action and quota.
- **propose** outcome for `write` on owner data.
- **guardian** path for child subjects (`children.Access` is a separate code path, `children/service.go:111-133`, owner or spouse family, read-only).
- **Per-domain max level and role/never-share filter** in one place. Today the never-share rules are spread across the view builders.

Positive: every member path re-checks server-side on every request. There is no client-trusted level.

### 2.4 Authorization audit — every endpoint that serves owner data to someone else

| Endpoint | Enforcement | Per domain+level | Per mode | Expiry | IDOR | Never-shared exposure |
|---|---|---|---|---|---|---|
| `GET /companions/links/:id/sections/:section` (`handlers.go:510-545`) | `CompanionLink` checks the caller is the active companion (`links.go:41-57`), then `Level` (`service.go:639-670`), then audit, then `ReadFor` | ✅ (view/edit) | ⚠️ read-time masking only (`mode.go`) | ❌ | ✅ uniform 404 | ❌ **mood** via `symptoms` (`shared.go:45`). ❌ med/appointment **notes**, appointment `topic`/`result_note`/`prep` (`care/medication.go:153`, `care/appointment.go:217-232`). ⚠️ owner's custom item labels in symptoms (CMP-L7 open, `docs/security/bloom-companion.md:127`). |
| `GET /companion/home` (`home/home.go:77-216`) | male-only gate (`:84-90`). Per link and per granted section: audit + `ReadFor`; the children card through `children.CompanionCard` (audited) | ✅ | ⚠️ same | ❌ | ✅ (only the caller's links) | Same as above. Tips derive from shared views only (`home.go:218-234`) ✅. |
| `GET/POST/PUT /care/medications[/:id]?for_user_id` (`internal/care/delegation.go:46-78`, `handlers.go:79-187`) | `subject()`: same 403 for unknown account or no grant ✅. Single-record reads limited to active meds (`care/delegation_view.go:10-13`). Throttled (`companion_guards.go:45-55`). | ✅ | ❌ | ❌ | ✅ | ❌ `notes`. ❌ **direct writes**, including activating, pausing and editing her reminder. |
| `GET/POST/PUT /care/appointments[/:id]?for_user_id` (`appointment_handlers.go:75-172`) | Same, plus private loss follow-ups hidden (`care/appointment.go:190-193`) and only upcoming (`delegation_view.go:15-19`) | ✅ | ❌ | ❌ | ✅ | ❌ notes/topic/result_note/prep. ❌ direct writes (a member can also move a telemed-booked visit's time; only `booking_id` is protected, `appointment_handlers.go:159-160`). |
| `GET /shop/checklist?for_user_id` (`routes_shop_checklist.go:40,88-94`) | `Delegation.Authorize(…, SectionPregnancy, false)`. Audited. | ✅ | ❌ (no mode or episode binding) | ❌ | ✅ | `due_date`, `weeks_left` (pregnancy signal) under a stale grant (§2.10). |
| `/children/*`, `/children/:id/{feeds,sleeps,diapers,baby-logs}` (`internal/http/routes_children.go:30-47`, `routes_babylog.go:29-49`) | `children.Access`: owner, or spouse through an *active* spouse link family (`db/queries/children/children.sql:44-53`). Audited fail-closed (`children/service.go:147-157`). Writes need `Editable` (owner only). | n/a (no grant level; family membership) | n/a | ❌ | ✅ uniform 404 (`children/service.go:62-63`) | — (child data only). |
| `GET /teen/linked` (parent card) | Teen sections, view only, audited (`internal/teen/service.go:379-460`) | ✅ | ✅ (teen) | ❌ | ✅ | — |

Never-shared checklist (v6 §1.7):

| Category | Reachable by a member? |
|---|---|
| Private notes | **Yes**: med `notes`, appointment `notes/topic/result_note/prep`; symptom custom labels (CMP-L7). |
| Mood | **Yes**, through `symptoms` (`SymptomCategories` includes `mood`, `shared.go:45`). |
| EPDS score | No. `mood.weekly_checkin` is a `Link` param and links are never stored (`internal/healthlog/taxonomy/taxonomy.go:195,344`). No other path. |
| Sex & desire | No (category not in `SymptomCategories`). |
| Lab results | No (`internal/companion/record_scope.go:9-17`; health-record routes owner-only). |
| Assistant/doctor chats | No (no companion reference in assistant/telemed). |
| Full cycle history | Mostly no (today + next prediction only). `cycle_length`/`cycle_day` are history-derived numbers (A1). |
| Fertility data | No raw LH/BBT/intercourse values. `main_phase=fertile` is shown (teen mapped away, `mode.go:77-84`). There is no TTC switch (A2). |

**IDOR:** no IDOR found in existing paths. Every owner-side route resolves the link with `c.OwnerID == ownerID` (`links.go:16-21`, `service.go:297,522,554`). Every member-side route checks `companion_user_id == viewer && active` (`links.go:41-46`). `for_user_id` returns one 403 for "no such account" and "no grant" (`care/delegation.go:43-77`). Shared-children ids are verified (`handlers.go:596-620`, `children/service.go:212-232`). The new v6 endpoints (proposal approve/reject, undo, nudge, alert ack, share_request) must follow the same pattern (§3, tests §2.18).

### 2.5 Is the member's view built from engine output?

Yes, it is built from the cycle **engine** and not from raw logs. `cycleView` runs `cycleservice.Load`, then `metrics.Calculate`, then `resolver.Resolve` (`shared.go:109-115`). Gaps:

- It bypasses the canonical `cycle_view` builder (`internal/cycle/view/builder.go:100`, `cycle/service/service.go:150-221`). Any v6 engine rule applied in the builder or snapshot layer, such as §2.42 hormonal contraception (no fertile/ovulation) or the §2.43 UI mapping, can be missed on the member side. Recommendation: one engine function `PublicPhase(ownerID, date) → {main_phase}`, consumed by both the owner card and every member projection.
- It exposes numbers beyond phase (A1).
- `days_late` up to 14 days is still shown (`mode.go:15-18`). A late period is a pregnancy signal, and v6 "phase only" should drop it.

### 2.6 Invites

| Requirement | Status | Evidence |
|---|---|---|
| 6-char code, uniform alphabet, HMAC-SHA256 at rest with pepper | ✅ | `code.go:13-33,64-69` |
| Single-use, 24 h, lock after 5 phone mismatches | ✅ | `service.go:24-25,391-433` |
| Generic refusal, per-user/IP throttles, global failure breaker, SMS caps | ✅ | `handlers.go:407-445`, `routes_companion.go:20-31`, `companion_guards.go:28-114`, `sms_gate.go:12-23` |
| Pepper mandatory outside production | ⚠️ | `PepperMissing` only for production (`internal/platform/config/config.go:334-335`). Stage keys codes with the public default (`service.go:31-32`). CMP-L3 is still open. |
| Member sees the role + granted domains (read-only summary) **before** accepting; may accept without mutual sharing; may answer a mutual request | ❌ | Accept = activate (`service.go:434-446`). |
| Decline; expired/declined → removed | ❌ | — |
| Phone → auto-bind on signup | ❌ | Hook infra: `auth.Handlers.OnSignup` (`internal/auth/handlers.go:80-110`), used by learning `ClaimPending` (`internal/learning/signup.go:10-18`, `routes_auth.go:21`). |
| Owner confirmation of an accepted code-only invite | ⚠️ | Not done (CMP-M3 was mitigated with the breaker, option b). With the wider v6 domains (child write, urgent alerts, vitals) → decision A11/M6. |

### 2.7 Proposals

❌ Nothing exists. There is no table, state machine or approval endpoint. The current "«ثبت برای …»" delegation writes straight into `reminders` (`care/handlers.go:122-126`, `appointment_handlers.go:115-119`) and then sends an inbox notice to the owner (`delegation.go:58-80`). v6 needs:

- `pending → approved (apply + notify member) | rejected | expired (7 d)`.
- Apply inside a transaction as the owner, with `created_by_member` provenance.
- A medication proposal must not create an **active** reminder until approved (§6.1 test).
- Expiry has no scheduler in the codebase except ticker loops (§5.4). It must also be **lazy** (a pending proposal older than 7 days is treated as expired on read and on approve), so correctness does not depend on the sweep.

### 2.8 Nudges

❌ Nothing exists. The inputs are available:

- Per-day dose slots `TodayDoses` (`care/today.go:31-60`): `Meta.Times` per active medication.
- The intake log `reminder_intakes` (`00002_reminder_intakes.sql:10`).
- Appointment `scheduled_at` and `remind_at` (`care/appointment.go:203-226`).

Required:

- Eligibility: dose not logged by `due + 60 min`; the appointment rule is ambiguous (A7).
- Max 1 per dose/appointment.
- 3 per day per member, by Tehran calendar day. The existing Redis limiter is a rolling window from the first hit, not calendar-aligned (`internal/platform/ratelimit/ratelimit.go:1-16`).
- Fixed quiet hours 23:00–08:00 Tehran. The existing `InQuietHours` is the *user's* configurable window (`notifications/policy.go:33-54`).
- Fixed kind copy with no drug name on the lock screen.
- An activity log entry.
- The §5.6 KPI "dose logged after nudge" needs `nudge.sent_at` joined to `reminder_intakes.created_at`.

### 2.9 Child co-guardianship, undo, both-parent approval

- **Guardians:** ❌. `Access.CanEdit()` is owner-only (`children/service.go:84`). Spouse writes return `ErrReadOnly`, mapped to 403 in children and babylog (`children/handlers.go:70`, `babylog/handlers.go:54`).
- **Actor stored on each write:** ❌. None of these tables has `created_by/updated_by`: `children`, `child_measurements`, `child_vaccine_doses`, `child_milestone_checks`, `baby_feeds`, `baby_sleeps`, `baby_diapers` (`00031`, `00033`).
- **Last write wins with both records in the log:** ❌. Writes are upserts (`children/service.go:313-324,384-395`), and the overwritten values are lost. "Both records stay in the log" needs a write snapshot in the activity log (decision A14).
- **Undo within 24 h:** ❌. Undoing an *update* needs a before-image, and undoing a *delete* needs a soft delete or a snapshot.
- **Both-parent approval** to delete a child or remove a guardian: ❌ (`children/service.go:287-293`).
- **Side effect to watch:** baby feed writes rewrite the **mother's** day-log slot `baby.feeds_count` (`internal/babylog/service.go:398-420`). With co-guardian writes, a father's actions change her health log (derived count). Keep it, but document it, and key it on the mother's postpartum profile, as today.
- **Delivery → suggest a child profile with shared parents:** ❌. Hook it into the postpartum start (`db/queries/postpartum/postpartum.sql:42-44` is where `life_mode='postpartum'` is written).

### 2.10 Mode-change deactivation

What exists:

- Read-time masking: a pregnancy/postpartum owner without the pregnancy grant gets a neutral cycle view, and menopause is always neutral (`mode.go:41-46`).
- Teen suspension of partner/spouse links without deleting them (`teen_guard.go:13-60`). This is a good precedent for "deactivate, don't delete".

What is wrong for v6:

1. **Postpartum counts as pregnancy.** `pregnancyMode()` returns true for postpartum (`mode.go:37-39`), and `symptomModes()` passes postpartum-only options to holders of the pregnancy grant (`mode.go:51-57`). v6 says pregnancy domains apply only while mode = pregnancy.
2. **There is no deactivation and no episode binding.** The pregnancy view keys only on `pregnancy_profiles.pregnancy_mode` (`shared.go:190-196`). There is one profile row per user (`internal/pregnancy/repo.go:19-28`). A member holding the grant therefore sees a **new** pregnancy as soon as the owner re-enters pregnancy mode. That includes a pregnancy after a loss: the loss flow only sends a one-line notice and keeps the grant (`internal/loss/service.go:263-286`).
3. **There is no owner prompt** ("set up new access" card) and no "member sees only children + to-do until the owner approves".
4. The life mode is written from several places: profile upsert (`db/queries/profile/life_profile.sql:11-21`), postpartum start (`postpartum.sql:42-44`), teen, and the pregnancy profile toggle (`enums.ResolveLifeMode`, `internal/enums/life_stage.go:86`). A hook-only design will miss transitions. Recommendation in §5.2: a mode epoch compared at check time, plus best-effort hooks.
5. The member-facing grant list shows every section, including `none` (`handlers.go:763-770,808-832`). Under v6 the member projection must list **only effective** grants. Showing "pregnancy: inactive/pending" would itself reveal the mode change.

### 2.11 Account-deletion cascade

| Case | Today | v6 |
|---|---|---|
| Owner deletes her account | One transaction (`profile/service.go:138-176`). FK cascades remove links, grants, invites, families and audit rows (`00026:51-52,74,87,101-103,132`). Loss notices in members' inboxes are purged (`:161`). | ✅ grants void instantly. ❌ **children cascade-deleted** (`00031_children.sql:51`) with all baby logs. Must transfer to the remaining guardian. |
| Admin deletes a user | Same pattern (`internal/admin/users/users.go:383-387`). | Same fix. |
| Member deletes their account | Link row cascades (`companions_companion_user_id_foreign`), audit actor SET NULL (`00026:130-131`). | ✅. Future member-inbox rows (alerts, proposals) must be purged by link id, not by JSON scans (§3 L4). |

### 2.12 Audit log (family_activity)

`companion_audit_logs` records read, write, invited, accepted, revoked and grants_changed (`companion.go:177-185`). It never stores a payload (`00026:30-32`). Reads are coalesced per (owner, actor, link, section) per 15 min (`audit.go:13-34`). The owner can page and filter it (`handlers.go:346-403`). Gaps for v6:

- Actions: nudge_sent, proposal_created/approved/rejected/expired, undo, mode_deactivated/reactivated, alert_relayed, member_left, consent_given/withdrawn, guardian_request.
- Columns: subject (child id), domain diff on grant changes, payload_ref (+ sealed before/after for child undo, A14), undone_at/undone_by.
- Retention: the table grows unbounded today; recommendation A19.
- The spec line "every allow/propose written" conflicts with read coalescing. Keep the coalescing and record a deviation (A16).

### 2.13 Urgent alert relay, member_alert, member_tip, member notifications

- **Source of urgent alerts:** the pregnancy alert engine persists per-owner rows with level `urgent` (`pregnancyalerts/engine.go:388-433`, `rules.go:24-30`). Alerts are evaluated when the owner uses the app (`handlers.go:75` `EvaluateFirst`, `AfterSave`). §5.4.1 generalizes this into an `alerts_inbox` slot (another gap report covers it). The member fan-out must hook wherever an urgent alerts_inbox row is **created**, never re-evaluate the owner's rules on the member's request.
- **Content:** the spec allows "title, time, call owner and 115, no medical details". The owner's alert **title is itself medical detail** (for example «کاهش حرکات جنین»). Use a fixed neutral member title per level (A10).
- **Placements:** `rules.go:42` has no `member_alert`/`member_tip`. A `member_tip` rule must evaluate against a **member-scoped fact set**: only fields derivable from the domains the member currently holds. A tip conditioned on `mode=pregnancy` would otherwise reveal a pregnancy the member was not granted.
- **Delivery channel:** there is no push anywhere. Repo-wide search found no webpush, FCM or `showNotification` in Go, the frontend or the android-shell; the PWA service worker never touches `/api` (`frontend/scripts/sw.template.js:18`). Member notifications are in-app inbox rows (`user_notifications`, `00001_baseline.sql:659-672`) until a push channel exists (A9).
- **Purging:** today member notices are purged with JSON scans that have no `user_id` filter (`db/queries/companion/companion.sql:161-173`). v6 member notices need an indexed `link_id`.

### 2.14 Consent (§4.2 family row)

§4.2 asks for consent "at every member invite and every access change, scoped by domain and level", with `consent(user_id, type, version, granted_at, revoked_at, scope_json)`. Today:

- The catalog is AI-centric (`consent/catalog.go:16-42`).
- `user_consents` holds one row per (user, code) (`00012_privacy_support.sql:20-31`), so it cannot represent per-member scope history.
- The invite and grant-change audit rows carry no scope or text version (`service.go:266,535`).

Needed: a versioned family consent text and a ledger row per invite and per grant change, with `scope_json = {member_id, role, domains: {domain: level}, expires_at}`. Use the generic consent ledger if the §4.2 gap task builds one; otherwise use a `family_consents` table.

### 2.15 Admin & KPIs

- Admin user overview shows `companions` = the count of active links owned (`internal/admin/users/v2.go:280-281`, raw `QueryRowContext`, not sqlc, against `backend-go/CLAUDE.md` "sqlc only"). The admin-web shows it (`admin-web/src/screens/users/ui/UserOverview.tsx:132-133`).
- The global companions list masks names and mobiles and shows `grants_count` (`internal/admin/companions/links.go:66-96`). CMP-L6 is fixed.
- v6 family tab needs: member count, roles, invite statuses (pending/expired/accepted/declined), and children-guardian counts. No domain or level content.
- KPIs ❌ beyond the distinct-companion count (`metrics.go:263-266`).

### 2.16 Plus for members

❌. `plus.Service.Status(ctx, userID, now)` (`internal/plus/service.go:169`) can be called for the **owner** id inside a member request, but nothing does it. Which "Plus features for shared data" apply is undefined (A18).

### 2.17 Household to-do, vitals, fetal activity

All ❌.

- **To-do:** user-scoped with a uniform 404 for foreign ids (`todo/model.go:7-8`). It needs either an `owner_id`-scoped household list readable and writable by grant holders, or a `list_scope` column.
- **Vitals:** the data exists in `internal/vitals`. The member projection should be minimal: latest value + category + date, no raw history (A1-style decision).
- **Fetal activity:** today's kick-session summary and the last contraction-session summary (with 5-1-1 status). Pregnancy mode only.

### 2.18 Tests — what exists and what is required

Existing tests (names from `grep ^func Test`):

- `internal/companion` (int): `TestInviteAccept_ActivatesWithChosenGrants`, `TestAccept_OneTimeUse`, `TestAccept_ConcurrentRedeemsOnlyOnce`, `TestAccept_Expiry`, `TestAccept_InvalidCodes`, `TestAccept_PhoneBoundInviteCountsAttemptsAndLocks`, `TestInvite_Guards`, `TestRevoke_RemovesAccess`, `TestRevoke_PendingInviteDies`, `TestRevoke_CompanionLeaves`, `TestGrantMatrix` (3 levels × 5 sections), `TestSpouse_FamilyAndSharedChildren`, `TestAudit_NoPayloadAndOwnerScoped`, `TestParent_*`.
- `internal/companion` (unit): `TestNewCode_*`, `TestNormalize*`, `TestHashCode_*`, `TestLevelsAndGrants`, `TestValidateInput`, `TestParseGrants*`, `TestIsInviteRefusal_OneGenericAnswer`, `TestWriteNotice`.
- `shared`: `TestSharedEntry_CycleModeOnly`, `TestScopeAndPhase`, `TestReader_SectionsAreFilteredSubsets`, `TestReadFor_PregnancyWithoutGrantIsNeutral`, `TestReadFor_PostpartumAndMenopause`.
- `home`: `TestCompanionHome_AccessCombinations`, `TestTipKeysMatchRegistry`.
- `internal/http` (int): `TestCompanionRoutes_LifecycleAndIDOR`, `_ExpiryAndRenew`, `_AcceptThrottle`, `_DisabledWithoutPepper`, `_InviteSMSLimits`, `_DelegatedWriteAuditFailClosed`; `TestCompanionSecurity_TeenOwnerSuspendsPartnerLinks`, `_DelegatedRecordsStayInView`, `_AcceptCircuitBreaker`, `_AuditCoalescedAndPaged`, `_NoticeNameSanitized`, `_PregnantOwnerCycleNeutral`; `TestChildrenRoutes_SharedThroughCompanionLink`.
- `children`: `TestChildren_SharedWithSpouseReadOnly`, `TestChildren_OtherUsersChildIsNotFound`.
- `babylog`: `TestBabyLogs_AccessMatrix`.
- `db/migrations`: `TestCompanionsMigration_UpDown`, `TestChildrenMigration_UpDown`.

All integration tests SKIP locally without `TEST_DB_DSN`. Each M9 task's `verify` must use `make test-int PKG=…`.

Required by §6.1 (none exist yet; proposed names):

| §6.1 test | Proposed test | Package |
|---|---|---|
| Mutual links are independent (revoking one does not touch the other) | `TestFamily_MutualLinksIndependent`: A→B and B→A; revoke A→B; B→A link, grants and activity are untouched | `internal/family` (int) |
| Accept without mutual sharing | `TestInvite_AcceptWithoutMutual`: accept with no reverse request creates no B→A link; with a request the member may decline it and still accept | `internal/family` (int) + http |
| check() for every level | `TestCheck_Matrix` (pure, table-driven): domain × level{off,view,nudge,write} × action{view,nudge,write} × role × mode × expiry → allow/propose/deny + reason; max-level clamps; owner fast path; child guardian and carer sub-scopes | `internal/family` (unit) |
| A med proposal is not applied without approval | `TestProposal_MedicationNotAppliedUntilApproved`: member `POST` → 202 proposal; owner's `reminders` count and `TodayDoses` unchanged; approve → an active reminder exists; reject/expire → nothing | `internal/family` + `internal/care` (int) |
| Nudge cap 3/day | `TestNudge_DailyCapAndPerDoseOnce`: 4th nudge on the same Tehran day refused; next day allowed; second nudge on the same dose refused; 59 min refused, 60 min allowed; 22:59 allowed, 23:00 refused, 07:59 refused, 08:00 allowed (fake clock); copy has no drug name | `internal/family` (int) |
| Pregnancy domains deactivated after delivery | `TestModeChange_PregnancyDomainsDeactivatedAfterDelivery`: postpartum start → check denies `mode_inactive`, grant row kept, member projection hides it, owner review list shows it; re-approve → active. Variant: loss, then a new pregnancy → still denied until re-approved | `internal/family` (int) |
| Deleting a child profile needs both parents | `TestChild_DeleteNeedsBothGuardians` (+ `_RemoveGuardianNeedsBoth`, `_SelfLeaveUnilateral`) | `internal/children` (int) |
| Urgent alert only with the switch and without details | `TestMemberAlert_OnlyWithSwitchAndNeutral`: urgent alert for the owner → member with switch on gets `{title: neutral, at, actions:[call_owner, 115]}`; assert no rule title, body or evidence in the JSON; switch off → none; non-urgent → none; revoke → member alerts purged | `internal/family` + `pregnancyalerts` (int) |

Security tests to add:

- `TestMember_NeverSharedNotReachable`: walks every member endpoint and asserts no `mood`, `notes`, `prep`, `topic`, `result_note`, EPDS, sex, labs, chats, `cycle_length`/history, or TTC data without the switch.
- `TestRevoke_DeniesImmediately`: also with the Redis cache enabled, if one is added.
- `TestLink_ExpiredDenies`, plus the sweep marking it and logging activity.
- IDOR tests for proposal approve/reject, undo, nudge target, alert ack and share_request (foreign id → 404).
- `TestOwnerDeletion_ChildKeptForCoGuardian`.
- `TestAdminFamilyTab_NoGrantContents`.
- `TestMemberTip_NoHiddenFieldConditions`.
- `TestGuardianWrite_RecheckInTx` (TOCTOU, CMP-L4).
- Data-migration test: legacy `symptoms` grants removed, `edit` mapped, `pregnancy` grants bound to the current epoch.

---

## 3. Security / privacy findings (ranked)

| Id | Severity | Finding | Evidence | Fix |
|---|---|---|---|---|
| FAM-H1 | **High** | **Mood (a v6 never-share category) is shared** with any member holding `symptoms`, together with pain/sleep/energy/skin. It shows on the companion home and through section reads. v6 has no symptoms domain at all. | `internal/companion/shared/shared.go:41-45,140-161`; `home/home.go:36-39,205-210` | Remove the `symptoms` section (projection + section code). Migrate stored grants to off and notify the owner (A3). Add a never-share regression test. |
| FAM-H2 | **High** | **Member writes land directly in the owner's medications and appointments**, including creating active reminders and pausing or resuming hers. v6 requires proposals that apply only after owner approval ("a medication proposal never activates a reminder"). | `internal/care/delegation.go:16-23`; `care/handlers.go:98-187`; `care/appointment_handlers.go:94-172` | Turn `for_user_id` writes into proposals (202 + proposal id). Apply them as the owner on approval, in a transaction with the grant re-checked. Members can never change `is_active/notify` on owner records. |
| FAM-H3 | **High** | **Stale pregnancy grant across episodes.** It is not deactivated on mode change, on a loss, or when a *new* pregnancy starts. A member who once had "pregnancy: view" sees a new (possibly very early) pregnancy immediately, including after a loss, through `/companions/.../pregnancy`, the companion home and `/shop/checklist` (`due_date`, `weeks_left`). | `shared.go:189-212`; `loss/service.go:263-286`; `routes_shop_checklist.go:88-94`; `shop/checklist/handlers.go:180-185` | Bind mode-scoped grants to an owner **mode epoch**. Deny when the epoch differs and require owner re-approval (§5.2). Gate the shop checklist on the same check. |
| FAM-H4 | **High** (data loss / spec) | **Owner account deletion deletes the children** and every measurement, dose, milestone and baby log, so the other parent loses everything. | `00031_children.sql:51`; `profile/service.go:138-176`; `admin/users/users.go:383-387` | Add a guardians table. On deletion, transfer the child to a remaining guardian (or delete when none). Change the FK from CASCADE to an explicit transfer in code, with a Laravel twin. |
| FAM-M1 | Medium | The cycle view exposes more than "overall phase": cycle day, cycle length, days to period, lateness ≤14 d (a pregnancy signal), next-period date, confidence. | `shared.go:116-137`; `mode.go:15-18` | Reduce to the v6 projection (A1) and drop `days_late` entirely. |
| FAM-M2 | Medium | **Private free text reaches members**: med `notes`, appointment `notes/topic/result_note/prep`, and symptom custom labels (CMP-L7 open). | `care/medication.go:142-170`; `care/appointment.go:203-237`; `docs/security/bloom-companion.md:127` | Member projections use an allow-list DTO (title, time, form/dose, location, with) and never the owner JSON. |
| FAM-M3 | Medium | **Postpartum is treated as pregnancy** for the pregnancy grant (postpartum-only options pass). | `mode.go:37-39,51-57` | v6: pregnancy domains active only in mode=pregnancy. |
| FAM-M4 | Medium | **No per-domain max level.** `edit` is accepted on cycle, pregnancy and symptoms. v6 caps cycle/pregnancy/vitals/fetal at view and meds at nudge. | `companion.go:103-112` | Domain registry with `MaxLevel`, enforced in validation **and** in check(). |
| FAM-M5 | Medium | **Code-only invites activate on first redemption with no member preview and no owner confirmation.** The wider v6 scope (child write, urgent alerts, vitals) raises the impact of a guessed or intercepted code (CMP-M3 residual). | `service.go:368-459`; `handlers.go:410-456` | Preview, then explicit accept. For templates with write/child/urgent/vitals domains, require a phone-bound invite or owner confirmation (A11). |
| FAM-M6 | Medium | **TOCTOU:** authorization and the write are separate statements (CMP-L4 open). With v6 guardian writes and proposal approvals, a revoke during an approval or a write must win. | `delegation.go:35-56`; `docs/security/bloom-companion.md:124` | Re-check the link/guardian with `SELECT … FOR SHARE` inside the write transaction. |
| FAM-M7 | Medium | **No scoped consent record** for invites or grant changes (§4.2). There is no evidence of what the owner agreed to share, or when. | `consent/catalog.go:16-42`; `00012_privacy_support.sql:30`; `service.go:535` | Family consent ledger with a text version + `scope_json` (§2.14). |
| FAM-M8 | Medium (product/privacy) | The companion home is male-only, so female carers and friends fall back to raw section endpoints. In v6 the member home must exist for every member and be built only from `check()`-allowed projections. | `home/home.go:84-90` | New member home keyed on links, not gender. |
| FAM-L1 | Low | Stage uses the public default pepper for invite codes (CMP-L3 open). | `config.go:334-335`; `service.go:31-32` | Fail closed outside local/testing/contract. |
| FAM-L2 | Low | A blocked owner (`users.blocked_at`) stays readable by members (CMP-I1 open). | `companion.sql:109-116` | Join the owner's `blocked_at IS NULL` in the check() snapshot query. |
| FAM-L3 | Low | Accept timing oracle (CMP-L2) and phone-bound invite lock DoS (CMP-L8) are still open. | `docs/security/bloom-companion.md:122,128` | As in the review. Re-evaluate with the preview endpoint, which is another code oracle and needs the same throttles and breaker. |
| FAM-L4 | Low (perf/privacy) | Member-notice purges use unindexed JSON scans without a `user_id` filter. They will slow down and are easy to miss for new notice kinds. | `companion.sql:161-173` | New member notices carry an indexed `link_id` (FK, cascade) and are purged on revoke and deletion. |
| FAM-L5 | Low | `grants_changed` has no diff, so the owner (and consent evidence) cannot see what changed. | `service.go:535` | Store `{domain: [old, new]}` (no health payload). |
| FAM-L6 | Low | Member endpoints don't send `Cache-Control: no-store`. This is defense in depth only: the SW never caches `/api` (`frontend/scripts/sw.template.js:18`). | `handlers.go` (no header); compare `internal/labs/handlers.go:527` | Add `private, no-store` on family responses. |
| FAM-I1 | Info | Raw SQL in the admin user summary breaks the sqlc-only rule. | `admin/users/v2.go:276-282` | Move to sqlc when the family tab is built. |

---

## 4. Spec ambiguities needing a user decision (recommended default)

| Id | Question | Recommended default |
|---|---|---|
| A1 | What exactly does "overall cycle phase" show (today it shows cycle day, length, days-to-period, lateness, next date, confidence)? | `main_phase` only, plus an optional coarse "period soon" flag (≤3 days) for the partner role. No dates, cycle day/length, lateness or confidence. The friend role gets phase only. |
| A2 | Is the `fertile` phase label "fertility data"? What does the partner's TTC switch include? | In cycle mode the `fertile` phase label is allowed in cycle_phase, as the existing product does. In TTC mode, fertile window/ovulation day/"best days" sit behind the separate partner-only switch (default off). Raw LH/BBT/intercourse logs are never shown. Hormonal-contraception owners never expose fertile (engine §2.42). |
| A3 | Drop the legacy `symptoms` section (mood is never-share; v6 has no symptoms domain)? | Yes. Migrate existing grants to off, give the owner a one-time "this sharing ended" card, and change nothing silently. |
| A4 | Meds domain: does "propose add" need a `write` level, or is it allowed at `nudge`? | Levels off/view/nudge/write, where `write` = propose-only (never a direct write). Spouse default = nudge, as in the spec table. |
| A5 | The pseudo-code makes every non-child write a proposal; the prose limits proposals to new meds/appointments. To-do? Carer child writes? | Proposals only for the owner's own health records (meds, appointments). Household to-do and carer child sub-scopes (feeding/diaper/sleep) are direct writes, logged with a 24 h undo. |
| A6 | Nudge cap scope: 1 per dose overall or per member? "3/day per member" across owners or per pair? | 1 per dose/appointment overall (the owner gets at most one nudge per dose). 3 per Tehran day per (member, owner) pair. |
| A7 | What is an appointment that "wasn't logged on time"? | One nudge allowed between the owner's own reminder time (`remind_at`) and `scheduled_at`, if the appointment is still `scheduled`. |
| A8 | Quiet hours: fixed 23–08 or the owner's notification preferences? A nudge tapped in quiet hours: queue or refuse? | 23:00–08:00 Tehran is a hard floor, plus the owner's own quiet hours on delivery. Refuse with `quiet_hours` and "available from 08:00"; never queue stale nudges. |
| A9 | Delivery channel (no push exists). Should urgent member alerts reach a member who isn't in the app? | In-app inbox for everything now. Add an opt-in neutral SMS («پیامی از خانواده در ریتمی — اپ را باز کن», discreet template) for urgent alerts only, per member, owner-controlled. Push is separate work. |
| A10 | Is the alert title (for example «کاهش حرکات جنین») a "medical detail"? | Yes. The member gets a fixed neutral title per level («{name} نیاز به پیگیری زودتر دارد»), the time, call owner and 115. Never the rule title, body or evidence. |
| A11 | Phone auto-bind: auto-activate or bind-then-accept? Phone invite TTL? Owner confirmation for code-only invites with high-impact domains? | Bind to the new account as a *pending acceptance*: the member sees the summary and accepts or declines. Phone invites live 7 days. Code-only invites that include child write, urgent alert or vitals require owner confirmation after acceptance. |
| A12 | Expired/declined invites: delete the row or keep it? | Mark revoked (`revoked_by = expired\|declined`), hide from lists, and keep it for the audit/KPI "invite acceptance rate". |
| A13 | Carer time-box default ("e.g. until 6 weeks postpartum")? | If pregnant: EDD + 42 days. If postpartum: delivery date + 42 days. Otherwise 30 days from acceptance. The owner can extend. |
| A14 | Undo scope and "both records stay in the log" (needs payloads in the log, which contradicts the current "never a payload" audit rule) | Undo for member/guardian creates, edits and deletes on child and household data (owner meds/appointments go through proposals). Store sealed (AES-GCM) before/after JSON in the activity row, purge it after 30 days, and keep actor/action/time 12 months. |
| A15 | "Removing a parent needs both parents": must the removed parent consent? Deadlock or abuse case? | Removing the *other* guardian needs both. Leaving yourself is unilateral. Pending requests expire after 7 days. Safety escalation through support (the clinical-support role, audited). |
| A16 | "Every allow is written to family_activity" vs read coalescing (CMP-L1) | Keep 15-min read coalescing and record it as a documented deviation. Writes, proposals, nudges and lifecycle events are always written. |
| A17 | Existing link types in v6 | partner → partner, spouse → partner + co-guardian of the shared children (migrate `family_children` to guardians), teen `parent` kept as a special role outside the v6 matrix. |
| A18 | Which Plus features reach view-level members? | Only Plus analyses of the domains the member already sees (for example the vitals trend if vitals: view). Never a PDF of the owner's record. Defer to a follow-up. |
| A19 | Activity retention | 12 months for rows. Sealed payloads 30 days (A14). |
| A20 | Which domains may a male owner grant? | meds, appointments, household (and children through guardianship). |
| A21 | The §6.1 wording "revoked member gets 403"; the current API gives 404 for a revoked link | Keep 404 for unknown/revoked links (no existence oracle) and 403 for an active link without the domain. Tests assert "no data". |

---

## 5. Architecture & performance

### 5.1 Where the code lives

- Keep the existing tables and **evolve them in place** with ALTERs and a data migration, instead of a big-bang copy: `companions` = family link, `companion_grants` = family_grant, `companion_audit_logs` = family_activity. Every schema change needs a **goose migration plus a Laravel twin** (`backend-go/CLAUDE.md` schema-change rule; T-M2-26/27 are blocked, so the rule stays in force), and `make schema-diff` must pass. The contract fixture `backend-go/contract/fixtures/dump.sql` also gets the new empty tables (D-47 precedent).
- New package **`internal/family`**: the domain registry (9 domains + TTC switch + teen sections), roles/templates/defaults, `Authorizer.Check`, proposals, nudges, activity, guardians, member projections. Make `internal/companion`'s `Level/CanRead/CanWrite/Delegation.Authorize` delegate to `family.Authorizer`, so every current consumer (care, shop checklist, teen, IVF, loss, children) gets v6 semantics in one switch. Keep `/api/v1/companions/*` working with an old→new mapping until the frontend moves to `/api/v1/family/*` (a Go-only addition: deviations entry + OpenAPI).
- **Parallel-safety warning for the orchestrator.** sqlc emits *all* tables into each of the 48 `store/models.go` files (`backend-go/sqlc.yaml`, 48 `queries:` entries; 47 `models.go` files are dirty in the current tree). Every M9 migration therefore rewrites ~48 generated files, so **schema tasks must be serialized**. Put new queries under the existing `db/queries/companion/` (or add one `family` sqlc entry once) to avoid `sqlc.yaml` churn.

### 5.2 check() design

```go
type Outcome uint8 // Deny, Allow, Propose
type Decision struct{ Outcome Outcome; Reason string; LinkID uint64 } // reason: no_link|expired|mode_inactive|level|never_shared|quota|not_guardian
// Pure, exhaustively unit-testable:
func Decide(s Snapshot, subj Subject, d Domain, a Action, now time.Time) Decision
// Snapshot = {Link{ID, Role, Status, ExpiresAt, OwnerBlocked}, Grants map[Domain]{Level, ModeEpoch},
//             Owner{Mode, ModeEpoch}, Guardian{Level, SubScopes} (child subjects)}
```

Order of evaluation:

1. Actor is the owner → allow.
2. Child subject + guardian → allow (carer: listed sub-scopes only).
3. Link missing, not active, or owner blocked → deny.
4. `now ≥ ExpiresAt` → deny `expired`.
5. Domain not grantable for the role, or never-share → deny.
6. Mode-scoped domain and (owner mode outside the domain's modes **or** `grant.ModeEpoch ≠ owner.ModeEpoch`) → deny `mode_inactive`.
7. Level below what the action needs → deny.
8. `nudge` → the quota check (a DB step) decides.
9. `write` on the owner's own record → propose.
10. `write` on household → allow.

The caller then logs to family_activity (reads coalesced).

**Mode epoch.** Store `family_owner_state(owner_id, mode, mode_epoch)`. A single `SyncMode(owner)` resolves the effective mode (`enums.ResolveLifeMode` + pregnancy profile) and bumps the epoch when it changes. Call it best-effort after every mode write (profile life-mode upsert, postpartum start, pregnancy start/end, loss, teen). **Also** run it lazily inside snapshot loading (self-healing), so a missed hook can only *deny*, never leak. A grant to a mode-scoped domain is created or re-approved with the current epoch. A new pregnancy is a new epoch, which closes FAM-H3.

### 5.3 Caching and DB cost

Today:

- One member section read costs `GetCompanion` + `GetAccess` + `GetUserLifeMode` (teen) + `RecentReadAudit` + maybe `InsertAudit`, before the view is built.
- The companion home repeats this per link × 5 sections (`home/home.go:170-185`) and runs the engine once per cycle view.

The owner's own requests never touch it (owner fast path).

Recommendation, in order:

1. **Batch, don't cache, first.** One query loads the full snapshot for a member: `links ⋈ grants ⋈ family_owner_state ⋈ users(blocked_at)`, for one owner or for all owners of the member (member home). Keep it in a request-scoped memo, so `Decide` runs in memory per domain. This cuts the member home from O(links × domains × 4) queries to O(1) for authorization. Add the composite index `companions(companion_user_id, status, owner_id)` (exists as `(companion_user_id, status)`; add `owner_id`) and `(owner_id, companion_user_id, status)`.
2. **Read-audit coalescing in Redis**: `SET ritme-go:family:ra:{owner}:{member}:{domain} 1 NX EX 900` before inserting. On a Redis error, fall back to the existing DB check (`audit.go:19-34`).
3. **Redis grant cache only if metrics show the need.** Volume is member traffic only, which is small. If added: owner-scoped version key `ritme-go:family:ver:{owner}` (INCR **after commit** on any link/grant/expiry/mode/deletion change) and snapshot key `family:snap:{owner}:{member}:v{ver}` with a 300 s TTL. Redis errors → DB. **Write paths** (proposal apply, guardian writes, nudges) always re-check inside the transaction with `FOR SHARE` (fixes CMP-L4). Revocation then stays "immediate" up to the INCR latency. Account deletion must also INCR.

### 5.4 Quotas and scheduling

- **Nudge quotas: DB, not Redis.** Use `family_nudges` with `UNIQUE(owner_id, target_type, target_id, due_date, slot)` (1 per dose/appointment, idempotent against double taps) plus a count of `(owner_id, member_id)` nudges in the Tehran day `[00:00, 24:00)`. Run it in one transaction under a lock on the link row. At 3/day the volume is trivial, and Redis INCR+EXPIREAT would lose the cap on a flush and would not provide the per-dose uniqueness or the KPI join (`sent_at` ↔ `reminder_intakes`).
- **Proposal expiry and link expiry: DB-driven sweeps, not delayed asynq tasks.** The codebase pattern is in-process tickers registered through `auth.OnLifecycle`: partners `internal/http/routes_partners.go:63`, sharelinks `internal/sharelinks/service.go:303`, learning outbox `internal/learning/dispatch.go:116-180`, safety `routes_admin_safety.go:27`. The queue wrapper only supports immediate `Dispatch` (`internal/platform/queue/queue.go:151-178`), with no `ProcessAt` or periodic scheduler, and delayed 7-day Redis tasks are lost on a flush. Use:
  - `UPDATE … SET status='expired' WHERE status='pending' AND expires_at <= now` hourly, idempotent, safe with one `backend-go` container per environment (`docker-compose.stage.yml:78-79`). Lazy expiry on read and approve keeps correctness independent of the sweep.
  - The carer `expires_at` sweep (status, activity row, owner inbox).
  - The sealed-payload purge (daily) and the invite cleanup (daily).
- **asynq is for immediate fan-out only**, where the owner's request must not wait:
  - Urgent alert → member_alert copies (job `family_member_alert`, retries, idempotent on `(member_id, source_alert_id)`).
  - Member notifications outbox dispatch, following the learning outbox model: DB outbox + `notifications.Decide` + deferral (`learning/dispatch.go:116-128`).

### 5.5 Indexes (new tables)

- `family_activity(owner_id, id)` (paging, already the pattern), `(subject_type, subject_id, created_at)`, `(undo_deadline)` for the sweeper.
- `family_proposals(owner_id, status, created_at)`, `(status, expires_at)`.
- `family_nudges` unique as above + `(member_id, owner_id, sent_at)`.
- `family_member_alerts(member_id, created_at)` + `link_id` FK + UNIQUE `(member_id, source_alert_id)`.
- `child_guardians` UNIQUE `(child_id, user_id)` + `(user_id, status)`.

---

## 6. Proposed task breakdown (backend first)

IDs are local (`FAM-xx`). The orchestrator maps them to `T-M9-xx`. Every backend task:

- type `backend`
- skills `[new-endpoint]` where routes change
- verify = `cd backend-go && make sqlc && go vet ./... && go test ./internal/family/... ./internal/companion/... && make test-int PKG=<pkgs> && make schema-diff` (schema-diff only when migrations change)
- updates `backend-go/api/openapi.yaml` + `docs/go-migration/deviations.md` for Go-only routes.

**Schema tasks (FAM-01a/b/c) must run serially** (§5.1). Decisions A1–A21 should be answered first. FAM-00 records them.

### FAM-00 — Family v6 decisions & domain registry doc (investigate)
- Scope: record the user's answers to A1–A21 in `docs/engine-v6/family.md`. Include the old→new mapping (sections/levels → domains/levels), the never-share allow-list per projection, and the deviation ids (read coalescing, 404-vs-403, Go-only `/family/*`).
- touches: `docs/engine-v6/family.md`, `docs/go-migration/deviations.md`
- deps: none · test plan: n/a (doc review).

### FAM-01a — Schema: links, grants, owner mode state, activity (+ data migration)
- Scope:
  - `companions` gets `role`, `template`, `expires_at`, `revoked_by` values `expired|declined|system`.
  - `companion_grants` gets `mode_epoch` and v6 domain/level values.
  - New `family_owner_state(owner_id PK, mode, mode_epoch, updated_at)`.
  - `companion_audit_logs` gets `subject_type`, `subject_id`, `payload_ref`, `payload_sealed` (nullable), `undo_deadline`, `undone_at`, `undone_by`, `proposal_id`, `diff_json`.
  - Data migration: `symptoms` grants deleted (A3); meds `edit` → `nudge` (or `write` = propose, per A4); appointments `edit` → `write`; cycle/pregnancy `edit` → `view`; `type` → `role`; pregnancy grants stamped with the current epoch only if the owner is pregnant now, else inactive.
  - Composite indexes (§5.3).
- touches: `backend-go/db/migrations`, `backend/database/migrations`, `backend-go/db/queries/companion`, `backend-go/internal/*/store/models.go` (regenerated), `backend-go/contract/fixtures/dump.sql`
- deps: FAM-00 · test plan: `TestFamilyMigration_UpDown` + mapping assertions (seeded legacy rows → expected v6 rows); `make schema-diff`.

### FAM-01b — Schema: proposals, share requests, nudges, member alerts, member notices, consents
- Scope:
  - `family_proposals` (owner, link, actor, domain, kind ∈ create_med|create_appointment|update_appointment|share_request|child_delete|guardian_remove, sealed payload, status, expires_at, decided_by/at, applied_ref).
  - `family_nudges` (unique per dose).
  - `family_member_alerts` (link FK, source alert id, level, created_at, seen/ack).
  - `family_notices` member outbox (link FK, kind, due_at, sent_at, defer reason) — or `user_notifications.link_id` (decide in FAM-00).
  - `family_consents` (owner, link, text version, scope_json, granted/revoked) — unless the §4.2 generic ledger lands first.
- touches: same as FAM-01a.
- deps: FAM-01a (serial) · test plan: UpDown + FK cascade tests (revoke/delete purges member alerts and notices).

### FAM-01c — Schema: child guardians + actor columns + child FK transfer
- Scope:
  - `child_guardians(child_id, user_id, level write|limited, sub_scopes json, status, added_by, created_at)`, backfilled from `children.owner_id` (write) + active spouse `families/family_children` (write).
  - `created_by/updated_by` on children, child_measurements, child_vaccine_doses, child_milestone_checks, baby_feeds, baby_sleeps, baby_diapers (+ `deleted_at` soft delete for undo, A14).
  - `children.owner_id` becomes the "primary guardian". The FK stays, but deletion goes through a transfer in code (FAM-08).
- touches: same as FAM-01a.
- deps: FAM-01b (serial) · test plan: UpDown + backfill assertions.

### FAM-02 — Domain registry + `family.Authorizer` (check())
- Scope:
  - `internal/family`: Domain/Level/Role/Template registry (max levels, role defaults, carer time-box A13, male-owner domains A20, TTC switch A2, teen roles passthrough).
  - Pure `Decide` (§5.2); batched snapshot loader with lazy `SyncMode`; Redis read-audit coalescing; activity writer.
  - Rewire `companion.Service.Level/CanRead/CanWrite` and `Delegation.Authorize` to the Authorizer.
  - Join the owner's `blocked_at` (FAM-L2).
- touches: `backend-go/internal/family/**`, `backend-go/internal/companion/{service.go,delegation.go,teen_guard.go,audit.go}`, `backend-go/db/queries/companion`
- deps: FAM-01a · test plan: `TestCheck_Matrix` (unit, exhaustive); `TestCheck_ModeEpoch`; `TestCheck_Expired`; `TestCheck_BlockedOwner`; existing `TestGrantMatrix` rewritten for 4 levels × 9 domains; all existing companion/care/shop/teen/children integration tests stay green.

### FAM-03 — v6 member projections; remove never-shared leaks
- Scope:
  - Allow-list DTOs per domain: `cycle_phase` (A1, from the single engine function `PublicPhase`, no `days_late`), `pregnancy` (week + EDD, mode pregnancy only), `appointments` (no notes/topic/result_note/prep), `meds` (no notes), `vitals` (latest value + category), `fetal_activity` (today's kicks summary + last contraction session status).
  - Delete the `symptoms` section and its home card.
  - Gate the shop checklist share on `pregnancy` via check().
  - The member grant list shows only effective grants.
  - `Cache-Control: private, no-store` on family responses.
- touches: `backend-go/internal/companion/shared/**`, `backend-go/internal/companion/home/**`, `backend-go/internal/companion/handlers.go`, `backend-go/internal/http/routes_shop_checklist.go`, `backend-go/internal/family/projections/**`, `backend-go/internal/cycle/**` (one exported helper only, coordinate with the engine tasks)
- deps: FAM-02 (+ the engine v6 resolver task, for `PublicPhase`) · test plan: `TestMember_NeverSharedNotReachable` (JSON key scan over every member endpoint), `TestCyclePhase_NoNumbers`, `TestPregnancyProjection_ModeOnly`, `TestShopChecklist_StaleGrantDenied`.

### FAM-04 — Mode-change deactivation + owner review + delivery suggestion
- Scope:
  - `SyncMode` hooks in profile life-mode upsert, postpartum start, pregnancy start/end, loss (`internal/loss/service.go`), teen.
  - `GET /api/v1/family/review` (owner: inactive grants per member + suggested new domains for the new mode).
  - `POST /family/members/:id/grants/reapprove`.
  - At delivery: a "create child profile with co-parent" suggestion.
  - The member sees only children/household while pending.
- touches: `backend-go/internal/family/**`, `backend-go/internal/profile/**` (hook call only), `backend-go/internal/postpartum/service.go`, `backend-go/internal/pregnancy/**` (hook call only), `backend-go/internal/loss/service.go`, `backend-go/internal/http/routes_family.go`
- deps: FAM-02, FAM-03 · test plan: `TestModeChange_PregnancyDomainsDeactivatedAfterDelivery`, `TestLoss_ThenNewPregnancy_StillDenied`, `TestMissedHook_SelfHealsToDeny`.

### FAM-05 — Invites v6: roles/templates, preview, accept/decline, mutual request, phone auto-bind, consent
- Scope:
  - `POST /family/members` (role + template + edits + expires_at + consent version).
  - `POST /family/invites/preview {code}` (throttled + breaker, same generic refusal).
  - `POST /family/invites/accept {code, mutual?: {domains, levels}}` and `POST /family/invites/decline`.
  - Signup hook `OnSignup` → bind phone invites as pending acceptance (A11); 7-day phone TTL.
  - Owner confirmation for high-impact code-only invites (A11).
  - `share_request` create/approve/decline (owner-only apply, rate-limited 1 per 7 days per domain per pair).
  - Consent ledger rows on invite and on every grant change, with a diff.
  - Expired/declined cleanup sweep.
  - Fail-closed pepper outside local/testing/contract (FAM-L1).
- touches: `backend-go/internal/family/**`, `backend-go/internal/companion/**` (compat layer), `backend-go/internal/http/routes_family.go`, `backend-go/internal/http/routes_auth.go` (register hook), `backend-go/internal/http/routes_companion.go`, `backend-go/internal/consent/**` (catalog entry), `backend-go/internal/platform/config/config.go`
- deps: FAM-01b, FAM-02 · test plan: `TestInvite_AcceptWithoutMutual`, `TestFamily_MutualLinksIndependent`, `TestInvite_PreviewIsThrottledAndGeneric`, `TestSignup_BindsPhoneInviteAsPending`, `TestShareRequest_NoChangeUntilOwnerApproves`, `TestConsent_RecordedPerInviteAndChange`.

### FAM-06 — Proposals (owner meds/appointments) + 7-day expiry
- Scope:
  - Replace direct `for_user_id` writes in `internal/care`: members get `202 {proposal}`.
  - `GET /family/proposals` (owner), `POST /family/proposals/:id/approve|reject` (IDOR-safe, re-check in tx, apply as owner, `created_by_member` meta, notify the member).
  - Lazy and ticker expiry.
  - A medication proposal never creates an active reminder before approval.
  - Members never toggle `is_active/notify`.
- touches: `backend-go/internal/care/{delegation.go,handlers.go,appointment_handlers.go}`, `backend-go/internal/family/proposals/**`, `backend-go/internal/http/routes_family.go`, `backend-go/internal/http/routes_care.go`
- deps: FAM-01b, FAM-02 · test plan: `TestProposal_MedicationNotAppliedUntilApproved`, `TestProposal_ExpiresAfter7Days` (fake clock, lazy + sweep), `TestProposal_IDOR`, `TestProposal_RevokeDuringApprove` (TOCTOU).

### FAM-07 — Nudges
- Scope:
  - `GET /family/members/me/owners/:owner/nudgeable` (doses ≥ 60 min overdue without an intake; appointment window A7).
  - `POST /family/nudges {target}`: 1 per dose, 3/day/(member, owner), 23–08 Tehran refusal (A8).
  - Owner inbox notice with fixed neutral copy (no drug name); the owner's notification preferences are respected.
  - Activity row.
  - KPI field for dose-after-nudge.
- touches: `backend-go/internal/family/nudges/**`, `backend-go/internal/care/today.go` (export helper only), `backend-go/internal/http/routes_family.go`
- deps: FAM-01b, FAM-02 · test plan: `TestNudge_DailyCapAndPerDoseOnce`, `TestNudge_QuietHoursTehran`, `TestNudge_60MinuteRule`, `TestNudge_CopyHasNoDrugName`, `TestNudge_IDOR`.

### FAM-08 — Child co-guardianship, actor, undo, both-parent approvals, deletion transfer
- Scope:
  - `children.Service.Access/Editable` via `child_guardians` (write for guardians; carer sub-scopes feeding/diaper/sleep).
  - Actor on every write; last write wins; activity rows with sealed before/after (A14).
  - `POST /family/activity/:id/undo` within 24 h (owner or guardian; IDOR-safe).
  - Delete-child and remove-guardian go through two-guardian approval (proposal kinds); self-leave is unilateral (A15).
  - Owner/admin account deletion transfers the child to a remaining guardian (`profile/service.go`, `admin/users/users.go`).
  - Guardian re-check inside the transaction.
- touches: `backend-go/internal/children/**`, `backend-go/internal/babylog/**`, `backend-go/internal/family/**`, `backend-go/internal/profile/service.go`, `backend-go/internal/admin/users/users.go`, `backend-go/db/queries/children`, `backend-go/db/queries/babylog`
- deps: FAM-01c, FAM-02, FAM-06 (proposal engine) · test plan: `TestChild_DeleteNeedsBothGuardians`, `TestChild_RemoveGuardianNeedsBoth`, `TestChild_SelfLeaveUnilateral`, `TestChild_ActorStoredAndLWWLogged`, `TestUndo_Within24hOnly`, `TestOwnerDeletion_ChildKeptForCoGuardian`, `TestCarer_LimitedSubScopes`.

### FAM-09 — Urgent alert relay (member_alert)
- Scope:
  - On creation of an urgent alert row (`pregnancyalerts` insert path now; the §5.4.1 `alerts_inbox` writer when it lands), dispatch an asynq fan-out to members with the urgent switch, as `family_member_alerts` rows: neutral title (A10), time, `call_owner` (owner mobile, revealed only to switch holders) and 115.
  - Member list/ack endpoints; purge on revoke and deletion.
  - Optional neutral SMS behind a switch (A9).
- touches: `backend-go/internal/messages/pregnancyalerts/engine.go` (hook call only), `backend-go/internal/family/alerts/**`, `backend-go/internal/http/routes_family.go`, `backend-go/internal/messages/rules/rules.go` (placement constant)
- deps: FAM-01b, FAM-02 (+ the alerts_inbox task from the message-engine gap report) · test plan: `TestMemberAlert_OnlyWithSwitchAndNeutral`, `TestMemberAlert_PurgedOnRevoke`, `TestMemberAlert_Idempotent`.

### FAM-10 — Member home (Fam_PHome API), member_tip, member notifications
- Scope:
  - `GET /api/v1/family/home` for **any** member (replaces the male-only gate). Aggregated per owner from FAM-03 projections + check(); appointments with "I'm coming" (`POST /family/appointments/:id/attend`, own RSVP, ≥ view); support tips from the message-engine `member_tip` placement, evaluated on a **member-scoped fact set** (only granted fields).
  - Member notifications outbox: phase/week change, appointment tomorrow, missed dose (nudge level only), urgent. Neutral copy «پیامی از خانواده در ریتمی». Ticker dispatcher + `notifications.Decide` (member's `companion` category).
  - Keep `/companion/home` as a compat alias.
- touches: `backend-go/internal/family/home/**`, `backend-go/internal/companion/home/**`, `backend-go/internal/messages/**` (placement + fact set), `backend-go/internal/http/routes_family.go`, `backend-go/internal/http/routes_companion_home.go`
- deps: FAM-03, FAM-07, FAM-09 · test plan: `TestFamilyHome_FemaleMemberAllowed`, `TestMemberTip_NoHiddenFieldConditions`, `TestMemberNotices_NeutralAndQuietHours`, `TestFamilyHome_QueryBudget` (assert O(1) authorization queries with a counting DB wrapper).

### FAM-11 — Household to-do domain
- Scope: owner-scoped household list (`todo` gains `owner_id`/`household` scope). Members with `household` view/write read and write directly (A5). Actor + activity + 24 h undo.
- touches: `backend-go/internal/todo/**`, `backend-go/db/queries/todo`, `backend-go/db/migrations`, `backend/database/migrations`, `backend-go/internal/family/**`
- deps: FAM-01c (serial schema), FAM-02 · test plan: `TestHousehold_MemberWriteDirectAndUndoable`, `TestHousehold_OffDenies`, `TestTodo_PersonalListsStayPrivate`.

### FAM-12 — Admin family tab + KPIs
- Scope:
  - `GET /api/admin/v1/users/:id/family` (member count, roles, invite statuses, guardian counts; **no** domains or levels), moving the raw SQL to sqlc.
  - Metrics: invite acceptance rate, average members per circle, proposal approval rate, dose-logged-after-nudge rate (§5.6), with suppression for small cohorts as in `metrics.go`.
- touches: `backend-go/internal/admin/users/**`, `backend-go/internal/admin/companions/**`, `backend-go/internal/metrics/**`, `backend-go/db/queries/admin`
- deps: FAM-05, FAM-06, FAM-07 · test plan: `TestAdminFamilyTab_NoGrantContents`, `TestFamilyKPIs_Definitions`.

### FAM-13 — (optional, after A18) Owner's Plus for view-level members
- Scope: in member projections, resolve `plus.Status(owner)` and unlock Plus analyses of domains the member already views. No PDF of the owner's record.
- touches: `backend-go/internal/family/**`, `backend-go/internal/plus/**` (read-only use)
- deps: FAM-03 · test plan: `TestPlusForMember_OnlySharedDomains`.

### Frontend / admin-web (separate tasks, after the matching backend)
- **FE-FAM-1 People hub & add-member flow** (Me_People, Fam_Hub, Fam_AddMember, Fam_Access): roles, templates, domain × level matrix with disabled levels, urgent switch, duration, "never" list, remove member. deps FAM-05.
- **FE-FAM-2 Member acceptance**: code entry → summary → accept/decline + mutual request; pending-invite card after signup. deps FAM-05.
- **FE-FAM-3 Fam_Activity & approvals**: log with member filter, undo ≤ 24 h, approve/reject proposals and share requests, "review access after mode change" card. deps FAM-04, FAM-06, FAM-08.
- **FE-FAM-4 Member home set** (Fam_PHome, PMeds with nudge/propose, PCalendar with "I'm coming", PAlert, PPostpartum). Remove the gender gate on the client route. deps FAM-07, FAM-09, FAM-10.
- **FE-FAM-5 Child co-guardian UI** (actor names on entries, two-guardian approvals, carer limited logging). deps FAM-08.
- **ADMIN-FAM-1 admin-web family tab + KPI cards**. deps FAM-12.

Dependency sketch:
- FAM-00 → 01a → 01b → 01c (serial schema).
- 01a → 02 → {03, 05, 06, 07}.
- 03 → 04.
- {01c, 02, 06} → 08.
- {01b, 02} → 09.
- {03, 07, 09} → 10.
- 01c → 11.
- {05, 06, 07} → 12.
- 03 → 13.
