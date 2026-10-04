# Security review — Companion «همدم» (bloom N4, B-N4-08)

Scope: the committed N4 companion feature on branch `stage` (HEAD `c3a848a`).

- Backend (Go): `backend-go/internal/companion/**` (service, handlers, links, delegation, account, sms_gate, code,
  `shared/`, `home/`), `internal/sms`, care delegation (`internal/care/delegation.go`, `for_user_id` on
  meds/appointments), `internal/http/routes_companion*.go`, `routes_admin_companions.go`, `routes_care.go`,
  `internal/admin/companions/**`, `db/queries/companion/*`, `db/queries/admin/companions.sql`, migration
  `00026_companions.sql`, config (`COMPANION_CODE_PEPPER`, `COMPANION_SMS_PROVIDER`).
- Frontend: `frontend/src/entities/companion/**`, `features/invite-companion/**`, `screens/companion-*/**`,
  onboarding partner steps, reminder forms record-for (`features/manage-medication|manage-appointment`),
  `admin-web/src/screens/companions/**`.

Method: code reading of every route, query and view. Nothing was executed against stage.

## 1. Threat model

### Assets

| Asset | Sensitivity |
|---|---|
| Owner's cycle state (cycle day, phase, days late, next period) | Health data (special category) |
| Owner's pregnancy / postpartum / menopause status | Health data, highest sensitivity (pregnancy disclosure can cause real-world harm) |
| Day-log symptoms, mood, sleep | Health data |
| Medication and appointment reminders (titles, notes, doctors) | Health data |
| The relationship itself (who is whose partner/spouse) | Personal data (intimate relationship) |
| Invite codes (plain: only in the create/renew response; at rest: HMAC-SHA256 under the pepper) | Bearer secret: a code turns into an active link |
| Phone numbers in invites | Personal data |
| Audit trail (`companion_audit_logs`) | Integrity of the owner's oversight |
| Owner inbox (`user_notifications`) | Integrity / phishing surface |

### Actors

