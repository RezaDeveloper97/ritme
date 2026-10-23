package etag_test

import (
	"io"
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
