package catalog_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

// env is the admin test app with the public route mounted on the same DB and Redis, so admin writes
// and the public cache meet. Public requests carry a real Passport token.
type env struct {
	*admintest.Env
	token string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := admintest.New(t)
	reader := catalog.NewReader(store.New(e.DB), e.Cache, 0, admintest.Quiet)
	catalog.NewAdmin(e.DB, reader, admintest.Quiet).Routes(e.Route(), e.Kit)

	e.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`, clientID)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(e.DB)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, admintest.Quiet).RequireUser
	e.App.Get("/api/v1/catalog/:group", i18n.Middleware(e.Registry), guard, catalog.NewHandlers(reader).Group)

	id, err := e.Exec("INSERT INTO users (mobile, created_at, updated_at) VALUES ('09120000001', NOW(), NOW())").LastInsertId()
	require.NoError(t, err)
	tok, err := passport.NewIssuer(key, q, clock.Real{}, 365).Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
	require.NoError(t, err)
	return &env{Env: e, token: tok.AccessToken}
}

type pub struct {
	status int
	body   map[string]any
}

func (p pub) data() map[string]any { d, _ := p.body["data"].(map[string]any); return d }

func (p pub) items() []map[string]any {
	var out []map[string]any
	list, _ := p.data()["items"].([]any)
	for _, x := range list {
		out = append(out, x.(map[string]any))
	}
	return out
}

func codes(p pub) []string {
	out := []string{}
	for _, it := range p.items() {
		out = append(out, it["code"].(string))
	}
	return out
}

func (e *env) get(t *testing.T, path, lang string, token ...string) pub {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	tok := e.token
	if len(token) > 0 {
		tok = token[0]
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	res, err := e.App.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return pub{status: res.StatusCode, body: m}
}

func (e *env) seed(group, code string, sort int, active bool, audiences, title, body, meta any) {
	e.T.Helper()
	enc := func(v any) any {
		if v == nil {
			return nil
		}
		b, err := json.Marshal(v)
		require.NoError(e.T, err)
		return string(b)
	}
	e.Exec("INSERT INTO catalog_items (`group`, code, sort_order, is_active, audiences, title, body, meta, needs_review, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, NOW(), NOW())",
		group, code, sort, active, enc(audiences), enc(title), enc(body), enc(meta))
}

// testGroup returns a fresh, test-only catalog group name. Migrations seed real groups (missed_pill_rules, teen_*,
// meno_*, child_*, …), so admin tests write to a group of their own and assert only on rows they created.
func testGroup(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return "test_" + hex.EncodeToString(b)
}

// groupRow is the GET /catalog entry of group (nil when the group has no items).
func groupRow(groups []any, group string) map[string]any {
	for _, x := range groups {
		if g, _ := x.(map[string]any); g["group"] == group {
			return g
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Public read

func TestPublic_RequiresAuth(t *testing.T) {
	e := newEnv(t)
	r := e.get(t, "/api/v1/catalog/teen_faq", "", "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestPublic_LocalizedFilteredOrdered(t *testing.T) {
	e := newEnv(t)
	e.seed("warn", "bleeding", 2, true, nil,
		map[string]any{"fa": "خونریزی شدید", "en": "Heavy bleeding"},
		map[string]any{"fa": "فوراً با پزشک تماس بگیر", "en": "Call your doctor now"},
		map[string]any{"severity": "urgent", "cta": map[string]any{"fa": "تماس", "en": "Call"}})
	e.seed("warn", "fever", 1, true, []string{"pregnancy", "postpartum"},
		map[string]any{"fa": "تب", "en": ""}, nil, nil)
	e.seed("warn", "old", 0, false, nil, map[string]any{"fa": "قدیمی", "en": "Old"}, nil, nil)
	e.seed("other", "x", 0, true, nil, map[string]any{"fa": "دیگر"}, nil, nil)

	fa := e.get(t, "/api/v1/catalog/warn", "fa")
	require.Equal(t, 200, fa.status)
	assert.Equal(t, true, fa.body["success"])
	assert.Equal(t, "warn", fa.data()["group"])
	assert.Equal(t, "fa", fa.data()["locale"])
	assert.Equal(t, []string{"fever", "bleeding"}, codes(fa), "active only, sort_order then id, own group only")
	assert.Equal(t, "تب", fa.items()[0]["title"])
	assert.Nil(t, fa.items()[0]["body"])
	assert.Equal(t, []any{"pregnancy", "postpartum"}, fa.items()[0]["audiences"])
	assert.Equal(t, true, fa.items()[0]["needs_review"])
	assert.Equal(t, "فوراً با پزشک تماس بگیر", fa.items()[1]["body"])
	assert.Equal(t, map[string]any{"severity": "urgent", "cta": "تماس"}, fa.items()[1]["meta"])

	en := e.get(t, "/api/v1/catalog/warn", "en")
	require.Equal(t, 200, en.status)
	assert.Equal(t, "en", en.data()["locale"])
	assert.Equal(t, "تب", en.items()[0]["title"], "empty en falls back to the default language")
	assert.Equal(t, "Heavy bleeding", en.items()[1]["title"])
	assert.Equal(t, "Call your doctor now", en.items()[1]["body"])
	assert.Equal(t, map[string]any{"severity": "urgent", "cta": "Call"}, en.items()[1]["meta"])
	assert.Equal(t, "en", e.get(t, "/api/v1/catalog/warn?locale=en", "fa").data()["locale"], "?locale= wins")

	assert.Equal(t, []string{"fever", "bleeding"}, codes(e.get(t, "/api/v1/catalog/warn?audience=pregnancy", "fa")))
	assert.Equal(t, []string{"bleeding"}, codes(e.get(t, "/api/v1/catalog/warn?audience=menopause", "fa")),
		"items for everyone always match")

	empty := e.get(t, "/api/v1/catalog/nothing_here", "fa")
	assert.Equal(t, 200, empty.status)
	assert.Equal(t, []any{}, empty.data()["items"])

	assert.Equal(t, 404, e.get(t, "/api/v1/catalog/Bad-Group", "fa").status)
}

// ---------------------------------------------------------------------------
// Admin CRUD

func validBody() map[string]any {
	return map[string]any{
		"code":      "missed_one",
		"title":     map[string]any{"fa": "یک قرص جا افتاده", "en": "One pill missed", "de": "ignored"},
		"body":      map[string]any{"fa": "همین حالا بخور", "en": "Take it now"},
		"audiences": []any{"contraception", "contraception"},
		"meta":      map[string]any{"hours": 24, "steps": []any{map[string]any{"fa": "بخور", "en": "Take"}}},
	}
}

func TestAdmin_Guards(t *testing.T) {
	e := newEnv(t)
	r := e.Anonymous().Get("/catalog")
	assert.Equal(t, 401, r.Status)
	assert.Equal(t, "unauthenticated", r.Code())
	assert.Equal(t, 200, e.As(admintest.EditorID).Get("/catalog").Status, "editors manage the catalog")

	g := testGroup(t)
	before := e.Int("SELECT COUNT(*) FROM catalog_items")
	noCSRF := e.As(admintest.EditorID)
	noCSRF.CSRF = ""
	r = noCSRF.JSON(fiber.MethodPost, "/catalog/"+g, validBody())
	assert.Equal(t, 419, r.Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM catalog_items WHERE `group` = ?", g))
	assert.Equal(t, before, e.Int("SELECT COUNT(*) FROM catalog_items"), "nothing written")

	a := e.As(admintest.SuperID)
	assert.Equal(t, 404, a.Get("/catalog/Bad-Group").Status)
	assert.Equal(t, 404, a.Get("/catalog/g/abc").Status)
	assert.Equal(t, 404, a.Get("/catalog/g/999999").Status)
}

func TestAdmin_CRUDFlushesPublicCache(t *testing.T) {
	e := newEnv(t)
	a := e.As(admintest.EditorID)
	grp, other := testGroup(t), testGroup(t)
	g, pubPath := "/catalog/"+grp, "/api/v1/catalog/"+grp

	// Warm the public cache with the empty group.
	assert.Empty(t, e.get(t, pubPath, "en").items())

	r := a.JSON(fiber.MethodPost, g, validBody())
	require.Equal(t, 201, r.Status, string(r.Raw))
	assert.Equal(t, "Catalog item created.", r.Body["message"])
	it := r.Obj("catalog_item")
	id := int(it["id"].(float64))
	assert.Equal(t, grp, it["group"])
	assert.Equal(t, "missed_one", it["code"])
	assert.Equal(t, map[string]any{"fa": "یک قرص جا افتاده", "en": "One pill missed"}, it["title"], "unknown languages dropped")
	assert.Equal(t, []any{"contraception"}, it["audiences"], "deduplicated")
	assert.Equal(t, true, it["is_active"])
	assert.Equal(t, true, it["needs_review"], "new copy awaits clinical review by default")
	assert.InDelta(t, 1, it["sort_order"], 0)
	assert.Equal(t, "{\"hours\":24,\"steps\":[{\"en\":\"Take\",\"fa\":\"بخور\"}]}",
		e.String("SELECT meta FROM catalog_items WHERE id = ?", id), "stored as raw UTF-8 JSON")

	pubEN := e.get(t, pubPath, "en")
	require.Len(t, pubEN.items(), 1, "the create flushed the cached empty group")
	assert.Equal(t, "One pill missed", pubEN.items()[0]["title"])
	assert.Equal(t, map[string]any{"hours": float64(24), "steps": []any{"Take"}}, pubEN.items()[0]["meta"])

	// Duplicate code in the same group → 422; the same code in another group is fine.
	r = a.JSON(fiber.MethodPost, g, validBody())
	assert.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "code")
	assert.Equal(t, 201, a.JSON(fiber.MethodPost, "/catalog/"+other, validBody()).Status)

	// Validation: title in the default language, code format, audience format, booleans.
	bad := map[string]any{"code": "Bad Code", "title": map[string]any{"en": "Only en"},
		"audiences": []any{"Not Valid"}, "is_active": "maybe", "meta": "string"}
	r = a.JSON(fiber.MethodPost, g, bad)
	assert.Equal(t, 422, r.Status)
	for _, f := range []string{"code", "title.fa", "audiences.0", "is_active", "meta"} {
		assert.Contains(t, r.Errors(), f)
	}

	// Show / list (search, status) / groups.
	assert.Equal(t, "missed_one", a.Get(g + "/" + itoa(id)).Obj("catalog_item")["code"])
	assert.Equal(t, 404, a.Get("/catalog/"+other+"/"+itoa(id)).Status, "an id of another group is not found")
	list := a.Get(g + "?q=" + "pill")
	require.Equal(t, 200, list.Status)
	assert.Len(t, list.Items(), 1)
	assert.Empty(t, a.Get(g+"?status=inactive").Items())
	groups := a.Get("/catalog").Items()
	assert.Equal(t, map[string]any{"group": grp, "items_count": float64(1), "active_count": float64(1)}, groupRow(groups, grp))
	assert.Equal(t, map[string]any{"group": other, "items_count": float64(1), "active_count": float64(1)}, groupRow(groups, other))

	// Update: partial — absent optional fields keep their value; null clears.
	r = a.JSON(fiber.MethodPut, g+"/"+itoa(id), map[string]any{
		"title": map[string]any{"fa": "جاافتادن قرص", "en": "Missed pill"}, "meta": nil, "needs_review": false,
		"code": "ignored_on_update",
	})
	require.Equal(t, 200, r.Status, string(r.Raw))
	it = r.Obj("catalog_item")
	assert.Equal(t, "missed_one", it["code"], "code is fixed at creation")
	assert.Nil(t, it["meta"])
	assert.Equal(t, []any{"contraception"}, it["audiences"], "kept")
	assert.Equal(t, map[string]any{"fa": "همین حالا بخور", "en": "Take it now"}, it["body"], "kept")
	assert.Equal(t, false, it["needs_review"])
	assert.Equal(t, true, it["is_active"], "kept")
	pubFA := e.get(t, pubPath, "fa")
	assert.Equal(t, "جاافتادن قرص", pubFA.items()[0]["title"], "the update flushed the cache")
	assert.Equal(t, false, pubFA.items()[0]["needs_review"])

	// Deactivate → gone from the public read.
	require.Equal(t, 200, a.JSON(fiber.MethodPut, g+"/"+itoa(id), map[string]any{
		"title": map[string]any{"fa": "جاافتادن قرص"}, "is_active": false,
	}).Status)
	assert.Empty(t, e.get(t, pubPath, "fa").items())
	assert.Len(t, a.Get(g+"?status=inactive").Items(), 1)

	// Delete.
	r = a.JSON(fiber.MethodDelete, g+"/"+itoa(id), nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, "Catalog item deleted.", r.Body["message"])
	assert.Equal(t, 404, a.JSON(fiber.MethodDelete, g+"/"+itoa(id), nil).Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM catalog_items WHERE `group` = ?", grp))
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM catalog_items WHERE `group` = ?", other), "other group untouched")
}

func itoa(n int) string { return strconv.Itoa(n) }

// ---------------------------------------------------------------------------
// Reorder (CB-CORE-03b)

func (e *env) orders(group string) map[string]int {
	e.T.Helper()
	rows, err := e.DB.QueryContext(e.T.Context(), "SELECT code, sort_order FROM catalog_items WHERE `group` = ?", group)
	require.NoError(e.T, err)
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var code string
		var n int
		require.NoError(e.T, rows.Scan(&code, &n))
		out[code] = n
	}
	require.NoError(e.T, rows.Err())
	return out
}

func (e *env) id(group, code string) int {
	return e.Int("SELECT id FROM catalog_items WHERE `group` = ? AND code = ?", group, code)
}

func TestAdmin_Reorder(t *testing.T) {
	e := newEnv(t)
	grp, other := testGroup(t), testGroup(t)
	for i, code := range []string{"a", "b", "c"} {
		e.seed(grp, code, 0, i != 1, nil, map[string]any{"fa": code}, nil, nil) // colliding orders
	}
	e.seed(other, "x", 0, true, nil, map[string]any{"fa": "x"}, nil, nil)
	a, b, c, x := e.id(grp, "a"), e.id(grp, "b"), e.id(grp, "c"), e.id(other, "x")
	path, pubPath := "/catalog/"+grp+"/reorder", "/api/v1/catalog/"+grp

	// Guards: auth, CSRF, group format.
	assert.Equal(t, 401, e.Anonymous().JSON(fiber.MethodPost, path, map[string]any{"ids": []int{c, a, b}}).Status)
	noCSRF := e.As(admintest.EditorID)
	noCSRF.CSRF = ""
	assert.Equal(t, 419, noCSRF.JSON(fiber.MethodPost, path, map[string]any{"ids": []int{c, a, b}}).Status)
	ed := e.As(admintest.EditorID)
	assert.Equal(t, 404, ed.JSON(fiber.MethodPost, "/catalog/Bad-Group/reorder", map[string]any{"ids": []int{}}).Status)

	// Validation: every id of the group exactly once.
	for name, body := range map[string]map[string]any{
		"missing":     {},
		"partial":     {"ids": []int{c, a}},
		"duplicate":   {"ids": []int{c, a, a}},
		"other group": {"ids": []int{c, a, x}},
		"not ints":    {"ids": []any{"z", a, b}},
	} {
		r := ed.JSON(fiber.MethodPost, path, body)
		assert.Equal(t, 422, r.Status, name)
	}
	r := ed.JSON(fiber.MethodPost, path, map[string]any{"ids": []int{c, a, a}})
	assert.Contains(t, r.Errors(), "ids.2")
	assert.Equal(t, map[string]int{"a": 0, "b": 0, "c": 0}, e.orders(grp), "a 422 writes nothing")

	// Warm the public cache, then reorder (inactive rows included).
	assert.Equal(t, []string{"a", "c"}, codes(e.get(t, pubPath, "fa")))
	r = ed.JSON(fiber.MethodPost, path, map[string]any{"ids": []int{c, b, a}})
	require.Equal(t, 200, r.Status, string(r.Raw))
	assert.Equal(t, "Order saved.", r.Body["message"])
	assert.Equal(t, []any{
		map[string]any{"id": float64(c), "sort_order": float64(1)},
		map[string]any{"id": float64(b), "sort_order": float64(2)},
		map[string]any{"id": float64(a), "sort_order": float64(3)},
	}, r.Items())
	assert.Equal(t, map[string]int{"c": 1, "b": 2, "a": 3}, e.orders(grp))
	assert.Equal(t, map[string]int{"x": 0}, e.orders(other), "other group untouched")
	assert.Equal(t, []string{"c", "a"}, codes(e.get(t, pubPath, "fa")), "the reorder flushed the cache")
	assert.Equal(t, "b", e.String("SELECT code FROM catalog_items WHERE id = ? AND is_active = 0", b), "other columns kept")
}

// A failure on a later row rolls back the rows already written: all or nothing.
func TestAdmin_ReorderIsAtomic(t *testing.T) {
	e := newEnv(t)
	grp := testGroup(t)
	for i, code := range []string{"a", "b", "boom"} {
		e.seed(grp, code, i+1, true, nil, map[string]any{"fa": code}, nil, nil)
	}
	// grp is generated ([a-z0-9_] only), so interpolating it into the trigger body is safe.
	e.Exec(fmt.Sprintf("CREATE TRIGGER catalog_boom BEFORE UPDATE ON catalog_items FOR EACH ROW "+
		"IF NEW.`group` = '%s' AND NEW.code = 'boom' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'boom'; END IF", grp))
	a, b, boom := e.id(grp, "a"), e.id(grp, "b"), e.id(grp, "boom")

	// b → 1 is written first, then writing boom → 2 fails: b's write must be rolled back.
	r := e.As(admintest.EditorID).JSON(fiber.MethodPost, "/catalog/"+grp+"/reorder",
		map[string]any{"ids": []int{b, boom, a}})
	assert.Equal(t, 500, r.Status, string(r.Raw))
	assert.Equal(t, map[string]int{"a": 1, "b": 2, "boom": 3}, e.orders(grp), "nothing written")
}

// The duplicate-code 422 names the catalog item code, not the shared `code` label (the OTP code).
func TestAdmin_CodeAttributeLabel(t *testing.T) {
	e := newEnv(t)
	a := e.As(admintest.EditorID)
	g := "/catalog/" + testGroup(t)
	require.Equal(t, 201, a.JSON(fiber.MethodPost, g, validBody()).Status)
	r := a.JSON(fiber.MethodPost, g, validBody())
	require.Equal(t, 422, r.Status)
	msgs, _ := r.Errors()["code"].([]any)
	require.Len(t, msgs, 1)
	assert.Equal(t, "کد آیتم قبلاً ثبت شده است.", msgs[0])
	shared, _ := lang.Default().Get("validation.attributes.code", "fa")
	assert.Equal(t, "کد تایید", shared, "the shared OTP label is unchanged")
	en, _ := lang.Default().Get("catalog.attributes.code", "en")
	assert.Equal(t, "item code", en)

	// Rule messages for code use the catalog label too.
	r = a.JSON(fiber.MethodPost, g, map[string]any{"title": map[string]any{"fa": "x"}})
	require.Equal(t, 422, r.Status)
	msgs, _ = r.Errors()["code"].([]any)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "کد آیتم")
}
