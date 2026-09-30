package catalog_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
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

	noCSRF := e.As(admintest.EditorID)
	noCSRF.CSRF = ""
	r = noCSRF.JSON(fiber.MethodPost, "/catalog/missed_pill_rules", validBody())
	assert.Equal(t, 419, r.Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM catalog_items"))

	a := e.As(admintest.SuperID)
	assert.Equal(t, 404, a.Get("/catalog/Bad-Group").Status)
	assert.Equal(t, 404, a.Get("/catalog/g/abc").Status)
	assert.Equal(t, 404, a.Get("/catalog/g/999999").Status)
}

func TestAdmin_CRUDFlushesPublicCache(t *testing.T) {
	e := newEnv(t)
	a := e.As(admintest.EditorID)
	const g = "/catalog/missed_pill_rules"

	// Warm the public cache with the empty group.
	assert.Empty(t, e.get(t, "/api/v1/catalog/missed_pill_rules", "en").items())

	r := a.JSON(fiber.MethodPost, g, validBody())
	require.Equal(t, 201, r.Status, string(r.Raw))
	assert.Equal(t, "Catalog item created.", r.Body["message"])
	it := r.Obj("catalog_item")
	id := int(it["id"].(float64))
	assert.Equal(t, "missed_pill_rules", it["group"])
	assert.Equal(t, "missed_one", it["code"])
	assert.Equal(t, map[string]any{"fa": "یک قرص جا افتاده", "en": "One pill missed"}, it["title"], "unknown languages dropped")
	assert.Equal(t, []any{"contraception"}, it["audiences"], "deduplicated")
	assert.Equal(t, true, it["is_active"])
	assert.Equal(t, true, it["needs_review"], "new copy awaits clinical review by default")
	assert.InDelta(t, 1, it["sort_order"], 0)
	assert.Equal(t, "{\"hours\":24,\"steps\":[{\"en\":\"Take\",\"fa\":\"بخور\"}]}",
		e.String("SELECT meta FROM catalog_items WHERE id = ?", id), "stored as raw UTF-8 JSON")

	pubEN := e.get(t, "/api/v1/catalog/missed_pill_rules", "en")
	require.Len(t, pubEN.items(), 1, "the create flushed the cached empty group")
	assert.Equal(t, "One pill missed", pubEN.items()[0]["title"])
	assert.Equal(t, map[string]any{"hours": float64(24), "steps": []any{"Take"}}, pubEN.items()[0]["meta"])

	// Duplicate code in the same group → 422; the same code in another group is fine.
	r = a.JSON(fiber.MethodPost, g, validBody())
	assert.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "code")
	assert.Equal(t, 201, a.JSON(fiber.MethodPost, "/catalog/teen_faq", validBody()).Status)

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
	assert.Equal(t, 404, a.Get("/catalog/teen_faq/"+itoa(id)).Status, "an id of another group is not found")
	list := a.Get(g + "?q=" + "pill")
	require.Equal(t, 200, list.Status)
	assert.Len(t, list.Items(), 1)
	assert.Empty(t, a.Get(g+"?status=inactive").Items())
	groups := a.Get("/catalog").Items()
	require.Len(t, groups, 2)
	assert.Equal(t, map[string]any{"group": "missed_pill_rules", "items_count": float64(1), "active_count": float64(1)}, groups[0])

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
	pubFA := e.get(t, "/api/v1/catalog/missed_pill_rules", "fa")
	assert.Equal(t, "جاافتادن قرص", pubFA.items()[0]["title"], "the update flushed the cache")
	assert.Equal(t, false, pubFA.items()[0]["needs_review"])

	// Deactivate → gone from the public read.
	require.Equal(t, 200, a.JSON(fiber.MethodPut, g+"/"+itoa(id), map[string]any{
		"title": map[string]any{"fa": "جاافتادن قرص"}, "is_active": false,
	}).Status)
	assert.Empty(t, e.get(t, "/api/v1/catalog/missed_pill_rules", "fa").items())
	assert.Len(t, a.Get(g+"?status=inactive").Items(), 1)

	// Delete.
	r = a.JSON(fiber.MethodDelete, g+"/"+itoa(id), nil)
	require.Equal(t, 200, r.Status)
	assert.Equal(t, "Catalog item deleted.", r.Body["message"])
	assert.Equal(t, 404, a.JSON(fiber.MethodDelete, g+"/"+itoa(id), nil).Status)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM catalog_items WHERE `group` = 'missed_pill_rules'"))
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM catalog_items WHERE `group` = 'teen_faq'"), "other group untouched")
}

func itoa(n int) string { return strconv.Itoa(n) }
