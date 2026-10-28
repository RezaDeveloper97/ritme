package http

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
)

// CB-LOSS-01 through the production wiring: recording a loss stops every pregnancy surface — the owner's home,
// Today / week / calendar / day log / analysis (409 pregnancy_not_active), the alert badge, the daily messages (even a
// forced ?mode=pregnancy), the life-stage mode — and the companion's pregnancy views; the companion holding the
// pregnancy grant gets exactly one one-line notice, a companion without it gets nothing.
func TestLoss_StopsPregnancyEverywhere(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120005001")
	ali := e.user("Ali", "09120005002")   // companion with the pregnancy grant
	reza := e.user("Reza", "09120005003") // companion with cycle only
	saraID := e.ids["Sara"]

	e.step(sara, "gender", map[string]any{"gender": "female"})
	e.step(sara, "goal", map[string]any{"goal": "cycle"})
	e.step(sara, "cycle", map[string]any{"last_period_start": "2026-08-02", "period_duration": 5, "cycle_duration": 28})
	r := e.do(fiber.MethodPost, "/api/v1/pregnancy/onboarding", sara, map[string]any{"age_source": "lmp", "lmp_date": "2026-08-02"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	for _, tok := range []string{ali, reza} {
		e.step(tok, "gender", map[string]any{"gender": "male"})
	}
	_, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"pregnancy": "view", "cycle": "view"}})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code}).Status)
	_, code = e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"cycle": "view"}})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", reza, map[string]any{"code": code}).Status)
	aliLink := idOfLink(t, e, ali)

	// Before: pregnancy everywhere.
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/pregnancy/v2/today", sara, nil).Status)
	assert.Equal(t, "pregnancy", e.do(fiber.MethodGet, "/api/v1/messages/mode", sara, nil).data()["mode"])
	assert.Equal(t, "pregnancy", e.do(fiber.MethodGet, "/api/v1/home", sara, nil).data()["mode"])
	sec := e.do(fiber.MethodGet, "/api/v1/companions/links/"+strconv.FormatUint(aliLink, 10)+"/sections/pregnancy", ali, nil)
	require.Equal(t, fiber.StatusOK, sec.Status, sec.Raw)
	assert.Equal(t, map[string]any{"is_active": true}, pick(sec.data()["data"], "is_active"))

	r = e.do(fiber.MethodPost, "/api/v1/loss", sara, map[string]any{"type": "early_miscarriage", "notify_companion": true})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["loss"].(map[string]any)["companion_notified"])

	// After: the owner's pregnancy surfaces are gone.
	for _, p := range []string{"/api/v1/pregnancy/v2/today", "/api/v1/pregnancy/v2/weeks/8", "/api/v1/pregnancy/v2/calendar",
		"/api/v1/pregnancy/v2/days/2026-09-23", "/api/v1/analysis/pregnancy"} {
		r := e.do(fiber.MethodGet, p, sara, nil)
		assert.Equal(t, fiber.StatusConflict, r.Status, p+" "+r.Raw)
		assert.Equal(t, "pregnancy_not_active", r.Body["error_code"], p)
	}
	assert.Equal(t, false, e.do(fiber.MethodGet, "/api/v1/pregnancy/status", sara, nil).data()["is_active"])
	assert.EqualValues(t, 0, e.do(fiber.MethodGet, "/api/v1/pregnancy/alerts/summary", sara, nil).data()["counts"].(map[string]any)["total"])
	assert.Equal(t, "cycle", e.do(fiber.MethodGet, "/api/v1/home", sara, nil).data()["mode"])
	assert.Equal(t, "cycle", e.do(fiber.MethodGet, "/api/v1/messages/mode", sara, nil).data()["mode"])
	assert.Equal(t, "cycle", e.do(fiber.MethodGet, "/api/v1/profile/life-stage", sara, nil).data()["mode"])
	daily := e.do(fiber.MethodGet, "/api/v1/messages/daily?mode=pregnancy", sara, nil)
	require.Equal(t, fiber.StatusOK, daily.Status, daily.Raw)
	assert.Equal(t, "cycle", daily.data()["mode"], "a forced pregnancy mode is ignored after a loss")

	// The companion's pregnancy view stops; the home leaves the pregnancy phase. T-M9-02 (D-133, FAM-H3): the loss
	// resets every pregnancy grant to none, so the section is now a 403 (it used to answer 200 {is_active:false}) and
	// the home carries no pregnancy block — a stale grant can never show the next pregnancy.
	sec = e.do(fiber.MethodGet, "/api/v1/companions/links/"+strconv.FormatUint(aliLink, 10)+"/sections/pregnancy", ali, nil)
	assert.Equal(t, fiber.StatusForbidden, sec.Status, sec.Raw)
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'pregnancy'`, aliLink))
	home := e.home(ali)
	require.Equal(t, fiber.StatusOK, home.Status, home.Raw)
	p := partnersOf(home)[0].(map[string]any)
	assert.NotEqual(t, "pregnancy", p["phase"])
	assert.Nil(t, p["pregnancy"], "no pregnancy block without the grant")

	// One line to the companion with the pregnancy grant, nothing to the other, nothing about type or date.
	aliID, rezaID := e.ids["Ali"], e.ids["Reza"]
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND type = 'companion'
		AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) = 'pregnancy_not_continuing' AND body IS NULL`, aliID))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ?
		AND JSON_UNQUOTE(JSON_EXTRACT(title, '$.en')) = 'Sara let you know that the pregnancy is not continuing.'`, aliID))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, rezaID))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND (title LIKE '%miscarriage%' OR data LIKE '%loss%')`, aliID))
	// No loss notice to herself; her only pregnancy notice is the T-M9-02 sharing reset («اشتراک بارداری … متوقف شد»).
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND data LIKE '%pregnancy%'
		AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) <> ?`, saraID, companion.EventPregnancySharingReset), "no notice to herself")
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) = ?`,
		saraID, companion.EventPregnancySharingReset), "the grant reset is told to her once")

	// Correcting the record the same day does not notify again.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/loss", sara, map[string]any{"notify_companion": true}).Status)
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, aliID))

	// The companion cannot read her loss (no route exposes it to him): his own /loss is empty.
	assert.Nil(t, e.do(fiber.MethodGet, "/api/v1/loss", ali, nil).data()["loss"])

	// Next step ttc: TTC mode, still no pregnancy content.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, "/api/v1/loss/next-step", sara, map[string]any{"choice": "ttc"}).Status)
	assert.Equal(t, "ttc", e.do(fiber.MethodGet, "/api/v1/profile/life-stage", sara, nil).data()["mode"])
	assert.Equal(t, fiber.StatusConflict, e.do(fiber.MethodGet, "/api/v1/pregnancy/v2/today", sara, nil).Status)
}

