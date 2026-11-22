package etag_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/etag"
)

func base() etag.Engine {
	return etag.Engine{
		UserID: 1004, Rev: 7, Today: civildate.New(2026, 10, 21), Locale: "fa",
		ParamsFP: "p1", ConfigFP: "a1b2c3", Build: "b1",
	}
}

func TestTag_Format(t *testing.T) {
	assert.Equal(t, `W/"urw.e7-2026-10-21-fa-p1-a1b2c3-b1"`, base().Tag()) // 1004 = "rw" in base 36
	e := base()
	e.Locale = "f\"a b\x00é"
	assert.Equal(t, `W/"urw.e7-2026-10-21-f_a_b___-p1-a1b2c3-b1"`, e.Tag(), "unsafe bytes never break the header")
}

// Privacy: another user with the same revision, day and locale gets another tag.
func TestTag_ChangesWithEveryInput(t *testing.T) {
	b := base().Tag()
	for name, mut := range map[string]func(*etag.Engine){
		"user":   func(e *etag.Engine) { e.UserID = 1005 },
		"rev":    func(e *etag.Engine) { e.Rev = 8 },
		"day":    func(e *etag.Engine) { e.Today = e.Today.AddDays(1) },
		"locale": func(e *etag.Engine) { e.Locale = "en" },
		"params": func(e *etag.Engine) { e.ParamsFP = "p1:engine.luteal@2" },
		"config": func(e *etag.Engine) { e.ConfigFP = "ffffff" },
		"build":  func(e *etag.Engine) { e.Build = "b2" },
	} {
		e := base()
		mut(&e)
		assert.NotEqual(t, b, e.Tag(), name)
	}
}

// Tehran midnight → a new tag (the user-local day is part of it).
func TestTag_TehranMidnight(t *testing.T) {
	tag := func(at time.Time) string {
		e := base()
		e.Today = civildate.LocalToday(t.Context(), clock.At(at))
		return e.Tag()
	}
	before := time.Date(2026, 10, 21, 23, 59, 59, 0, civildate.Tehran)
	assert.Equal(t, tag(before), tag(before.Add(-time.Hour)))
	assert.NotEqual(t, tag(before), tag(before.Add(time.Second)))
}

func TestMatch(t *testing.T) {
	tag := base().Tag()
	strong := tag[2:]
	for hdr, want := range map[string]bool{
		tag:                           true,
		strong:                        true, // weak comparison ignores W/
		`"x", ` + tag:                 true,
		`"x",` + tag + `, "y"`:        true,
		"*":                           true,
		`W/"urw.e6-2026-10-21-fa-p1"`: false,
		"":                            false,
		"garbage":                     false,
		tag[:len(tag)-1]:              false,
	} {
		assert.Equal(t, want, etag.Match(hdr, tag), hdr)
	}
	assert.False(t, etag.Match("*", "not-a-tag"))
}

func newApp(tag string, hits *int) *fiber.App {
	app := fiber.New()
	app.Get("/e", etag.Middleware(func(fiber.Ctx) (string, error) { return tag, nil }), func(c fiber.Ctx) error {
		*hits++
		return c.JSON(fiber.Map{"data": "body"})
	})
	return app
}

func do(t *testing.T, app *fiber.App, inm string) (int, string, map[string]string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "/e", nil)
	if inm != "" {
		req.Header.Set(fiber.HeaderIfNoneMatch, inm)
	}
	res, err := app.Test(req)
	require.NoError(t, err)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, string(body), map[string]string{
		"etag": res.Header.Get(fiber.HeaderETag), "cc": res.Header.Get(fiber.HeaderCacheControl),
		"vary": res.Header.Get(fiber.HeaderVary),
	}
}

// Same inputs → 304 with no body before the handler runs; another tag → 200 with the headers.
func TestMiddleware_304(t *testing.T) {
	tag := base().Tag()
	hits := 0
	app := newApp(tag, &hits)

	code, body, h := do(t, app, "")
	assert.Equal(t, fiber.StatusOK, code)
	assert.JSONEq(t, `{"data":"body"}`, body)
	assert.Equal(t, tag, h["etag"])
	assert.Equal(t, "private, no-cache", h["cc"])
	assert.Equal(t, "Authorization, Accept-Language", h["vary"])
	assert.Equal(t, 1, hits)

	code, body, h = do(t, app, tag)
	assert.Equal(t, fiber.StatusNotModified, code)
	assert.Empty(t, body)
	assert.Equal(t, tag, h["etag"])
	assert.Equal(t, "private, no-cache", h["cc"])
	assert.Equal(t, 1, hits, "the handler never ran")

	other := base()
	other.UserID = 1005
	code, _, _ = do(t, app, other.Tag())
	assert.Equal(t, fiber.StatusOK, code, "another account's tag never revalidates")
	assert.Equal(t, 2, hits)
}

// Vary keeps what earlier middleware (CORS: Origin) set.
func TestSetHeaders_AppendsVary(t *testing.T) {
	app := fiber.New()
	app.Get("/v", func(c fiber.Ctx) error {
		c.Vary(fiber.HeaderOrigin)
		etag.SetHeaders(c, base().Tag())
		return c.SendStatus(fiber.StatusNoContent)
	})
	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/v", nil))
	require.NoError(t, err)
	assert.Equal(t, "Origin, Authorization, Accept-Language", res.Header.Get(fiber.HeaderVary))
}

func TestMiddleware_TagErrorIsNot304(t *testing.T) {
	app := fiber.New()
	app.Get("/e", etag.Middleware(func(fiber.Ctx) (string, error) { return "", fiber.ErrInternalServerError }),
		func(c fiber.Ctx) error { return c.SendString("x") })
	req := httptest.NewRequest(fiber.MethodGet, "/e", nil)
	req.Header.Set(fiber.HeaderIfNoneMatch, "*")
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
}

