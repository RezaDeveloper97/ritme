package children_test

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adminchildren "github.com/ritme/backend-go/internal/admin/children"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/messages"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/children"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/seeds/who"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// newEnv mounts this module plus the catalog and messages APIs wired as in production (child catalogs and postpartum
// copy are super-only writes).
func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	adminchildren.New(admintest.Quiet).Routes(e.Route(), e.Kit)
	catalog.NewAdmin(e.DB, nil, admintest.Quiet).WithSuperGroups(children.Groups...).Routes(e.Route(), e.Kit)
	messages.New(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	return e
}

func TestWHO_GuardsAndDefaults(t *testing.T) {
	e := newEnv(t)
	r := e.Anonymous().Get("/children/who")
	assert.Equal(t, 401, r.Status)
	assert.Equal(t, "unauthenticated", r.Code())

	r = e.As(admintest.EditorID).Get("/children/who")
	require.Equal(t, 200, r.Status)
	f := r.Obj("filters")
	assert.Equal(t, "weight", f["indicator"])
	assert.Equal(t, "girl", f["sex"])
	assert.Equal(t, "month", f["step"])
	assert.Equal(t, "kg", r.Data()["unit"])
	assert.Equal(t, true, r.Data()["read_only"])
	assert.Equal(t, who.Source, r.Data()["source"])
	meta := r.Obj("meta")
	assert.InDelta(t, 61, meta["total"], 0, "months 0–60")
	assert.InDelta(t, 20, meta["per_page"], 0)
	require.Len(t, r.Items(), 20)

	// Row 0 is the WHO birth row; P50 is the median M and P3 < P50 < P97.
	first := r.Items()[0].(map[string]any)
	lms, ok := who.At(who.Weight, who.Girl, 0)
	require.True(t, ok)
	assert.InDelta(t, 0, first["age"], 0)
	assert.InDelta(t, 0, first["day"], 0)
	assert.InDelta(t, lms.L, first["l"], 1e-9)
	assert.InDelta(t, lms.M, first["m"], 1e-9)
	assert.InDelta(t, lms.S, first["s"], 1e-9)
	assert.InDelta(t, lms.M, first["p50"], 0.001)
	assert.InDelta(t, 2.4, first["p3"], 0.05, "WHO girls weight-for-age P3 at birth ≈ 2.4 kg")
	assert.InDelta(t, 4.2, first["p97"], 0.05, "WHO girls weight-for-age P97 at birth ≈ 4.2 kg")
}

func TestWHO_StepsPagingAndValidation(t *testing.T) {
	e := newEnv(t)
	c := e.As(admintest.EditorID)

	r := c.Get("/children/who?indicator=head&sex=boy&step=week&per_page=100&page=3")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, "cm", r.Data()["unit"])
	assert.InDelta(t, 266, r.Obj("meta")["total"], 0, "weeks 0–265")
	row := r.Items()[0].(map[string]any)
	assert.InDelta(t, 200, row["age"], 0)
	assert.InDelta(t, 1400, row["day"], 0)

	r = c.Get("/children/who?indicator=length&step=day&per_page=100&page=19")
	require.Equal(t, 200, r.Status)
	assert.InDelta(t, who.MaxDay+1, r.Obj("meta")["total"], 0)
	last := r.Items()[len(r.Items())-1].(map[string]any)
	assert.InDelta(t, who.MaxDay, last["day"], 0)

	// The month sampling is round(month × 30.4375).
	pts := adminchildren.Points(adminchildren.StepMonth)
	assert.Equal(t, 731, pts[24].Day, "24 × 30.4375 = 730.5 → 731")
	assert.Equal(t, 1826, pts[60].Day)

	// Past the last page: empty items, same total.
	r = c.Get("/children/who?page=9")
	require.Equal(t, 200, r.Status)
	assert.Empty(t, r.Items())

	r = c.Get("/children/who?indicator=bmi&sex=x&step=year")
	assert.Equal(t, 422, r.Status)
	for _, k := range []string{"indicator", "sex", "step"} {
		assert.Contains(t, r.Errors(), k)
	}
}

func vaccineBody(code string) map[string]any {
	return map[string]any{
		"code":  code,
		"title": map[string]any{"fa": "واکسن آزمایشی", "en": "Test vaccine"},
		"body":  map[string]any{"fa": "سل", "en": "Tuberculosis"},
		"meta":  map[string]any{"visit": "m2", "age_months": 2},
	}
}

