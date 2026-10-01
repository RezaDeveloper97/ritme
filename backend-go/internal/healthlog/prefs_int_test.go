package healthlog_test

// Log preferences and custom items (/api/v1/logs/preferences, /logs/custom-items, B-N3-02) end to end:
// defaults per mode and cycle phase, saving / resetting per mode, limits, custom items in the day save,
// soft delete keeping history, IDOR.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func strs(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, x := range list {
		s, _ := x.(string)
		out = append(out, s)
	}
	return out
}

// cycleUser gives the user a 28-day profile (user_goal: goal) and a confirmed period starting on lmp.
func (e *logsEnv) cycleUser(t *testing.T, userID uint64, lmp, goal string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO user_profiles (user_id, period_duration, cycle_duration, last_period_start, user_goal, created_at, updated_at)
		VALUES (?, 5, 28, ?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, lmp, goal)
	require.NoError(t, err)
	end := civildate.MustParse(lmp).AddDays(4).String()
	_, err = e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source, created_at, updated_at)
		VALUES (?, ?, ?, 5, 1, 0, 'user_logged', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, lmp, end)
	require.NoError(t, err)
}

func TestPrefs_Unauthenticated(t *testing.T) {
	e := setupLogs(t)
	for _, c := range [][2]string{
		{"GET", "/api/v1/logs/preferences"}, {"PUT", "/api/v1/logs/preferences"}, {"DELETE", "/api/v1/logs/preferences"},
		{"GET", "/api/v1/logs/custom-items"}, {"POST", "/api/v1/logs/custom-items"},
		{"PATCH", "/api/v1/logs/custom-items/1"}, {"DELETE", "/api/v1/logs/custom-items/1"},
	} {
		r := e.do(t, c[0], c[1], "", "", `{}`)
		assert.Equal(t, 401, r.status, c)
		assert.Equal(t, "unauthenticated", r.body["error_code"], c)
	}
}

