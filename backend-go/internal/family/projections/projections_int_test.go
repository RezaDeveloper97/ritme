package projections_test

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/family"
)

var allDomains = []string{"cycle_phase", "ttc_fertility", "pregnancy", "fetal_activity", "appointments", "meds", "vitals"}

// seedSentinels writes never-shared data for owner: private notes, mood, EPDS, sex, labs, chats, cycle history and
// length, TTC data — and free text on the records members may see a summary of.
func (e *env) seedSentinels(owner uint64) {
	e.t.Helper()
	entry := `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at)
		VALUES (?, '2026-09-22', ?, ?, ?, ?, NULL, ?, 'manual', '2026-09-22 09:00:00', '2026-09-22 09:00:00')`
	for _, r := range [][]any{
		{"mood", "moods", "SENTINEL_MOOD", "1", nil},
		{"note", "text", "", nil, "SENTINEL_PRIVATE_NOTE"},
		{"sex", "intercourse", "", "SENTINEL_SEX", nil},
		{"sex", "desire", "", "SENTINEL_DESIRE", nil},
		{"fertility", "lh", "", "SENTINEL_LH", nil},
		{"fertility", "bbt", "", nil, "SENTINEL_BBT"},
		{"discharge", "consistency", "", "SENTINEL_MUCUS", nil},
	} {
		e.exec(entry, append([]any{owner}, r...)...)
	}
	e.exec(`INSERT INTO epds_checks (user_id, kind, taken_on, answers, total, created_at, updated_at)
		VALUES (?, 'epds', '2026-09-20', '{"SENTINEL_EPDS":3}', 27, NOW(), NOW())`, owner)
	lab := e.exec(`INSERT INTO lab_reports (user_id, source, status, created_at, updated_at) VALUES (?, 'manual', 'confirmed', NOW(), NOW())`, owner)
	e.exec(`INSERT INTO lab_markers (lab_id, user_id, name, value_text, source, created_at, updated_at)
		VALUES (?, ?, 'SENTINEL_LAB', 'SENTINEL_LAB_VALUE', 'manual', NOW(), NOW())`, lab, owner)
	dep := e.exec(`INSERT INTO assistant_departments (code, name, system_prompt, created_at, updated_at)
		VALUES (CONCAT('d', ?), '{"en":"Dept"}', 'p', NOW(), NOW())`, owner)
	conv := e.exec(`INSERT INTO assistant_conversations (user_id, department_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, owner, dep)
	e.exec(`INSERT INTO assistant_messages (conversation_id, role, body, created_at, updated_at) VALUES (?, 'user', 'SENTINEL_CHAT', NOW(), NOW())`, conv)
	for _, d := range []string{"2026-05-23", "2026-06-20", "2026-07-18"} {
		e.exec(`INSERT INTO cycle_histories (user_id, period_start_date, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, owner, d)
	}
	e.exec(`INSERT INTO reminders (user_id, type, title, subtitle, notes, recurrence, recurrence_time, is_active, meta, created_at, updated_at)
		VALUES (?, 'medication', 'Iron', 'SENTINEL_SUBTITLE', 'SENTINEL_MED_NOTE', 'daily', '08:00:00', 1, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`,
		owner, `{"v":1,"dose":"SENTINEL_DOSE","unit":"SENTINEL_UNIT","form":"tablet","times":["08:00"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"ongoing","notify":true}`)
	e.exec(`INSERT INTO reminders (user_id, type, title, notes, scheduled_at, recurrence, is_active, meta, created_at, updated_at)
		VALUES (?, 'appointment', 'Checkup', 'SENTINEL_APPT_NOTE', '2026-09-30 10:00:00', 'none', 1, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`,
		owner, `{"v":1,"kind":"in_person","with":"Dr. Rahimi","specialty":"SENTINEL_SPECIALTY","topic":"ultrasound","location":"Clinic A","remind_before":"1d","add_to_calendar":false,"prep":[{"id":"p1","text":"SENTINEL_PREP","done":false}],"status":"scheduled","result_note":"SENTINEL_RESULT","booking_id":987654}`)
	e.exec(`INSERT INTO vital_readings (user_id, type, measured_at, systolic, diastolic, note, created_at, updated_at)
		VALUES (?, 'bp', '2026-09-22 08:00:00', 118, 76, 'SENTINEL_VITAL_NOTE', NOW(), NOW())`, owner)
}

