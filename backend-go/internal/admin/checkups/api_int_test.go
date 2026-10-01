package checkups_test

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	admincheckups "github.com/ritme/backend-go/internal/admin/checkups"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	admincheckups.New(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	// These tests pin the M4 default catalog; audience-scoped seeds (menopause, goose 00022) are not part of it.
	_, err := e.DB.Exec("DELETE FROM checkup_types WHERE user_id IS NULL AND audiences IS NOT NULL")
	require.NoError(t, err)
	return e
}

func newUser(e *admintest.Env, mobile string) uint64 {
	e.T.Helper()
	id, err := e.Exec("INSERT INTO users (mobile, created_at, updated_at) VALUES (?, NOW(), NOW())", mobile).LastInsertId()
	require.NoError(e.T, err)
	return uint64(id) //nolint:gosec // test ids
}

func typeID(e *admintest.Env, key string) int {
	e.T.Helper()
	return e.Int("SELECT id FROM checkup_types WHERE `key` = ?", key)
}

func path(id int) string { return "/checkup-types/" + strconv.Itoa(id) }

func customType(e *admintest.Env, userID uint64) int {
	e.T.Helper()
	res := e.Exec(`INSERT INTO checkup_types (user_id, category, title, performed_by, interval_months, created_at, updated_at)
		VALUES (?, 'custom', '{"fa":"سفارشی","en":"Mine"}', 'doctor', 12, NOW(), NOW())`, userID)
	id, err := res.LastInsertId()
	require.NoError(e.T, err)
	return int(id)
}

func today() civildate.Date { return civildate.InTehran(time.Now()) }

// validBody is a complete create/update form.
func validBody() map[string]any {
	return map[string]any{
		"key":             "eye_exam",
		"category":        "multi_year",
		"title":           map[string]any{"fa": "معاینه چشم", "en": "Eye exam"},
		"subtitle":        map[string]any{"fa": "بینایی\u200cسنجی", "en": "Vision check"},
		"why":             map[string]any{"fa": "چرا", "en": "Why"},
		"performed_by":    "doctor",
		"icon":            "stetho",
		"tone":            "teal",
		"interval_months": 24,
		"age_min":         18,
		"prep_steps": []any{
			map[string]any{"fa": "عینک را بیاور", "en": "Bring your glasses", "de": "ignored"},
		},
		"guide_steps": []any{
			map[string]any{"title": map[string]any{"fa": "گام", "en": "Step"}, "body": map[string]any{"fa": "متن", "en": "Text"}},
		},
		"finding_options": []any{
			map[string]any{"key": "none", "exclusive": true, "label": map[string]any{"fa": "هیچ", "en": "None"}},
			map[string]any{"key": "blur", "label": map[string]any{"fa": "تاری", "en": "Blur"}},
		},
		"is_active":   true,
		"source_note": "Reviewed",
	}
}

func without(m map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	for _, k := range keys {
		delete(out, k)
	}
	return out
}

func with(m map[string]any, kv ...any) map[string]any {
	out := without(m)
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i].(string)] = kv[i+1]
	}
	return out
}

// ---------------------------------------------------------------------------

func TestGuards(t *testing.T) {
	e := newEnv(t)
	r := e.Anonymous().Get("/checkup-types")
	assert.Equal(t, 401, r.Status)
	assert.Equal(t, "unauthenticated", r.Code())

	editor := e.As(admintest.EditorID)
	assert.Equal(t, 200, editor.Get("/checkup-types").Status, "editors manage the catalog")
	assert.Equal(t, 200, e.As(admintest.SuperID).Get("/checkup-types/stats").Status)

	noToken := e.As(admintest.EditorID)
	noToken.CSRF = ""
	r = noToken.JSON(fiber.MethodPost, "/checkup-types", validBody())
	assert.Equal(t, 419, r.Status)
	assert.Equal(t, "csrf_mismatch", r.Code())
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/checkup-types/reorder", map[string]any{"ids": []int{1}}).Status)

	wrong := e.As(admintest.EditorID)
	wrong.CSRF = "wrong"
	assert.Equal(t, 419, wrong.JSON(fiber.MethodDelete, path(typeID(e, "dentist")), nil).Status)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM checkup_types WHERE `key` = 'dentist'"))

	// Custom (user-owned) types are not part of the admin catalog.
	custom := customType(e, newUser(e, "09120000001"))
	assert.Equal(t, 404, editor.Get(path(custom)).Status)
	assert.Equal(t, 404, editor.JSON(fiber.MethodPut, path(custom), with(validBody(), "key", nil)).Status)
	r = editor.JSON(fiber.MethodDelete, path(custom), nil)
	assert.Equal(t, 404, r.Status)
	assert.Equal(t, "not_found", r.Code())
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM checkup_types WHERE id = ?", custom))
	assert.Equal(t, 404, editor.Get("/checkup-types/abc").Status)
	assert.Equal(t, 404, editor.Get("/checkup-types/999999").Status)
}

