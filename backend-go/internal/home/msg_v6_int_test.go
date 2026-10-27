package home_test

// T-M9-38b: under CYCLE_ENGINE=v6 Home's smart tip is the /messages/daily message of the same user and day — the
// home's msgSource reads the same engine view as /messages/daily (EngineDay), not the legacy display window — and the
// daily has no 400 for a user whose periods are logged but whose profile has no last_period_start.
// Integration: `make test-int PKG=./internal/home/...`.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/home"
	homestore "github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages"
	msgstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/usertz"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

type msgEnv struct {
	t      *testing.T
	db     *sql.DB
	app    *fiber.App
	iss    *passport.Issuer
	tokens map[uint64]string
}

// newMsgEnv mounts /home and /messages/daily like routes_home.go and routes_messages.go under CYCLE_ENGINE=v6: one
// shared engine state source (engineStates), request clock → user zone → locale → auth guard → handler.
func newMsgEnv(t *testing.T) *msgEnv {
	t.Helper()
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))
	keysPath, err := filepath.Abs(contractKeys)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rc := cache.NewFromClient(rdb, "ritme-go-t-m9-38b:")
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))

	cq := cyclestore.New(db)
	p, err := state.New(cq, rc, clock.Real{}, state.Options{Enabled: true, L1Entries: 100, Build: "test", Logger: quiet})
	require.NoError(t, err)
	states := state.Source{P: p, Revs: cq}
	hh := home.NewHandlers(home.Deps{
		Queries: homestore.New(db), Messages: msgstore.New(db), Pregnancy: pstore.New(db),
		AppURL: "https://api.ritme.app", Logger: quiet,
	}, cycleservice.New(db, nil), clock.Real{}).WithEngine(states, cq)
	mh := messages.NewHandlers(db, clock.Real{}).WithEngine(states)

	cfg := &config.Config{
		App:         config.App{Env: "testing", URL: "http://localhost"},
		Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
		StoragePath: keysPath,
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	tz := usertz.New(profilestore.New(db), quiet)
	t.Cleanup(tz.Wait)
	app.Use(tz.Handler())
	guard := auth.MustGuard(app, cfg, db, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), rc, quiet))
	app.Get("/api/v1/home", locale, guard, hh.Index)
	app.Get("/api/v1/messages/daily", locale, guard, mh.Daily)
	return &msgEnv{t: t, db: db, app: app, tokens: map[uint64]string{},
		iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

func (e *msgEnv) get(uid uint64, path string) (int, map[string]any) {
	e.t.Helper()
	tok, ok := e.tokens[uid]
	if !ok {
		issued, err := e.iss.Issue(context.Background(), uid, time.Now())
		require.NoError(e.t, err)
		tok = issued.AccessToken
		e.tokens[uid] = tok
	}
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "fa")
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set(clock.Header, fixtureNow)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	var body map[string]any
	require.NoError(e.t, json.Unmarshal(raw, &body), string(raw))
	return resp.StatusCode, body
}

// smartTip is /home's smart_tip section (text, source), ok=false without one.
func smartTip(body map[string]any) (text, source any, ok bool) {
	data, _ := body["data"].(map[string]any)
	sections, _ := data["sections"].([]any)
	for _, s := range sections {
		m, _ := s.(map[string]any)
		if m["key"] == "smart_tip" {
			d, _ := m["data"].(map[string]any)
			return d["text"], d["source"], true
		}
	}
	return nil, nil, false
}

// dailyTip is the text /home's smart tip takes from a /messages/daily payload (SmartTipSection: primary long → short
// message, else the first correlation's insight), "" when it has none.
func dailyTip(body map[string]any) (string, string) {
	data, _ := body["data"].(map[string]any)
	if pm, ok := data["primary_message"].(map[string]any); ok {
		for _, k := range []string{"long_message", "short_message"} {
			if s, _ := pm[k].(string); s != "" {
				return s, "message_engine"
			}
		}
	}
	if cs, _ := data["correlations"].([]any); len(cs) > 0 {
		if c, ok := cs[0].(map[string]any); ok {
			if s, _ := c["insight_message"].(string); s != "" {
				return s, "correlation"
			}
		}
	}
	return "", ""
}

