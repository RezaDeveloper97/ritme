package auth_test

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// setupWithConsents is setup() with the mandatory signup consents wired (T-M10-23, D-174).
func setupWithConsents(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	q := store.New(db)
	rec := &recorder{}
	h := auth.NewHandlers(q, passport.NewIssuer(testKey, q, clock.Real{}, 365), rec, clock.Real{}, 30, quiet)
	h.RequireSignupConsents(consent.NewSignup(consent.NewService(db)))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/api/v1/auth/send-otp", h.SendOTP)
	app.Post("/api/v1/auth/verify-otp", h.VerifyOTP)
	return &env{db: db, app: app, jobs: rec}
}

func (e *env) consentEvents(t *testing.T, mobile string) []string {
	t.Helper()
	rows, err := e.db.Query(`SELECT CONCAT(ce.consent, ':', ce.version, ':', ce.action, ':', ce.source) FROM consent_events ce
		JOIN users u ON u.id = ce.user_id WHERE u.mobile = ? ORDER BY ce.id`, mobile)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(t, rows.Err())
	return out
}

func TestSignupConsents_NewAccountNeedsThem(t *testing.T) {
	e := setupWithConsents(t)
	const mobile = "09121239001"
	r := e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"`+mobile+`"}`, t0)
	require.Equal(t, 200, r.status, r.raw)
	code := e.otp(t, mobile)

	for _, body := range []string{
		`{"mobile":"` + mobile + `","code":"` + code + `"}`,
		`{"mobile":"` + mobile + `","code":"` + code + `","consents":{"terms":1}}`,
		`{"mobile":"` + mobile + `","code":"` + code + `","consents":{"terms":1,"health_data_processing":2}}`,
		`{"mobile":"` + mobile + `","code":"` + code + `","consents":"yes"}`,
	} {
		r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "", body, t0)
		require.Equal(t, 422, r.status, r.raw)
		assert.Equal(t, "consent_required", r.body["error_code"])
		assert.Equal(t, map[string]any{"terms": float64(1), "health_data_processing": float64(1)}, r.body["consents"])
		assert.NotEmpty(t, r.body["message"])
		assert.Nil(t, r.body["data"], "no token")
	}
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM users WHERE mobile = ?`, mobile).Scan(&n))
	assert.Zero(t, n, "no account without the consents")

	// The same code still works once the consents are sent (it was not spent).
	r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "",
		`{"mobile":"`+mobile+`","code":"`+code+`","consents":{"terms":1,"health_data_processing":"1"}}`, t0)
	require.Equal(t, 200, r.status, r.raw)
	data := r.body["data"].(map[string]any)
	assert.Equal(t, true, data["new_user"])
	assert.NotEmpty(t, data["access_token"])
	assert.Equal(t, []string{"terms:1:grant:signup", "health_data_processing:1:grant:signup"}, e.consentEvents(t, mobile))
	var granted int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM user_consents uc JOIN users u ON u.id = uc.user_id
		WHERE u.mobile = ? AND uc.granted = 1 AND uc.version = 1`, mobile).Scan(&granted))
	assert.Equal(t, 2, granted, "the projection follows the events")
}

func TestSignupConsents_ExistingAccountNeverRefused(t *testing.T) {
	e := setupWithConsents(t)
	const mobile = "09121239002"
	_, err := e.db.Exec(`INSERT INTO users (mobile, created_at, updated_at) VALUES (?, '2026-01-01 10:00:00', '2026-01-01 10:00:00')`, mobile)
	require.NoError(t, err)

	e.login(t, mobile, t0) // no consents: H-18, the pending sheet asks later
	assert.Empty(t, e.consentEvents(t, mobile))

	// Ticking the boxes again on the phone screen records what was not accepted yet, once.
	for _, now := range []string{t1, "2026-09-23T10:02:02+03:30"} {
		r := e.do(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"`+mobile+`"}`, now)
		require.Equal(t, 200, r.status, r.raw)
		r = e.do(t, http.MethodPost, "/api/v1/auth/verify-otp", "",
			`{"mobile":"`+mobile+`","code":"`+e.otp(t, mobile)+`","consents":{"terms":1,"health_data_processing":1}}`, now)
		require.Equal(t, 200, r.status, r.raw)
		assert.Equal(t, false, r.body["data"].(map[string]any)["new_user"])
	}
	assert.Equal(t, []string{"terms:1:grant:signup", "health_data_processing:1:grant:signup"}, e.consentEvents(t, mobile))
}
