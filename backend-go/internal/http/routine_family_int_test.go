package http_test

// T-M12-17 (T-05): the family path as a member really lives it, end to end through the real routes — owner invites
// with one domain at view → member accepts → reads only that domain → cannot write → owner raises the level → the
// write goes through → owner revokes the domain, then the link → the very next member request is refused with no
// data → the member's own data is untouched; and a code-only write-level invite that waits for the owner (F-27)
// followed by an owner mode change that pauses every owner-data domain while the urgent alert stays (F-23).
//
// Expected answers come from docs/engine-v6/family.md and the F decisions (DECISIONS.md F-01 … F-27), not from what
// the API happens to return: F-21 (404 for an unknown / revoked link, 403 family_domain_denied with no data for an
// active link without the domain), F-04/F-05 (write on the owner's records), F-23 (mode change), F-27 (code-only
// write-level invites wait as awaiting_owner).

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/family/invites"
	"github.com/ritme/backend-go/internal/http/testenv"
)

const (
	famMembers   = "/api/v1/family/members"
	famMe        = "/api/v1/family/me"
	famAccept    = "/api/v1/family/invites/accept"
	famPreview   = "/api/v1/family/invites/preview"
	famCareAppts = "/api/v1/care/appointments"
	famCareMeds  = "/api/v1/care/medications"
	// famNoteLeak is in the owner's appointment notes / topic: it must never reach a member (family.md §4 "never").
	famNoteLeak = "famsecretnote"
)

// famWant is the expected answer of one member request: the status and, for a refusal, the error_code.
type famWant struct {
	status int
	code   string // "" = no error_code expected (200, or the 404 that carries none)
}

var (
	famOK     = famWant{status: fiber.StatusOK}
	famDenied = famWant{status: fiber.StatusForbidden, code: "family_domain_denied"} // F-21: active link, no domain
	famGone   = famWant{status: fiber.StatusNotFound}                                // F-21: no / revoked link, no route
)

// famFlow is one owner with her members on a booted API.
type famFlow struct {
	t     *testing.T
	e     *testenv.Env
	owner testenv.Account
}

func (f *famFlow) domainPath(ownerID uint64, domain string) string {
	return fmt.Sprintf("/api/v1/family/owners/%d/domains/%s", ownerID, domain)
}

// read is GET /family/owners/{owner}/domains/{domain} as tok, revalidating with ifNoneMatch when set.
func (f *famFlow) read(tok, domain, ifNoneMatch string) testenv.Reply {
	f.t.Helper()
	h := map[string]string{}
	if ifNoneMatch != "" {
		h[fiber.HeaderIfNoneMatch] = ifNoneMatch
	}
	return f.e.Send(testenv.Request{Method: fiber.MethodGet, Path: f.domainPath(f.owner.ID, domain), Token: tok, Header: h})
}

// refused asserts a member refusal: the wanted status and code, and no data anywhere in the body (family.md §5:
// "responses for denied member reads carry no data").
func famRefused(t *testing.T, label string, r testenv.Reply, w famWant) {
	t.Helper()
	assert.Equal(t, w.status, r.Status, "%s: %s", label, r.Raw)
	if w.code != "" {
		assert.Equal(t, w.code, r.Body["error_code"], "%s: %s", label, r.Raw)
	}
	assert.Nil(t, r.Body["data"], "%s: a refusal carries no data", label)
	assert.NotContains(t, r.Raw, famNoteLeak, label)
	assert.NotContains(t, r.Raw, "Ultrasound visit", label)
}

// check asserts one member request against w (an allow is a 200 with data; a refusal goes through famRefused).
func famCheck(t *testing.T, label string, r testenv.Reply, w famWant) {
	t.Helper()
	if w.status == fiber.StatusOK {
		require.Equal(t, fiber.StatusOK, r.Status, "%s: %s", label, r.Raw)
		assert.NotNil(t, r.Body["data"], label)
		return
	}
	famRefused(t, label, r, w)
}

