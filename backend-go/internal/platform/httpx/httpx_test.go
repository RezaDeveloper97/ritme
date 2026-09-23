package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Goldens in testdata/laravel were captured from the Laravel app by
// testdata/laravel/capture.php (APP_ENV=production, APP_DEBUG=false, Accept:
// application/json, PHP 8.4.6): <name>.body is the exact response body,
// <name>.meta.json the status and relevant headers, paginator_*.json the
// json_encode of a LengthAwarePaginator.

type golden struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string
}

func loadGolden(t *testing.T, name string) golden {
	t.Helper()
	dir := filepath.Join("testdata", "laravel")
	body, err := os.ReadFile(filepath.Join(dir, name+".body"))
	require.NoError(t, err)
	meta, err := os.ReadFile(filepath.Join(dir, name+".meta.json"))
	require.NoError(t, err)
	var g golden
	require.NoError(t, json.Unmarshal(meta, &g))
	g.Body = string(body)
	return g
}

func newApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))})
	noop := func(c fiber.Ctx) error { return c.SendString("ok") }
	// The routes the captures hit (backend/routes/api.php).
	app.Post("/api/v1/auth/send-otp", noop)
	app.Get("/api/v1/languages", noop)
	app.Put("/api/v1/reminders/:id", noop)
	app.Delete("/api/v1/reminders/:id", noop)
	return app
}

type result struct {
	status int
	header http.Header
	body   string
}

func do(t *testing.T, app *fiber.App, method, target string, hdr ...string) result {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Accept", "application/json")
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return result{resp.StatusCode, resp.Header, string(b)}
}

func assertGolden(t *testing.T, name string, got result) {
	t.Helper()
	g := loadGolden(t, name)
	assert.Equal(t, g.Status, got.status, name)
	assert.Equal(t, g.Body, got.body, name) // byte-for-byte
	for k, v := range g.Headers {
		assert.Equal(t, v, got.header.Get(k), "%s header %s", name, k)
	}
}

func TestErrorHandler_RouteAndMethodGoldens(t *testing.T) {
	app := newApp(t)
	assertGolden(t, "route_404", do(t, app, "GET", "/api/v1/does-not-exist"))
	assertGolden(t, "route_404_nested", do(t, app, "GET", "/api/v1/foo/bar/baz"))
	assertGolden(t, "route_404_encoded", do(t, app, "GET", "/api/v1/%D8%B3%D9%84%D8%A7%D9%85/x?y=1"))
	assertGolden(t, "route_404_trailing_slash", do(t, app, "GET", "/api/v1/nope/"))
	assertGolden(t, "route_404_root_level", do(t, app, "GET", "/nope"))
	assertGolden(t, "method_405_single", do(t, app, "GET", "/api/v1/auth/send-otp"))
	assertGolden(t, "method_405_multi", do(t, app, "GET", "/api/v1/reminders/5"))
	assertGolden(t, "method_405_post_on_get", do(t, app, "POST", "/api/v1/languages"))
}

func TestErrorHandler_ExceptionGoldens(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler(nil)})
	errs := map[string]error{
		"throttle_429":            TooManyRequests(42, 5, 0, 1790000042),
		"server_500":              errors.New("boom"),
		"maintenance_503":         ServiceUnavailable(),
		"abort_404":               NotFound(),
		"model_404":               ModelNotFound(`App\Models\TaskTemplate`, 99),
		"model_404_noid":          ModelNotFound(`App\Models\TaskTemplate`),
		"validation_422":          required("a", "b", "c"),
		"validation_422_one_more": required("a", "b"),
		"validation_422_fa":       faValidation(t),
	}
	for name, e := range errs {
		app.Get("/"+name, func(fiber.Ctx) error { return fmt.Errorf("wrapped: %w", e) })
	}
	for name := range errs {
		t.Run(name, func(t *testing.T) {
			assertGolden(t, name, do(t, app, "GET", "/"+name))
		})
	}
}

// required mimics Validator::make with a 'required' rule per field (en messages).
func required(fields ...string) *ValidationError {
	v := NewValidationError()
	for _, f := range fields {
		v.Add(f, "The "+f+" field is required.")
	}
	return v
}

// faValidation rebuilds the fa capture's errors (fa messages from lang/fa/validation.php)
// so the test checks the summary suffix, which stays English in fa.
func faValidation(t *testing.T) *ValidationError {
	t.Helper()
	var body struct {
		Errors map[string][]string `json:"errors"`
	}
	require.NoError(t, json.Unmarshal([]byte(loadGolden(t, "validation_422_fa").Body), &body))
	v := NewValidationError()
	for _, f := range []string{"a", "b", "c"} {
		for _, m := range body.Errors[f] {
			v.Add(f, m)
		}
	}
	return v
}

