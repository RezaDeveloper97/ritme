package messages_test

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

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// CB-LOSS-01: once her pregnancy has ended (profile with pregnancy mode off — a loss or a deactivation) a forced
// ?mode=pregnancy no longer produces pregnancy messages, whether or not a loss record exists (she may erase it); during
// a new active pregnancy the forced mode is honoured again.
func TestDaily_ForcedPregnancyAfterLoss(t *testing.T) {
	db := testdb.New(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ff10f', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	aq := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, aq, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/messages/daily", locale, guard, messages.NewHandlers(db, clock.Real{}).Daily)

	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120006001', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := passport.NewIssuer(key, aq, clock.Real{}, 365).Issue(context.Background(), uint64(uid), time.Now()) //nolint:gosec // test id
	require.NoError(t, err)
	exec := func(q string, args ...any) {
		_, err := db.Exec(q, args...)
		require.NoError(t, err)
	}
	exec(`INSERT INTO user_profiles (user_id, last_period_start, period_duration, cycle_duration, created_at, updated_at)
		VALUES (?, '2026-09-10', 5, 28, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, age_source, lmp_date, onboarding_completed, created_at, updated_at)
		VALUES (?, 0, 1, 'lmp', '2026-08-02', 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)

	mode := func() string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/daily?mode=pregnancy", nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		req.Header.Set(clock.Header, "2026-09-23T10:00:00+03:30")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(raw))
		var body struct {
			Data struct {
				Mode string `json:"mode"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(raw, &body))
		return body.Data.Mode
	}

	assert.Equal(t, "cycle", mode(), "ended pregnancy, no loss record: the forced pregnancy mode is ignored")
	exec(`INSERT INTO pregnancy_losses (user_id, loss_type, content_stopped_at, created_at, updated_at)
		VALUES (?, 'unspecified', '2026-09-22 09:00:00', '2026-09-22 09:00:00', '2026-09-22 09:00:00')`, uid)
	assert.Equal(t, "cycle", mode(), "after a loss the forced pregnancy mode is ignored")
	exec(`UPDATE pregnancy_profiles SET pregnancy_mode = 1, cycle_mode = 0 WHERE user_id = ?`, uid)
	assert.Equal(t, "pregnancy", mode(), "a new active pregnancy is hers again")
}