// table checks every member-facing domain code on the v6 projection route against want (a code left out of want
// expects famDenied when it has a projection, famGone otherwise). Every answer is private, no-store.
func (f *famFlow) table(label, tok string, want map[string]famWant) {
	f.t.Helper()
	// The member-readable projections (family.md §4) and every other code a member could try: grantable domains
	// without a member projection, the teen card (F-17), children (guardianship), never-shared categories (§4,
	// F-03) and junk.
	projected := []string{"cycle_phase", "ttc_fertility", "pregnancy", "fetal_activity", "appointments", "meds", "vitals"}
	unprojected := []string{"urgent_alert", "household", "teen_period_week", "teen_kit", "teen_notes", "children",
		"symptoms", "mood", "private_notes", "lab_results", "cycle_history", "bogus"}
	for _, d := range append(append([]string{}, projected...), unprojected...) {
		w, ok := want[d]
		if !ok {
			w = famGone
			for _, p := range projected {
				if p == d {
					w = famDenied
				}
			}
		}
		r := f.read(tok, d, "")
		famCheck(f.t, label+" "+d, r, w)
		assert.Equal(f.t, "private, no-store", r.Header.Get(fiber.HeaderCacheControl), "%s %s: never cached", label, d)
	}
}

// meAsMember is the caller's as_member entries of GET /family/me, keyed by owner id → domain → level.
func (f *famFlow) meAsMember(tok string) map[uint64]map[string]string {
	f.t.Helper()
	r := f.e.MustOK(fiber.StatusOK, fiber.MethodGet, famMe, nil, tok)
	out := map[uint64]map[string]string{}
	list, _ := r.Data()["as_member"].([]any)
	for _, x := range list {
		m, _ := x.(map[string]any)
		owner, _ := testenv.Path(m, "owner", "id").(float64)
		doms := map[string]string{}
		ds, _ := m["domains"].([]any)
		for _, d := range ds {
			dm, _ := d.(map[string]any)
			code, _ := dm["domain"].(string)
			lv, _ := dm["level"].(string)
			doms[code] = lv
		}
		out[uint64(owner)] = doms
	}
	return out
}

// ownerGrants is the owner's view of one link in GET /family/me: status, review_pending and domain → state.
func (f *famFlow) ownerGrants(linkID uint64) (status string, reviewPending bool, states map[string]string) {
	f.t.Helper()
	r := f.e.MustOK(fiber.StatusOK, fiber.MethodGet, famMe, nil, f.owner.Token)
	list, _ := r.Data()["as_owner"].([]any)
	states = map[string]string{}
	for _, x := range list {
		m, _ := x.(map[string]any)
		if id, _ := m["id"].(float64); uint64(id) != linkID {
			continue
		}
		status, _ = m["status"].(string)
		reviewPending, _ = m["review_pending"].(bool)
		gs, _ := m["grants"].([]any)
		for _, g := range gs {
			gm, _ := g.(map[string]any)
			code, _ := gm["domain"].(string)
			st, _ := gm["state"].(string)
			states[code] = st
		}
		return status, reviewPending, states
	}
	f.t.Fatalf("link %d is not in the owner's /family/me", linkID)
	return "", false, nil
}

// invite is POST /family/members as the owner (code only); it returns the link id and the code.
func (f *famFlow) invite(body map[string]any) (uint64, string) {
	f.t.Helper()
	body["consent_version"] = invites.ConsentVersion
	r := f.e.MustOK(fiber.StatusCreated, fiber.MethodPost, famMembers, body, f.owner.Token)
	link, _ := r.Data()["link"].(map[string]any)
	inv, _ := r.Data()["invite"].(map[string]any)
	id, _ := link["id"].(float64)
	code, _ := inv["code"].(string)
	require.NotEmpty(f.t, code, r.Raw)
	return uint64(id), code
}

// legacyGrants is the owner's grant edit (PUT /companions/{id}/grants, the only grant-edit route today).
func (f *famFlow) legacyGrants(linkID uint64, grants map[string]any) {
	f.t.Helper()
	f.e.MustOK(fiber.StatusOK, fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", linkID),
		map[string]any{"grants": grants}, f.owner.Token)
}

// apptBody is a care appointment body; forUser != 0 makes it a «ثبت برای …» member write.
func famApptBody(title, at string, forUser uint64) map[string]any {
	b := map[string]any{"kind": "in_person", "topic": "ultrasound", "title": title, "scheduled_at": at,
		"location": "Clinic", "notes": famNoteLeak}
	if forUser != 0 {
		b["for_user_id"] = forUser
	}
	return b
}

func famMedBody(title string, forUser uint64) map[string]any {
	b := map[string]any{"title": title, "form": "tablet", "times": []string{"21:00"}, "notes": famNoteLeak}
	if forUser != 0 {
		b["for_user_id"] = forUser
	}
	return b
}