| Actor | Capabilities |
|---|---|
| Owner (usually a woman's account) | Invites, sets grants, revokes, reads her own audit trail |
| Companion (male account, or any account that accepted a code) | Reads granted sections, writes meds/appointments with `edit`, leaves |
| Former companion (revoked / left) | Holds a valid 365-day session token, no link |
| Arbitrary registered user | Can call `/companions/accept` and guess codes, call `for_user_id` with any id |
| Code interceptor (someone who saw a shared code: chat, shoulder-surfing, SMS forward) | Can redeem a code-only invite |
| Admin (any active admin) | Masked links overview, tips copy |
| Super admin | Everything an admin can do, plus more |
| Infrastructure attacker with a DB dump | Code hashes, relationships |

### Trust boundaries

1. Browser / PWA ↔ Go API (`/api/v1/companions*`, `/api/v1/companion/home`, `/api/v1/care/*` with `for_user_id`),
   behind the Passport RS256 guard (`auth.MustGuard(...).RequireUser`).
2. Owner's data domain ↔ companion's account: crossed only through `GetAccess` (active link + grant row, checked in
   SQL by `owner_id`, `companion_user_id`, `status='active'`), always followed by an audit insert before the data is
   built.
3. Admin web ↔ admin API (`/api/admin/v1/companions/*`): `httpadmin.Handle` + `kit.Admin`, session cookie, CSRF on writes.
4. Go API ↔ SMS gateway (Kavenegar verify/lookup; the API key in the URL path is never logged).
5. Go API ↔ Redis (rate limits, SMS send gate; the gate fails closed).

### Data flows

1. Invite: owner → `POST /companions` → `CreateInvite` (owner row lock, caps, HMAC code at rest) → response carries
   the plain code once → optional SMS (3 gates: per link 2 min, per recipient 3/day, per owner 5/day).
2. Accept: companion → `POST /companions/accept` (per-IP 30/h, per-user 10/h) → hash lookup → owner lock → checks
   (used/expired/locked/self/phone binding/already linked) → activate → audit `accepted` → owner inbox notice.
3. Read: companion → `GET /companions/links/:id/sections/:section` or `GET /companion/home` → `CompanionLink` (viewer is
   the active companion) → `GetAccess` → audit `read` → `shared.Reader` builds a minimised view.
4. Delegated write: companion → `POST|PUT /care/medications|appointments` with `for_user_id` → `Delegation.Authorize`
   (live grant, audit `write` first, fail closed) → the write, scoped by the owner's `user_id` → owner inbox notice.
5. Revoke: owner `DELETE /companions/:id` or companion `DELETE /companions/links/:id` → link `revoked`, grants and
   family deleted, open invites revoked, audit `revoked`. Access ends on the next request (no server cache).
6. Admin: `GET /companions/links` (masked names/mobiles, counts), tips CRUD on `message_contents` group `companion_tip`.

## 2. What holds (verified)

- **IDOR, owner routes**: `Show`, `Renew`, `UpdateGrants`, `UpdateChildren`, `Destroy` all go through
  `OwnerLink` / `SetGrants` / `SetSharedChildren` / `RenewInvite`, which compare `c.OwnerID` with the caller
  (`links.go:17-22`, `service.go:289-291,502-505,530-533`). A foreign or revoked id gives the same 404.
- **IDOR, companion routes**: `Leave` and `Section` use `CompanionLink` (active link where the caller is the
  companion, `links.go:41-49`); the grant comes from `GetAccess`, which filters owner, viewer and `status='active'` in
  the SQL itself (`companion.sql` GetAccess). Unknown sections are 404 before any read.
- **IDOR, delegation**: `care.subject` (`internal/care/delegation.go:47-80`) hands `for_user_id` to
  `Delegation.Authorize`; a missing grant or an unknown account is the same 403 `companion_forbidden`; the record is
  then loaded by `id AND user_id = owner AND type` (`db/queries/care/*.sql`). Delete, intake ticks, cancel and prep stay
  owner-only.
- **Privilege escalation**: grants are only written by the owner (`SetGrants`/`CreateInvite`); the companion cannot
  change grants, children or invites; `revoked_by` and `status` are not request fields. Admin tips cannot touch roles.
- **Previous M-1 (SMS abuse) is fixed**: `sms_gate.go` applies per-link 2-minute resend, 3 SMS/day per recipient
  (hashed key, across all owners), 5/day per owner, and fails closed on a Redis error; invite creation/renewal is also
  capped at 20/h per user (`routes_companion.go:28`). The gateway error path strips the URL (API key, number, code)
  (`internal/auth/sms/providers.go:168-176`).
- **Previous M-2 (audit before write) is fixed**: `Delegation.Authorize` inserts the audit row before returning
  `true` and returns an error if the insert fails (`delegation.go:36-57`); `Section` audits before `Reader.Read`
  (`handlers.go:496-499`); the companion home audits each section before building it (`home/home.go:161-167`).
- **Revocation latency**: zero on the server. Each request re-runs `GetAccess`/`ListForCompanion`; there is no
  server-side cache of grants or views. Client caches are in memory only (React Query, 30-60 s staleTime) and
  `public/sw.js` never caches `/api/*`. A 403 on a delegated write drops the picker back to «خودم».
- **Code handling**: 32-symbol alphabet, uniform from crypto/rand, HMAC at rest, unique hash; one generic 422 for
  every refusal; codes never logged (handlers log a masked mobile only). Frontend keeps the code in component state
  and the request body only; no localStorage/sessionStorage, no URL, no console, no query persister.
- **Config**: `COMPANION_SMS_PROVIDER=fake` is refused in production; production without `COMPANION_CODE_PEPPER`
  answers 503 (fail closed); a production pepper under 32 bytes fails config load.
- **XSS**: `display_name`, `users.name`, tip copy and notification titles are rendered as React text nodes (frontend
  and admin-web); no `dangerouslySetInnerHTML` in any companion screen. Article excerpts on the companion home go
  through `sanitizer.PlainText`.
- **Phone enumeration**: invite creation returns the same shape whether or not the number has an account; accept
  refusals are identical; masked numbers only (`0912****789`) on the user side.
- **Data minimisation**: the cycle view drops logs and fertility tests; the symptoms view drops sex, discharge,
  urogenital, measurements, bleeding detail and notes; audit rows carry no payload; owner notices name the kind of
  record only; the admin overview has no codes, hashes or grant contents.
- **Male/female confusion**: the women's home and messages answer a male account 409; `GET /companion/home` answers
  a non-male account 403; all section reads are gated by grants, not by gender, so a woman who accepts a code sees only
  what was granted. Tip phase on the companion home is derived only from views the companion may read (`tipPhase`).

## 3. Findings

Severity: Critical / High / Medium / Low / Info. "Release blocker" = must be fixed before N4 ships beyond stage.

| ID | Severity | File:line | Description | Fix | Status |
|---|---|---|---|---|---|
| CMP-H1 | **High** (release blocker) | `backend-go/internal/companion/shared/shared.go:84-110`, `:112-131`; `backend-go/internal/companion/home/home.go:196-213` | **Pregnancy / postpartum status leaks to a companion who was not given the pregnancy section.** The cycle view runs the cycle engine without looking at the owner's life mode. A pregnant owner stops logging periods, so `cycle_day` keeps growing and `days_late` climbs (`resolver.go:125`, `daysLate = max(0, -rawDaysToPeriod)`), and `main_phase` stays at period-expected. A partner with only `cycle: view` sees «۴۵ روز تاخیر» and can infer the pregnancy that the owner chose to withhold (`pregnancy: none`). The symptoms view also passes on mode-only options: `symptoms.general.leg_cramps` and `pain.location.leg` (pregnancy only), `pain.location.stitches` and `mood.weekly_checkin` (postpartum only), and the menopause-only items. Exploit: the owner shares cycle + symptoms with a boyfriend and keeps pregnancy private. He opens the companion home, sees days_late = 38 and «گرفتگی ساق پا». | In `shared.Reader.Read`, resolve the owner's life mode (`enums.ResolveLifeMode` with the pregnancy profile, as `home/handlers.go` does). When the mode is pregnancy or postpartum and the link has no pregnancy grant, return the neutral cycle view (`has_data:false`, every number null, `main_phase` null). Do the same for menopause, where the cycle view means nothing. In the symptoms view keep only (category, param, option) entries that `Category.Available(p, ModeCycle)` and the cycle option list allow, so mode-only items never leave. `Read` needs the link's grants (or a `hidePregnancy bool`). Separately, clip `days_late` to null past `maxLateDays` for every companion. Add an integration test: pregnant owner, cycle+symptoms grant, no pregnancy grant → no days_late, no pregnancy-only items. | **fixed** (B-N4-08b) |
| CMP-M1 | Medium (release blocker: child safeguarding) | `backend-go/internal/companion/service.go:199-271`; `shared/shared.go:84-110` | **A teen-mode owner (a minor) can link an adult "partner"/"spouse" and share her cycle, including the fertile phase.** `CreateInvite`/`Accept` never check the owner's life mode. `EngineProfile` sets `NoFertilityCopy` for teen, but the companion cycle view still returns `main_phase: fertile` and the home picks the fertile tip. The canvas plans a `parent` type for teens (CB-TEEN-01); partner/spouse links for minors are not intended. | Refuse `partner`/`spouse` invites and their acceptance while the owner's resolved mode is `teen` (new `ErrModeNotAllowed` → 422 `companion_mode_not_allowed`). In the cycle view, map `fertile` to its neighbouring phase (or null) whenever `profile.NoFertilityCopy` is true. | **fixed** (CB-TEEN-01 + B-N4-08b) |
| CMP-M2 | Medium | `backend-go/internal/care/handlers.go:80-93`; `backend-go/internal/care/appointment_handlers.go:76-89` | **Delegated single-record reads get around the minimised section views.** The meds section shows only *active* medications and the appointments section only *upcoming* ones (`shared.go` package doc, `medsView`/`appointmentsView`). `GET /care/medications/:id?for_user_id=<owner>` and `GET /care/appointments/:id?for_user_id=<owner>` with a view grant return any of the owner's records, including paused or ended medications and past or cancelled appointments. Reminder ids are global and sequential, and the GET is not throttled, so a companion can walk ids and pull the owner's medication and visit history (each hit is audited, but the owner has to notice). | In `ShowMedication`/`ShowAppointment`, when `subjectID != actorID`: answer 404 unless `m.Row.IsActive` (meds) or `a.Upcoming(now)` (appointments). Apply the same rule to delegated `Update*` so an edit grant cannot reopen history. Add a per-user throttle on delegated reads (for example 120/h). | **fixed** (B-N4-08b) |
| CMP-M3 | Medium (re-rated from L-1) | `backend-go/internal/http/routes_companion.go:19-29,62`; `backend-go/internal/companion/service.go:358-366` | **Untargeted brute force of code-only invites.** Throttles are per user (10/h) and per IP (30/h) only. There is no global failure budget, and a wrong guess at a code-only invite touches no row, so nothing locks. With *N* open code-only invites the chance per guess is N/32^6. At 2,000 open invites, 100 cheap accounts on 35 IPs (24k guesses/day) get about one active link every 22 days, to a random owner's granted health data. Mitigations already present: the owner gets an inbox notice on accept and can revoke, and every read is audited. | Pick one, in order of strength: (a) owner confirmation: accept moves the link to `pending_confirm` and no grant takes effect until the owner taps «تأیید» in-app (also closes the shared-in-chat interception case); (b) a global accept-failure circuit breaker (Redis counter of refusals per minute → 429 for everyone + an alert above a threshold); (c) 8-character codes (32^8 ≈ 1.1e12; the UI has 6 boxes); (d) make phone binding the default in the invite flow. | **fixed** (B-N4-08b, option b) |
| CMP-L1 | Low | `backend-go/internal/companion/home/home.go:156-168`; `handlers.go:496`; `service.go:663-669`; `handlers.go:344` | **Audit trail flooding and unthrottled reads.** Every companion-home load writes one `read` row per granted section (up to 5), with no coalescing and no throttle. The owner's audit endpoint returns only the newest 50 (max 200) rows with no pagination or filter. A companion who makes a delegated write and then reloads the home about 40 times pushes the `write` row out of the owner's visible trail. The inbox notice still exists. Normal use also grows the table without limit. | Coalesce read audits: skip the insert when the same (actor, owner, section, `read`) was written within the last 15 minutes (a Redis key or a `SELECT … LIMIT 1`). Give `GET /companions/audit` cursor pagination and an `action` filter so `write`/lifecycle rows stay reachable. Throttle `GET /companion/home` and `GET /companions/links/:id/sections/*` (for example 300/h/user). | **fixed** (B-N4-08b) |
| CMP-L2 | Low (was L-2) | `backend-go/internal/companion/service.go:363-370` | Timing oracle on accept: an unknown code returns after one indexed SELECT, while an existing code opens a transaction with a row lock. The response body is identical. Value to an attacker is small (it only shows that a bound, used or expired code exists). | Run a dummy `LockOwner`/transaction path for unknown codes, or add a fixed minimum response time to `Accept`. | open |
| CMP-L3 | Low (was L-3) | `backend-go/internal/companion/service.go:31`; `backend-go/internal/platform/config/config.go:232`; `.env.stage.example:88`; `docker-compose.stage.yml:120` | Stage runs without a pepper, so codes are keyed with the public default `ritme-companion-invite`. A stage DB dump reverses every open code in seconds (32^6 HMACs). Stage holds real tester data. | Treat "pepper missing" as fatal (or 503) for every `APP_ENV` except `local`/`testing`/`contract`; set a random ≥32-byte `COMPANION_CODE_PEPPER` on stage. | open |
| CMP-L4 | Low (was L-6) | `backend-go/internal/companion/delegation.go:36-57`; `backend-go/internal/care/handlers.go:96-135,140-180` | TOCTOU: `Authorize` and the INSERT/UPDATE are separate statements with no lock, so a delegated write that started just before a revoke can still commit just after it. | Re-check the grant inside the write's transaction (`SELECT … FROM companions … FOR SHARE` joined to the grant), or accept and document the window (milliseconds). | open |
| CMP-L5 | Low | `backend-go/internal/companion/links.go:111-150` | Owner-inbox content spoofing: when the owner gave no label, the notice title embeds the companion's own `users.name` (up to 255 chars, `profile/handlers.go:129`). A malicious companion can set a name like «پشتیبانی ریتمی: حساب شما مسدود شد، با ۰۹۱۲… تماس بگیرید» and it appears as an app notification. Text only, no XSS. | Clip the name to about 40 runes in `NotifyOwner`; drop digit runs and URLs, or fall back to the neutral word when the name contains them; prefer the owner's label. | **fixed** (B-N4-08b) |
| CMP-L6 | Low | `backend-go/internal/admin/companions/companions.go:47-53`; `links.go:38-47` | The admin overview is open to every admin (`kit.Admin`). It shows raw owner and companion user ids (the relationship graph; ids pivot to `/admin/users/:id`, also `kit.Admin`) and masks mobiles to 4+4 digits (8 of 11 visible, 1,000 candidates) for the *invited* phone, which the owner typed and which may belong to someone who never signed up. | Restrict `/companions/links` to `kit.Super`, or drop the user ids. Mask mobiles as the user side does (`0912****789`) or show only the last 2 digits. | **fixed** (B-N4-08b) |
| CMP-L7 | Low | `backend-go/internal/companion/shared/shared.go:118-129`; `internal/healthlog/taxonomy/view.go:45-49` | The symptoms view passes through unknown or legacy params of the allowed categories (with `guessType`, which can be text) and the owner's custom item labels (free text the owner typed for herself, e.g. a private condition name). | Whitelist known params with non-text types; send custom items as a count, or drop them unless the owner opts in. | open |
| CMP-L8 | Low | `backend-go/internal/companion/service.go:438-442` | Anyone who holds a phone-bound code can lock it with 5 wrong-account attempts (invite DoS). The owner has to renew. | Accept as is, or count only one mismatch per account. | open |
| CMP-I1 | Info | `backend-go/db/queries/companion/companion.sql` (GetAccess, ListCompanionLinks) | An owner blocked by an admin (`users.blocked_at`) keeps being readable by her companions: links are not suspended. | Join `users o ON o.id = c.owner_id AND o.blocked_at IS NULL` in `GetAccess` / `ListCompanionLinks`, or revoke links on block. | open |
| CMP-I2 | Info | `backend-go/internal/companion/sms_gate.go:15-18` | Any owner can use up a number's 3 SMS/day (the cap is global per recipient), so a real invite to that number falls back to sharing the code by hand. This trade-off is intended (it is what stops SMS bombing). | None; keep it. | open |

Previous notes from the B-N4-02 review: L-4 (prep checklist owner-only) is confirmed as designed
(`care/delegation.go:20-23`). L-5 (notify the owner on accept) is implemented (`handlers.go:411-415`).

### Release gate

- **Must fix before the N4 release:** CMP-H1 (pregnancy inference) and CMP-M1 (teen partner links). Both fixed.
- **Strongly recommended before GA (can follow in N4.x):** CMP-M2, CMP-M3, CMP-L1 (fixed), CMP-L3 (open).
- Others: backlog.

### Resolution (B-N4-08b)

- **CMP-H1** — `shared.Reader.ReadFor` takes the link (`companion.SectionReader`), resolves the owner's life mode
  (`healthlog.Service.LifeMode` = `enums.ResolveLifeMode`, an active pregnancy wins) and builds the views through
  `shared/mode.go`: pregnancy / postpartum without a `pregnancy` grant, and menopause always, give the neutral cycle
  view (`has_data:false`, every number and `main_phase` null); `days_late` and `cycle_day` are null past
  `MaxLateDays` (14, the engine's cap) for every companion; the symptoms view keeps only (category, param, option)
  entries offered in the cycle mode (plus the owner's pregnancy / postpartum options with the grant); unknown params
  and options are dropped (custom items of hosting params still pass, see CMP-L7). `Read(ownerID)` without a link is
  the strictest view. Tests: `shared/mode_int_test.go`, `shared/mode_test.go`,
  `http/companion_security_int_test.go` (`PregnantOwnerCycleNeutral`).
- **CMP-M1** — new partner / spouse invites and accepts of a teen owner were already refused by CB-TEEN-01
  (`teen_parent_only`, `invite_not_allowed`). B-N4-08b: the companion views map `fertile` to `follicular` when the
  owner's profile has `NoFertilityCopy`; an ACTIVE partner / spouse link of an owner whose stored mode is teen grants
  nothing while she is a teen (QUESTIONS #98 default: no revoke) — `Service.access` answers none (section reads 403,
  «ثبت برای …» 403), `ListForCompanion` / `CompanionLink` clear the effective grants (the home shows no card data);
  the owner's list keeps the stored grants, which apply again when she leaves teen mode; `parent` links unaffected
  (`companion/teen_guard.go`). Test: `TeenOwnerSuspendsPartnerLinks`.
- **CMP-M2** — `care/delegation_view.go`: with `for_user_id` ≠ caller, `ShowMedication` / `UpdateMedication` answer
  404 unless the medication is active, `ShowAppointment` / `UpdateAppointment` unless it is upcoming and not private.
  Delegated GETs are throttled 120/hour per account (`http/companion_guards.go`, `delegatedReadThrottle`). Test:
  `DelegatedRecordsStayInView`.
- **CMP-M3** — option (b), a global failed-accept circuit breaker on Redis (`companion/breaker.go`,
  `http/companion_guards.go`): every refused accept of any caller counts; 30 refusals within a minute or 200 within an
  hour pause `POST /companions/accept` for everyone for 15 minutes (the generic 429 `too_many_requests`, even for a
  valid code) and log an error-level alert («companion accept circuit breaker tripped»). The CMP-M3 scenario (~1,000
  guesses/hour) trips it within minutes, so a guesser gets at most a few hundred guesses per 15 minutes instead of
  24k/day. Accepted trade-off: ~30 wrong codes can pause accepts for 15 minutes (the owner's code stays valid for
  24 h). A Redis error while checking fails closed (500). Owner confirmation (a) and 8-character codes (c) remain
  options for later. Test: `AcceptCircuitBreaker`.
- **CMP-L1** — `Service.Audit` skips a `read` row when the same actor, owner, link and section has one within
  `ReadAuditWindow` (15 min; query `RecentReadAudit`); writes and lifecycle rows are always written.
  `GET /companions/audit` gained `before_id` (cursor) and `action` filters. `GET /companion/home` and section reads are
  throttled 300/hour per account. Test: `AuditCoalescedAndPaged`.
- **CMP-L5** — `NotifyOwner` names the companion through `noticeName`: control / bidi / zero-width characters removed
  (ZWNJ kept), whitespace collapsed, clipped to 40 runes; an account name with a run of 3+ digits, a URL, domain,
  e-mail or handle falls back to the neutral word. Tests: `notice_name_test.go`, `NoticeNameSanitized`.
- **CMP-L6** — `/api/admin/v1/companions/links` is `kit.Super`; user ids are no longer sent; mobiles keep only the
  last 2 digits (`•••••••••67`). admin-web's link list follows (no id fallback). Test: `admin/companions/api_int_test.go`.

## 4. Residual risks and assumptions

- **Consent model:** grants are what the owner explicitly picked. Data in a granted section (medication names,
  appointment notes, mood) reaching the companion is intended. The review only flags data beyond the intended
  view or inferable outside the granted sections.
- **Device trust:** a companion who had access can keep screenshots and data cached in memory until reload. Revocation
  cannot undo past disclosure; the audit trail is the only record.
- **Sessions:** tokens last 365 days by design; revocation is enforced per request on the link, not on the token.
- **Children / family:** `family_children` has no FK and `OwnsChildren` refuses every non-empty list until B-N5-02. That
  task must add the real ownership check, and the shared-child card must go through the same grant/audit path.
- **Production routing:** prod still runs Laravel and `deploy/go-routes.inc` does not route `/api/v1/companions*`,
  `/api/v1/companion/home` or `/api/v1/care*` to Go. The feature is live only on stage (whole-vhost Basic auth) until
  cutover; production needs `COMPANION_CODE_PEPPER` set (else 503) and `COMPANION_SMS_PROVIDER=gateway|none`.
- **Gender as account kind:** `user_life_profiles.gender = male` decides companion vs owner. Changing gender never
  widens data access, since every read is grant-gated, but it changes which home the user gets.
- **Rate limits need Redis:** without `d.Cache` (tests) every companion throttle and the SMS gate are no-ops; a prod
  deploy without Redis would remove the M-1 protections (assumed: deployed environments always run Redis; not verified in this review).
- **SMS provider trust:** Kavenegar sees the recipient number and the code (inherent to SMS delivery).
- Not covered: the Android app (out of scope), Laravel `backend/` (no companion code), load and DoS testing, and dynamic
  testing against stage.