func TestPrefs_DefaultsPerMode(t *testing.T) {
	e := setupLogs(t)
	_, tok := e.user(t, "09120000001", "")
	preg, pregTok := e.user(t, "09120000002", "")
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, created_at, updated_at) VALUES (?, 1, NOW(), NOW())`, preg)
	require.NoError(t, err)
	_, postTok := e.user(t, "09120000003", "postpartum")
	_, menoTok := e.user(t, "09120000004", "menopause")

	r := e.do(t, "GET", "/api/v1/logs/preferences", tok, "fa", "")
	require.Equal(t, 200, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "cycle", d["mode"])
	assert.Nil(t, d["phase"], "no cycle data → no phase")
	assert.Equal(t, true, d["is_default"])
	assert.Equal(t, float64(8), d["max_pinned"])
	assert.Equal(t, []string{"bleeding", "pain", "mood", "symptoms", "discharge", "sleep", "measurements.weight", "note"}, strs(d["pinned"]))
	cats, _ := d["categories"].([]any)
	first, _ := cats[0].(map[string]any)
	assert.Equal(t, "bleeding", first["code"])
	assert.Equal(t, "پریود و لکه\u200cبینی", first["label"])
	assert.Equal(t, true, first["pinned"])
	assert.Equal(t, false, first["hidden"])
	assert.Nil(t, first["custom_param"])
	assert.NotContains(t, r.raw, `"code":"baby"`)
	assert.Contains(t, r.raw, `"custom_param":"general"`)
	assert.Equal(t, []any{}, d["custom_items"])

	r = e.do(t, "GET", "/api/v1/logs/preferences", pregTok, "en", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "pregnancy", r.data()["mode"])
	assert.Equal(t, []string{"pregnancy.kicks", "pregnancy.contractions", "symptoms", "mood", "measurements.weight",
		"measurements.bp_systolic", "sleep", "note"}, strs(r.data()["pinned"]))

	r = e.do(t, "GET", "/api/v1/logs/preferences", postTok, "en", "")
	assert.Equal(t, []string{"baby.feeding", "bleeding", "pain", "mood", "sleep", "measurements.weight", "meds", "note"}, strs(r.data()["pinned"]))
	r = e.do(t, "GET", "/api/v1/logs/preferences", menoTok, "en", "")
	assert.Equal(t, "symptoms", strs(r.data()["pinned"])[0])

	r = e.do(t, "GET", "/api/v1/logs/preferences?mode=ttc", tok, "en", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "ttc", r.data()["mode"])
	assert.Equal(t, "measurements.bbt", strs(r.data()["pinned"])[0])

	for _, m := range []string{"all", "martian"} {
		r = e.do(t, "GET", "/api/v1/logs/preferences?mode="+m, tok, "en", "")
		assert.Equal(t, 422, r.status, m)
		assert.Equal(t, []string{"mode"}, r.errorKeys())
	}
}

func TestPrefs_DefaultsPerPhase(t *testing.T) {
	e := setupLogs(t)
	period, tok := e.user(t, "09120000001", "")
	e.cycleUser(t, period, "2026-09-22", "non_ttc") // today (2026-09-23) is day 2
	fertile, fertileTok := e.user(t, "09120000002", "")
	e.cycleUser(t, fertile, "2026-09-11", "non_ttc") // day 13
	ttc, ttcTok := e.user(t, "09120000003", "")
	e.cycleUser(t, ttc, "2026-09-11", "ttc")

	r := e.do(t, "GET", "/api/v1/logs/preferences", tok, "en", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "menstrual", r.data()["phase"])
	assert.Equal(t, []string{"bleeding", "pain", "mood", "symptoms", "sleep", "appetite_energy", "measurements.weight", "note"}, strs(r.data()["pinned"]))

	r = e.do(t, "GET", "/api/v1/logs/preferences", fertileTok, "en", "")
	assert.Equal(t, "fertile", r.data()["phase"])
	assert.Equal(t, "discharge", strs(r.data()["pinned"])[0])

	r = e.do(t, "GET", "/api/v1/logs/preferences", ttcTok, "en", "")
	assert.Equal(t, "fertile", r.data()["phase"])
	assert.Equal(t, "measurements.lh_test", strs(r.data()["pinned"])[0])

	// the phase is not used outside the cycle modes
	r = e.do(t, "GET", "/api/v1/logs/preferences?mode=pregnancy", tok, "en", "")
	assert.Nil(t, r.data()["phase"])
}

func TestPrefs_SaveResetPerMode(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120000001", "")

	r := e.do(t, "PUT", "/api/v1/logs/preferences", tok, "en", `{"order":["note","mood"],"hidden":["sex"],"pinned":["note","pain","measurements.bbt"]}`)
	require.Equal(t, 200, r.status, r.raw)
	d := r.data()
	assert.Equal(t, false, d["is_default"])
	assert.Equal(t, map[string]any{"order": true, "hidden": true, "pinned": true}, d["customized"])
	assert.Equal(t, []string{"note", "pain", "measurements.bbt"}, strs(d["pinned"]))
	cats, _ := d["categories"].([]any)
	assert.Equal(t, "note", cats[0].(map[string]any)["code"])
	assert.Equal(t, "mood", cats[1].(map[string]any)["code"])
	assert.Equal(t, "bleeding", cats[2].(map[string]any)["code"])
	assert.Contains(t, r.raw, `{"code":"sex","label":"Sex & libido","hidden":true,"pinned":false,"custom_param":null}`)

	g := e.do(t, "GET", "/api/v1/logs/preferences", tok, "en", "")
	assert.JSONEq(t, mustJSON(t, d), mustJSON(t, g.data()))

	// a list not sent is kept; null resets one list to its default
	r = e.do(t, "PUT", "/api/v1/logs/preferences", tok, "en", `{"pinned":null}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, map[string]any{"order": true, "hidden": true, "pinned": false}, r.data()["customized"])
	assert.Equal(t, []string{"bleeding", "pain", "mood", "symptoms", "discharge", "sleep", "measurements.weight", "note"}, strs(r.data()["pinned"]))

	// hiding a category drops its tiles (defaults top up from the next ones)
	r = e.do(t, "PUT", "/api/v1/logs/preferences", tok, "en", `{"hidden":["discharge"]}`)
	assert.NotContains(t, strs(r.data()["pinned"]), "discharge")
	assert.Len(t, strs(r.data()["pinned"]), 8)

	// another mode is untouched
	p := e.do(t, "GET", "/api/v1/logs/preferences?mode=pregnancy", tok, "en", "")
	assert.Equal(t, true, p.data()["is_default"])
	r = e.do(t, "PUT", "/api/v1/logs/preferences?mode=pregnancy", tok, "en", `{"pinned":["pregnancy.kicks"]}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 2, e.count(t, "SELECT COUNT(*) FROM health_log_preferences WHERE user_id = ?", uid))

	// reset
	r = e.do(t, "DELETE", "/api/v1/logs/preferences", tok, "en", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, true, r.data()["is_default"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM health_log_preferences WHERE user_id = ?", uid))

	// resetting every list removes the row
	r = e.do(t, "PUT", "/api/v1/logs/preferences?mode=pregnancy", tok, "en", `{"pinned":null}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM health_log_preferences WHERE user_id = ?", uid))

	// an empty tile list is a choice too
	r = e.do(t, "PUT", "/api/v1/logs/preferences", tok, "en", `{"pinned":[]}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Empty(t, strs(r.data()["pinned"]))
}

func TestPrefs_Validation(t *testing.T) {
	e := setupLogs(t)
	_, tok := e.user(t, "09120000001", "")

	nine := `{"pinned":["bleeding","pain","mood","symptoms","discharge","sleep","note","meds","activity"]}`
	r := e.do(t, "PUT", "/api/v1/logs/preferences", tok, "en", nine)
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, []string{"pinned"}, r.errorKeys())
	assert.Contains(t, r.raw, "must not have more than 8 items")

	r = e.do(t, "PUT", "/api/v1/logs/preferences", tok, "en", `{"pinned":["baby.feeding","pain.nope"],"order":["horoscope"],"hidden":"sex"}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.ElementsMatch(t, []string{"pinned.0", "pinned.1", "order.0", "hidden"}, r.errorKeys())

	r = e.do(t, "PUT", "/api/v1/logs/preferences", tok, "en", `{"order":["note","note"],"pinned":["pain","pain"]}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.ElementsMatch(t, []string{"order.1", "pinned.1"}, r.errorKeys())
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM health_log_preferences"), "nothing is written on a 422")
}

func TestCustomItems_Lifecycle(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120000001", "")

	r := e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"custom","label":"  قهوه   زیاد "}`)
	require.Equal(t, 201, r.status, r.raw)
	item := r.data()
	code, _ := item["code"].(string)
	id := int(item["id"].(float64))
	assert.Equal(t, "custom_"+strconv.Itoa(id), code)
	assert.Equal(t, "items", item["param"])
	assert.Equal(t, "قهوه زیاد", item["label"], "trimmed, whitespace collapsed")
	assert.Nil(t, item["deleted_at"])

	r = e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"symptoms","label":"Jaw pain"}`)
	require.Equal(t, 201, r.status, r.raw)
	symptom, _ := r.data()["code"].(string)
	assert.Equal(t, "general", r.data()["param"])

	// the day save accepts them in their param
	day := `{"categories":{"custom":{"items":{"` + code + `":"yes"}},"symptoms":{"general":{"` + symptom + `":"moderate"}}}}`
	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tok, "en", day)
	require.Equal(t, 200, r.status, r.raw)
	assert.Contains(t, r.raw, code)
	// … but not in another param
	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tok, "en", `{"categories":{"mood":{"moods":["`+code+`"]}}}`)
	assert.Equal(t, 422, r.status)

	// preferences list the active items
	p := e.do(t, "GET", "/api/v1/logs/preferences", tok, "en", "")
	assert.Len(t, p.data()["custom_items"], 2)

	// rename
	r = e.do(t, "PATCH", "/api/v1/logs/custom-items/"+strconv.Itoa(id), tok, "en", `{"label":"Coffee"}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "Coffee", r.data()["label"])
	assert.Equal(t, code, r.data()["code"], "the code never changes")

	// delete is soft: logged days keep the item, new input refuses it, re-saving the stored value passes
	r = e.do(t, "DELETE", "/api/v1/logs/custom-items/"+strconv.Itoa(id), tok, "en", "")
	require.Equal(t, 200, r.status, r.raw)
	assert.NotNil(t, r.data()["deleted_at"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM health_log_custom_items WHERE user_id = ?  AND deleted_at IS NOT NULL", uid))
	g := e.do(t, "GET", "/api/v1/logs/days/2026-09-22", tok, "en", "")
	assert.Contains(t, g.categories(), "custom")
	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-21", tok, "en", `{"categories":{"custom":{"items":{"`+code+`":"yes"}}}}`)
	assert.Equal(t, 422, r.status, "a deleted item takes no new input")
	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tok, "en", `{"categories":{"custom":{"items":{"`+code+`":"no"}}}}`)
	assert.Equal(t, 200, r.status, "but stays editable where it is logged")

	l := e.do(t, "GET", "/api/v1/logs/custom-items", tok, "en", "")
	require.Equal(t, 200, l.status, l.raw)
	items, _ := l.data()["custom_items"].([]any)
	require.Len(t, items, 2, "deleted items are listed for history labels")
	assert.NotNil(t, items[0].(map[string]any)["deleted_at"])
	p = e.do(t, "GET", "/api/v1/logs/preferences", tok, "en", "")
	assert.Len(t, p.data()["custom_items"], 1, "preferences list only active items")

	// a deleted item cannot be renamed or deleted again; its label is free again
	assert.Equal(t, 404, e.do(t, "PATCH", "/api/v1/logs/custom-items/"+strconv.Itoa(id), tok, "en", `{"label":"x"}`).status)
	assert.Equal(t, 404, e.do(t, "DELETE", "/api/v1/logs/custom-items/"+strconv.Itoa(id), tok, "en", "").status)
	assert.Equal(t, 201, e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"custom","label":"coffee"}`).status)
	assert.Equal(t, 404, e.do(t, "DELETE", "/api/v1/logs/custom-items/abc", tok, "en", "").status)
}

func TestCustomItems_Validation(t *testing.T) {
	e := setupLogs(t)
	_, tok := e.user(t, "09120000001", "")

	r := e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{}`)
	require.Equal(t, 422, r.status)
	assert.ElementsMatch(t, []string{"category", "label"}, r.errorKeys())
	r = e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"bleeding","label":"`+strings.Repeat("x", 41)+`"}`)
	require.Equal(t, 422, r.status)
	assert.ElementsMatch(t, []string{"category", "label"}, r.errorKeys(), "bleeding takes no custom items; label too long")
	r = e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"custom","label":"   "}`)
	require.Equal(t, 422, r.status)
	assert.Equal(t, []string{"label"}, r.errorKeys())

	require.Equal(t, 201, e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"custom","label":"Coffee"}`).status)
	r = e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"custom","label":"coffee "}`)
	require.Equal(t, 422, r.status, "same label in the category (case-insensitive)")
	assert.Equal(t, []string{"label"}, r.errorKeys())
	assert.Equal(t, 201, e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"mood","label":"Coffee"}`).status,
		"another category may reuse it")

	// limit: 20 active items
	for i := 3; i <= 20; i++ {
		require.Equal(t, 201, e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"custom","label":"item `+strconv.Itoa(i)+`"}`).status)
	}
	r = e.do(t, "POST", "/api/v1/logs/custom-items", tok, "en", `{"category":"custom","label":"one too many"}`)
	require.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, []string{"custom_items"}, r.errorKeys())
	assert.Contains(t, r.raw, "The custom items field must not have more than 20 items.")
}

