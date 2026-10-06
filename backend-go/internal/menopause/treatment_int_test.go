package menopause_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/resources/translations"
)

// now = Thursday 2026-10-01; its week is Saturday 2026-09-26 … Friday 2026-10-02.

func items(r response, kind string) []any {
	l, _ := r.data()["items"].(map[string]any)[kind].([]any)
	return l
}

func first(r response, kind string) map[string]any {
	l := items(r, kind)
	if len(l) == 0 {
		return nil
	}
	return l[0].(map[string]any)
}

func (e *env) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

func TestTreatmentFlow(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000101")

	r := e.do(t, http.MethodGet, "/api/v1/menopause/treatment", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-09-26", r.data()["week"].(map[string]any)["from"])
	assert.Empty(t, items(r, "hrt"))
	assert.Nil(t, r.data()["review"])
	assert.NotEmpty(t, r.data()["tips"], "treatment tips from meno_tips")

	// HRT: a care medication reminder at the morning slot; the review is suggested 3 months after the start
	r = e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tok, "fa",
		`{"kind":"hrt","name":"استروژن (ژل پوستی)","dose":"۱ پمپ","schedule":"morning","started_on":"2026-08-01"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "درمان ذخیره شد", r.body["message"])
	hrt := first(r, "hrt")
	require.NotNil(t, hrt)
	hrtID := num(hrt["id"])
	rem := hrt["reminder"].(map[string]any)
	assert.Equal(t, "08:00", rem["time"])
	assert.Equal(t, true, rem["notify"])
	review := r.data()["review"].(map[string]any)
	assert.Equal(t, "2026-11-01", review["on"])
	assert.Equal(t, true, review["suggested"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'medication' AND title = 'استروژن (ژل پوستی)'`, uid))

	// supplement with remind off; lifestyle minutes goal (no reminder)
	r = e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tok, "en",
		`{"kind":"supplement","name":"Calcium + vitamin D","schedule":"noon","remind":false}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	sup := first(r, "supplement")
	supID := num(sup["id"])
	assert.Equal(t, false, sup["reminder"].(map[string]any)["notify"])
	r = e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tok, "en",
		`{"kind":"lifestyle","name":"Brisk walk","weekly_goal":150,"goal_unit":"minutes","started_on":"2026-09-01"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	walk := first(r, "lifestyle")
	walkID := num(walk["id"])
	assert.Nil(t, walk["reminder"])
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM reminders WHERE user_id = ?`, uid))

	// intakes: today + yesterday for HRT (ticks the care dose), 30 minutes walk
	for _, d := range []string{"2026-10-01", "2026-09-30"} {
		r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/%s", hrtID, d), tok, "en", `{}`)
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/2026-10-01", walkID), tok, "en", `{"amount":30}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	hrt = first(r, "hrt")
	assert.Equal(t, true, hrt["taken_today"])
	assert.Equal(t, 2, num(hrt["days_taken"]))
	assert.Equal(t, 6, num(hrt["days"]), "Saturday … Thursday")
	assert.Equal(t, 33, num(hrt["adherence_pct"]))
	week := hrt["week"].([]any)
	require.Len(t, week, 7)
	assert.Equal(t, true, week[5].(map[string]any)["taken"])
	assert.Equal(t, false, week[6].(map[string]any)["taken"])
	goal := first(r, "lifestyle")["goal"].(map[string]any)
	assert.Equal(t, 30, num(goal["amount"]))
	assert.Equal(t, 150, num(goal["target"]))
	assert.Equal(t, false, goal["done"])
	assert.Equal(t, 2, e.count(t, `SELECT COUNT(*) FROM reminder_intakes WHERE user_id = ? AND slot = '08:00'`, uid))

	// a tick in /care counts here too
	var supReminder uint64
	require.NoError(t, e.db.QueryRow(`SELECT reminder_id FROM treatment_items WHERE id = ?`, supID).Scan(&supReminder))
	e.exec(t, `INSERT INTO reminder_intakes (user_id, reminder_id, intake_date, slot, taken_at, created_at, updated_at)
		VALUES (?, ?, '2026-10-01', '13:00', '2026-10-01 13:05:00', '2026-10-01 13:05:00', '2026-10-01 13:05:00')`, uid, supReminder)
	r = e.do(t, http.MethodGet, "/api/v1/menopause/treatment", tok, "en", "")
	assert.Equal(t, true, first(r, "supplement")["taken_today"])

	// unlog removes both
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/2026-09-30", hrtID), tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-09-30", r.data()["date"], "the screen of the day touched")
	assert.Equal(t, 0, num(first(r, "hrt")["days_taken"]), "Saturday … Wednesday")
	assert.Equal(t, true, first(r, "hrt")["week"].([]any)[5].(map[string]any)["taken"])
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM reminder_intakes WHERE user_id = ? AND slot = '08:00'`, uid))

	// validation: minutes need an amount; future / too old / before start
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/2026-10-01", walkID), tok, "en", `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "amount")
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/2026-10-02", hrtID), tok, "en", `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "date")
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/2026-08-15", hrtID), tok, "en", `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/2026-09-05", walkID), tok, "en", `{"amount":20}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/bad", hrtID), tok, "en", `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tok, "en", `{"kind":"hrt","name":"x"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "schedule")
	r = e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tok, "en", `{"kind":"lifestyle","name":"x","weekly_goal":30,"goal_unit":"sessions"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "weekly_goal")
	r = e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tok, "en", `{"kind":"pill","name":"x","started_on":"2026-09-10","stopped_on":"2026-09-01"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "kind")

	// side effects of a day as a set
	r = e.do(t, http.MethodPut, "/api/v1/menopause/treatment/side-effects/2026-10-01", tok, "en",
		fmt.Sprintf(`{"codes":["spotting","breast_tenderness"],"treatment_item_id":%d}`, hrtID))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"breast_tenderness", "spotting"}, r.data()["side_effects"].(map[string]any)["today"])
	r = e.do(t, http.MethodPut, "/api/v1/menopause/treatment/side-effects/2026-10-01", tok, "en", `{"codes":["headache"]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"headache"}, r.data()["side_effects"].(map[string]any)["today"])
	r = e.do(t, http.MethodPut, "/api/v1/menopause/treatment/side-effects/2026-10-01", tok, "en", `{"codes":["rash"]}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)

	// stop HRT with a review date: it moves to «stopped», its care reminder is switched off
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d", hrtID), tok, "en",
		`{"name":"Estrogen gel","schedule":"morning","started_on":"2026-08-01","review_on":"2026-11-20","stopped_on":"2026-10-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Empty(t, items(r, "hrt"))
	stopped := r.data()["stopped"].([]any)
	require.Len(t, stopped, 1)
	assert.Equal(t, "Estrogen gel", stopped[0].(map[string]any)["name"])
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM reminders r JOIN treatment_items i ON i.reminder_id = r.id WHERE i.id = ? AND r.is_active = 1`, hrtID))

	// deleting an item deletes its care reminder; deleting the reminder in /care leaves the item (re-created on edit)
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/menopause/treatment/items/%d", supID), tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM reminders WHERE id = ?`, supReminder))
	e.exec(t, `DELETE FROM reminders WHERE id = (SELECT reminder_id FROM treatment_items WHERE id = ?)`, hrtID)
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d", hrtID), tok, "en",
		`{"name":"Estrogen gel","schedule":"evening","started_on":"2026-08-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "18:00", first(r, "hrt")["reminder"].(map[string]any)["time"])
	r = e.do(t, http.MethodDelete, "/api/v1/menopause/treatment/items/999999", tok, "en", "")
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)

	// the home's treatment card reads the same intakes
	r = e.do(t, http.MethodGet, "/api/v1/menopause/today", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
}

func TestTreatmentUserIsolation(t *testing.T) {
	e := setup(t)
	_, tokA := e.user(t, "09120000102")
	_, tokB := e.user(t, "09120000103")

	r := e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tokA, "en",
		`{"kind":"hrt","name":"Estrogen","schedule":"morning","started_on":"2026-09-01"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	id := num(first(r, "hrt")["id"])
	e.do(t, http.MethodPut, "/api/v1/menopause/treatment/side-effects/2026-10-01", tokA, "en", `{"codes":["headache"]}`)

	for _, c := range []struct{ method, path, body string }{
		{http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d", id), `{"name":"x","schedule":"night"}`},
		{http.MethodDelete, fmt.Sprintf("/api/v1/menopause/treatment/items/%d", id), ""},
		{http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/2026-10-01", id), `{}`},
		{http.MethodDelete, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/2026-10-01", id), ""},
	} {
		r = e.do(t, c.method, c.path, tokB, "en", c.body)
		assert.Equal(t, http.StatusNotFound, r.status, "%s %s: %s", c.method, c.path, r.raw)
	}
	r = e.do(t, http.MethodPut, "/api/v1/menopause/treatment/side-effects/2026-10-01", tokB, "en",
		fmt.Sprintf(`{"codes":["headache"],"treatment_item_id":%d}`, id))
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "treatment_item_id")

	r = e.do(t, http.MethodGet, "/api/v1/menopause/treatment", tokB, "en", "")
	assert.Empty(t, items(r, "hrt"))
	assert.Equal(t, []any{}, r.data()["side_effects"].(map[string]any)["today"])
	r = e.do(t, http.MethodGet, "/api/v1/menopause/report", tokB, "en", "")
	assert.Equal(t, true, r.data()["empty"])

	// A's item is untouched
	r = e.do(t, http.MethodGet, "/api/v1/menopause/treatment", tokA, "en", "")
	assert.Equal(t, "Estrogen", first(r, "hrt")["name"])
	assert.Equal(t, false, first(r, "hrt")["taken_today"])
	assert.Equal(t, []any{"headache"}, r.data()["side_effects"].(map[string]any)["today"])
}

// seedReport gives uid a month of menopause data: stage meno, flashes, logs, two scores, BP, HRT with intakes.
func (e *env) seedReport(t *testing.T, uid uint64, tok string) int {
	t.Helper()
	e.exec(t, `INSERT INTO user_life_profiles (user_id, life_mode, gender, menopause_stage, menopause_last_period, created_at, updated_at)
		VALUES (?, 'menopause', 'female', 'meno', '2025-08-01', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	for _, at := range []string{"2026-09-28 03:00:00", "2026-09-28 11:00:00", "2026-09-29 12:00:00"} {
		e.exec(t, `INSERT INTO hot_flashes (user_id, started_at, duration_s, night, sweat, created_at, updated_at)
			VALUES (?, ?, 120, ?, ?, ?, ?)`, uid, at, at[11:13] == "03", at[11:13] == "03", at, at)
	}
	e.logEntry(t, uid, "2026-09-28", "symptoms", "general", "insomnia", "moderate")
	e.logEntry(t, uid, "2026-09-28", "sleep", "duration", "", "3_6")
	e.logEntry(t, uid, "2026-09-30", "sleep", "duration", "", "6_9")
	e.logEntry(t, uid, "2026-09-30", "urogenital", "symptoms", "vaginal_dryness", "mild")
	e.logEntry(t, uid, "2026-09-24", "bleeding", "presence", "", "spotting")
	e.logEntry(t, uid, "2026-09-25", "bleeding", "presence", "", "spotting")
	e.exec(t, `INSERT INTO menopause_scores (user_id, month, answers, total, somatic, psychological, urogenital, created_at, updated_at)
		VALUES (?, '2026-08-23', '{}', 22, 10, 7, 5, '2026-09-01 09:00:00', '2026-09-01 09:00:00'),
		       (?, '2026-09-23', '{}', 14, 7, 4, 3, '2026-09-30 09:00:00', '2026-09-30 09:00:00')`, uid, uid)
	e.exec(t, `INSERT INTO vital_readings (user_id, type, measured_at, systolic, diastolic, created_at, updated_at)
		VALUES (?, 'bp', '2026-09-27 08:00:00', 124, 80, '2026-09-27 08:00:00', '2026-09-27 08:00:00'),
		       (?, 'bp', '2026-09-29 08:00:00', 128, 80, '2026-09-29 08:00:00', '2026-09-29 08:00:00')`, uid, uid)
	r := e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tok, "en",
		`{"kind":"hrt","name":"Estrogen gel","schedule":"morning","started_on":"2026-09-27"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	id := num(first(r, "hrt")["id"])
	for _, d := range []string{"2026-09-27", "2026-09-28", "2026-09-29", "2026-09-30"} {
		e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/menopause/treatment/items/%d/intakes/%s", id, d), tok, "en", `{}`)
	}
	e.do(t, http.MethodPost, "/api/v1/menopause/treatment/items", tok, "en", `{"kind":"supplement","name":"Calcium + vitamin D","schedule":"noon"}`)
	e.do(t, http.MethodPut, "/api/v1/menopause/treatment/side-effects/2026-09-29", tok, "en", `{"codes":["breast_tenderness"]}`)
	return id
}

func TestReport(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000104")
	e.seedReport(t, uid, tok)

	r := e.do(t, http.MethodGet, "/api/v1/menopause/report?months=1", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["empty"])
	rep := r.data()["report"].(map[string]any)
	assert.Equal(t, "2026-09-01", rep["window"].(map[string]any)["from"])
	assert.Equal(t, "meno", rep["stage"].(map[string]any)["code"])
	assert.Equal(t, 14, num(rep["stage"].(map[string]any)["months_without_period"]))
	score := rep["score"].(map[string]any)
	assert.Equal(t, 22, num(score["first"].(map[string]any)["total"]))
	assert.Equal(t, 14, num(score["last"].(map[string]any)["total"]))
	assert.Equal(t, -8, num(score["change"]))
	fl := rep["hot_flashes"].(map[string]any)
	assert.Equal(t, 3, num(fl["total"]))
	assert.Equal(t, 5, num(fl["tracked_days"]), "24, 25, 28, 29, 30 Sep")
	assert.InDelta(t, 0.6, fl["per_day"], 0.001)
	assert.Equal(t, 1, num(rep["night_sweats"].(map[string]any)["nights"]))
	assert.InDelta(t, 6.0, rep["sleep"].(map[string]any)["avg_hours"], 0.001)
	bl := rep["bleeding"].(map[string]any)
	assert.Equal(t, 1, num(bl["events"]))
	assert.Equal(t, 2, num(bl["days"]))
	assert.Equal(t, []any{"2026-09-24"}, bl["dates"])
	bp := rep["blood_pressure"].(map[string]any)
	assert.Equal(t, 126, num(bp["systolic"]))
	assert.Equal(t, 80, num(bp["diastolic"]))
	top := rep["symptoms"].(map[string]any)["top"].([]any)
	require.NotEmpty(t, top)
	assert.Equal(t, "symptoms.general.hot_flashes", top[0].(map[string]any)["key"])
	assert.Equal(t, 40, num(top[0].(map[string]any)["percent"]), "2 of 5 tracked days")
	tr := rep["treatment"].(map[string]any)
	assert.Equal(t, 80, num(tr["hrt_adherence_pct"]), "4 of 27 Sep … 1 Oct")
	assert.Equal(t, []any{"Calcium + vitamin D"}, rep["supplements"])
	se := rep["side_effects"].([]any)
	require.Len(t, se, 1)
	assert.Equal(t, "breast_tenderness", se[0].(map[string]any)["code"])
	assert.NotContains(t, r.raw, `"id"`, "no row ids in the report")

	r = e.do(t, http.MethodGet, "/api/v1/menopause/report?months=2", tok, "en", "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
}

func phpvalMap(t *testing.T, kv map[string]any) phpval.Map {
	t.Helper()
	m := phpval.NewMap()
	for k, v := range kv {
		m.Set(k, v)
	}
	return m
}

func TestReportSectionInBuilder(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000105")
	e.seedReport(t, uid, tok)
	other, _ := e.user(t, "09120000106") // not in menopause mode

	cat := catalog.NewReader(catalogstore.New(e.db), nil, 0, quiet)
	labels := i18n.NewTranslationStore(translations.FS, "")
	svc := healthrecord.NewService(e.db, nil).WithProviders(menopause.NewReportSection(menopause.NewService(e.db, cat), labels))
	at, err := time.ParseInLocation("2006-01-02 15:04", "2026-10-01 10:00", civildate.Tehran)
	require.NoError(t, err)
	ctx := context.Background()

	req, err := healthrecord.ParseReportRequest(phpvalMap(t, map[string]any{"range": "3m", "sections": "vitals,menopause"}), "en", at, false)
	require.NoError(t, err)
	assert.Equal(t, []string{healthrecord.SectionVitals, healthrecord.SectionMenopause}, req.Sections)

	rec, err := svc.Build(ctx, uid, healthrecord.AudienceShare, req.Options(civildate.InTehran(at), "en", "fa"))
	require.NoError(t, err)
	sec, ok := rec.Section(healthrecord.SectionMenopause)
	require.True(t, ok)
	assert.False(t, sec.Editable)
	assert.False(t, sec.Empty)
	body := healthrecord.ShareJSON(rec)
	raw, err := body.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"key":"menopause"`)
	assert.Contains(t, string(raw), `"Estrogen gel"`)
	from, _ := req.Window(civildate.InTehran(at))
	assert.Contains(t, string(raw), `"window":{"from":"`+from.String()+`"`, "the builder's 3-month window")

	// another user: no section (not menopause mode); the summary (no section filter) leaves it out
	rec, err = svc.Build(ctx, other, healthrecord.AudienceShare, req.Options(civildate.InTehran(at), "en", "fa"))
	require.NoError(t, err)
	_, ok = rec.Section(healthrecord.SectionMenopause)
	assert.False(t, ok)
	rec, err = svc.Build(ctx, uid, healthrecord.AudienceOwner, healthrecord.Options{Today: civildate.InTehran(at)})
	require.NoError(t, err)
	_, ok = rec.Section(healthrecord.SectionMenopause)
	assert.False(t, ok)
}