// Every fixture persona on the contract day and around it: Home's smart tip is the daily message (same user, same
// day).
func TestHomeV6_MessageEqualsDaily(t *testing.T) {
	e := newMsgEnv(t)
	compared := 0
	for uid := uint64(1001); uid <= 1018; uid++ {
		for _, date := range []string{"", "2026-09-10", "2026-09-30", "2026-10-15"} {
			q := ""
			if date != "" {
				q = "?date=" + date
			}
			name := fmt.Sprintf("%d%s", uid, q)
			hs, hb := e.get(uid, "/api/v1/home"+q)
			if hs == fiber.StatusConflict { // a companion (male) fixture account
				continue
			}
			require.Equal(t, fiber.StatusOK, hs, name)
			data, _ := hb["data"].(map[string]any)
			if data["mode"] != nil && data["mode"] != "cycle" {
				continue // pregnancy / postpartum homes: not the cycle message
			}
			ds, db := e.get(uid, "/api/v1/messages/daily"+q)
			text, source, ok := smartTip(hb)
			// 400: no engine anchor. 500: a low-energy cycle day (Laravel quirk D-05, ErrUndefinedOverrideCase) — Home
			// swallows the same error. Either way no message on either side.
			if ds == fiber.StatusBadRequest || ds == fiber.StatusInternalServerError {
				assert.NotEqual(t, "message_engine", source, name)
				continue
			}
			require.Equal(t, fiber.StatusOK, ds, "%s: %v", name, db)
			dd, _ := db["data"].(map[string]any)
			if dd["mode"] != "cycle" {
				continue
			}
			want, wantSource := dailyTip(db)
			if want == "" {
				assert.NotEqual(t, "message_engine", source, name)
				continue
			}
			require.True(t, ok, "%s: daily has a message, home has no smart tip", name)
			assert.Equal(t, want, text, name)
			assert.Equal(t, wantSource, source, name)
			compared++
		}
	}
	assert.Positive(t, compared, "at least one persona compared")
	t.Logf("home smart tip == daily message on %d persona-days", compared)
}

// Periods logged, profile without last_period_start: v6 has an anchor, so the daily answers (Laravel's 400 «complete
// your profile» is a v1 gate) and Home shows the same message.
func TestHomeV6_DailyWithoutLastPeriodStart(t *testing.T) {
	e := newMsgEnv(t)
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Test', '09120006391', NOW(), NOW())`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	uid := uint64(id) //nolint:gosec // test id
	_, err = e.db.Exec(`INSERT INTO user_profiles (user_id, birthday, last_period_start, cycle_duration, period_duration, user_goal, created_at, updated_at)
		VALUES (?, '1998-01-01', NULL, 28, 5, 'non_ttc', NOW(), NOW())`, uid)
	require.NoError(t, err)
	last := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC) // day 11 on the contract day
	for i := 5; i >= 0; i-- {
		s := last.AddDate(0, 0, -28*i)
		_, err = e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed,
			is_estimated, source, created_at, updated_at) VALUES (?, ?, ?, 5, 1, 0, 'user_logged', NOW(), NOW())`,
			uid, s.Format(time.DateOnly), s.AddDate(0, 0, 4).Format(time.DateOnly))
		require.NoError(t, err)
	}

	ds, db := e.get(uid, "/api/v1/messages/daily")
	require.Equal(t, fiber.StatusOK, ds, "%v", db)
	dd, _ := db["data"].(map[string]any)
	assert.Equal(t, "cycle", dd["mode"])
	ci, _ := dd["context_info"].(map[string]any)
	assert.EqualValues(t, 11, ci["cycle_day"])

	hs, hb := e.get(uid, "/api/v1/home")
	require.Equal(t, fiber.StatusOK, hs)
	want, wantSource := dailyTip(db)
	require.NotEmpty(t, want)
	text, source, ok := smartTip(hb)
	require.True(t, ok)
	assert.Equal(t, want, text)
	assert.Equal(t, wantSource, source)
}

// A user with neither logged periods nor a declared start still gets the 400 (no anchor).
func TestHomeV6_DailyWithoutAnchorIs400(t *testing.T) {
	e := newMsgEnv(t)
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Test', '09120006392', NOW(), NOW())`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	uid := uint64(id) //nolint:gosec // test id
	_, err = e.db.Exec(`INSERT INTO user_profiles (user_id, birthday, cycle_duration, period_duration, user_goal, created_at, updated_at)
		VALUES (?, '1998-01-01', 28, 5, 'non_ttc', NOW(), NOW())`, uid)
	require.NoError(t, err)
	ds, _ := e.get(uid, "/api/v1/messages/daily")
	assert.Equal(t, fiber.StatusBadRequest, ds)
}
