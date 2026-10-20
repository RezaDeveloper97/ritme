---
id: T-M11-01
title: Family decisions, family design doc and deviation rows (user OK)
milestone: M11
type: investigate
status: todo
depends_on: [T-M9-01]
parallel_group: M11-A
touches: [docs/engine-v6/DECISIONS.md, docs/engine-v6/family.md, docs/go-migration/deviations.md]
skills: []
verify: test -f docs/engine-v6/family.md && grep -q '^| D-206 ' docs/go-migration/deviations.md && ! grep -q 'All F rows: .*Status\*\* proposed' docs/engine-v6/DECISIONS.md
---

# T-M11-01 — Family decisions, family design doc and deviation rows (user OK)
**Model:** main session (opus). Product and privacy decisions; do not delegate.

## Why
SPEC v6 §1.7–§1.8 turn the companion feature into a family circle: directional links, nine data domains with four
levels, proposals, nudges, child co-guardians, mode-bound access, urgent-alert relay and a family activity log. The
current code covers roughly a quarter of it and leaks never-shared data (T-M9-02 closed the worst leaks). Every open
question has a recommended default in `docs/engine-v6/DECISIONS.md` section F (F-01 … F-26). Evidence:
`docs/engine-v6/gap/05-family-access.md` (summary table §1, findings §3, ambiguities §4, design §5).

## Scope
1. Re-read section F and gap 05. Spot-check that the cited code is unchanged (at least
   `internal/companion/service.go`, `internal/companion/shared/shared.go`, `internal/care/delegation.go`,
   `internal/children/service.go`) and that T-M9-02 has shipped.
2. Ask the user in Persian, one line per decision with the recommended default, grouped: what members see (F-01,
   F-02, F-03, F-18, F-21), writes and proposals (F-04, F-05, F-14, F-26), nudges and alerts (F-06 … F-10), invites
   and links (F-11, F-12, F-13, F-17), children and household (F-15, F-24, F-25), mode change (F-23), screens without
   artboards (F-22), admin and retention (F-16, F-19, F-20). Offer "OK to all defaults"; accept per-item changes.
3. Record every answer in DECISIONS.md (`accepted` or `changed — <what>`, date) and replace the "All F rows" status
   line with the approval date.
4. Write `docs/engine-v6/family.md`, the reference every M11 task reads:
   - domain registry: code, max level, partner/carer/friend/other-parent defaults, mode scope, roles allowed, male-owner
     rule (F-20), the TTC switch (F-02);
   - templates (cycle_buddy, meds_appointments, full_family, custom) and role defaults (SPEC §1.7 tables);
   - legacy → v6 mapping: link `type` → `role`, section → domain, level none/view/edit → off/view/nudge/write;
   - per domain, the allow-list of fields a member may receive (and the never-shared list as negative tests);
   - the `check()` order (gap 05 §5.2 with F-23) and the mode-epoch rule;
   - activity actions, payload sealing and retention (F-14, F-16, F-19);
   - copy keys for owner and member notices (neutral member wording «پیامی از خانواده در ریتمی»).
5. Write rows **D-190 … D-206** (table "M11 deviation ids" in DECISIONS.md) in `docs/go-migration/deviations.md`,
   status `accepted — engine v6 M11, not shipped yet`; D-207…D-209 stay reserved.
6. If an answer changes a task's scope, edit that task file now and list the edits in `tasks/PROGRESS.md`. If F-18 is
   "no", mark T-M11-16 `skipped` with the reason.

## Out of scope
- Code.

## Acceptance
- No F row is open; family.md complete; D-190…D-206 exist; edited tasks listed in PROGRESS.md.
