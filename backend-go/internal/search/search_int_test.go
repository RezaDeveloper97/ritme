package search_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/search"
	"github.com/ritme/backend-go/resources/translations"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	h := search.NewHandlers(search.NewService(search.NewSources(db)), clock.Real{},
		i18n.NewTranslationStore(translations.FS, ""))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/search", locale, guard, h.Search)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) exec(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	res, err := e.db.Exec(query, args...)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	return id
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	id := e.exec(t, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

// pain logs one pain location with a score and a cycle starting on start.
func (e *env) pain(t *testing.T, userID uint64, start, date, item, score string) {
	t.Helper()
	e.exec(t, `INSERT IGNORE INTO cycle_histories (user_id, period_start_date, is_confirmed, is_estimated, created_at, updated_at)
		VALUES (?, ?, 1, 0, '2026-09-14 09:00:00', '2026-09-14 09:00:00')`, userID, start)
	e.exec(t, `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, source, created_at, updated_at)
		VALUES (?, ?, 'pain', 'location', ?, 'moderate', ?, 'manual', '2026-09-15 09:00:00', '2026-09-15 09:00:00')`, userID, date, item, score)
}

func (e *env) reminder(t *testing.T, userID uint64, typ, title string) int64 {
	t.Helper()
	return e.exec(t, `INSERT INTO reminders (user_id, type, title, recurrence, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 'daily', 1, '2026-09-15 09:00:00', '2026-09-15 09:00:00')`, userID, typ, title)
}

func (e *env) checkup(t *testing.T, userID any, key any, title string) int64 {
	t.Helper()
	return e.exec(t, `INSERT INTO checkup_types (`+"`key`"+`, user_id, category, title, performed_by, interval_months, is_active, sort_order, created_at, updated_at)
		VALUES (?, ?, 'screening', ?, 'doctor', 12, 1, 0, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, key, userID, title)
}

type response struct {
	status int
	body   map[string]any
	raw    string
	text   string // the body decoded and re-encoded without escapes, for Contains checks
}

func (r response) group(key string) map[string]any {
	data, _ := r.body["data"].(map[string]any)
	groups, _ := data["groups"].([]any)
	for _, g := range groups {
		if m, _ := g.(map[string]any); m["key"] == key {
			return m
		}
	}
	return nil
}

func items(g map[string]any) []map[string]any {
	list, _ := g["items"].([]any)
	out := make([]map[string]any, 0, len(list))
	for _, it := range list {
		m, _ := it.(map[string]any)
		out = append(out, m)
	}
	return out
}

func (e *env) get(t *testing.T, params url.Values, token, lang string) response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?"+params.Encode(), strings.NewReader(""))
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	req.Header.Set(clock.Header, now)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	var text strings.Builder
	enc := json.NewEncoder(&text)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(m)
	return response{status: resp.StatusCode, body: m, raw: string(raw), text: text.String()}
}

func TestSearch_Unauthenticated(t *testing.T) {
	e := setup(t)
	r := e.get(t, url.Values{"q": {"درد"}}, "", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw)
}

func TestSearch_Validation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	for _, c := range []struct {
		params url.Values
		field  string
	}{
		{url.Values{}, "q"},
		{url.Values{"q": {"د"}}, "q"},
		{url.Values{"q": {strings.Repeat("د", search.MaxQueryLength+1)}}, "q"},
		{url.Values{"q": {"درد"}, "scope": {"shop"}}, "scope"},
		{url.Values{"q": {"درد"}, "limit": {"0"}}, "limit"},
		{url.Values{"q": {"درد"}, "limit": {"21"}}, "limit"},
		{url.Values{"q[]": {"درد"}}, "q"},
	} {
		r := e.get(t, c.params, tok, "")
		assert.Equal(t, http.StatusUnprocessableEntity, r.status, c.params)
		errs, _ := r.body["errors"].(map[string]any)
		assert.Contains(t, errs, c.field, c.params)
	}
	r := e.get(t, url.Values{"scope": {"shop"}}, tok, "en")
	assert.Contains(t, r.text, "search term")
	assert.Contains(t, r.text, "search scope")
}

func TestSearch_MineIsUserScoped(t *testing.T) {
	e := setup(t)
	a, tokA := e.user(t, "09120000001")
	b, tokB := e.user(t, "09120000002")
	e.pain(t, a, "2026-09-14", "2026-09-15", "abdomen", "4.00")
	e.pain(t, a, "2026-09-14", "2026-09-16", "head", "6.00")
	e.pain(t, b, "2026-09-10", "2026-09-11", "back", "9.00")
	medA := e.reminder(t, a, "medication", "مسکن درد")
	e.reminder(t, b, "medication", "درد دندان")
	e.checkup(t, nil, "search_test_shared", `{"fa":"معاینه مشترک","en":"Shared exam"}`)
	e.checkup(t, b, nil, `{"fa":"معاینه درد لگن"}`)

	r := e.get(t, url.Values{"q": {"درد"}}, tokA, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	mine := items(r.group("mine"))
	require.NotEmpty(t, mine)
	assert.Equal(t, "log_insight", mine[0]["type"])
	assert.Equal(t, "pain", mine[0]["id"])
	meta, _ := mine[0]["meta"].(map[string]any)
	assert.EqualValues(t, 2, meta["days"])
	assert.Equal(t, map[string]any{"kind": "cycle", "from": "2026-09-14", "to": "2026-09-23"}, meta["window"])
	assert.Equal(t, map[string]any{"score": float64(6), "date": "2026-09-16", "cycle_day": float64(3)}, meta["peak"])

	// B's logs, reminders and custom checkup never show for A.
	assert.NotContains(t, r.text, "درد دندان")
	assert.NotContains(t, r.text, "معاینه درد لگن")
	assert.NotContains(t, r.text, `"back"`)
	assert.Contains(t, r.text, "/reminders/medication/"+strconv.FormatInt(medA, 10))

	// And B sees her own data only.
	r = e.get(t, url.Values{"q": {"درد"}, "scope": {"mine"}}, tokB, "")
	require.Equal(t, http.StatusOK, r.status)
	assert.Contains(t, r.text, "درد دندان")
	assert.NotContains(t, r.text, "مسکن درد")
	mineB := items(r.group("mine"))
	require.NotEmpty(t, mineB)
	metaB, _ := mineB[0]["meta"].(map[string]any)
	assert.EqualValues(t, 1, metaB["days"])
	assert.Len(t, r.body["data"].(map[string]any)["groups"], 1)

	r = e.get(t, url.Values{"q": {"معاينه"}, "scope": {"services"}}, tokB, "")
	assert.Contains(t, r.text, "معاینه درد لگن")
}

func TestSearch_GroupsEducationAndLocale(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	for i, s := range []string{"a", "b", "c", "d"} {
		e.exec(t, `INSERT INTO articles (slug, title, excerpt, category, read_time_minutes, is_published, published_at, sort_order, created_at, updated_at)
			VALUES (?, ?, NULL, 'body', 4, 1, '2026-09-01 09:00:00', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`,
			"yoga-"+s, `{"fa":"يوگاي ملايم `+s+`","en":"Gentle yoga `+s+`"}`, i)
	}
	e.exec(t, `INSERT INTO articles (slug, title, is_published, sort_order, created_at, updated_at)
		VALUES ('draft-yoga', '{"fa":"یوگای پیش\u200cنویس"}', 0, 0, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`)

	r := e.get(t, url.Values{"q": {"یوگای ملایم"}}, tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	data := r.body["data"].(map[string]any)
	assert.Equal(t, "all", data["scope"])
	assert.Equal(t, "یوگای ملایم", data["query"])
	keys := []any{}
	for _, g := range data["groups"].([]any) {
		keys = append(keys, g.(map[string]any)["key"])
	}
	assert.Equal(t, []any{"mine", "programs", "education", "services"}, keys)
	edu := r.group("education")
	assert.EqualValues(t, 4, edu["total"])
	list := items(edu)
	require.Len(t, list, search.DefaultLimitAll)
	assert.Equal(t, "/articles/yoga-a", list[0]["route"])
	assert.NotContains(t, r.text, "draft-yoga")

	r = e.get(t, url.Values{"q": {"gentle"}, "scope": {"education"}, "limit": {"2"}}, tok, "en")
	require.Equal(t, http.StatusOK, r.status)
	edu = r.group("education")
	assert.EqualValues(t, 4, edu["total"])
	list = items(edu)
	require.Len(t, list, 2)
	assert.Equal(t, "Gentle yoga a", list[0]["title"])
}
