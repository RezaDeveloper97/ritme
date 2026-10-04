package http

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// B-N4-08b: companion security fixes (docs/security/bloom-companion.md CMP-M1, CMP-M2, CMP-M3, CMP-L1, CMP-L5).

// securityPair is Sara (cycling owner) with Ali (male companion) on an active partner link: cycle + symptoms view,
// meds + appointments edit.
func securityPair(t *testing.T, e *companionEnv) (sara, ali string, linkID uint64) {
	t.Helper()
	sara = e.user("Sara", "09120008001")
	ali = e.user("Ali", "09120008002")
	e.step(sara, "gender", map[string]any{"gender": "female"})
	e.step(sara, "goal", map[string]any{"goal": "cycle"})
	e.step(sara, "cycle", map[string]any{"last_period_start": "2026-09-05", "period_duration": 5, "cycle_duration": 28})
	e.step(ali, "gender", map[string]any{"gender": "male"})
	linkID, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{
		"cycle": "view", "symptoms": "view", "meds": "edit", "appointments": "edit"}})
	r := e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	return sara, ali, linkID
}

// clearThrottles drops the throttle counters (not the breaker), so a test can look at one guard at a time.
func (e *companionEnv) clearThrottles(match string) {
	for _, k := range e.mr.Keys() {
		if strings.Contains(k, "throttle:") && strings.Contains(k, match) {
			e.mr.Del(k)
		}
	}
}

// CMP-M1 / QUESTIONS #98: an active partner link of an owner who switches to teen mode grants nothing while she is a
// teen (no revoke); her stored grants come back when she leaves teen mode.
func TestCompanionSecurity_TeenOwnerSuspendsPartnerLinks(t *testing.T) {
	e := newCompanionEnv(t)
	sara, ali, linkID := securityPair(t, e)
	saraID := e.ids["Sara"]
	sec := func(s string) string { return fmt.Sprintf("/api/v1/companions/links/%d/sections/%s", linkID, s) }
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, sec("cycle"), ali, nil).Status)

	_, err := e.db.Exec(`UPDATE user_life_profiles SET life_mode = 'teen' WHERE user_id = ?`, saraID)
	require.NoError(t, err)

	for _, s := range []string{"cycle", "symptoms", "meds", "appointments"} {
		r := e.do(fiber.MethodGet, sec(s), ali, nil)
		assert.Equal(t, fiber.StatusForbidden, r.Status, s)
		assert.Equal(t, "section_not_shared", r.Body["error_code"], s)
	}
	h := e.home(ali)
	require.Equal(t, fiber.StatusOK, h.Status, h.Raw)
	p := partnersOf(h)
	require.Len(t, p, 1, "the link is not revoked")
	card := p[0].(map[string]any)
	for _, s := range []string{"cycle", "pregnancy", "symptoms", "meds", "appointments"} {
		assert.Nil(t, card[s], s)
	}
	r := e.do(fiber.MethodPost, "/api/v1/care/medications", ali, medBody(map[string]any{"for_user_id": saraID}))
	assert.Equal(t, fiber.StatusForbidden, r.Status, "no «ثبت برای …» either")
	assert.Equal(t, "companion_forbidden", r.Body["error_code"])
	links := e.do(fiber.MethodGet, "/api/v1/companions/links", ali, nil).list()
	require.Len(t, links, 1)
	assert.Equal(t, map[string]any{"cycle": "none", "symptoms": "none", "meds": "none", "appointments": "none", "pregnancy": "none"},
		links[0].(map[string]any)["grants"])

	// The owner still sees the link with what she chose.
	own := e.do(fiber.MethodGet, "/api/v1/companions", sara, nil).list()
	require.Len(t, own, 1)
	assert.Equal(t, "active", own[0].(map[string]any)["status"])
	assert.Equal(t, "view", own[0].(map[string]any)["grants"].(map[string]any)["cycle"])

	_, err = e.db.Exec(`UPDATE user_life_profiles SET life_mode = 'cycle' WHERE user_id = ?`, saraID)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, sec("cycle"), ali, nil).Status, "back from teen mode")
}

