package pregnancy_test

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	adminpregnancy "github.com/ritme/backend-go/internal/admin/pregnancy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// newEnv mounts the routes on a database with the T-M7-01 seeds (weeks 1–42, 5 care items,
// the pregnancy_alert rows).
func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	adminpregnancy.New(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	return e
}

func tr(fa, en string) map[string]any { return map[string]any{"fa": fa, "en": en} }

func weekBody() map[string]any {
	return map[string]any{
		"size_label":       tr("تمشک", "raspberry"),
		"illustration_key": "raspberry",
		"length_cm":        "1.6",
		"weight_g":         "<1",
		"heart_rate":       "150-170",
		"headline":         tr("تیتر", "Headline"),
		"highlights": []any{
			map[string]any{"icon": "hand", "tone": "pink", "title": tr("دست", "Hands"), "body": tr("متن", "Text")},
			map[string]any{"icon": nil, "title": tr("قلب", "Heart"), "body": tr("متن", "Text")},
		},
		"body_symptoms": []any{
			map[string]any{"key": "nausea", "label": tr("تهوع", "Nausea")},
			map[string]any{"key": "braxton_hicks", "label": tr("انقباض", "Braxton Hicks")},
		},
		"body_text":     tr("بدن", "Body"),
		"tasks":         []any{map[string]any{"key": "folic_acid", "text": tr("فولیک", "Folic acid")}},
		"warning":       tr("هشدار", "Warning"),
		"reviewer_name": tr("دکتر", "Dr X"),
		"reviewed_at":   "2026-09-01",
		"sources": []any{
			map[string]any{"title": tr("منبع", "Source"), "url": "https://who.int/x"},
			map[string]any{"title": tr("کتاب", "Book"), "url": nil},
		},
	}
}

func clone(m map[string]any, kv ...any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i].(string)] = kv[i+1]
	}
	return out
}

// ---------------------------------------------------------------------------

func TestGuards(t *testing.T) {
	e := newEnv(t)
	for _, p := range []string{"/pregnancy-week-details", "/pregnancy-weeks/8/details", "/pregnancy-care-items",
		"/pregnancy-alert-rules", "/pregnancy-alert-rules/bp_high"} {
		r := e.Anonymous().Get(p)
		assert.Equal(t, 401, r.Status, p)
		assert.Equal(t, "unauthenticated", r.Code(), p)
		assert.Equal(t, 200, e.As(admintest.EditorID).Get(p).Status, "editors manage pregnancy content: "+p)
		assert.Equal(t, 200, e.As(admintest.SuperID).Get(p).Status, p)
	}
	noToken := e.As(admintest.EditorID)
	noToken.CSRF = ""
	for _, req := range []struct{ m, p string }{
		{fiber.MethodPut, "/pregnancy-weeks/8/details"},
		{fiber.MethodPost, "/pregnancy-care-items"},
		{fiber.MethodPost, "/pregnancy-care-items/reorder"},
		{fiber.MethodDelete, "/pregnancy-care-items/1"},
		{fiber.MethodPut, "/pregnancy-alert-rules/bp_high"},
	} {
		r := noToken.JSON(req.m, req.p, map[string]any{})
		assert.Equal(t, 419, r.Status, req.p)
		assert.Equal(t, "csrf_mismatch", r.Code(), req.p)
	}
	assert.Equal(t, 5, e.Int("SELECT COUNT(*) FROM pregnancy_care_items"))
}

// ---------------------------------------------------------------------------
// Week details

func TestWeekDetailsReadAndOptions(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.Get("/pregnancy-weeks/8/details")
	require.Equal(t, 200, r.Status, r.Body)
	w := r.Obj("week_details")
	assert.Equal(t, float64(8), w["week_number"])
	assert.Equal(t, true, w["exists"])
	assert.Equal(t, "raspberry", w["illustration_key"])
	assert.Equal(t, map[string]any{"fa": "تمشک", "en": "raspberry"}, w["size_label"])
	assert.Equal(t, "150-170", w["heart_rate"])
	assert.Len(t, w["highlights"], 3)
	assert.Nil(t, w["reviewed_at"], "seeded copy is not clinically reviewed")

	for _, bad := range []string{"0", "43", "abc", "08"} {
		assert.Equal(t, 404, c.Get("/pregnancy-weeks/"+bad+"/details").Status, bad)
	}

	r = c.Get("/pregnancy-week-details")
	require.Equal(t, 200, r.Status)
	assert.Len(t, r.Items(), 42)
	first := r.Items()[0].(map[string]any)
	assert.Equal(t, float64(1), first["week_number"])
	assert.Nil(t, first["illustration_key"])

	r = c.Get("/pregnancy-week-details/options")
	require.Equal(t, 200, r.Status)
	assert.Len(t, r.Data()["illustration_keys"], len(adminpregnancy.IllustrationKeys))
	assert.Equal(t, float64(42), r.Data()["max_week"])
	assert.Len(t, r.Data()["highlight_icons"], len(adminpregnancy.HighlightIcons))
}

