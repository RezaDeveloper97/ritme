package http_test

// T-M12-34 (P-10): ETag / 304 on the slow-changing GETs of the home (etag.ByBody: the tag is the hash of the body
// built for this request), through the production registry.

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/http/testenv"
	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/etag"
)

// bodyTagged are the routes tagged by etag.ByBody (routes_*.go).
var bodyTagged = []string{
	"/api/v1/modules",
	"/api/v1/banners",
	"/api/v1/banners/ads?mode=cycle",
	"/api/v1/logs/taxonomy",
	"/api/v1/health-logs/enums",
	"/api/v1/learning/courses",
	"/api/v1/checkups/home",
}

func getINM(e *testenv.Env, path, token, inm string) testenv.Reply {
	e.T.Helper()
	h := map[string]string{}
	if inm != "" {
		h[fiber.HeaderIfNoneMatch] = inm
	}
	return e.Send(testenv.Request{Method: fiber.MethodGet, Path: path, Token: token, Header: h})
}

// Every tagged route: 200 with the hash of its own body; the same body revalidates to a 304 with no body and the
// tag echoed; no If-None-Match (what the contract sends) is a plain 200 with the full body.
func TestETagBody_EveryRoute304OnTheSameBody(t *testing.T) {
	e := testenv.New(t)
	u := e.User()
	for _, p := range bodyTagged {
		r := getINM(e, p, u.Token, "")
		require.Equal(t, fiber.StatusOK, r.Status, "%s: %s", p, r.Raw)
		require.NotEmpty(t, r.Raw, p)
		tag := r.ETag()
		assert.Equal(t, etag.BodyTag([]byte(r.Raw)), tag, p)
		assert.Contains(t, r.Header.Get(fiber.HeaderVary), fiber.HeaderAuthorization, p)
		assert.Contains(t, r.Header.Get(fiber.HeaderVary), fiber.HeaderAcceptLanguage, p)

		nm := getINM(e, p, u.Token, tag)
		assert.Equal(t, fiber.StatusNotModified, nm.Status, p)
		assert.Empty(t, nm.Raw, p)
		assert.Equal(t, tag, nm.ETag(), p)

		// Boundary: a tag one byte off is a 200 with the full body.
		last := byte('0')
		if tag[len(tag)-2] == '0' {
			last = '1'
		}
		off := getINM(e, p, u.Token, tag[:len(tag)-2]+string(last)+`"`)
		assert.Equal(t, fiber.StatusOK, off.Status, p)
		assert.Equal(t, r.Raw, off.Raw, p)
	}
	// The public modules map keeps its own shared-cache policy; the authenticated ones are private.
	assert.Equal(t, "public, max-age=60", getINM(e, "/api/v1/modules", "", "").Header.Get(fiber.HeaderCacheControl))
	assert.Equal(t, etag.CacheControl, getINM(e, "/api/v1/banners", u.Token, "").Header.Get(fiber.HeaderCacheControl))

	// 401 (no token) is never tagged nor a 304, even for "*".
	r := getINM(e, "/api/v1/banners", "", "*")
	assert.Equal(t, fiber.StatusUnauthorized, r.Status)
	assert.Empty(t, r.ETag())
}

// An admin module toggle changes the map → a new tag: the old one gets a 200 with the new body.
func TestETagBody_ModuleToggleIsANewTag(t *testing.T) {
	e := testenv.New(t)
	key := modules.Keys[0]
	before := getINM(e, "/api/v1/modules", "", "")
	require.Equal(t, fiber.StatusOK, before.Status)
	require.Equal(t, fiber.StatusNotModified, getINM(e, "/api/v1/modules", "", before.ETag()).Status)

	setModule(t, e, key, false)
	after := getINM(e, "/api/v1/modules", "", before.ETag())
	require.Equal(t, fiber.StatusOK, after.Status, after.Raw)
	assert.Equal(t, false, after.Data()[key], after.Raw)
	assert.NotEqual(t, before.ETag(), after.ETag())
	assert.Equal(t, fiber.StatusNotModified, getINM(e, "/api/v1/modules", "", after.ETag()).Status)

	// Back on: the original body and tag again.
	setModule(t, e, key, true)
	assert.Equal(t, fiber.StatusNotModified, getINM(e, "/api/v1/modules", "", before.ETag()).Status)
}

// An admin banner edit → new tag; a teen account (banners suppressed) has another body and another tag, and her
// request carrying the adult account's tag gets a 200 with her own (empty) body — never a 304 on someone else's body.
func TestETagBody_BannerEditAndCrossUser(t *testing.T) {
	e := testenv.New(t)
	adult, teen := e.User(), e.User()
	_, err := e.DB.Exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, 'female', 'teen', NOW(), NOW())`, teen.ID)
	require.NoError(t, err)
	res, err := e.DB.Exec(`INSERT INTO banners (title, image_path, position, link_url, link_type, is_active, sort_order, kind, created_at, updated_at)
		VALUES ('{"fa":"بنر","en":"Banner"}', 'banners/b.webp', 'home_top', 'https://example.com', 'external', 1, 1, 'banner', NOW(), NOW())`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	a := getINM(e, "/api/v1/banners", adult.Token, "")
	require.Equal(t, fiber.StatusOK, a.Status, a.Raw)
	require.Contains(t, a.Raw, "b.webp")

	b := getINM(e, "/api/v1/banners", teen.Token, a.ETag())
	require.Equal(t, fiber.StatusOK, b.Status, "the adult's tag never revalidates the teen's body")
	assert.NotContains(t, b.Raw, "b.webp")
	assert.NotEqual(t, a.ETag(), b.ETag())
	assert.Equal(t, etag.BodyTag([]byte(b.Raw)), b.ETag())
	assert.Equal(t, fiber.StatusNotModified, getINM(e, "/api/v1/banners", teen.Token, b.ETag()).Status)
	assert.Equal(t, fiber.StatusNotModified, getINM(e, "/api/v1/banners", adult.Token, a.ETag()).Status)

	// Admin edit of the banner → the adult's old tag is stale.
	_, err = e.DB.Exec(`UPDATE banners SET title = '{"fa":"بنر نو","en":"New banner"}', updated_at = NOW() WHERE id = ?`, id)
	require.NoError(t, err)
	a2 := getINM(e, "/api/v1/banners", adult.Token, a.ETag())
	require.Equal(t, fiber.StatusOK, a2.Status)
	assert.Contains(t, a2.Raw, "New banner")
	assert.NotEqual(t, a.ETag(), a2.ETag())

	// The query is part of the body: another position, another tag.
	p := getINM(e, "/api/v1/banners?position=home_bottom", adult.Token, a2.ETag())
	assert.Equal(t, fiber.StatusOK, p.Status)
	assert.NotEqual(t, a2.ETag(), p.ETag())
}

// Locale change → another body and another tag; the fa tag never revalidates the en body.
func TestETagBody_LocaleIsANewTag(t *testing.T) {
	e := testenv.New(t)
	u := e.User()
	e.Lang = "fa"
	fa := getINM(e, "/api/v1/logs/taxonomy", u.Token, "")
	require.Equal(t, fiber.StatusOK, fa.Status, fa.Raw)
	e.Lang = "en"
	en := getINM(e, "/api/v1/logs/taxonomy", u.Token, fa.ETag())
	require.Equal(t, fiber.StatusOK, en.Status)
	assert.NotEqual(t, fa.Raw, en.Raw)
	assert.NotEqual(t, fa.ETag(), en.ETag())
	assert.Equal(t, fiber.StatusNotModified, getINM(e, "/api/v1/logs/taxonomy", u.Token, en.ETag()).Status)
}