func TestListShowOptions(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	customType(e, newUser(e, "09120000002"))

	r := c.Get("/checkup-types")
	require.Equal(t, 200, r.Status, r.Body)
	items := r.Items()
	require.Len(t, items, 6, "custom types are excluded")
	keys := []string{}
	for _, it := range items {
		keys = append(keys, it.(map[string]any)["key"].(string))
	}
	assert.Equal(t, []string{"breast_self_exam", "clinical_breast_exam", "pap_smear", "blood_test", "dentist", "mammography"}, keys)
	assert.Equal(t, map[string]any{"current_page": 1.0, "last_page": 1.0, "per_page": 20.0, "total": 6.0}, r.Data()["meta"])
	assert.Equal(t, map[string]any{"q": "", "status": "all"}, r.Data()["filters"])

	first := items[0].(map[string]any)
	assert.Equal(t, map[string]any{"fa": "خودآزمایی سینه", "en": "Breast self-exam"}, first["title"])
	assert.Len(t, first["guide_steps"], 3)
	assert.Len(t, first["finding_options"], 5)
	assert.Equal(t, 7.0, first["cycle_day_from"])
	assert.Equal(t, 0.0, first["records_count"])
	second := items[1].(map[string]any)
	assert.Equal(t, []any{}, second["guide_steps"], "NULL lists are []")

	r = c.Get("/checkup-types?q=mammo")
	require.Len(t, r.Items(), 1)
	r = c.Get("/checkup-types?q=" + url.QueryEscape("دندان"))
	require.Len(t, r.Items(), 1, "searches the Persian title")
	assert.Equal(t, "dentist", r.Items()[0].(map[string]any)["key"])
	assert.Empty(t, c.Get("/checkup-types?status=inactive").Items())
	assert.Len(t, c.Get("/checkup-types?status=active&per_page=2").Items(), 2)

	r = c.Get(path(typeID(e, "pap_smear")))
	require.Equal(t, 200, r.Status)
	pap := r.Obj("checkup_type")
	assert.Equal(t, 21.0, pap["age_min"])
	assert.Equal(t, 65.0, pap["age_max"])
	assert.Len(t, pap["prep_steps"], 3)
	assert.NotNil(t, pap["created_at"])

	r = c.Get("/checkup-types/options")
	require.Equal(t, 200, r.Status)
	o := r.Data()
	assert.Equal(t, []any{"monthly", "six_monthly", "annual", "multi_year", "age_based"}, o["categories"])
	assert.Equal(t, []any{"self", "doctor", "lab", "dentist"}, o["performed_by"])
	assert.Contains(t, o["icons"], "shieldCheck")
	assert.Equal(t, 7.0, o["next_sort_order"])
	assert.Equal(t, 45.0, o["max_cycle_day"])
}