func TestWeekDetailsUpdate(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.JSON(fiber.MethodPut, "/pregnancy-weeks/8/details", weekBody())
	require.Equal(t, 200, r.Status, r.Body)
	w := r.Obj("week_details")
	assert.Equal(t, "2026-09-01", w["reviewed_at"])
	assert.Equal(t, "<1", w["weight_g"])
	assert.Equal(t, []any{
		map[string]any{"icon": "hand", "tone": "pink", "title": tr("دست", "Hands"), "body": tr("متن", "Text")},
		map[string]any{"icon": nil, "title": tr("قلب", "Heart"), "body": tr("متن", "Text")},
	}, w["highlights"])
	assert.Equal(t, []any{
		map[string]any{"title": tr("منبع", "Source"), "url": "https://who.int/x"},
		map[string]any{"title": tr("کتاب", "Book"), "url": nil},
	}, w["sources"])
	var tasks []map[string]any
	require.NoError(t, json.Unmarshal([]byte(e.String("SELECT tasks FROM pregnancy_week_details WHERE week_number = 8")), &tasks))
	assert.Equal(t, "folic_acid", tasks[0]["key"])

	// Partial update: absent fields keep their value; [] / null clear.
	r = c.JSON(fiber.MethodPut, "/pregnancy-weeks/8/details", map[string]any{
		"headline": tr("نو", "New"), "highlights": []any{}, "warning": nil,
	})
	require.Equal(t, 200, r.Status, r.Body)
	w = r.Obj("week_details")
	assert.Equal(t, tr("نو", "New"), w["headline"])
	assert.Equal(t, []any{}, w["highlights"])
	assert.Nil(t, w["warning"])
	assert.Equal(t, "raspberry", w["illustration_key"], "absent keeps")
	assert.Len(t, w["tasks"], 1)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM pregnancy_week_details WHERE week_number = 8 AND highlights IS NULL"))
	assert.Equal(t, 42, e.Int("SELECT COUNT(*) FROM pregnancy_week_details"), "upsert, no new row")

	// Validation.
	tooMany := make([]any, 11)
	for i := range tooMany {
		tooMany[i] = map[string]any{"key": "k" + strconv.Itoa(i), "text": tr("a", "b")}
	}
	tomorrow := civildate.InTehran(time.Now()).AddDays(1).String()
	r = c.JSON(fiber.MethodPut, "/pregnancy-weeks/8/details", clone(weekBody(),
		"illustration_key", "dragon_fruit",
		"length_cm", "about 2",
		"highlights", []any{map[string]any{"icon": "rocket", "title": map[string]any{"fa": "x"}, "body": tr("a", "b")}},
		"body_symptoms", []any{
			map[string]any{"key": "nausea", "label": tr("a", "b")},
			map[string]any{"key": "nausea", "label": tr("a", "b")},
		},
		"tasks", tooMany,
		"sources", []any{map[string]any{"title": tr("a", "b"), "url": "javascript:alert(1)"}},
		"reviewed_at", tomorrow,
		"headline", map[string]any{"fa": "فقط فارسی"},
	))
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"illustration_key", "length_cm", "highlights.0.icon", "highlights.0.title.en",
		"body_symptoms.1.key", "tasks", "sources.0.url", "reviewed_at", "headline.en"} {
		assert.Contains(t, r.Errors(), f)
	}
	assert.Equal(t, "validation_failed", r.Code())
}

// ---------------------------------------------------------------------------
// Care items

func careBody() map[string]any {
	return map[string]any{
		"key": "iron_check", "title": tr("آزمایش آهن", "Iron check"), "prep": tr("ناشتا", "Fasting"),
		"kind": "test", "week_from": 24, "week_to": 28, "remind_before": 2, "is_active": true,
	}
}

func careID(e *admintest.Env, key string) int {
	return e.Int("SELECT id FROM pregnancy_care_items WHERE `key` = ?", key)
}

func linkAppointment(e *admintest.Env, key string) {
	e.T.Helper()
	res := e.Exec("INSERT INTO users (mobile, created_at, updated_at) VALUES ('09120000009', NOW(), NOW())")
	uid, err := res.LastInsertId()
	require.NoError(e.T, err)
	e.Exec(`INSERT INTO reminders (user_id, type, title, scheduled_at, meta, created_at, updated_at)
		VALUES (?, 'appointment', 'Visit', NOW(), JSON_OBJECT('care_item_key', ?, 'stage', 'booked'), NOW(), NOW())`, uid, key)
}