// The companion home's tip `note` is admin copy, so only `"notes"` is a key sentinel; SENTINEL_PRIVATE_NOTE covers notes.
var sentinels = []string{"SENTINEL", "epds", "EPDS", `"mood"`, `"notes"`, `"topic"`, "ultrasound", `"prep"`,
	`"specialty"`, `"result_note"`, `"booking_id"`, "987654", `"cycle_day"`, `"cycle_length"`, `"days_late"`,
	`"days_to_period"`, `"predicted_next_period_start"`, `"confidence"`, `"history"`, "2026-05-23", "2026-06-20",
	"2026-07-18", `"lh"`, `"bbt"`, `"intercourse"`, `"symptoms"`}

// TestMember_NeverSharedNotReachable — SPEC §1.7: with every domain granted, no member endpoint (family and
// companion) carries a never-shared value or key, for a cycling and for a pregnant owner.
func TestMember_NeverSharedNotReachable(t *testing.T) {
	e := newEnv(t)
	cyc, _ := e.account("09120116001", "female", "cycle")
	preg, _ := e.account("09120116002", "female", "pregnancy")
	memberID, tok := e.account("09120116003", "male", "")
	e.cyclePeriod(cyc, "2026-09-10")
	e.pregnant(preg, "2026-06-01")
	all := map[string]string{"cycle_phase": "view", "pregnancy": "view", "appointments": "write", "fetal_activity": "view",
		"vitals": "view", "meds": "write", "symptoms": "view"}
	links := map[uint64]uint64{}
	for _, o := range []uint64{cyc, preg} {
		e.seedSentinels(o)
		links[o] = e.link(o, memberID, "partner", all)
	}

	paths := []string{"/api/v1/family/me", "/api/v1/companions/links", "/api/v1/companion/home"}
	for _, o := range []uint64{cyc, preg} {
		for _, d := range allDomains {
			paths = append(paths, domainPath(o, d))
		}
		for _, s := range []string{"cycle", "symptoms", "meds", "appointments", "pregnancy"} {
			paths = append(paths, sectionPath(links[o], s))
		}
	}
	served := 0
	for _, p := range paths {
		r := e.get(p, tok)
		require.Less(t, r.Status, 500, p+" "+r.Raw)
		if r.Status == fiber.StatusOK {
			served++
		}
		assert.Empty(t, contains(r.Raw, sentinels...), "%s (%d): %s", p, r.Status, r.Raw)
	}
	assert.GreaterOrEqual(t, served, 12, "most endpoints answered with data, so the walk proves something")
	assert.Equal(t, fiber.StatusNotFound, e.get(sectionPath(links[cyc], "symptoms"), tok).Status, "F-03: symptoms is gone")
}

// TestCyclePhase_NoNumbers — F-01: the member cycle phase is main_phase (+ period_soon for a partner) only.
func TestCyclePhase_NoNumbers(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116011", "female", "cycle")
	member, tok := e.account("09120116012", "male", "")
	e.cyclePeriod(owner, "2026-09-10")
	link := e.link(owner, member, "partner", map[string]string{"cycle_phase": "view"})
	for _, p := range []string{domainPath(owner, "cycle_phase"), sectionPath(link, "cycle")} {
		r := e.get(p, tok)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		assert.ElementsMatch(t, []string{"main_phase", "period_soon"}, keysOf(r.inner()), p)
		assert.Equal(t, family.CacheControlNoStore, r.Cache, p)
	}
}

