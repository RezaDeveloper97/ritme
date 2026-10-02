package companions_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/companions"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	companions.New(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	return e
}

func newUser(e *admintest.Env, mobile, name string) uint64 {
	e.T.Helper()
	id, err := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES (?, ?, NOW(), NOW())",
		mobile, name).LastInsertId()
	require.NoError(e.T, err)
	return uint64(id) //nolint:gosec // test ids
}

func newLink(e *admintest.Env, owner uint64, comp any, typ, status string, label any, created string) int64 {
	e.T.Helper()
	id, err := e.Exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, display_name, invited_at,
		accepted_at, revoked_at, revoked_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, IF(? = 'invited', NULL, ?), IF(? = 'revoked', ?, NULL), IF(? = 'revoked', 'owner', NULL), ?, ?)`,
		owner, comp, typ, status, label, created, status, created, status, created, status, created, created).LastInsertId()
	require.NoError(e.T, err)
	return id
}

func phaseOf(t *testing.T, r admintest.Resp, phase string) map[string]any {
	t.Helper()
	for _, it := range r.Items() {
		if m := it.(map[string]any); m["phase"] == phase {
			return m
		}
	}
	t.Fatalf("phase %s missing", phase)
	return nil
}

func texts(m map[string]any, code string) map[string]any {
	return m["texts"].(map[string]any)[code].(map[string]any)
}

func TestGuardsAndCSRF(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/companions/tips", "/companions/tips/luteal", "/companions/links"} {
		r := e.Anonymous().Get(path)
		assert.Equal(t, 401, r.Status, path)
		assert.Equal(t, "unauthenticated", r.Code(), path)
	}
	ed := e.As(admintest.EditorID)
	assert.Equal(t, 200, ed.Get("/companions/tips").Status, "editors manage the tips")
	assert.Equal(t, 200, ed.Get("/companions/links").Status, "any active admin reads the overview")

	noToken := e.As(admintest.SuperID)
	noToken.CSRF = ""
	body := map[string]any{"texts": map[string]any{"fa": map[string]any{"note": "x", "tips": []any{}}}}
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPut, "/companions/tips/luteal", body).Status)
	assert.Equal(t, 419, noToken.JSON(fiber.MethodDelete, "/companions/tips/luteal?locale=fa", nil).Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM message_contents WHERE `group` = 'companion_tip'"))

	assert.Equal(t, 404, ed.Get("/companions/tips/bogus").Status)
	assert.Equal(t, 404, ed.JSON(fiber.MethodPut, "/companions/tips/bogus", body).Status)

	e.Exec("UPDATE admins SET is_active = 0 WHERE id = ?", admintest.EditorID)
	assert.Equal(t, 401, e.As(admintest.EditorID).Get("/companions/links").Status, "a deactivated admin is out")
}

func TestTipsBuiltInThenSaveAndReset(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.Get("/companions/tips")
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, []any{"menstrual", "follicular", "fertile", "luteal", "pregnancy", "general"}, r.Data()["phases"])
	assert.EqualValues(t, 3, r.Data()["tips_per_phase"])
	assert.Equal(t, []any{"name"}, r.Data()["placeholders"])
	require.Len(t, r.Items(), 6)
	fa := texts(phaseOf(t, r, "menstrual"), "fa")
	assert.Equal(t, false, fa["customized"])
	note := fa["note"].(map[string]any)
	assert.Equal(t, "built_in", note["source"])
	assert.Contains(t, note["body"], "{name}", "the built-in copy keeps the placeholder for the editor")
	tips := fa["tips"].([]any)
	require.Len(t, tips, 3)
	assert.Equal(t, "یک نوشیدنی گرم برایش آماده کن", tips[0].(map[string]any)["title"])
	en := texts(phaseOf(t, r, "menstrual"), "en")
	assert.NotEmpty(t, en["tips"].([]any)[0].(map[string]any)["title"])

	// Save fa: note + two tips (the third slot is emptied = hidden).
	body := map[string]any{"texts": map[string]any{"fa": map[string]any{
		"note": "  {name} این روزها خسته است.  ",
		"tips": []any{
			map[string]any{"title": "چای گرم", "body": "برای {name} چای دم کن."},
			map[string]any{"title": "Short walk", "body": nil},
		},
	}}}
	r = c.JSON(fiber.MethodPut, "/companions/tips/menstrual", body)
	require.Equal(t, 200, r.Status, r.Body)
	got := r.Obj("tips")
	fa = texts(got, "fa")
	assert.Equal(t, true, fa["customized"])
	assert.Equal(t, map[string]any{"body": "{name} این روزها خسته است.", "source": "locale"}, fa["note"])
	tips = fa["tips"].([]any)
	assert.Equal(t, map[string]any{"slot": float64(1), "title": "چای گرم", "body": "برای {name} چای دم کن.", "source": "locale"}, tips[0])
	assert.Equal(t, "", tips[2].(map[string]any)["title"], "unused slot emptied")
	assert.Len(t, got["rows"].([]any), 4)
	// en has no rows of its own: the default language's (fa) rows now win over the built-in copy, like the panel.
	en = texts(got, "en")
	assert.Equal(t, false, en["customized"])
	assert.Equal(t, "default_language", en["note"].(map[string]any)["source"])

	assert.Equal(t, 4, e.Int("SELECT COUNT(*) FROM message_contents WHERE `group` = 'companion_tip' AND locale = 'fa' AND is_active = 1 AND is_approved = 1"))
	var p map[string]any
	require.NoError(t, json.Unmarshal([]byte(e.String("SELECT payload FROM message_contents WHERE item_key = 'menstrual_tip_2' AND locale = 'fa'")), &p))
	assert.Equal(t, map[string]any{"title": "Short walk", "body": ""}, p)

	// A second save updates the same rows (and makes a row disabled elsewhere live again).
	e.Exec("UPDATE message_contents SET is_active = 0 WHERE item_key = 'menstrual_note' AND locale = 'fa'")
	body["texts"].(map[string]any)["fa"].(map[string]any)["note"] = ""
	r = c.JSON(fiber.MethodPut, "/companions/tips/menstrual", body)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 4, e.Int("SELECT COUNT(*) FROM message_contents WHERE `group` = 'companion_tip'"))
	assert.Equal(t, 1, e.Int("SELECT is_active FROM message_contents WHERE item_key = 'menstrual_note' AND locale = 'fa'"))
	assert.Equal(t, "", texts(r.Obj("tips"), "fa")["note"].(map[string]any)["body"], "empty note hides it")

	// Reset fa → back to the built-in copy.
	r = c.JSON(fiber.MethodDelete, "/companions/tips/menstrual?locale=fa", nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "built_in", texts(r.Obj("tips"), "fa")["note"].(map[string]any)["source"])
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM message_contents WHERE `group` = 'companion_tip'"))

	r = c.JSON(fiber.MethodDelete, "/companions/tips/menstrual?locale=xx", nil)
	assert.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "locale")
}

func TestTipsValidation(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.SuperID)
	cases := []struct {
		name  string
		body  any
		field string
	}{
		{"missing texts", map[string]any{}, "texts"},
		{"unknown language", map[string]any{"texts": map[string]any{"xx": map[string]any{"note": "a"}}}, "texts.xx"},
		{"too many tips", map[string]any{"texts": map[string]any{"fa": map[string]any{"tips": []any{
			map[string]any{"title": "a"}, map[string]any{"title": "b"}, map[string]any{"title": "c"}, map[string]any{"title": "d"},
		}}}}, "texts.fa.tips"},
		{"tip without title", map[string]any{"texts": map[string]any{"fa": map[string]any{"tips": []any{
			map[string]any{"title": "", "body": "x"},
		}}}}, "texts.fa.tips.0.title"},
		{"note too long", map[string]any{"texts": map[string]any{"fa": map[string]any{"note": strings.Repeat("ن", 501)}}}, "texts.fa.note"},
		{"body too long", map[string]any{"texts": map[string]any{"en": map[string]any{"tips": []any{
			map[string]any{"title": "a", "body": strings.Repeat("b", 1001)},
		}}}}, "texts.en.tips.0.body"},
	}
	for _, tc := range cases {
		r := c.JSON(fiber.MethodPut, "/companions/tips/general", tc.body)
		assert.Equal(t, 422, r.Status, tc.name)
		assert.Contains(t, r.Errors(), tc.field, tc.name)
	}
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM message_contents WHERE `group` = 'companion_tip'"))
}

func TestLinksOverviewMaskedAndCounted(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)
	owner := newUser(e, "09121234567", "سارا رضایی")
	owner2 := newUser(e, "09129876543", "Mina")
	man := newUser(e, "09351112233", "Ali Ahmadi")

	active := newLink(e, owner, man, "spouse", "active", "علی", "2026-09-20 10:00:00")
	invited := newLink(e, owner2, nil, "partner", "invited", "Reza", "2026-09-25 10:00:00")
	revoked := newLink(e, owner2, man, "partner", "revoked", nil, "2026-09-10 10:00:00")
	e.Exec(`INSERT INTO companion_invites (companion_id, owner_id, phone, code_hash, expires_at, created_at, updated_at)
		VALUES (?, ?, '09358889900', ?, '2037-01-01 00:00:00', NOW(), NOW())`, invited, owner2, strings.Repeat("a", 64))
	e.Exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at)
		VALUES (?, 'cycle', 'view', NOW(), NOW()), (?, 'pregnancy', 'view', NOW(), NOW())`, active, active)

	r := c.Get("/companions/links")
	require.Equal(t, 200, r.Status, r.Body)
	items := r.Items()
	require.Len(t, items, 3)
	assert.EqualValues(t, invited, items[0].(map[string]any)["id"], "newest first")
	assert.EqualValues(t, revoked, items[2].(map[string]any)["id"])

	a := items[1].(map[string]any)
	assert.Equal(t, "spouse", a["type"])
	assert.Equal(t, "active", a["status"])
	assert.Equal(t, "ع•••", a["label"])
	assert.Equal(t, map[string]any{"id": float64(owner), "name": "س•••", "mobile": "0912•••4567"}, a["owner"])
	assert.Equal(t, map[string]any{"id": float64(man), "name": "A•••", "mobile": "0935•••2233"}, a["companion"])
	assert.EqualValues(t, 2, a["grants_count"])
	assert.Nil(t, a["invite"])

	inv := items[0].(map[string]any)
	assert.Nil(t, inv["companion"])
	assert.Equal(t, map[string]any{"phone": "0935•••9900", "expires_at": "2037-01-01T00:00:00+03:30", "expired": false}, inv["invite"])
	assert.Equal(t, "owner", items[2].(map[string]any)["revoked_by"])

	// Nothing identifying in full, no code, no shared sections.
	raw, _ := json.Marshal(r.Body)
	for _, leak := range []string{"09121234567", "09358889900", "09351112233", "سارا", "Ali Ahmadi", "Reza", "code", "aaaa", "cycle", "pregnancy", "section", "grants\""} {
		assert.NotContains(t, string(raw), leak)
	}

	counts := r.Data()["counts"].(map[string]any)
	assert.Equal(t, map[string]any{"all": float64(3), "invited": float64(1), "active": float64(1), "revoked": float64(1)}, counts["by_status"])
	assert.Equal(t, map[string]any{
		"partner": map[string]any{"invited": float64(1), "active": float64(0), "revoked": float64(1), "all": float64(2)},
		"spouse":  map[string]any{"invited": float64(0), "active": float64(1), "revoked": float64(0), "all": float64(1)},
	}, counts["by_type"])
	assert.Equal(t, map[string]any{"status": "all", "type": "all"}, r.Data()["filters"])

	r = c.Get("/companions/links?status=revoked")
	require.Len(t, r.Items(), 1)
	assert.EqualValues(t, revoked, r.Items()[0].(map[string]any)["id"])
	assert.EqualValues(t, 1, r.Data()["meta"].(map[string]any)["total"])
	assert.EqualValues(t, 3, r.Data()["counts"].(map[string]any)["by_status"].(map[string]any)["all"], "counts ignore the filter")

	r = c.Get("/companions/links?type=partner&status=invited")
	require.Len(t, r.Items(), 1)
	assert.EqualValues(t, invited, r.Items()[0].(map[string]any)["id"])

	r = c.Get("/companions/links?status=bogus&type=bogus&per_page=2&page=2")
	require.Len(t, r.Items(), 1)
	assert.Equal(t, map[string]any{"status": "all", "type": "all"}, r.Data()["filters"])

	// An expired open invite is flagged.
	e.Exec("UPDATE companion_invites SET expires_at = '2020-01-01 00:00:00'")
	r = c.Get("/companions/links?status=invited")
	assert.Equal(t, true, r.Items()[0].(map[string]any)["invite"].(map[string]any)["expired"])
}
