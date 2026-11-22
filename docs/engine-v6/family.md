# Engine v6 — Family reference (M11)

Every M11 task reads this file. It turns SPEC §1.7–§1.8, the gap analysis `gap/05-family-access.md` and the F decisions
(`DECISIONS.md` section F, accepted by the user on 2026-10-21 — every recommended default) into one contract.
Where this file and older code disagree, this file wins; where this file and SPEC disagree, the F decision cited wins.

Model in one line: a family link is **directional** (owner → member). Mutual sharing = two independent links; revoking one
never touches the other. A member never sees why access changed.

---

## 1. Levels

| Level | Code | Member can |
|---|---|---|
| خاموش | `off` | nothing |
| مشاهده | `view` | read the domain's allow-listed summary |
| یادآوری | `nudge` | view + send a gentle nudge (quotas §6) |
| ثبت | `write` | view + nudge + add. On the **owner's own records** a write is a **proposal** the owner approves (F-04, F-05). On child and household data it is a direct write with a 24 h undo (F-05, F-14). |

Order: `off < view < nudge < write`. A grant above the domain's max level is rejected (422), never clamped silently.

## 2. Domain registry

`M` = mode scope (`any`, `cycle`, `ttc`, `pregnancy`, `postpartum`). A mode-scoped domain is visible only while the owner's
effective mode is in scope **and** the grant's `mode_epoch` equals the owner's current epoch (§5).

| Code | Domain (fa) | Max | Partner | Carer | Friend | Other parent | M | Male owner (F-20) |
|---|---|---|---|---|---|---|---|---|
| `cycle_phase` | فاز کلی چرخه | view | view | off | view | off | cycle, ttc | — |
| `pregnancy` | هفته و تاریخ تقریبی زایمان | view | view | view | off | off | pregnancy | — |
| `appointments` | ویزیت‌ها و نوبت‌های صاحب | write (proposal) | write | view | off | off | any | ✓ |
| `fetal_activity` | حرکات جنین و انقباض (خلاصه) | view | view | off | off | off | pregnancy | — |
| `vitals` | وزن، فشار، قند | view | off | off | off | off | any | — |
| `meds` | دارو و مکمل صاحب | write (proposal) | nudge (F-04) | off | off | off | any | ✓ |
| `urgent_alert` | هشدار فوری | view (receive) | on (`view`) | off | off | off | any | — |
| `children` | پروفایل کودک | write | write (co-guardian) | write, sub-scopes `feeding`, `diaper`, `sleep` only | off | write (shared children only) | any | through guardianship |
| `household` | لیست کارهای خانه | write (direct) | write | off | off | off | any | ✓ |

Switch **`ttc_fertility`** (F-02): partner role only, default off, visible only in mode `ttc`. When on, the partner sees the
fertile window, estimated ovulation and best days. Raw LH, BBT and intercourse logs are never shown. Under hormonal
contraception (`limits.fertility_suppressed`) nothing fertile is exposed, switch or not.

Teen parent card (`teen_period_week`, `teen_kit`, `teen_notes`) stays the legacy `parent` role, outside this matrix (F-17).

### Roles
`partner` · `other_parent` (shared children only) · `carer` (time-boxed, F-13) · `friend` (cycle phase only) · legacy
`parent` (teen). The role only seeds defaults; every row is editable up to the max level.

### Templates (SPEC §1.7)

| Template | Grants |
|---|---|
| `cycle_buddy` — همدم چرخه | `cycle_phase: view` |
| `meds_appointments` — نوبت و دارو | `appointments: write`, `meds: nudge`, `cycle_phase: off` |
| `full_family` — مراقبت کامل خانواده | `pregnancy: view`, `appointments: write`, `meds: nudge`, `urgent_alert: view`, `children: write` (+ `fetal_activity: view` for partner) |
| `custom` — سفارشی | any combination within max levels |

### Carer time-box (F-13)
`expires_at` default: owner pregnant → EDD + 42 days; postpartum → delivery + 42 days; otherwise now + 30 days. The owner can
extend. At expiry the hourly sweep marks the link expired, writes an activity row and an owner inbox notice.

## 3. Legacy → v6 mapping (data migration, T-M11-02)

| Legacy | v6 |
|---|---|
| `companions.type = partner` | role `partner` |
| `companions.type = spouse` | role `partner` + co-guardian of every `family_children` child (→ `child_guardians`) |
| `companions.type = parent` | role `parent` (teen, unchanged) |
| section `cycle` | `cycle_phase` (level `edit` → `view`; max is view) |
| section `pregnancy` | `pregnancy` (`edit` → `view`); grant epoch = the owner's current epoch only if she is pregnant now, else inactive |
| section `appointments` | `appointments` (`view` → view, `edit` → write) |
| section `meds` | `meds` (`view` → view, `edit` → write) |
| section `symptoms` | **removed** (F-03): grant set to off; the owner gets a one-time «این اشتراک تمام شد» card |
| level `none` / `view` / `edit` | `off` / `view` / `write` (except where the domain max lowers it, rows above) |