func TestCareItemsCRUD(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.Get("/pregnancy-care-items")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 5)
	first := r.Items()[0].(map[string]any)
	assert.Equal(t, "first_visit", first["key"])
	assert.Equal(t, float64(0), first["appointments_count"])

	r = c.Get("/pregnancy-care-items/options")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, []any{"visit", "test", "scan", "vaccine"}, r.Data()["kinds"])
	assert.Equal(t, float64(6), r.Data()["next_sort_order"])

	r = c.JSON(fiber.MethodPost, "/pregnancy-care-items", careBody())
	require.Equal(t, 201, r.Status, r.Body)
	it := r.Obj("care_item")
	assert.Equal(t, "iron_check", it["key"])
	assert.Equal(t, float64(6), it["sort_order"], "appended")
	assert.Equal(t, float64(2), it["remind_before"])
	id := int(it["id"].(float64))
	path := "/pregnancy-care-items/" + strconv.Itoa(id)

	r = c.JSON(fiber.MethodPost, "/pregnancy-care-items", clone(careBody(), "week_from", 30, "week_to", 20,
		"kind", "party", "title", map[string]any{"fa": "x"}))
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"key", "week_to", "kind", "title.en"} {
		assert.Contains(t, r.Errors(), f)
	}

	// Update: the key is fixed; absent prep / remind_before keep their value.
	upd := clone(careBody(), "key", "renamed", "title", tr("آهن", "Iron"), "is_active", false)
	delete(upd, "prep")
	delete(upd, "remind_before")
	r = c.JSON(fiber.MethodPut, path, upd)
	require.Equal(t, 200, r.Status, r.Body)
	it = r.Obj("care_item")
	assert.Equal(t, "iron_check", it["key"])
	assert.Equal(t, tr("آهن", "Iron"), it["title"])
	assert.Equal(t, tr("ناشتا", "Fasting"), it["prep"])
	assert.Equal(t, float64(2), it["remind_before"])
	assert.Equal(t, false, it["is_active"])

	r = c.JSON(fiber.MethodPost, path+"/toggle", nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, true, r.Obj("care_item")["is_active"])

	assert.Equal(t, 404, c.Get("/pregnancy-care-items/999999").Status)
	assert.Equal(t, 404, c.Get("/pregnancy-care-items/abc").Status)

	// Delete: refused while appointments reference the key.
	linkAppointment(e, "first_visit")
	fv := "/pregnancy-care-items/" + strconv.Itoa(careID(e, "first_visit"))
	r = c.JSON(fiber.MethodDelete, fv, nil)
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "in_use", r.Code())
	assert.Equal(t, float64(1), r.Body["appointments_count"])
	assert.Equal(t, float64(1), c.Get(fv).Obj("care_item")["appointments_count"])
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM pregnancy_care_items WHERE `key` = 'first_visit'"))

	r = c.JSON(fiber.MethodDelete, path, nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM pregnancy_care_items WHERE `key` = 'iron_check'"))
}

func TestCareItemsReorder(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	keys := []string{"tdap", "gtt", "anomaly_scan", "nt_scan", "first_visit"}
	ids := make([]int, 0, len(keys))
	for _, k := range keys {
		ids = append(ids, careID(e, k))
	}

	r := c.JSON(fiber.MethodPost, "/pregnancy-care-items/reorder", map[string]any{"ids": ids[:4]})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "ids")
	r = c.JSON(fiber.MethodPost, "/pregnancy-care-items/reorder", map[string]any{"ids": append([]int{ids[0]}, ids[:4]...)})
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "ids.1")

	r = c.JSON(fiber.MethodPost, "/pregnancy-care-items/reorder", map[string]any{"ids": ids})
	require.Equal(t, 200, r.Status, r.Body)
	list := c.Get("/pregnancy-care-items").Items()
	for i, k := range keys {
		assert.Equal(t, k, list[i].(map[string]any)["key"])
		assert.Equal(t, float64(i+1), list[i].(map[string]any)["sort_order"])
	}
}

// ---------------------------------------------------------------------------
// Alert rules

func payload(e *admintest.Env, key, locale string) map[string]any {
	e.T.Helper()
	var p map[string]any
	require.NoError(e.T, json.Unmarshal([]byte(e.String(
		"SELECT payload FROM message_contents WHERE `group` = 'pregnancy_alert' AND item_key = ? AND locale = ?",
		key, locale)), &p))
	return p
}

