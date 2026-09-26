package calendar_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"database/sql"

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
	"github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/pregnancy/v2/calendar"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

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
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/cal", locale, guard, calendar.NewHandlers(store.New(db), clock.Real{}).Calendar)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile, lmp string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('T', ?, NOW(), NOW())`, mobile)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	if lmp != "" {
		_, err = e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
			VALUES (?, 1, 'lmp', ?, NOW(), NOW())`, id, lmp)
		require.NoError(t, err)
	}
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

func (e *env) appointment(t *testing.T, uid uint64, at, meta string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO reminders (user_id, type, title, scheduled_at, recurrence, is_active, meta, created_at, updated_at)
		VALUES (?, 'appointment', 'Visit', ?, 'none', 1, ?, NOW(), NOW())`, uid, at, meta)
	require.NoError(t, err)
}

func (e *env) get(t *testing.T, path, tok, lang string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Accept-Language", lang)
	req.Header.Set(clock.Header, "2026-09-23T10:00:00+03:30")
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return resp.StatusCode, m, string(raw)
}

func planItem(t *testing.T, data map[string]any, key string) map[string]any {
	t.Helper()
	for _, x := range data["care_plan"].([]any) {
		if it := x.(map[string]any); it["key"] == key {
			return it
		}
	}
	t.Fatalf("care item %s missing", key)
	return nil
}

func TestCalendar(t *testing.T) {
	e := setup(t)
	_, err := e.db.Exec(`INSERT INTO pregnancy_care_items (` + "`key`" + `, title, kind, week_from, week_to, sort_order, is_active, created_at, updated_at) VALUES
		('t_nt', '{"fa":"NT","en":"NT scan"}', 'scan', 11, 14, 1, 1, NOW(), NOW()),
		('t_gtt', '{"fa":"قند","en":"GTT"}', 'test', 24, 28, 2, 1, NOW(), NOW()),
		('t_first', '{"fa":"اول","en":"First"}', 'visit', 6, 8, 3, 1, NOW(), NOW())`)
	require.NoError(t, err)
	uid, tok := e.user(t, "09120000901", "2026-07-01")
	other, otherTok := e.user(t, "09120000902", "2026-07-01")
	e.appointment(t, uid, "2026-09-30 10:30:00", `{"v":1,"kind":"in_person","topic":"ultrasound","remind_before":"1d","prep":[],"status":"scheduled","care_item_key":"t_nt","stage":"booked"}`)
	e.appointment(t, uid, "2026-08-15 09:00:00", `{"v":1,"kind":"in_person","topic":"checkup","remind_before":"1d","prep":[],"status":"scheduled","care_item_key":"t_first","stage":"result","result_note":"ok"}`)
	e.appointment(t, uid, "2026-10-01 09:00:00", `{"v":1,"kind":"in_person","topic":"checkup","remind_before":"1d","prep":[],"status":"cancelled"}`)
	e.appointment(t, other, "2026-10-02 09:00:00", `{"v":1,"kind":"in_person","topic":"lab","remind_before":"1d","prep":[],"status":"scheduled","care_item_key":"t_gtt"}`)

	status, body, raw := e.get(t, "/cal", tok, "fa")
	require.Equal(t, http.StatusOK, status, raw)
	data := body["data"].(map[string]any)
	assert.Equal(t, "1405-07", data["month"])
	assert.Len(t, data["days"], 30)
	visits := data["visits"].([]any)
	require.Len(t, visits, 1, "cancelled and other user's visits hidden")
	v := visits[0].(map[string]any)
	assert.Equal(t, "2026-09-30", v["date"])
	assert.Equal(t, "10:30", v["time"])
	assert.Equal(t, "booked", v["stage"])
	assert.EqualValues(t, 14, v["week"])
	assert.Equal(t, "2026-09-30", data["next_visit"].(map[string]any)["date"])

	assert.Equal(t, "booked", planItem(t, data, "t_nt")["state"])
	first := planItem(t, data, "t_first")
	assert.Equal(t, "done", first["state"])
	assert.Equal(t, "2026-08-15", first["date"])
	gtt := planItem(t, data, "t_gtt")
	assert.Equal(t, "to_book", gtt["state"], "other user's linked visit does not count")
	assert.Equal(t, "2026-12-09", gtt["suggested_date"])
	assert.Equal(t, map[string]any{"from": "2026-12-09", "to": "2027-01-12"}, gtt["window"])

	// Mordad 1405 holds the done visit.
	status, body, raw = e.get(t, "/cal?month=1405-05", tok, "fa")
	require.Equal(t, http.StatusOK, status, raw)
	assert.Len(t, body["data"].(map[string]any)["visits"], 1)

	// Gregorian month for en.
	status, body, _ = e.get(t, "/cal?month=2026-10", tok, "en")
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "October 2026", body["data"].(map[string]any)["month_label"])
	assert.Len(t, body["data"].(map[string]any)["days"], 31)

	status, body, _ = e.get(t, "/cal?month=2026-13", tok, "en")
	assert.Equal(t, http.StatusUnprocessableEntity, status)
	assert.Contains(t, body["errors"], "month")

	// The other user sees only her own visit.
	status, body, _ = e.get(t, "/cal?month=1405-07", otherTok, "fa")
	require.Equal(t, http.StatusOK, status)
	assert.Len(t, body["data"].(map[string]any)["visits"], 1)
	assert.Equal(t, "booked", planItem(t, body["data"].(map[string]any), "t_gtt")["state"])

	_, noTok := e.user(t, "09120000903", "")
	status, body, _ = e.get(t, "/cal", noTok, "fa")
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, "pregnancy_not_active", body["error_code"])

	status, _, _ = e.get(t, "/cal", "", "fa")
	assert.Equal(t, http.StatusUnauthorized, status)
}