func TestValidation(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.JSON(fiber.MethodPost, "/checkup-types", map[string]any{})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, "validation_failed", r.Code())
	for _, f := range []string{"key", "title", "title.fa", "title.en", "category", "performed_by", "interval_months"} {
		assert.Contains(t, r.Errors(), f)
	}

	eleven := make([]any, 11)
	for i := range eleven {
		eleven[i] = map[string]any{"fa": "x", "en": "y"}
	}
	r = c.JSON(fiber.MethodPost, "/checkup-types", with(validBody(),
		"key", "Bad Key",
		"title", map[string]any{"fa": "فقط فارسی"},
		"subtitle", map[string]any{"fa": "زیرعنوان"},
		"category", "custom",
		"icon", "rocket",
		"tone", "black",
		"interval_months", 0,
		"interval_months_max", 200,
		"age_min", 50, "age_max", 40,
		"cycle_day_from", 20, "cycle_day_to", 50,
		"prep_steps", eleven,
		"guide_steps", []any{map[string]any{"title": map[string]any{"fa": "x"}}},
		"finding_options", []any{
			map[string]any{"key": "a", "label": map[string]any{"fa": "x", "en": "y"}},
			map[string]any{"key": "a", "label": map[string]any{"fa": "x", "en": "y"}},
		},
	))
	require.Equal(t, 422, r.Status)
	errs := r.Errors()
	for _, f := range []string{
		"key", "title.en", "subtitle.en", "category", "icon", "tone", "interval_months", "interval_months_max",
		"age_max", "cycle_day_to", "prep_steps", "guide_steps.0.title.en", "guide_steps.0.body",
		"finding_options.1.key",
	} {
		assert.Contains(t, errs, f, f)
	}
	assert.NotContains(t, errs, "title.fa")
	assert.NotContains(t, errs, "why.en")

	// Cross-field rules on otherwise valid input.
	r = c.JSON(fiber.MethodPost, "/checkup-types", with(validBody(), "cycle_day_from", 10, "interval_months_max", 12))
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "cycle_day_to", "both ends of the cycle window")
	assert.Contains(t, r.Errors(), "interval_months_max", "max ≥ interval_months (24)")
	r = c.JSON(fiber.MethodPost, "/checkup-types", with(validBody(), "cycle_day_from", 12, "cycle_day_to", 10))
	assert.Contains(t, r.Errors(), "cycle_day_to")

	// Unique key (seeded rows included).
	r = c.JSON(fiber.MethodPost, "/checkup-types", with(validBody(), "key", "dentist"))
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "key")
}

