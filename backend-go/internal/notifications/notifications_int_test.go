package notifications_test

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
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	path     = "/api/v1/profile/notification-settings"
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
	h := notifications.NewHandlers(profilestore.New(db), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Get(path, locale, guard, h.Show)
	app.Put(path, locale, guard, h.Update)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (e *env) do(t *testing.T, method, token, lang, body string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func TestUnauthenticated(t *testing.T) {
	e := setup(t)
	for _, m := range []string{http.MethodGet, http.MethodPut} {
		r := e.do(t, m, "", "", `{}`)
		assert.Equal(t, http.StatusUnauthorized, r.status, m)
		assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw, m)
	}
}

func TestShow_DefaultsWithoutRow(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodGet, tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"success":true,"data":{
		"groups":[
			{"code":"cycle","items":[{"code":"before_period","enabled":true},{"code":"pms","enabled":true},{"code":"fertile_window","enabled":false},{"code":"daily_log","enabled":true}]},
			{"code":"health","items":[{"code":"medications","enabled":true},{"code":"appointments","enabled":true},{"code":"checkups","enabled":true},{"code":"vitals","enabled":false}]},
			{"code":"other","items":[{"code":"learning","enabled":true},{"code":"companion","enabled":true},{"code":"articles","enabled":false}]}
		],
		"quiet_hours":{"enabled":true,"start":"23:00","end":"08:00"},
		"neutral_copy":true}}`, r.raw)
}

func TestUpdate_PartialSaveAndUserIsolation(t *testing.T) {
	e := setup(t)
	aID, a := e.user(t, "09120000001")
	_, b := e.user(t, "09120000002")

	r := e.do(t, http.MethodPut, a, "en", `{"categories":{"vitals":true,"pms":false},"quiet_hours":{"enabled":false,"start":"22:30"},"neutral_copy":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Notification settings saved", r.body["message"])

	r = e.do(t, http.MethodPut, a, "en", `{"categories":{"articles":true}}`) // second save keeps the first one's keys
	require.Equal(t, http.StatusOK, r.status, r.raw)

	prefs, err := notifications.Load(context.Background(), profilestore.New(e.db), aID)
	require.NoError(t, err)
	assert.True(t, prefs.Enabled(notifications.Vitals))
	assert.True(t, prefs.Enabled(notifications.Articles))
	assert.False(t, prefs.Enabled(notifications.PMS))
	assert.False(t, prefs.QuietEnabled)
	assert.Equal(t, 22*60+30, prefs.QuietStart)
	assert.Equal(t, notifications.DefaultQuietEnd, prefs.QuietEnd)
	assert.False(t, prefs.NeutralCopy)

	// user B still sees the defaults and cannot reach A's row
	r = e.do(t, http.MethodGet, b, "en", "")
	require.Equal(t, http.StatusOK, r.status)
	data := r.body["data"].(map[string]any)
	assert.Equal(t, true, data["neutral_copy"])
	assert.Equal(t, map[string]any{"enabled": true, "start": "23:00", "end": "08:00"}, data["quiet_hours"])
}

func TestUpdate_Validation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodPut, tok, "en", `{"categories":{"pms":"maybe"},"quiet_hours":{"start":"25:00"},"neutral_copy":"yes"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "categories.pms")
	assert.Contains(t, errs, "quiet_hours.start")
	assert.Contains(t, errs, "neutral_copy")

	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM notification_preferences`).Scan(&n))
	assert.Zero(t, n, "a rejected body writes nothing")
}