func TestCustomItems_IDOR(t *testing.T) {
	e := setupLogs(t)
	a, tokA := e.user(t, "09120000001", "")
	_, tokB := e.user(t, "09120000002", "")
	r := e.do(t, "POST", "/api/v1/logs/custom-items", tokA, "en", `{"category":"custom","label":"A's"}`)
	require.Equal(t, 201, r.status, r.raw)
	id := strconv.Itoa(int(r.data()["id"].(float64)))
	code, _ := r.data()["code"].(string)
	require.Equal(t, 200, e.do(t, "PUT", "/api/v1/logs/preferences", tokA, "en", `{"pinned":["note"]}`).status)

	assert.Equal(t, 404, e.do(t, "PATCH", "/api/v1/logs/custom-items/"+id, tokB, "en", `{"label":"B"}`).status)
	assert.Equal(t, 404, e.do(t, "DELETE", "/api/v1/logs/custom-items/"+id, tokB, "en", "").status)
	l := e.do(t, "GET", "/api/v1/logs/custom-items", tokB, "en", "")
	assert.Equal(t, []any{}, l.data()["custom_items"])
	p := e.do(t, "GET", "/api/v1/logs/preferences", tokB, "en", "")
	assert.Equal(t, true, p.data()["is_default"], "B never sees A's preferences")
	assert.Equal(t, []any{}, p.data()["custom_items"])

	r = e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tokB, "en", `{"categories":{"custom":{"items":{"`+code+`":"yes"}}}}`)
	assert.Equal(t, 422, r.status, "B cannot log A's custom item")
	require.Equal(t, 200, e.do(t, "DELETE", "/api/v1/logs/preferences", tokB, "en", "").status)

	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM health_log_custom_items WHERE user_id = ? AND label = 'A''s' AND deleted_at IS NULL", a))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM health_log_preferences WHERE user_id = ?", a))
}