// TestCyclePhase_PeriodSoonPartnerOnly — F-01: period_soon only for the partner role; a friend gets the phase alone.
func TestCyclePhase_PeriodSoonPartnerOnly(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116021", "female", "cycle")
	partner, ptok := e.account("09120116022", "male", "")
	friend, ftok := e.account("09120116023", "female", "cycle")
	e.cyclePeriod(owner, "2026-08-28") // 28 days → next 2026-09-25, two days after "today"
	e.link(owner, partner, "partner", map[string]string{"cycle_phase": "view"})
	e.link(owner, friend, "friend", map[string]string{"cycle_phase": "view"})

	r := e.get(domainPath(owner, "cycle_phase"), ptok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	in, _ := r.inner().(map[string]any)
	assert.Equal(t, true, in["period_soon"], r.Raw)

	r = e.get(domainPath(owner, "cycle_phase"), ftok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, []string{"main_phase"}, keysOf(r.inner()), "a friend sees the phase only")
}

// TestPregnancyProjection_ModeAndEpochOnly — the pregnancy domain answers only while the owner is pregnant and the
// grant carries her current mode epoch; a stale (paused) or revoked grant returns nothing.
func TestPregnancyProjection_ModeAndEpochOnly(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116031", "female", "pregnancy")
	member, tok := e.account("09120116032", "male", "")
	e.pregnant(owner, "2026-06-01")
	link := e.link(owner, member, "partner", map[string]string{"pregnancy": "view"})

	r := e.get(domainPath(owner, "pregnancy"), tok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.ElementsMatch(t, []string{"current_week", "trimester", "edd"}, keysOf(r.inner()))
	in, _ := r.inner().(map[string]any)
	assert.NotNil(t, in["current_week"])
	assert.NotNil(t, in["edd"])

	// A new epoch (a new pregnancy, or any mode change, F-23): the grant is paused → 403, no data.
	e.exec(`UPDATE family_owner_state SET mode_epoch = mode_epoch + 1 WHERE owner_id = ?`, owner)
	for _, p := range []string{domainPath(owner, "pregnancy"), sectionPath(link, "pregnancy")} {
		r = e.get(p, tok)
		assert.Equal(t, fiber.StatusForbidden, r.Status, p+" "+r.Raw)
		assert.Nil(t, r.Body["data"], p)
		assert.Empty(t, contains(r.Raw, "current_week", "edd", "trimester"), p)
	}
	// Re-stamped, but the owner is no longer pregnant: still nothing.
	e.exec(`UPDATE companion_grants g JOIN family_owner_state s ON s.owner_id = ? SET g.mode_epoch = s.mode_epoch WHERE g.companion_id = ?`, owner, link)
	assert.Equal(t, fiber.StatusOK, e.get(domainPath(owner, "pregnancy"), tok).Status)
	e.exec(`UPDATE pregnancy_profiles SET pregnancy_mode = 0 WHERE user_id = ?`, owner)
	e.exec(`UPDATE user_life_profiles SET life_mode = 'cycle' WHERE user_id = ?`, owner)
	r = e.get(domainPath(owner, "pregnancy"), tok)
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Nil(t, r.Body["data"])
	// Revoked: 404, no data, no existence hint.
	e.exec(`UPDATE companions SET status = 'revoked' WHERE id = ?`, link)
	r = e.get(domainPath(owner, "pregnancy"), tok)
	assert.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)
	assert.Nil(t, r.Body["data"])
	assert.Equal(t, family.CacheControlNoStore, r.Cache, "errors are no-store too")
}

// TestShopChecklist_StaleGrantDenied — the layette share is check() on `pregnancy`, epoch-bound: a grant from an
// earlier epoch (a new pregnancy) shows nothing.
func TestShopChecklist_StaleGrantDenied(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116041", "female", "pregnancy")
	member, tok := e.account("09120116042", "male", "")
	e.pregnant(owner, "2026-06-01")
	e.link(owner, member, "partner", map[string]string{"pregnancy": "view"})
	path := fmt.Sprintf("/api/v1/shop/checklist?for_user_id=%d", owner)

	r := e.get(path, tok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotNil(t, r.data()["due_date"])

	e.exec(`UPDATE family_owner_state SET mode_epoch = mode_epoch + 1 WHERE owner_id = ?`, owner)
	r = e.get(path, tok)
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Empty(t, contains(r.Raw, "due_date", "weeks_left"))
}

// TestVitalsProjection_LatestOnly — the latest weight, BP and glucose, value + category + date; no history, no note.
func TestVitalsProjection_LatestOnly(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116051", "female", "cycle")
	member, tok := e.account("09120116052", "male", "")
	e.link(owner, member, "partner", map[string]string{"vitals": "view"})
	for _, r := range [][]any{{"2026-09-10 08:00:00", 150, 95, "old"}, {"2026-09-21 08:00:00", 118, 76, "SENTINEL_VITAL_NOTE"}} {
		e.exec(`INSERT INTO vital_readings (user_id, type, measured_at, systolic, diastolic, note, created_at, updated_at) VALUES (?, 'bp', ?, ?, ?, ?, NOW(), NOW())`,
			owner, r[0], r[1], r[2], r[3])
	}
	e.exec(`INSERT INTO vital_readings (user_id, type, measured_at, glucose_mg_dl, glucose_unit, context, created_at, updated_at) VALUES (?, 'glucose', '2026-09-20 07:00:00', 90.0, 'mmol_l', 'fasting', NOW(), NOW())`, owner)
	for _, w := range [][]any{{"2026-09-01", "61.00"}, {"2026-09-19", "60.40"}} {
		e.exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_num, source, created_at, updated_at) VALUES (?, ?, 'measurements', 'weight', '', ?, 'manual', NOW(), NOW())`,
			owner, w[0], w[1])
	}

	r := e.get(domainPath(owner, "vitals"), tok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	in, _ := r.inner().(map[string]any)
	assert.ElementsMatch(t, []string{"weight", "bp", "glucose"}, keysOf(in))
	bp, _ := in["bp"].(map[string]any)
	assert.Equal(t, map[string]any{"systolic": float64(118), "diastolic": float64(76), "category": bp["category"], "date": "2026-09-21"}, bp)
	assert.NotNil(t, bp["category"])
	weight, _ := in["weight"].(map[string]any)
	assert.Equal(t, 60.4, weight["value"])
	assert.Equal(t, "2026-09-19", weight["date"])
	glucose, _ := in["glucose"].(map[string]any)
	assert.Equal(t, "mmol_l", glucose["unit"], "the owner's unit")
	assert.Equal(t, 5.0, glucose["value"])
	assert.Empty(t, contains(r.Raw, "150", "61", "SENTINEL", `"note"`, "fasting", "measured_at", "08:00"))
}

// TestFetalActivity_PregnancyOnly — kick and contraction summaries for a pregnant owner; nothing in any other mode.
func TestFetalActivity_PregnancyOnly(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116061", "female", "pregnancy")
	member, tok := e.account("09120116062", "male", "")
	e.pregnant(owner, "2026-03-01")
	e.link(owner, member, "partner", map[string]string{"fetal_activity": "view"})
	e.exec(`INSERT INTO pregnancy_kick_sessions (user_id, started_at, ended_at, kicks, tenth_kick_at, last_kick_at, created_at, updated_at)
		VALUES (?, '2026-09-23 08:00:00', '2026-09-23 08:30:00', 10, '2026-09-23 08:25:00', '2026-09-23 08:25:00', NOW(), NOW())`, owner)
	sess := e.exec(`INSERT INTO pregnancy_contraction_sessions (user_id, started_at, ended_at, created_at, updated_at)
		VALUES (?, '2026-09-22 20:00:00', '2026-09-22 20:30:00', NOW(), NOW())`, owner)
	for _, c := range [][]string{{"2026-09-22 20:00:00", "2026-09-22 20:01:00"}, {"2026-09-22 20:10:00", "2026-09-22 20:11:00"}} {
		e.exec(`INSERT INTO pregnancy_contractions (session_id, user_id, started_at, ended_at, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())`,
			sess, owner, c[0], c[1])
	}

	r := e.get(domainPath(owner, "fetal_activity"), tok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	in, _ := r.inner().(map[string]any)
	kicks, _ := in["kicks_today"].(map[string]any)
	assert.Equal(t, map[string]any{"date": "2026-09-23", "sessions": float64(1), "status": "completed"}, kicks)
	cs, _ := in["last_contraction_session"].(map[string]any)
	assert.ElementsMatch(t, []string{"count", "last_at", "in_progress", "five_one_one_met"}, keysOf(cs))
	assert.Equal(t, float64(2), cs["count"])
	assert.Equal(t, false, cs["five_one_one_met"])

	e.exec(`UPDATE pregnancy_profiles SET pregnancy_mode = 0 WHERE user_id = ?`, owner)
	e.exec(`UPDATE user_life_profiles SET life_mode = 'postpartum' WHERE user_id = ?`, owner)
	r = e.get(domainPath(owner, "fetal_activity"), tok)
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Nil(t, r.Body["data"])
}

// TestMeds_DoseStatusOnlyAtNudge — today's dose status needs the meds grant at nudge or above.
func TestMeds_DoseStatusOnlyAtNudge(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116071", "female", "cycle")
	member, tok := e.account("09120116072", "male", "")
	link := e.link(owner, member, "partner", map[string]string{"meds": "view"})
	e.exec(`INSERT INTO reminders (user_id, type, title, recurrence, recurrence_time, is_active, meta, created_at, updated_at)
		VALUES (?, 'medication', 'Iron', 'daily', '08:00:00', 1, ?, NOW(), NOW())`,
		owner, `{"v":1,"dose":"1","unit":"tablet","form":"tablet","times":["08:00","21:00"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"ongoing","notify":true}`)

	r := e.get(domainPath(owner, "meds"), tok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	list, _ := r.inner().([]any)
	require.Len(t, list, 1)
	assert.NotContains(t, keysOf(list[0]), "today", "view level: no dose status")

	e.exec(`UPDATE companion_grants SET level = 'nudge' WHERE companion_id = ? AND section = 'meds'`, link)
	r = e.get(domainPath(owner, "meds"), tok)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	list, _ = r.inner().([]any)
	require.Len(t, list, 1)
	today, _ := list[0].(map[string]any)["today"].([]any)
	assert.Equal(t, []any{
		map[string]any{"time": "08:00", "status": "missed"},
		map[string]any{"time": "21:00", "status": "pending"},
	}, today)
}

// TestMemberLinkView_EffectiveDomainsOnly — the member's link views list what Decide gives right now: no `none`
// rows, no symptoms, nothing paused by a mode change.
func TestMemberLinkView_EffectiveDomainsOnly(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116081", "female", "cycle")
	member, tok := e.account("09120116082", "male", "")
	e.cyclePeriod(owner, "2026-09-10")
	e.link(owner, member, "partner", map[string]string{"cycle_phase": "view", "meds": "view", "symptoms": "view", "pregnancy": "view"})

	links := e.get("/api/v1/companions/links", tok)
	require.Equal(t, fiber.StatusOK, links.Status, links.Raw)
	list, _ := links.Body["data"].([]any)
	require.Len(t, list, 1)
	assert.Equal(t, map[string]any{"cycle": "view", "meds": "view"}, list[0].(map[string]any)["grants"],
		"pregnancy is out of the owner's mode, symptoms is gone")
	me := e.get("/api/v1/family/me", tok)
	require.Equal(t, fiber.StatusOK, me.Status, me.Raw)
	assert.Equal(t, family.CacheControlNoStore, me.Cache)
	asMember, _ := me.data()["as_member"].([]any)
	require.Len(t, asMember, 1)
	assert.ElementsMatch(t, []any{
		map[string]any{"domain": "cycle_phase", "level": "view"}, map[string]any{"domain": "meds", "level": "view"},
	}, asMember[0].(map[string]any)["domains"])

	// The owner becomes pregnant: a new mode epoch pauses every owner-data grant (F-23) until she reviews.
	e.pregnant(owner, "2026-09-01")
	links = e.get("/api/v1/companions/links", tok)
	list, _ = links.Body["data"].([]any)
	require.Len(t, list, 1)
	assert.Equal(t, map[string]any{}, list[0].(map[string]any)["grants"], links.Raw)
	me = e.get("/api/v1/family/me", tok)
	asMember, _ = me.data()["as_member"].([]any)
	require.Len(t, asMember, 1)
	assert.Equal(t, []any{}, asMember[0].(map[string]any)["domains"])
	for _, d := range allDomains {
		r := e.get(domainPath(owner, d), tok)
		assert.Equal(t, fiber.StatusForbidden, r.Status, d+" "+r.Raw)
		assert.Nil(t, r.Body["data"], d)
	}
}

// Unknown owner, domain or a stranger: 404 with no data; no token: 401.
func TestFamilyDomain_NotFoundAndAuth(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116091", "female", "cycle")
	_, tok := e.account("09120116092", "male", "")
	for _, p := range []string{domainPath(owner, "cycle_phase"), domainPath(owner, "symptoms"), domainPath(owner, "urgent_alert"),
		domainPath(owner, "bank"), "/api/v1/family/owners/abc/domains/meds", domainPath(999999, "meds")} {
		r := e.get(p, tok)
		assert.Equal(t, fiber.StatusNotFound, r.Status, p+" "+r.Raw)
		assert.Nil(t, r.Body["data"], p)
		assert.Equal(t, family.CacheControlNoStore, r.Cache, p)
	}
	r := e.get(domainPath(owner, "cycle_phase"), "")
	assert.Equal(t, fiber.StatusUnauthorized, r.Status)
	assert.Equal(t, family.CacheControlNoStore, r.Cache)
}

// TestFertileLabel_HiddenUnderHormonalContraception — F-02 through the production wiring (CYCLE_ENGINE v1 here):
// cycle day 10 reads fertile, and follicular once the owner tracks a hormonal method.
func TestFertileLabel_HiddenUnderHormonalContraception(t *testing.T) {
	e := newEnv(t)
	owner, _ := e.account("09120116101", "female", "cycle")
	member, tok := e.account("09120116102", "male", "")
	e.cyclePeriod(owner, "2026-09-14")
	link := e.link(owner, member, "partner", map[string]string{"cycle_phase": "view"})
	phase := func(p string) any {
		r := e.get(p, tok)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		in, _ := r.inner().(map[string]any)
		return in["main_phase"]
	}
	assert.Equal(t, "fertile", phase(domainPath(owner, "cycle_phase")))
	e.exec(`INSERT INTO contraception_methods (user_id, method, created_at, updated_at) VALUES (?, 'combined_pill', NOW(), NOW())`, owner)
	assert.Equal(t, "follicular", phase(domainPath(owner, "cycle_phase")))
	assert.Equal(t, "follicular", phase(sectionPath(link, "cycle")))
}