// CMP-M2: delegated single-record reads and updates stay inside the section view; delegated GETs are throttled.
func TestCompanionSecurity_DelegatedRecordsStayInView(t *testing.T) {
	e := newCompanionEnv(t)
	sara, ali, _ := securityPair(t, e)
	saraID := e.ids["Sara"]

	create := func(path string, body map[string]any) uint64 {
		r := e.do(fiber.MethodPost, path, sara, body)
		require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
		return idOf(r.data()["id"])
	}
	active := create("/api/v1/care/medications", medBody(nil))
	paused := create("/api/v1/care/medications", medBody(map[string]any{"title": "Old pill"}))
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/care/medications/%d", paused), sara,
		map[string]any{"is_active": false}).Status)
	upcoming := create("/api/v1/care/appointments", map[string]any{"kind": "in_person", "topic": "ultrasound", "scheduled_at": "2026-09-30 10:00"})
	cancelled := create("/api/v1/care/appointments", map[string]any{"kind": "in_person", "topic": "ultrasound", "scheduled_at": "2026-10-02 10:00"})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, fmt.Sprintf("/api/v1/care/appointments/%d/cancel", cancelled), sara, nil).Status)

	med := func(id uint64) string { return fmt.Sprintf("/api/v1/care/medications/%d?for_user_id=%d", id, saraID) }
	appt := func(id uint64) string { return fmt.Sprintf("/api/v1/care/appointments/%d?for_user_id=%d", id, saraID) }

	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, med(active), ali, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, med(paused), ali, nil).Status, "a paused medication is history")
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d", paused), sara, nil).Status, "the owner sees it")
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, appt(upcoming), ali, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, appt(cancelled), ali, nil).Status, "a cancelled visit is history")
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/appointments/%d", cancelled), sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, appt(upcoming), ali, nil, "2026-10-01T10:00:00+03:30").Status,
		"once past, an appointment leaves the view")

	// Edit grants cannot reopen history.
	put := func(path string, id uint64, body map[string]any) reply {
		body["for_user_id"] = saraID
		return e.do(fiber.MethodPut, fmt.Sprintf(path, id), ali, body)
	}
	assert.Equal(t, fiber.StatusNotFound, put("/api/v1/care/medications/%d", paused, map[string]any{"is_active": true}).Status)
	assert.Equal(t, fiber.StatusNotFound, put("/api/v1/care/appointments/%d", cancelled, map[string]any{"topic": "lab"}).Status)
	assert.Equal(t, fiber.StatusOK, put("/api/v1/care/medications/%d", active, map[string]any{"title": "Iron 2"}).Status)
	assert.Equal(t, fiber.StatusOK, put("/api/v1/care/appointments/%d", upcoming, map[string]any{"topic": "lab"}).Status)

	// Throttle: CompanionDelegatedReadsPerUser delegated GETs per hour, then 429; the owner's own reads are not counted.
	e.clearThrottles("companion-delegated-read")
	for i := range CompanionDelegatedReadsPerUser {
		require.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, med(active), ali, nil).Status, i)
	}
	r := e.do(fiber.MethodGet, med(active), ali, nil)
	assert.Equal(t, fiber.StatusTooManyRequests, r.Status)
	assert.Equal(t, "too_many_requests", r.Body["error_code"])
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d", active), sara, nil).Status)
}