func TestAlertRulesRead(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.Get("/pregnancy-alert-rules")
	require.Equal(t, 200, r.Status, r.Body)
	require.Len(t, r.Items(), 8)
	vs := r.Items()[0].(map[string]any)
	assert.Equal(t, "vomiting_streak", vs["key"])
	assert.Equal(t, true, vs["configured"])
	assert.Equal(t, "follow_up", vs["level"])
	assert.Equal(t, map[string]any{"min_streak_days": float64(3), "severe_min_count": float64(2)}, vs["params"])
	assert.Equal(t, []any{}, vs["missing_locales"])
	we := r.Items()[4].(map[string]any)
	assert.Equal(t, "week_entered", we["key"])
	assert.Equal(t, map[string]any{}, we["params"], "param-less rule answers {}")
	assert.Len(t, r.Data()["legend"].(map[string]any)["rows"], 2)

	r = c.Get("/pregnancy-alert-rules/bp_high")
	require.Equal(t, 200, r.Status, r.Body)
	rule := r.Obj("rule")
	assert.Equal(t, "urgent", rule["level"])
	texts := rule["texts"].(map[string]any)
	assert.Contains(t, texts, "fa")
	assert.Contains(t, texts, "en")
	assert.Len(t, rule["params_schema"], 2)
	assert.Equal(t, []any{"systolic", "diastolic"}, rule["placeholders"])

	r = c.Get("/pregnancy-alert-rules/options")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, []any{"info", "suggestion", "follow_up", "urgent"}, r.Data()["levels"])
	assert.Len(t, r.Data()["rules"], 8)

	assert.Equal(t, 404, c.Get("/pregnancy-alert-rules/legend").Status, "the legend is not a rule")
	assert.Equal(t, 404, c.Get("/pregnancy-alert-rules/nope").Status)
}

func TestAlertRulesUpdate(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	enBefore := payload(e, "bp_high", "en")

	r := c.JSON(fiber.MethodPut, "/pregnancy-alert-rules/bp_high", map[string]any{
		"enabled": false, "level": "follow_up", "window_days": 14,
		"params": map[string]any{"systolic_min": 135, "diastolic_min": 85, "junk": 1},
		"texts": map[string]any{"fa": map[string]any{
			"title": "فشار بالا", "what_we_saw": "{systolic}/{diastolic}", "how_sure": "یک عدد", "advice": "استراحت",
			"actions": []any{map[string]any{"key": "call", "label": "تماس"}}, "contact": "",
		}},
	})
	require.Equal(t, 200, r.Status, r.Body)
	rule := r.Obj("rule")
	assert.Equal(t, false, rule["enabled"])
	assert.Equal(t, float64(14), rule["window_days"])

	fa, en := payload(e, "bp_high", "fa"), payload(e, "bp_high", "en")
	for _, p := range []map[string]any{fa, en} {
		assert.Equal(t, false, p["enabled"], "behaviour written to every locale")
		assert.Equal(t, "follow_up", p["level"])
		assert.Equal(t, map[string]any{"systolic_min": float64(135), "diastolic_min": float64(85)}, p["params"])
	}
	assert.Equal(t, "فشار بالا", fa["title"])
	assert.Nil(t, fa["contact"])
	assert.Equal(t, enBefore["title"], en["title"], "texts of an unsent locale are kept")
	assert.Equal(t, enBefore["actions"], en["actions"])

	// Validation.
	r = c.JSON(fiber.MethodPut, "/pregnancy-alert-rules/fetal_movement", map[string]any{
		"enabled": "maybe", "level": "loud", "window_days": 0,
		"params": map[string]any{"from_week": 50, "statuses": []any{"reduced", "reduced", "gone"}},
		"texts": map[string]any{"en": map[string]any{"title": "", "actions": []any{
			map[string]any{"key": "ack", "label": "a"}, map[string]any{"key": "ack", "label": "b"},
		}}},
	})
	require.Equal(t, 422, r.Status)
	for _, f := range []string{"enabled", "level", "window_days", "params.from_week", "params.statuses.1",
		"params.statuses.2", "texts.en.title", "texts.en.advice", "texts.en.actions.1.key"} {
		assert.Contains(t, r.Errors(), f)
	}

	// A language without a row gets one when its texts are sent; the default language's
	// texts are required while its row is missing.
	e.Exec("DELETE FROM message_contents WHERE `group` = 'pregnancy_alert' AND item_key = 'week_entered'")
	body := map[string]any{"enabled": true, "level": "info", "window_days": 7, "params": map[string]any{}}
	r = c.JSON(fiber.MethodPut, "/pregnancy-alert-rules/week_entered", body)
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "texts.fa")
	body["texts"] = map[string]any{"fa": map[string]any{
		"title": "هفته {week}", "what_we_saw": "w", "how_sure": "h", "advice": "a", "actions": []any{},
	}}
	r = c.JSON(fiber.MethodPut, "/pregnancy-alert-rules/week_entered", body)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, []any{"en"}, r.Obj("rule")["missing_locales"])
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM message_contents WHERE `group` = 'pregnancy_alert' AND item_key = 'week_entered' AND is_active = 1 AND is_approved = 1"))
	assert.Equal(t, map[string]any{}, payload(e, "week_entered", "fa")["params"])
}