func famID(t *testing.T, r testenv.Reply) uint64 {
	t.Helper()
	id, _ := r.Data()["id"].(float64)
	require.Positive(t, id, r.Raw)
	return uint64(id)
}

// newFamFlow boots the API and onboards the owner: a woman tracking her cycle (cycle mode).
func newFamFlow(t *testing.T) *famFlow {
	t.Helper()
	e := testenv.New(t)
	owner := e.User()
	f := &famFlow{t: t, e: e, owner: owner}
	for _, s := range []struct {
		step string
		body map[string]any
	}{
		{"gender", map[string]any{"gender": "female"}},
		{"goal", map[string]any{"goal": "cycle"}},
		{"cycle", map[string]any{"last_period_start": "2026-09-05", "period_duration": 5, "cycle_duration": 28}},
	} {
		e.MustOK(fiber.StatusOK, fiber.MethodPut, "/api/v1/onboarding/steps/"+s.step, s.body, owner.Token)
	}
	return f
}

// member is a new account (male, onboarded gender only) that will be a family member.
func (f *famFlow) member() testenv.Account {
	f.t.Helper()
	a := f.e.User()
	f.e.MustOK(fiber.StatusOK, fiber.MethodPut, "/api/v1/onboarding/steps/gender", map[string]any{"gender": "male"}, a.Token)
	return a
}