## 4. What a member may receive (allow-lists)

Every member projection is a dedicated DTO built from the allow-list below; nothing is copied from the owner DTO.
`never` lists are negative tests: each must be asserted absent from every member payload.

| Domain | Allowed fields | Never |
|---|---|---|
| `cycle_phase` | `main_phase` (from the v6 engine `PublicPhase`; `fertile` allowed in cycle mode only when not suppressed); partner only: `period_soon` (bool, next start ≤ 3 days) (F-01) | dates, cycle day, cycle length, days late / lateness, confidence, predictions, history, mood, symptoms |
| `period_forecast` (owner consent, companion calendar) | partner role only, with a readable `cycle_phase` grant **and** the owner's per-link consent `companions.share_period_days` (default off): the days in the requested window her period is likely to fall on, **today onwards only** (`GET /companion/calendar`, clamped to today … +92 days); `[]` while she is late | logged / past period days, cycle day, cycle length, lateness, confidence, fertile window, history |
| `ttc_fertility` (switch) | `fertile_start`, `fertile_end`, `ovulation_date`, `best_days` | LH/BBT/intercourse values, test results |
| `pregnancy` | `current_week`, `trimester`, `edd` (approximate, date only) | LMP, dating source, scans, notes, alerts content |
| `fetal_activity` | per day: kick sessions count + "completed/short" flag; contraction session summary (count, last time) | free text, alert evidence |
| `vitals` | latest weight, BP and glucose value + date, with the owner's chosen unit | targets, doctor notes, alert reasons |
| `meds` | name, schedule times, today's dose status (taken/missed/pending) | `notes`, dose text beyond the label, history beyond 7 days, reminder settings |
| `appointments` | title, date/time, place name, «من هم می‌آیم» state | `notes`, `topic`, `result_note`, `prep`, private loss follow-ups, booking ids |
| `urgent_alert` | fixed neutral title per level, time, owner's call action, 115 (F-10, M-11) | rule title, body, evidence, measured values |
| `children` | per child: vaccines, growth, feeding, diaper, sleep, child dose log (F-26), writer name on each entry; carer: feeding/diaper/sleep only | other children not shared with this guardian |
| `household` | items: title, assignee, due, done | — |

Never shared, any domain, any role (SPEC §1.7): private notes · mood and EPDS · sex and desire · lab results · assistant and
doctor chats · full cycle history · fertility data (except the TTC switch above). The legacy `symptoms` section is gone (F-03).

## 5. `check()` (the `family.Authorizer`)

Pure function `Decide(snapshot, subject, domain, action, now) → {Allow | Propose | Deny, reason}`, evaluated in this order
(gap 05 §5.2, with F-23):

1. Actor is the owner → **allow**.
2. Subject is a child and the actor is a guardian → **allow** (carer: listed sub-scopes only, else deny `not_guardian`).
3. Link missing, not active, or owner blocked/deleted → **deny** `no_link` (HTTP **404**, F-21).
4. `now ≥ expires_at` → **deny** `expired` (404).
5. Domain not grantable for the role, or never-shared → **deny** `never_shared` (403).
6. Mode-scoped domain and (owner mode out of scope **or** `grant.mode_epoch ≠ owner.mode_epoch`) → **deny** `mode_inactive` (403).
   **F-23:** on an owner mode change *every owner-data domain* (`cycle_phase`, `pregnancy`, `fetal_activity`, `vitals`,
   `meds`, `appointments`) pauses until the owner reviews; `children` and `household` stay live; `urgent_alert` stays on.
7. Level below what the action needs → **deny** `level` (403).
8. `nudge` → quota check (DB, §6) → allow or deny `quota` (429).
9. `write` on the owner's own record → **propose** (HTTP 202, F-05).
10. `write` on household or child data → **allow**, logged with a 24 h undo.

Responses for denied member reads carry **no data** (tests assert empty bodies).

### Mode epoch
`family_owner_state(owner_id, mode, mode_epoch)`. `SyncMode(owner)` resolves the effective mode and bumps the epoch on a change;
it runs after every mode write and lazily inside snapshot loading, so a missed hook can only deny, never leak. A new pregnancy
is a new epoch (closes the stale-grant bug fixed for v1 in T-M9-02). The owner review card offers one tap «مثل قبل نگه دار»,
which re-stamps the paused grants with the new epoch.

## 6. Nudges (F-06 … F-08)

- Allowed only when a dose is unlogged 60 minutes after its time, or (F-07) an appointment is still scheduled between the
  owner's reminder time and the appointment time.