// CMP-M3: refused accepts of every caller count against one global budget; past it accepts pause for everyone (the
// generic 429), even with a valid code, until the cool-down ends.
func TestCompanionSecurity_AcceptCircuitBreaker(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008101")
	_, code := e.invite(sara, map[string]any{"type": "partner"})

	guessers := []string{e.user("G1", "09120008102"), e.user("G2", "09120008103"), e.user("G3", "09120008104"), e.user("G4", "09120008105")}
	failures := 0
	for failures < CompanionAcceptFailuresPerMinute {
		tok := guessers[failures/CompanionAcceptPerUser]
		r := e.do(fiber.MethodPost, "/api/v1/companions/accept", tok, map[string]any{"code": "ZZZZZZ"})
		require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
		require.Equal(t, "invite_invalid", r.Body["error_code"])
		failures++
	}
	open := "ritme-go-b4n2:companion-accept-breaker:open"
	assert.True(t, e.mr.Exists(open), "the breaker tripped")
	e.clearThrottles("companion-accept") // only the breaker is left to say no

	newcomer := e.user("Ali", "09120008106")
	r := e.do(fiber.MethodPost, "/api/v1/companions/accept", newcomer, map[string]any{"code": code})
	assert.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	assert.Equal(t, "too_many_requests", r.Body["error_code"], "the throttles' generic answer")
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companions WHERE owner_id = ? AND status = 'invited'`, e.ids["Sara"]))

	e.mr.FastForward(CompanionAcceptBreakerCooldown + time.Second)
	r = e.do(fiber.MethodPost, "/api/v1/companions/accept", newcomer, map[string]any{"code": code})
	assert.Equal(t, fiber.StatusOK, r.Status, r.Raw)
}

// CMP-L1: companion reads are audited once per section per 15 minutes; the owner's trail pages back and filters by
// action, so a write stays reachable.
func TestCompanionSecurity_AuditCoalescedAndPaged(t *testing.T) {
	e := newCompanionEnv(t)
	sara, ali, linkID := securityPair(t, e)
	saraID, aliID := e.ids["Sara"], e.ids["Ali"]
	reads := func(section string) int {
		return e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND actor_id = ? AND action = 'read' AND section = ?`,
			saraID, aliID, section)
	}
	for range 3 {
		require.Equal(t, fiber.StatusOK, e.home(ali).Status)
	}
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/companions/links/%d/sections/cycle", linkID), ali, nil).Status)
	assert.Equal(t, 1, reads("cycle"), "one read row per section per window")
	assert.Equal(t, 1, reads("meds"))
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/companion/home", ali, nil, "2026-09-23T10:16:00+03:30").Status)
	assert.Equal(t, 2, reads("cycle"), "a new row after 15 minutes")

	r := e.do(fiber.MethodPost, "/api/v1/care/medications", ali, medBody(map[string]any{"for_user_id": saraID}))
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	for range 3 {
		require.Equal(t, fiber.StatusOK, e.home(ali).Status)
	}

	r = e.do(fiber.MethodGet, "/api/v1/companions/audit?action=write", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	require.Len(t, r.list(), 1)
	assert.Equal(t, "write", r.list()[0].(map[string]any)["action"])
	assert.Equal(t, "meds", r.list()[0].(map[string]any)["section"])

	all := e.do(fiber.MethodGet, "/api/v1/companions/audit?limit=200", sara, nil).list()
	require.Greater(t, len(all), 3)
	page := e.do(fiber.MethodGet, "/api/v1/companions/audit?limit=2", sara, nil).list()
	require.Len(t, page, 2)
	last := idOf(page[1].(map[string]any)["id"])
	next := e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/companions/audit?limit=2&before_id=%d", last), sara, nil).list()
	require.NotEmpty(t, next)
	assert.Equal(t, idOf(all[2].(map[string]any)["id"]), idOf(next[0].(map[string]any)["id"]), "the page after the cursor")
	for _, x := range next {
		assert.Less(t, idOf(x.(map[string]any)["id"]), last)
	}

	for _, bad := range []string{"action=delete", "before_id=0", "before_id=abc"} {
		r := e.do(fiber.MethodGet, "/api/v1/companions/audit?"+bad, sara, nil)
		assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, bad)
	}
}

// CMP-L5: a companion cannot use the account name to put a phone number or a long message in the owner's inbox.
func TestCompanionSecurity_NoticeNameSanitized(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008201")
	scam := e.user("Ritme support: account blocked, call 0912 345 6789", "09120008202")
	_, code := e.invite(sara, map[string]any{"type": "partner"})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", scam, map[string]any{"code": code}).Status)
	var title string
	require.NoError(t, e.db.QueryRow(`SELECT title FROM user_notifications WHERE user_id = ? ORDER BY id DESC LIMIT 1`, e.ids["Sara"]).Scan(&title))
	assert.NotContains(t, title, "0912")
	assert.NotContains(t, title, "support")
	assert.Contains(t, title, "Your companion accepted")
}

// CMP-H1 over HTTP: a pregnant owner's partner without the pregnancy grant gets the neutral cycle view on the section
// route and on the companion home (general tips, no phase).
func TestCompanionSecurity_PregnantOwnerCycleNeutral(t *testing.T) {
	e := newCompanionEnv(t)
	_, ali, linkID := securityPair(t, e)
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-09-05', NOW(), NOW())`, e.ids["Sara"])
	require.NoError(t, err)

	r := e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/companions/links/%d/sections/cycle", linkID), ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	view := r.data()["data"].(map[string]any)
	assert.Equal(t, false, view["has_data"])
	assert.Nil(t, view["cycle_day"])
	assert.Nil(t, view["days_late"])
	assert.Nil(t, view["main_phase"])

	h := e.home(ali)
	require.Equal(t, fiber.StatusOK, h.Status, h.Raw)
	card := partnersOf(h)[0].(map[string]any)
	assert.Equal(t, false, card["cycle"].(map[string]any)["has_data"])
	assert.Nil(t, card["pregnancy"])
	assert.Equal(t, "general", card["phase"])
}