func TestCreateUpdateVisibleToUserAPI(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	ctx := context.Background()

	r := c.JSON(fiber.MethodPost, "/checkup-types", validBody())
	require.Equal(t, 201, r.Status, r.Body)
	ct := r.Obj("checkup_type")
	id := int(ct["id"].(float64))
	assert.Equal(t, "eye_exam", ct["key"])
	assert.Equal(t, 7.0, ct["sort_order"], "appended after the seeded catalog")
	assert.Equal(t, 7.0, ct["remind_lead_days"], "default lead")
	assert.Equal(t, []any{map[string]any{"fa": "عینک را بیاور", "en": "Bring your glasses"}}, ct["prep_steps"],
		"only active languages are stored")
	assert.Equal(t, []any{
		map[string]any{"key": "none", "exclusive": true, "label": map[string]any{"fa": "هیچ", "en": "None"}},
		map[string]any{"key": "blur", "label": map[string]any{"fa": "تاری", "en": "Blur"}},
	}, ct["finding_options"])
	assert.Nil(t, ct["cycle_day_from"])
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM checkup_types WHERE `key` = 'eye_exam' AND user_id IS NULL"))

	// The user API's catalog query sees it at once (no cache between admin and user API).
	userID := newUser(e, "09120000003")
	in, err := checkups.EngineInputs(ctx, store.New(e.DB), userID)
	require.NoError(t, err)
	assert.Len(t, in.Types, 7)

	// Update: new title and interval; absent optional fields keep their value; key is fixed.
	body := without(validBody(), "subtitle", "why", "prep_steps", "age_min", "source_note")
	body["key"] = "renamed"
	body["title"] = map[string]any{"fa": "چشم\u200cپزشکی", "en": "Eye doctor"}
	body["interval_months"] = 12
	body["cycle_day_from"], body["cycle_day_to"] = 5, 9
	r = c.JSON(fiber.MethodPut, path(id), body)
	require.Equal(t, 200, r.Status, r.Body)
	ct = r.Obj("checkup_type")
	assert.Equal(t, "eye_exam", ct["key"])
	assert.Equal(t, map[string]any{"fa": "چشم\u200cپزشکی", "en": "Eye doctor"}, ct["title"])
	assert.Equal(t, map[string]any{"fa": "بینایی\u200cسنجی", "en": "Vision check"}, ct["subtitle"], "kept")
	assert.Equal(t, 18.0, ct["age_min"], "kept")
	assert.Len(t, ct["prep_steps"], 1, "kept")
	assert.Equal(t, "Reviewed", ct["source_note"], "kept")
	assert.Equal(t, 5.0, ct["cycle_day_from"])
	assert.Equal(t, 7.0, ct["sort_order"], "kept")

	row, err := store.New(e.DB).GetAdminCheckupType(ctx, uint64(id)) //nolint:gosec // test id
	require.NoError(t, err)
	var title map[string]string
	require.NoError(t, json.Unmarshal(row.Title, &title))
	assert.Equal(t, "Eye doctor", title["en"])
	in, err = checkups.EngineInputs(ctx, store.New(e.DB), userID)
	require.NoError(t, err)
	for _, ty := range in.Types {
		if ty.ID == uint64(id) { //nolint:gosec // test id
			assert.Equal(t, 12, ty.IntervalMonths, "the engine reads the edited interval")
			assert.Equal(t, 5, ty.CycleDayFrom)
		}
	}

	// Sending null clears; the age window is checked against the stored value when absent.
	r = c.JSON(fiber.MethodPut, path(id), with(body, "subtitle", nil, "age_max", 10))
	require.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "age_max", "age_max ≥ stored age_min (18)")
	r = c.JSON(fiber.MethodPut, path(id), with(body, "subtitle", nil, "prep_steps", []any{}, "is_active", false))
	require.Equal(t, 200, r.Status, r.Body)
	ct = r.Obj("checkup_type")
	assert.Nil(t, ct["subtitle"])
	assert.Equal(t, []any{}, ct["prep_steps"])
	assert.Equal(t, false, ct["is_active"])

	// Deactivated → gone from the user API's catalog.
	in, err = checkups.EngineInputs(ctx, store.New(e.DB), userID)
	require.NoError(t, err)
	assert.Len(t, in.Types, 6)
}

func TestDeleteRefusedWhileInUse(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	dentist := typeID(e, "dentist")
	u := newUser(e, "09120000004")
	e.Exec("INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, created_at, updated_at) VALUES (?, ?, ?, 'normal', NOW(), NOW())",
		u, dentist, today().AddDays(-10).String())

	r := c.JSON(fiber.MethodDelete, path(dentist), nil)
	require.Equal(t, 422, r.Status, r.Body)
	assert.Equal(t, "in_use", r.Code())
	assert.Equal(t, 1.0, r.Body["records_count"])
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM checkup_records WHERE checkup_type_id = ?", dentist), "records untouched")
	assert.Equal(t, 1.0, c.Get(path(dentist)).Obj("checkup_type")["records_count"])

	// An unused type can be deleted (its settings rows go with it).
	r = c.JSON(fiber.MethodPost, "/checkup-types", validBody())
	require.Equal(t, 201, r.Status)
	id := int(r.Obj("checkup_type")["id"].(float64))
	e.Exec("INSERT INTO user_checkup_settings (user_id, checkup_type_id, enabled, remind) VALUES (?, ?, 0, 0)", u, id)
	r = c.JSON(fiber.MethodDelete, path(id), nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, float64(id), r.Data()["id"])
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM checkup_types WHERE id = ?", id))
	assert.Equal(t, 404, c.JSON(fiber.MethodDelete, path(id), nil).Status)
}