// The child catalogs are clinical: editors read them, only super admins write (create, update, reorder, delete).
func TestChildCatalog_SuperOnlyWrites(t *testing.T) {
	e := newEnv(t)
	ed, su := e.As(admintest.EditorID), e.As(admintest.SuperID)
	before := e.Int("SELECT COUNT(*) FROM catalog_items WHERE `group` = 'child_vaccines'")

	assert.Equal(t, 200, ed.Get("/catalog/child_vaccines").Status, "editors read")
	assert.Equal(t, 403, ed.JSON(fiber.MethodPost, "/catalog/child_vaccines", vaccineBody("zz_test")).Status)
	assert.Equal(t, before, e.Int("SELECT COUNT(*) FROM catalog_items WHERE `group` = 'child_vaccines'"))

	r := su.JSON(fiber.MethodPost, "/catalog/child_vaccines", vaccineBody("zz_test"))
	require.Equal(t, 201, r.Status, r.Body)
	item := r.Obj("catalog_item")
	assert.Equal(t, true, item["needs_review"], "new clinical rows start unreviewed")
	id := uint64(item["id"].(float64))
	path := fmt.Sprintf("/catalog/child_vaccines/%d", id)

	review := map[string]any{"title": map[string]any{"fa": "واکسن آزمایشی"}, "needs_review": false}
	assert.Equal(t, 403, ed.JSON(fiber.MethodPut, path, review).Status)
	assert.Equal(t, 1, e.Int("SELECT needs_review FROM catalog_items WHERE id = ?", id))
	r = su.JSON(fiber.MethodPut, path, review)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, false, r.Obj("catalog_item")["needs_review"], "«بازبینی شد»")
	assert.Equal(t, "m2", r.Obj("catalog_item")["meta"].(map[string]any)["visit"], "absent meta is kept")

	ids := []any{}
	for _, it := range su.Get("/catalog/child_vaccines?per_page=100").Items() {
		ids = append(ids, it.(map[string]any)["id"])
	}
	assert.Equal(t, 403, ed.JSON(fiber.MethodPost, "/catalog/child_vaccines/reorder", map[string]any{"ids": ids}).Status)
	assert.Equal(t, 200, su.JSON(fiber.MethodPost, "/catalog/child_vaccines/reorder", map[string]any{"ids": ids}).Status)

	assert.Equal(t, 403, ed.JSON(fiber.MethodDelete, path, nil).Status)
	assert.Equal(t, 200, su.JSON(fiber.MethodDelete, path, nil).Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM catalog_items WHERE id = ?", id))

	// CSRF still applies to super admins.
	noToken := e.As(admintest.SuperID)
	noToken.CSRF = ""
	assert.Equal(t, 419, noToken.JSON(fiber.MethodPost, "/catalog/child_learn", vaccineBody("zz_csrf")).Status)

	// Every child group is gated; other groups keep editor writes.
	for _, g := range children.Groups {
		assert.Equal(t, 403, ed.JSON(fiber.MethodPost, "/catalog/"+g, vaccineBody("zz_gate")).Status, g)
	}
	r = ed.JSON(fiber.MethodPost, "/catalog/zz_editorial", vaccineBody("zz_open"))
	assert.Equal(t, 201, r.Status, r.Body)
}

// Postpartum copy (week tips, alerts, EPDS safety) is clinical: editors read, only super admins write.
func TestPostpartumMessages_SuperOnlyWrites(t *testing.T) {
	e := newEnv(t)
	ed, su := e.As(admintest.EditorID), e.As(admintest.SuperID)

	r := ed.Get("/messages?group=postpartum_week_tip")
	require.Equal(t, 200, r.Status)
	assert.Contains(t, r.Data()["super_only_groups"], "postpartum_safety")
	missing := r.Data()["missing"].([]any)
	require.NotEmpty(t, missing, "unseeded: every registered tip is missing")
	m := missing[0].(map[string]any)
	key := m["item_key"].(string)

	body := map[string]any{"group": "postpartum_week_tip", "item_key": key, "locale": "fa",
		"payload": map[string]any{"title": "هفته اول", "body": "استراحت کن"}}
	assert.Equal(t, 403, ed.JSON(fiber.MethodPost, "/messages", body).Status)
	r = su.JSON(fiber.MethodPost, "/messages", body)
	require.Equal(t, 201, r.Status, r.Body)
	id := uint64(r.Obj("message")["id"].(float64))
	path := fmt.Sprintf("/messages/%d", id)

	assert.Equal(t, 200, ed.Get(path).Status, "editors read")
	upd := map[string]any{"payload": map[string]any{"title": "هفته اول", "body": "استراحت و آب کافی"}}
	assert.Equal(t, 403, ed.JSON(fiber.MethodPut, path, upd).Status)
	assert.Equal(t, 403, ed.JSON(fiber.MethodPost, path+"/approve", nil).Status)
	assert.Equal(t, 403, ed.JSON(fiber.MethodPost, path+"/toggle", nil).Status)
	assert.Equal(t, 200, su.JSON(fiber.MethodPut, path, upd).Status)
	assert.Equal(t, 200, su.JSON(fiber.MethodPost, path+"/toggle", nil).Status)
	assert.Equal(t, 0, e.Int("SELECT is_active FROM message_contents WHERE id = ?", id))
}