func TestErrorHandler_OtherFiberErrorsAndLogging(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))})
	app.Get("/too-large", func(fiber.Ctx) error { return fiber.ErrRequestEntityTooLarge })
	app.Get("/boom", func(fiber.Ctx) error { return errors.New("db down") })
	// Laravel: PostTooLargeException → 413 {"message": ""} (pretty).
	r := do(t, app, "GET", "/too-large")
	assert.Equal(t, 413, r.status)
	assert.Equal(t, "{\n    \"message\": \"\"\n}", r.body)
	r = do(t, app, "GET", "/boom")
	assert.Equal(t, 500, r.status)
	assert.Equal(t, "{\n    \"message\": \"Server Error\"\n}", r.body)
	assert.Equal(t, "application/json", r.header.Get("Content-Type"))
}

func TestEnvelopes(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler(nil)})
	app.Get("/ok", func(c fiber.Ctx) error { return OK(c, jsonx.Obj("url", "https://x/y", "fa", "<b>")) })
	app.Get("/ok-msg", func(c fiber.Ctx) error { return OK(c, []int{1}, "Saved") })
	app.Get("/created", func(c fiber.Ctx) error { return Created(c, jsonx.Obj("id", 1), "Created") })
	app.Get("/message", func(c fiber.Ctx) error { return Message(c, "Logged out") })
	app.Get("/fail", func(fiber.Ctx) error {
		return Fail(422, "Invalid date format. Use YYYY-MM-DD.")
	})
	app.Get("/fail-extras", func(fiber.Ctx) error {
		return Fail(429, "Wait", "data", jsonx.Obj("retry_after", 42)).WithHeader("Retry-After", "42")
	})
	app.Get("/custom", func(c fiber.Ctx) error {
		// health-logs POST literal order: success, message, data, warning
		return JSON(c, 201, jsonx.Obj("success", true, "message", "m", "data", nil, "warning", "w"))
	})

	cases := []struct {
		path   string
		status int
		body   string
	}{
		{"/ok", 200, `{"success":true,"data":{"url":"https:\/\/x\/y","fa":"<b>"}}`},
		{"/ok-msg", 200, `{"success":true,"message":"Saved","data":[1]}`},
		{"/created", 201, `{"success":true,"message":"Created","data":{"id":1}}`},
		{"/message", 200, `{"success":true,"message":"Logged out"}`},
		{"/fail", 422, `{"success":false,"message":"Invalid date format. Use YYYY-MM-DD."}`},
		{"/fail-extras", 429, `{"success":false,"message":"Wait","data":{"retry_after":42}}`},
		{"/custom", 201, `{"success":true,"message":"m","data":null,"warning":"w"}`},
	}
	for _, c := range cases {
		r := do(t, app, "GET", c.path)
		assert.Equal(t, c.status, r.status, c.path)
		assert.Equal(t, c.body, r.body, c.path)
		assert.Equal(t, "application/json", r.header.Get("Content-Type"), c.path)
	}
	assert.Equal(t, "42", do(t, app, "GET", "/fail-extras").header.Get("Retry-After"))

	// Encoding failures surface as 500 through the error handler.
	app.Get("/bad", func(c fiber.Ctx) error { return OK(c, make(chan int)) })
	r := do(t, app, "GET", "/bad")
	assert.Equal(t, 500, r.status)
}

func TestStatusOfAndErrors(t *testing.T) {
	assert.Equal(t, 422, StatusOf(Fail(422, "x")))
	assert.Equal(t, 429, StatusOf(fmt.Errorf("w: %w", TooManyRequests(1, 1, 0, 1))))
	assert.Equal(t, 422, StatusOf(NewValidationError()))
	assert.Equal(t, 404, StatusOf(fiber.ErrNotFound))
	assert.Equal(t, 500, StatusOf(errors.New("x")))
	assert.Contains(t, Fail(400, "bad").Error(), "400 bad")
	assert.Contains(t, NotFound().Error(), "404")
	assert.Contains(t, required("f").Error(), "The f field is required.")
}

// ValidationException::summarize():
//
//	0 messages → "The given data was invalid."; 1 → the message; 2 → "(and 1 more error)"; 3+ → "(and N more errors)".
func TestValidationSummary(t *testing.T) {
	v := NewValidationError()
	assert.True(t, v.Empty())
	assert.Equal(t, "The given data was invalid.", v.Summary())
	v.Add("x", "X is bad.")
	assert.Equal(t, "X is bad.", v.Summary())
	v.Add("x", "X again.")
	assert.Equal(t, "X is bad. (and 1 more error)", v.Summary())
	v.Add("y", "Y.")
	assert.Equal(t, "X is bad. (and 2 more errors)", v.Summary())
	assert.Equal(t, []string{"X is bad.", "X again."}, v.Messages("x"))
	assert.False(t, v.Empty())
	b, err := jsonx.Marshal(v.Body(), 0)
	require.NoError(t, err)
	assert.Equal(t, `{"message":"X is bad. (and 2 more errors)","errors":{"x":["X is bad.","X again."],"y":["Y."]}}`, string(b))

	var zero ValidationError
	zero.Add("z", "Z.")
	assert.Equal(t, "Z.", zero.Summary())
}