func TestReorder(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	custom := customType(e, newUser(e, "09120000005"))
	keys := []string{"mammography", "dentist", "blood_test", "pap_smear", "clinical_breast_exam", "breast_self_exam"}
	ids := make([]int, 0, len(keys))
	for _, k := range keys {
		ids = append(ids, typeID(e, k))
	}

	r := c.JSON(fiber.MethodPost, "/checkup-types/reorder", map[string]any{"ids": ids})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Len(t, r.Items(), 6)
	list := c.Get("/checkup-types").Items()
	got := []string{}
	for _, it := range list {
		got = append(got, it.(map[string]any)["key"].(string))
	}
	assert.Equal(t, keys, got)
	assert.Equal(t, 1, e.Int("SELECT sort_order FROM checkup_types WHERE `key` = 'mammography'"))

	for name, body := range map[string]any{
		"partial":   map[string]any{"ids": ids[:3]},
		"duplicate": map[string]any{"ids": append(append([]int{}, ids[:5]...), ids[0])},
		"custom":    map[string]any{"ids": append(append([]int{}, ids[:5]...), custom)},
		"missing":   map[string]any{},
		"strings":   map[string]any{"ids": []any{"a"}},
	} {
		r = c.JSON(fiber.MethodPost, "/checkup-types/reorder", body)
		assert.Equal(t, 422, r.Status, name)
		assert.Equal(t, "validation_failed", r.Code(), name)
	}
	assert.Equal(t, 1, e.Int("SELECT sort_order FROM checkup_types WHERE `key` = 'mammography'"), "unchanged by failures")
}

func TestStats(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	now := today()
	dentist, blood, self := typeID(e, "dentist"), typeID(e, "blood_test"), typeID(e, "breast_self_exam")
	rec := func(user uint64, typeID int, done civildate.Date, nextDue any) {
		e.Exec(`INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, next_due_on, created_at, updated_at)
			VALUES (?, ?, ?, 'normal', ?, NOW(), NOW())`, user, typeID, done.String(), nextDue)
	}
	u1, u2, u3, u4 := newUser(e, "09120000011"), newUser(e, "09120000012"), newUser(e, "09120000013"), newUser(e, "09120000014")

	rec(u1, dentist, now.AddDays(-400), nil) // old record …
	rec(u1, dentist, now.AddDays(-20), nil)  // … superseded by a recent one: not overdue
	rec(u2, dentist, now.AddDays(-250), nil) // 6 months passed: overdue
	rec(u3, dentist, now.AddDays(-250), nil) // overdue, but switched off
	e.Exec("INSERT INTO user_checkup_settings (user_id, checkup_type_id, enabled, remind) VALUES (?, ?, 0, 0)", u3, dentist)
	rec(u4, blood, now.AddDays(-40), now.AddDays(-1).String()) // user override in the past: overdue
	rec(u4, self, now.AddDays(-5), nil)
	custom := customType(e, u4)
	rec(u4, custom, now.AddDays(-5), nil) // custom types are not reported

	r := c.Get("/checkup-types/stats")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 30.0, r.Data()["window_days"])
	assert.Equal(t, now.String(), r.Data()["today"])
	items := r.Items()
	require.Len(t, items, 6)
	by := map[string]map[string]any{}
	for _, it := range items {
		m := it.(map[string]any)
		by[m["key"].(string)] = m
	}
	assert.Equal(t, 4.0, by["dentist"]["records_total"])
	assert.Equal(t, 3.0, by["dentist"]["users_with_records"])
	assert.Equal(t, 1.0, by["dentist"]["records_last_30_days"])
	assert.Equal(t, 1.0, by["dentist"]["overdue_users"])
	assert.Equal(t, 1.0, by["blood_test"]["overdue_users"])
	assert.Equal(t, 0.0, by["blood_test"]["records_last_30_days"])
	assert.Equal(t, 1.0, by["breast_self_exam"]["records_last_30_days"])
	assert.Equal(t, 0.0, by["breast_self_exam"]["overdue_users"])
	assert.Equal(t, 0.0, by["mammography"]["users_with_records"])
	assert.Equal(t, map[string]any{"fa": "دندان\u200cپزشکی", "en": "Dentist"}, by["dentist"]["title"])
}