// TestRoutineFamily_InviteViewRaiseRevoke: invite one domain at view → accept → domain-limited reads over every
// domain → no write → owner raises the level → write → owner revokes the domain, then the link → the very next
// request is refused (no cache, no stale ETag) → the member's own data untouched.
func TestRoutineFamily_InviteViewRaiseRevoke(t *testing.T) {
	f := newFamFlow(t)
	e, sara := f.e, f.owner
	ali := f.member()

	// The owner's data: one upcoming appointment (with notes and a topic a member must never see) and one medication.
	apptID := famID(t, e.MustOK(fiber.StatusCreated, fiber.MethodPost, famCareAppts,
		famApptBody("Ultrasound visit", "2026-09-30 10:00", 0), sara.Token))
	medID := famID(t, e.MustOK(fiber.StatusCreated, fiber.MethodPost, famCareMeds, famMedBody("Iron", 0), sara.Token))
	// The member's own data.
	aliMed := famID(t, e.MustOK(fiber.StatusCreated, fiber.MethodPost, famCareMeds, famMedBody("Ali vitamin", 0), ali.Token))
	aliAppt := famID(t, e.MustOK(fiber.StatusCreated, fiber.MethodPost, famCareAppts,
		famApptBody("Ali dentist", "2026-10-01 09:00", 0), ali.Token))
	aliRows := func() int {
		return e.Count(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, ali.ID)
	}
	aliBefore := aliRows()
	require.Equal(t, 2, aliBefore)

	// Before any link: the owner is a stranger to Ali — every domain is a 404 (no existence oracle, F-21).
	f.table("no link", ali.Token, map[string]famWant{"cycle_phase": famGone, "ttc_fertility": famGone,
		"pregnancy": famGone, "fetal_activity": famGone, "appointments": famGone, "meds": famGone, "vitals": famGone})

	// 1. Owner invites with exactly one domain at view (cycle_buddy minus its cycle row, plus appointments: view).
	// A code-only invite without any write level, urgent alert or vitals activates on accept (F-11, F-27).
	linkID, code := f.invite(map[string]any{"role": "partner", "template": "cycle_buddy",
		"grants": map[string]any{"cycle_phase": "off", "appointments": "view"}})
	// Invited, not accepted: still nothing.
	famRefused(t, "invited only", f.read(ali.Token, "appointments", ""), famGone)
	pv := e.MustOK(fiber.StatusOK, fiber.MethodPost, famPreview, map[string]any{"code": code}, ali.Token)
	assert.Equal(t, false, pv.Data()["owner_confirmation"], "a view-only code invite needs no owner confirmation")
	acc := e.MustOK(fiber.StatusOK, fiber.MethodPost, famAccept, map[string]any{"code": code}, ali.Token)
	assert.Equal(t, "active", testenv.Path(acc.Data(), "link", "status"), acc.Raw)

	// Member reads: only appointments, the allow-listed DTO (family.md §4: no notes, no topic).
	f.table("view", ali.Token, map[string]famWant{"appointments": famOK})
	r := f.read(ali.Token, "appointments", "")
	items, _ := r.Data()["data"].([]any)
	require.Len(t, items, 1, r.Raw)
	item, _ := items[0].(map[string]any)
	assert.Equal(t, "Ultrasound visit", item["title"])
	assert.Equal(t, "2026-09-30 10:00:00", item["scheduled_at"])
	for _, never := range []string{"notes", "topic", "result_note", "prep", "remind_before"} {
		assert.NotContains(t, item, never, "appointments DTO never carries %s", never)
	}
	assert.NotContains(t, r.Raw, famNoteLeak)
	assert.Equal(t, map[uint64]map[string]string{sara.ID: {"appointments": "view"}}, f.meAsMember(ali.Token))

	// The «ثبت برای …» routes say the same: the appointment record readable, the medication not.
	e.MustOK(fiber.StatusOK, fiber.MethodGet, fmt.Sprintf("%s/%d?for_user_id=%d", famCareAppts, apptID, sara.ID), nil, ali.Token)
	r = e.Do(fiber.MethodGet, fmt.Sprintf("%s/%d?for_user_id=%d", famCareMeds, medID, sara.ID), nil, ali.Token)
	famRefused(t, "meds record at no grant", r, famWant{status: fiber.StatusForbidden, code: "companion_forbidden"})
	// The /companions compatibility layer (D-204) agrees section by section.
	sec := func(s string) testenv.Reply {
		return e.Do(fiber.MethodGet, fmt.Sprintf("/api/v1/companions/links/%d/sections/%s", linkID, s), nil, ali.Token)
	}
	require.Equal(t, fiber.StatusOK, sec("appointments").Status)
	for _, s := range []string{"cycle", "pregnancy", "meds"} {
		famRefused(t, "compat "+s, sec(s), famWant{status: fiber.StatusForbidden, code: "section_not_shared"})
	}
	famRefused(t, "compat symptoms (F-03)", sec("symptoms"), famGone)

	// Cannot write at view: create, edit — appointments (granted at view) and meds (not granted).
	before := e.Count(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, sara.ID)
	r = e.Do(fiber.MethodPost, famCareAppts, famApptBody("Ali's idea", "2026-10-03 11:00", sara.ID), ali.Token)
	famRefused(t, "create appointment at view", r, famWant{status: fiber.StatusForbidden, code: "companion_forbidden"})
	r = e.Do(fiber.MethodPut, fmt.Sprintf("%s/%d", famCareAppts, apptID), map[string]any{"title": "Changed", "for_user_id": sara.ID}, ali.Token)
	famRefused(t, "edit appointment at view", r, famWant{status: fiber.StatusForbidden, code: "companion_forbidden"})
	r = e.Do(fiber.MethodPost, famCareMeds, famMedBody("Ali's pill", sara.ID), ali.Token)
	famRefused(t, "create medication, no grant", r, famWant{status: fiber.StatusForbidden, code: "companion_forbidden"})
	assert.Equal(t, before, e.Count(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, sara.ID), "nothing written")

	// 2. Owner raises appointments to write. It is the owner's own act on a link whose member she already knows, so
	// the link stays active (F-27's owner confirmation is for a code redeemed by whoever holds it — see the next test).
	f.legacyGrants(linkID, map[string]any{"appointments": "edit"})
	st, _, _ := f.ownerGrants(linkID)
	assert.Equal(t, "active", st)
	assert.Equal(t, map[uint64]map[string]string{sara.ID: {"appointments": "write"}}, f.meAsMember(ali.Token))
	f.table("write", ali.Token, map[string]famWant{"appointments": famOK})
	// The write goes through. F-04/F-05 decide it is a proposal (202) the owner approves; until proposals ship
	// (T-M11-09, todo) the compatibility path writes directly (201) with the owner notice — both are "allowed".
	r = e.Do(fiber.MethodPost, famCareAppts, famApptBody("Ali booked", "2026-10-04 12:00", sara.ID), ali.Token)
	assert.Contains(t, []int{fiber.StatusCreated, fiber.StatusAccepted}, r.Status, r.Raw)
	r = e.Do(fiber.MethodPut, fmt.Sprintf("%s/%d", famCareAppts, apptID), map[string]any{"location": "Clinic 2", "for_user_id": sara.ID}, ali.Token)
	assert.Contains(t, []int{fiber.StatusOK, fiber.StatusAccepted}, r.Status, r.Raw)
	// Still no meds: the raise touched one domain only.
	r = e.Do(fiber.MethodPost, famCareMeds, famMedBody("Ali's pill", sara.ID), ali.Token)
	famRefused(t, "create medication after the raise", r, famWant{status: fiber.StatusForbidden, code: "companion_forbidden"})

	// 3a. Owner turns the domain off: the very next read is a 403 with no data — not a cached 200, not a 304 for the
	// tag of the last good answer.
	ok := f.read(ali.Token, "appointments", "")
	require.Equal(t, fiber.StatusOK, ok.Status, ok.Raw)
	tag := ok.ETag()
	if tag == "" {
		tag = `W/"family-appointments"` // no ETag is issued (no-store); a client that sends one anyway gets no 304
	}
	f.legacyGrants(linkID, map[string]any{})
	famRefused(t, "domain revoked", f.read(ali.Token, "appointments", tag), famDenied)
	f.table("domain revoked", ali.Token, map[string]famWant{})
	assert.Equal(t, map[uint64]map[string]string{sara.ID: {}}, f.meAsMember(ali.Token), "the link stays, no domain")
	r = e.Do(fiber.MethodPost, famCareAppts, famApptBody("Ali again", "2026-10-05 12:00", sara.ID), ali.Token)
	famRefused(t, "write after domain revoke", r, famWant{status: fiber.StatusForbidden, code: "companion_forbidden"})

	// 3b. Owner gives it back, then revokes the whole link: the very next request is a 404 (F-21, no oracle).
	f.legacyGrants(linkID, map[string]any{"appointments": "view"})
	require.Equal(t, fiber.StatusOK, f.read(ali.Token, "appointments", "").Status)
	e.MustOK(fiber.StatusOK, fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/%d", linkID), nil, sara.Token)
	famRefused(t, "link revoked", f.read(ali.Token, "appointments", tag), famGone)
	f.table("link revoked", ali.Token, map[string]famWant{"cycle_phase": famGone, "ttc_fertility": famGone,
		"pregnancy": famGone, "fetal_activity": famGone, "appointments": famGone, "meds": famGone, "vitals": famGone})
	assert.Empty(t, f.meAsMember(ali.Token), "a revoked link is not listed")
	r = e.Do(fiber.MethodGet, fmt.Sprintf("%s/%d?for_user_id=%d", famCareAppts, apptID, sara.ID), nil, ali.Token)
	assert.Contains(t, []int{fiber.StatusForbidden, fiber.StatusNotFound}, r.Status, r.Raw)
	assert.Nil(t, r.Body["data"])
	famRefused(t, "compat after revoke", sec("appointments"), famGone)

	// 4. The member's own data is untouched by every grant change and the revoke, and still his alone.
	assert.Equal(t, aliBefore, aliRows())
	e.MustOK(fiber.StatusOK, fiber.MethodGet, fmt.Sprintf("%s/%d", famCareMeds, aliMed), nil, ali.Token)
	e.MustOK(fiber.StatusOK, fiber.MethodGet, fmt.Sprintf("%s/%d", famCareAppts, aliAppt), nil, ali.Token)
	list := e.MustOK(fiber.StatusOK, fiber.MethodGet, famCareMeds, nil, ali.Token)
	assert.Contains(t, list.Raw, "Ali vitamin")
	assert.NotContains(t, list.Raw, `"Iron"`, "the owner's medication never lands in the member's own list")
	// Links are directional: the owner never got anything of Ali's.
	r = e.Do(fiber.MethodGet, fmt.Sprintf("%s/%d?for_user_id=%d", famCareMeds, aliMed, ali.ID), nil, sara.Token)
	famRefused(t, "reverse direction", r, famWant{status: fiber.StatusForbidden, code: "companion_forbidden"})
	assert.Equal(t, fiber.StatusNotFound, e.Do(fiber.MethodGet,
		fmt.Sprintf("/api/v1/family/owners/%d/domains/meds", ali.ID), nil, sara.Token).Status)
}

// TestRoutineFamily_CodeWriteInviteWaitsThenModeChangePauses: a code-only invite with a write level waits as
// awaiting_owner and shows nothing until the owner confirms (F-27); then the owner's mode change (cycle → pregnancy)
// pauses every owner-data domain until she reviews while the urgent-alert switch stays on (F-23).
func TestRoutineFamily_CodeWriteInviteWaitsThenModeChangePauses(t *testing.T) {
	f := newFamFlow(t)
	e, sara := f.e, f.owner
	reza := f.member()
	e.MustOK(fiber.StatusCreated, fiber.MethodPost, famCareAppts, famApptBody("Ultrasound visit", "2026-09-30 10:00", 0), sara.Token)

	// F-27 on its own: a code-only invite whose only high-impact part is a write level (appointments: write, no
	// urgent alert, no vitals, no children) still waits for the owner.
	omid := f.member()
	writeOnly, wcode := f.invite(map[string]any{"role": "partner", "template": "custom",
		"grants": map[string]any{"cycle_phase": "off", "pregnancy": "off", "fetal_activity": "off", "meds": "off",
			"household": "off", "appointments": "write"}, "urgent_alert": false})
	pv := e.MustOK(fiber.StatusOK, fiber.MethodPost, famPreview, map[string]any{"code": wcode}, omid.Token)
	assert.Equal(t, true, pv.Data()["owner_confirmation"], "F-27: write level alone needs the owner")
	acc := e.MustOK(fiber.StatusOK, fiber.MethodPost, famAccept, map[string]any{"code": wcode}, omid.Token)
	assert.Equal(t, "awaiting_owner", testenv.Path(acc.Data(), "link", "status"), acc.Raw)
	famRefused(t, "write-only invite before confirm", f.read(omid.Token, "appointments", ""), famGone)
	assert.Empty(t, f.meAsMember(omid.Token))
	e.MustOK(fiber.StatusOK, fiber.MethodPost, fmt.Sprintf("%s/%d/cancel", famMembers, writeOnly), nil, sara.Token)
	famRefused(t, "write-only invite cancelled", f.read(omid.Token, "appointments", ""), famGone)

	// meds_appointments = appointments: write, meds: nudge; plus cycle_phase and the urgent alert.
	linkID, code := f.invite(map[string]any{"role": "partner", "template": "meds_appointments",
		"grants": map[string]any{"cycle_phase": "view"}, "urgent_alert": true})
	pv = e.MustOK(fiber.StatusOK, fiber.MethodPost, famPreview, map[string]any{"code": code}, reza.Token)
	assert.Equal(t, true, pv.Data()["owner_confirmation"], "F-27: a code-only write invite needs the owner")
	acc = e.MustOK(fiber.StatusOK, fiber.MethodPost, famAccept, map[string]any{"code": code}, reza.Token)
	assert.Equal(t, "awaiting_owner", testenv.Path(acc.Data(), "link", "status"), acc.Raw)

	// Before confirmation the member sees nothing at all: every domain 404, nothing listed, no write.
	f.table("awaiting_owner", reza.Token, map[string]famWant{"cycle_phase": famGone, "ttc_fertility": famGone,
		"pregnancy": famGone, "fetal_activity": famGone, "appointments": famGone, "meds": famGone, "vitals": famGone})
	assert.Empty(t, f.meAsMember(reza.Token))
	r := e.Do(fiber.MethodPost, famCareAppts, famApptBody("Reza booked", "2026-10-04 12:00", sara.ID), reza.Token)
	assert.Contains(t, []int{fiber.StatusForbidden, fiber.StatusNotFound}, r.Status, r.Raw)
	// Only the owner confirms: the member confirming his own link is a 404.
	confirm := fmt.Sprintf("%s/%d/confirm", famMembers, linkID)
	assert.Equal(t, fiber.StatusNotFound, e.Do(fiber.MethodPost, confirm, nil, reza.Token).Status)
	st, _, _ := f.ownerGrants(linkID)
	assert.Equal(t, "awaiting_owner", st)

	cr := e.MustOK(fiber.StatusOK, fiber.MethodPost, confirm, nil, sara.Token)
	assert.Equal(t, "active", cr.Data()["status"])
	f.table("confirmed", reza.Token, map[string]famWant{"cycle_phase": famOK, "appointments": famOK, "meds": famOK})
	assert.Equal(t, map[uint64]map[string]string{sara.ID: {"cycle_phase": "view", "appointments": "write",
		"meds": "nudge", "urgent_alert": "view"}}, f.meAsMember(reza.Token))
	// Cycle phase allow-list (F-01): main_phase (+ the partner's period_soon), never dates or numbers.
	cp := f.read(reza.Token, "cycle_phase", "")
	for _, never := range []string{"cycle_day", "cycle_length", "days_late", "confidence", "last_period_start", "2026-09-05"} {
		assert.NotContains(t, cp.Raw, never)
	}
	e.MustOK(fiber.StatusCreated, fiber.MethodPost, famCareAppts, famApptBody("Reza booked", "2026-10-04 12:00", sara.ID), reza.Token)

	// The owner's mode changes: she starts a pregnancy (cycle → pregnancy, a new mode epoch).
	e.MustOK(fiber.StatusCreated, fiber.MethodPost, "/api/v1/pregnancy/onboarding",
		map[string]any{"age_source": "lmp", "lmp_date": "2026-09-05"}, sara.Token)

	// F-23: every owner-data domain pauses until she reviews (cycle_phase is also out of the pregnancy scope);
	// the member sees only what is not owner data — here the urgent-alert switch, which stays on.
	f.table("mode changed", reza.Token, map[string]famWant{})
	assert.Equal(t, map[uint64]map[string]string{sara.ID: {"urgent_alert": "view"}}, f.meAsMember(reza.Token),
		"after a mode change only the urgent alert stays (F-23)")
	r = e.Do(fiber.MethodPost, famCareAppts, famApptBody("Reza again", "2026-10-06 12:00", sara.ID), reza.Token)
	famRefused(t, "write while paused", r, famWant{status: fiber.StatusForbidden, code: "companion_forbidden"})
	st, review, states := f.ownerGrants(linkID)
	assert.Equal(t, "active", st, "a mode change pauses, it never revokes")
	assert.True(t, review, "the owner gets the review card")
	assert.Equal(t, map[string]string{"cycle_phase": "inactive", "appointments": "paused", "meds": "paused",
		"urgent_alert": "active"}, states)

	// The owner reviews and keeps appointments and meds (through the grant edit, the review card's re-approval):
	// those come back at once, cycle_phase stays out of the pregnancy scope.
	f.legacyGrants(linkID, map[string]any{"appointments": "edit", "meds": "view", "cycle": "view"})
	f.table("reviewed", reza.Token, map[string]famWant{"appointments": famOK, "meds": famOK})
	e.MustOK(fiber.StatusCreated, fiber.MethodPost, famCareAppts, famApptBody("Reza after review", "2026-10-07 12:00", sara.ID), reza.Token)
	// The urgent-alert switch stays on through the change and its review (F-23, T-M12-17b).
	assert.Equal(t, map[uint64]map[string]string{sara.ID: {"appointments": "write", "meds": "view", "urgent_alert": "view"}},
		f.meAsMember(reza.Token))
}

// TestRoutineFamily_GrantEditKeepsUrgentAlert: the owner's grant edit must not switch the urgent alert off. F-23
// says the urgent-alert switch stays on through a mode change and its review; the review re-approval today is the
// /companions grant edit, which only knows the legacy sections.
// Found by T-M12-17, fixed by T-M12-17b (the edit is scoped to the legacy domains, family.Change.Scope).
func TestRoutineFamily_GrantEditKeepsUrgentAlert(t *testing.T) {
	f := newFamFlow(t)
	e := f.e
	reza := f.member()
	linkID, code := f.invite(map[string]any{"role": "partner", "template": "meds_appointments", "urgent_alert": true})
	e.MustOK(fiber.StatusOK, fiber.MethodPost, famAccept, map[string]any{"code": code}, reza.Token)
	e.MustOK(fiber.StatusOK, fiber.MethodPost, fmt.Sprintf("%s/%d/confirm", famMembers, linkID), nil, f.owner.Token)
	require.Equal(t, "view", f.meAsMember(reza.Token)[f.owner.ID]["urgent_alert"])

	// The owner changes an unrelated level (meds → view) with the grant edit.
	f.legacyGrants(linkID, map[string]any{"appointments": "edit", "meds": "view"})
	got := f.meAsMember(reza.Token)[f.owner.ID]
	assert.Equal(t, "view", got["meds"])
	assert.Equal(t, "view", got["urgent_alert"], "editing the meds level must not turn the urgent alert off: %v", got)
	_, _, states := f.ownerGrants(linkID)
	assert.Contains(t, states, "urgent_alert", "the owner's own list still shows the urgent alert")
}