// Security audit of CB-LOSS-01: the loss follow-up appointments are private — a companion holding appointments
// (view or edit) sees neither of them in the section view, the companion home, or «ثبت برای …» show / update, while
// her ordinary appointments stay shared. The one-line notice goes when the link is revoked, when she erases the loss
// and when her account is deleted.
func TestLoss_PrivacyOfFollowupsAndNotices(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120005101")
	ali := e.user("Ali", "09120005102")
	reza := e.user("Reza", "09120005103")
	saraID := e.ids["Sara"]
	e.step(sara, "gender", map[string]any{"gender": "female"})
	r := e.do(fiber.MethodPost, "/api/v1/pregnancy/onboarding", sara, map[string]any{"age_source": "lmp", "lmp_date": "2026-08-02"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	for _, tok := range []string{ali, reza} {
		e.step(tok, "gender", map[string]any{"gender": "male"})
	}
	_, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"appointments": "edit", "pregnancy": "view"}})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code}).Status)
	_, code = e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"appointments": "view", "pregnancy": "view"}})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", reza, map[string]any{"code": code}).Status)

	// An ordinary appointment of hers, then the loss with both follow-ups.
	r = e.do(fiber.MethodPost, "/api/v1/care/appointments", sara, map[string]any{"kind": "in_person", "topic": "other", "scheduled_at": "2026-10-02 10:00"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	r = e.do(fiber.MethodPost, "/api/v1/loss", sara, map[string]any{"notify_companion": true})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	r = e.do(fiber.MethodPut, "/api/v1/loss/followup", sara, map[string]any{"beta_next_on": "2026-09-30", "visit_at": "2026-10-07 11:30"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	followup := r.data()["followup"].(map[string]any)
	betaID := idOf(followup["beta"].(map[string]any)["appointment"].(map[string]any)["id"])
	visitID := idOf(followup["visit"].(map[string]any)["appointment"].(map[string]any)["id"])
	// She sees all three in her own list.
	assert.Len(t, e.do(fiber.MethodGet, "/api/v1/care/appointments", sara, nil).list(), 3)

	// The loss ends the pregnancy mode: every owner-data grant pauses until she reviews (T-M11-05, F-23) — the
	// members see nothing, then she keeps their appointments access through the real API.
	for tok, level := range map[string]string{ali: "edit", reza: "view"} {
		link := idOfLink(t, e, tok)
		sec := e.do(fiber.MethodGet, "/api/v1/companions/links/"+strconv.FormatUint(link, 10)+"/sections/appointments", tok, nil)
		require.Equal(t, fiber.StatusForbidden, sec.Status, "paused after the mode change: "+sec.Raw)
		assert.Nil(t, sec.Body["data"])
		r := e.do(fiber.MethodPut, "/api/v1/companions/"+strconv.FormatUint(link, 10)+"/grants", sara,
			map[string]any{"grants": map[string]any{"appointments": level}})
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	}
	for _, tok := range []string{ali, reza} {
		link := idOfLink(t, e, tok)
		sec := e.do(fiber.MethodGet, "/api/v1/companions/links/"+strconv.FormatUint(link, 10)+"/sections/appointments", tok, nil)
		require.Equal(t, fiber.StatusOK, sec.Status, sec.Raw)
		appts, _ := sec.data()["data"].([]any)
		require.Len(t, appts, 1, sec.Raw)
		// T-M9-02 (D-133, FAM-M2): the member projection never carries the owner's topic (or notes / prep / result).
		assert.NotContains(t, appts[0].(map[string]any), "topic")
		home := e.home(tok)
		require.Equal(t, fiber.StatusOK, home.Status, home.Raw)
		hp, _ := partnersOf(home)[0].(map[string]any)["appointments"].([]any)
		assert.Len(t, hp, 1, "companion home: only the ordinary appointment")
		for _, id := range []uint64{betaID, visitID} {
			r := e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/appointments/%d?for_user_id=%d", id, saraID), tok, nil)
			assert.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)
		}
	}
	for _, id := range []uint64{betaID, visitID} {
		r := e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/care/appointments/%d", id), ali, map[string]any{"for_user_id": saraID, "title": "x"})
		assert.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)
	}
	// Her own edit keeps the follow-up private.
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/care/appointments/%d", visitID), sara, map[string]any{"location": "Clinic"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM reminders WHERE id = ? AND JSON_EXTRACT(meta, '$.private') = true`, visitID))
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/appointments/%d?for_user_id=%d", visitID, saraID), ali, nil).Status)

	notices := func(id uint64) int {
		return e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) = 'pregnancy_not_continuing'`, id)
	}
	aliID, rezaID := e.ids["Ali"], e.ids["Reza"]
	require.Equal(t, 1, notices(aliID))
	require.Equal(t, 1, notices(rezaID))

	// Revoking Reza's link removes his notice only.
	r = e.do(fiber.MethodDelete, "/api/v1/companions/links/"+strconv.FormatUint(idOfLink(t, e, reza), 10), reza, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, 0, notices(rezaID))
	assert.Equal(t, 1, notices(aliID))

	// Erasing the loss removes Ali's notice. T-M9-02 (D-133, FAM-H3): the first loss reset his pregnancy grant to
	// none, so a second loss without a fresh grant sends no new notice (it used to notify again). A fresh grant brings
	// the notice back, and her account deletion removes it.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, "/api/v1/loss", sara, nil).Status)
	assert.Equal(t, 0, notices(aliID))
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'pregnancy'`, idOfLink(t, e, ali)))
	require.Equal(t, fiber.StatusCreated, e.do(fiber.MethodPost, "/api/v1/loss", sara, map[string]any{"notify_companion": true}).Status)
	assert.Equal(t, 0, notices(aliID), "no pregnancy grant → no notice")
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, "/api/v1/loss", sara, nil).Status)
	e.exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, 'pregnancy', 'view', NOW(), NOW())`, idOfLink(t, e, ali))
	require.Equal(t, fiber.StatusCreated, e.do(fiber.MethodPost, "/api/v1/loss", sara, map[string]any{"notify_companion": true}).Status)
	assert.Equal(t, 1, notices(aliID), "a fresh grant → the notice again")
	r = e.do(fiber.MethodDelete, "/api/v1/account", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, 0, notices(aliID))
}

// pick is the subset of m (a JSON object) with keys.
func pick(v any, keys ...string) map[string]any {
	m, _ := v.(map[string]any)
	out := map[string]any{}
	for _, k := range keys {
		out[k] = m[k]
	}
	return out
}

// idOfLink is the viewer's first companion link id.
func idOfLink(t *testing.T, e *companionEnv, token string) uint64 {
	t.Helper()
	r := e.do(fiber.MethodGet, "/api/v1/companions/links", token, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	links := r.list()
	if links == nil {
		if l, ok := r.data()["links"].([]any); ok {
			links = l
		}
	}
	require.NotEmpty(t, links, r.Raw)
	return idOf(links[0].(map[string]any)["id"])
}