func TestMethodNotAllowed_Order(t *testing.T) {
	e := MethodNotAllowed("GET", "/api/v1/x/", []string{"DELETE", " PATCH", "PUT"})
	assert.Equal(t, "The GET method is not supported for route api/v1/x. Supported methods: PUT, PATCH, DELETE.", e.Message)
	assert.Equal(t, "PUT, PATCH, DELETE", e.Headers["Allow"])
	assert.Equal(t, "The route / could not be found.", RouteNotFound("/").Message)
	assert.Equal(t, "The route a/b could not be found.", RouteNotFound("/a/b?c=1").Message)
	assert.Equal(t, 418, Abort(418).Status)
}

// testdata/laravel/paginator_*.json: new LengthAwarePaginator($items, $total, 30|2, $page,
// ['path' => 'https://api.ritme.app/api/v1/health-logs']) with $items = [['id' => n], …].
func TestPaginator_RawMatchesLaravel(t *testing.T) {
	cases := []struct {
		file                 string
		page, total, perPage int
	}{
		{"paginator_p1", 1, 45, 30},
		{"paginator_p2", 2, 95, 30},
		{"paginator_empty", 1, 0, 30},
		{"paginator_many", 5, 400, 30},
		{"paginator_beyond", 7, 95, 30},
		{"paginator_slider", 20, 1500, 30},
		{"paginator_near_end", 48, 1500, 30},
		{"paginator_small", 1, 5, 2},
	}
	type item struct {
		ID int `json:"id"`
	}
	for _, c := range cases {
		want, err := os.ReadFile(filepath.Join("testdata", "laravel", c.file+".json"))
		require.NoError(t, err)
		n := max(0, min(c.perPage, c.total-(c.page-1)*c.perPage))
		var items []item
		for i := range n {
			items = append(items, item{ID: (c.page-1)*c.perPage + i + 1})
		}
		p := NewPage(items, c.total, c.perPage, c.page, "https://api.ritme.app/api/v1/health-logs")
		got, err := jsonx.Marshal(p.Raw(), 0)
		require.NoError(t, err)
		assert.Equal(t, string(want), string(got), c.file)
	}
}

func TestPaginator_Meta(t *testing.T) {
	p := NewPage([]int{1, 2}, 95, 30, 2, "https://x")
	b, err := jsonx.Marshal(p.Meta(), 0)
	require.NoError(t, err)
	assert.JSONEq(t, `{"current_page":2,"last_page":4,"per_page":30,"total":95}`, string(b))
	assert.Equal(t, `{"current_page":2,"last_page":4,"per_page":30,"total":95}`, string(b))
	assert.Equal(t, 30, p.Offset())
	assert.Equal(t, 1, Page{PerPage: 0}.LastPage())
	// Items left nil encode as [].
	b, err = jsonx.Marshal(Page{Total: 0, PerPage: 10, CurrentPage: 1, Path: "https://x/a?b=1"}.Raw(), jsonx.UnescapedSlashes)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"data":[]`)
	assert.Contains(t, string(b), `"first_page_url":"https://x/a?b=1&page=1"`)
}

func TestPageParam(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error { return c.SendString(fmt.Sprint(PageParam(c))) })
	// filter_var($page, FILTER_VALIDATE_INT) !== false && (int) $page >= 1 ? $page : 1
	for q, want := range map[string]string{
		"": "1", "?page=3": "3", "?page=0": "1", "?page=-2": "1", "?page=abc": "1",
		"?page=2.5": "1", "?page=+4": "4", "?page=07": "1", "?page=%207": "7",
	} {
		r := do(t, app, "GET", "/"+q)
		assert.Equal(t, want, r.body, q)
	}
}

func TestRequestURL(t *testing.T) {
	app := fiber.New()
	app.Get("/*", func(c fiber.Ctx) error { return c.SendString(RequestURL(c)) })
	cases := []struct {
		target string
		hdr    []string
		want   string
	}{
		{"http://api.ritme.app/api/v1/health-logs?page=2", nil, "http://api.ritme.app/api/v1/health-logs"},
		{"http://backend:8000/api/v1/health-logs/", []string{"X-Forwarded-Proto", "https", "X-Forwarded-Host", "api.ritme.app"},
			"https://api.ritme.app/api/v1/health-logs"},
		{"http://backend:8000/api/v1/x", []string{"X-Forwarded-Proto", "https, http", "X-Forwarded-Host", "stage.ritmeapp.ir", "X-Forwarded-Port", "443"},
			"https://stage.ritmeapp.ir/api/v1/x"},
		{"http://backend:8000/api/v1/x", []string{"X-Forwarded-Proto", "http", "X-Forwarded-Host", "localhost:9999", "X-Forwarded-Port", "8010"},
			"http://localhost:8010/api/v1/x"},
	}
	for _, c := range cases {
		r := do(t, app, "GET", c.target, c.hdr...)
		assert.Equal(t, c.want, r.body, c.target)
	}
}