func TestBuild_Stable(t *testing.T) {
	b := etag.Build()
	assert.NotEmpty(t, b)
	assert.Equal(t, b, etag.Build())
}

// bodyApp serves GET /b with the body of the current ?who= (stands for a per-user / per-locale body) through ByBody.
func bodyApp(hits *int, status int, cc string) *fiber.App {
	app := fiber.New()
	app.Get("/b", etag.ByBody(), func(c fiber.Ctx) error {
		*hits++
		if cc != "" {
			c.Set(fiber.HeaderCacheControl, cc)
		}
		return c.Status(status).JSON(fiber.Map{"data": c.Query("who")})
	})
	return app
}

func getB(t *testing.T, app *fiber.App, who, inm string) (int, string, http.Header) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, "/b?who="+who, nil)
	if inm != "" {
		req.Header.Set(fiber.HeaderIfNoneMatch, inm)
	}
	res, err := app.Test(req)
	require.NoError(t, err)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, string(body), res.Header
}

func TestBodyTag(t *testing.T) {
	a := etag.BodyTag([]byte(`{"data":"a"}`))
	assert.Regexp(t, `^W/"b[0-9a-f]{32}"$`, a)
	assert.Equal(t, a, etag.BodyTag([]byte(`{"data":"a"}`)))
	assert.NotEqual(t, a, etag.BodyTag([]byte(`{"data":"b"}`)))
	assert.NotEqual(t, a, etag.BodyTag([]byte(`{"data":"a"} `)), "one byte more is another tag")
}

// The handler always runs; the tag is the hash of this request's body, so a tag of another body never matches.
func TestByBody_304OnlyForTheSameBody(t *testing.T) {
	hits := 0
	app := bodyApp(&hits, fiber.StatusOK, "")
	code, body, h := getB(t, app, "a", "")
	require.Equal(t, fiber.StatusOK, code)
	assert.JSONEq(t, `{"data":"a"}`, body)
	tagA := h.Get(fiber.HeaderETag)
	assert.Equal(t, etag.BodyTag([]byte(body)), tagA)
	assert.Equal(t, etag.CacheControl, h.Get(fiber.HeaderCacheControl))
	assert.Equal(t, "Authorization, Accept-Language", h.Get(fiber.HeaderVary))

	for _, inm := range []string{tagA, tagA[2:], `"x", ` + tagA, "*"} { // weak, strong form, list, wildcard
		code, body, h = getB(t, app, "a", inm)
		assert.Equal(t, fiber.StatusNotModified, code, inm)
		assert.Empty(t, body, inm)
		assert.Equal(t, tagA, h.Get(fiber.HeaderETag), inm)
	}
	assert.Equal(t, 5, hits, "the handler runs on every request (bandwidth, not server work)")

	// Another body (another user / locale / query): A's tag → 200 with B's own body and tag.
	code, body, h = getB(t, app, "b", tagA)
	assert.Equal(t, fiber.StatusOK, code)
	assert.JSONEq(t, `{"data":"b"}`, body)
	assert.NotEqual(t, tagA, h.Get(fiber.HeaderETag))

	// Boundaries: a truncated tag, garbage, an empty list member.
	for _, inm := range []string{tagA[:len(tagA)-1], "garbage", `W/""`, ","} {
		code, _, _ = getB(t, app, "a", inm)
		assert.Equal(t, fiber.StatusOK, code, inm)
	}
}

func TestByBody_KeepsHandlerCacheControl(t *testing.T) {
	hits := 0
	app := bodyApp(&hits, fiber.StatusOK, "public, max-age=60")
	_, _, h := getB(t, app, "a", "")
	tag := h.Get(fiber.HeaderETag)
	assert.Equal(t, "public, max-age=60", h.Get(fiber.HeaderCacheControl))
	code, _, h := getB(t, app, "a", tag)
	assert.Equal(t, fiber.StatusNotModified, code)
	assert.Equal(t, "public, max-age=60", h.Get(fiber.HeaderCacheControl))
}

// Non-200 answers are never tagged nor turned into a 304, even for "*".
func TestByBody_NeverTagsNon200(t *testing.T) {
	for _, st := range []int{fiber.StatusCreated, fiber.StatusForbidden, fiber.StatusUnprocessableEntity} {
		hits := 0
		app := bodyApp(&hits, st, "")
		code, body, h := getB(t, app, "a", "*")
		assert.Equal(t, st, code)
		assert.NotEmpty(t, body)
		assert.Empty(t, h.Get(fiber.HeaderETag))
	}
	app := fiber.New()
	app.Get("/b", etag.ByBody(), func(fiber.Ctx) error { return fiber.ErrInternalServerError })
	req := httptest.NewRequest(fiber.MethodGet, "/b", nil)
	req.Header.Set(fiber.HeaderIfNoneMatch, "*")
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
	assert.Empty(t, res.Header.Get(fiber.HeaderETag))
}

// A handler's own tag (an engine tag) wins; ByBody never overwrites it.
func TestByBody_KeepsAnExistingTag(t *testing.T) {
	app := fiber.New()
	app.Get("/b", etag.ByBody(), func(c fiber.Ctx) error {
		c.Set(fiber.HeaderETag, `W/"own"`)
		return c.SendString("x")
	})
	req := httptest.NewRequest(fiber.MethodGet, "/b", nil)
	req.Header.Set(fiber.HeaderIfNoneMatch, "*")
	res, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
	assert.Equal(t, `W/"own"`, res.Header.Get(fiber.HeaderETag))
}