- Caps: 1 per dose/appointment overall (`UNIQUE(owner_id, target_type, target_id, due_date, slot)`), 3 per owner-local
  day per (member, owner) pair.
- Quiet hours: 23:00–08:00 owner-local is a hard floor, plus the owner's own quiet hours at delivery. Refused with
  `quiet_hours`, never queued.
- Text fixed and kind; no medicine name on the lock screen.

## 7. Activity, sealing, retention (F-14, F-16, F-19)

| Action | Logged | Payload |
|---|---|---|
| `read` | coalesced: one row per (owner, member, domain) per 15 minutes (Redis `SET NX EX 900`, DB fallback) — D-203 | none |
| `grant_change`, `link_create`, `link_revoke`, `link_expire`, `invite_*`, `mode_pause`, `mode_review` | always | per-domain diff (levels only) |
| `proposal_create`, `proposal_approve`, `proposal_reject`, `proposal_expire` | always | sealed proposed record |
| `nudge_send`, `nudge_refused` | always | target ref only |
| `child_write`, `household_write` (create/edit/delete), `undo` | always | sealed before/after JSON |
| `alert_relay`, `alert_ack` | always | source alert id only |

- Sealing: AES-256-GCM with `PRIVATE_NOTE_KEY`, AD `family:activity:<owner_id>`; 503 without a key outside local/testing.
- Retention: rows 12 months; sealed payloads purged after 30 days (daily sweep); undo window 24 h.
- Proposals expire after 7 days unanswered (hourly sweep + lazy expiry on read/approve). A medication proposal never
  creates an active reminder before the owner approves.
- Invites (F-11, F-12): code 6 chars, single use, 24 h; phone invites 7 days and bind as a *pending acceptance*.
  Code-only invites that include child write, urgent alert or vitals need owner confirmation after acceptance. Expired or
  declined invites → `revoked` with `revoked_by = expired|declined`, hidden from lists, kept for audit and KPIs.
- Removing the other guardian of a child needs both guardians; leaving yourself is unilateral; pending requests expire after
  7 days (F-15). Owner account deletion revokes every link at once; child data stays with the other guardian.

## 8. Notices and copy keys

All member-facing lock-screen text is neutral (SPEC §1.8). Namespaces: backend `internal/family/lang/{fa,en}/family.json`,
frontend `frontend/messages/{fa,en}/family.json`.

| Key | fa | Used for |
|---|---|---|
| `family.notice.neutral` | پیامی از خانواده در ریتمی | every member push / lock screen |
| `family.notice.sms_urgent` | پیامی از خانواده در ریتمی — اپ را باز کن | optional urgent SMS (F-09, per member, owner-controlled) |
| `family.alert.title.urgent` | یک موضوع فوری | member alert title, level urgent (F-10) |
| `family.alert.title.follow_up` | یک موضوع برای پیگیری زودتر | member alert title, level follow_up |
| `family.alert.call_owner` | بهش زنگ بزن | member alert action |
| `family.alert.call_115` | تماس با ۱۱۵ | member alert action |
| `family.nudge.meds` | یادت نره داروی امروزت رو بخوری | owner receives a meds nudge (no medicine name) |
| `family.nudge.appointment` | نوبت امروزت یادت هست؟ | owner receives an appointment nudge |
| `family.owner.mode_review` | دسترسی‌های جدید رو تنظیم کن | owner card after a mode change |
| `family.owner.mode_review.keep` | مثل قبل نگه دار | one-tap keep |
| `family.owner.symptoms_ended` | اشتراک «علائم» تمام شد | one-time card (F-03) |
| `family.owner.proposal_new` | {name} یک مورد پیشنهاد داده | proposal waiting |
| `family.member.proposal_approved` | پیشنهادت تأیید شد | proposal outcome |
| `family.member.proposal_rejected` | پیشنهادت ثبت نشد | proposal outcome (no reason shown) |
| `family.owner.carer_expired` | دسترسی {name} تمام شد | carer time-box end |

Final Persian wording goes through the `persian-writing` skill in the frontend tasks (written-warm «تو» register); keys are
fixed here, text may be polished there.

## 9. Plus for members (F-18)

When the owner has Plus, a view-level member gets the Plus analyses and a PDF summary of **only the domains they already see**;
never the owner's full record PDF and never never-shared data. Optional task T-M11-16 (D-206).

## 10. Screens without artboards (F-22)

Fam_Activity, Fam_PMeds, Fam_PCalendar, Fam_PAlert, Fam_PPostpartum, the review card and the approvals list are built now in the
language of the Hamdam artboards (`docs/design/night-bloom/companion-family/`) and the design system; the main session reviews
light and dark screenshots. «شیفت شب» on Fam_PPostpartum is a household to-do item assigned to one person for tonight (F-24).
A mutual pair sees one household list per owner, labelled with the owner's name; lists are never merged (F-25).
