package vitals_test

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"strings"
	"testing"

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
	"github.com/ritme/backend-go/internal/vitals"
)

// T-M10-19: GET/PUT/DELETE /vitals/targets (owner only) and the diabetic classification end to end.
func setupTargets(t *testing.T) *env {
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
	h := vitals.NewHandlers(vitals.NewService(db), clock.Fixed(fixed))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	p := "/api/v1/vitals"
	app.Post(p+"/readings", locale, guard, h.Store)
	app.Get(p+"/targets", locale, guard, h.Targets)
	app.Put(p+"/targets", locale, guard, h.SaveTargets)
	app.Delete(p+"/targets", locale, guard, h.ClearTargets)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) diabetes(t *testing.T, userID uint64, illnesses string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, chronic_illnesses, created_at, updated_at)
		VALUES (?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, illnesses)
	require.NoError(t, err)
}

func classOf(t *testing.T, r response) any {
	t.Helper()
	c, _ := r.data()["reading"].(map[string]any)["classification"].(map[string]any)
	if c == nil {
		return nil
	}
	return c["code"]
}

func TestVitals_TargetsOwnerOnlyAndDiabeticClasses(t *testing.T) {
	e := setupTargets(t)
	idA, tokA := e.user(t, "09120000011")
	idB, tokB := e.user(t, "09120000012")
	e.diabetes(t, idA, `["diabetes"]`)
	e.diabetes(t, idB, `["migraine"]`)

	r := e.do(t, http.MethodGet, "/api/v1/vitals/targets", "", "", nil)
	assert.Equal(t, http.StatusUnauthorized, r.status)

	r = e.do(t, http.MethodGet, "/api/v1/vitals/targets", tokA, "en", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, true, r.data()["diabetic"])
	assert.Equal(t, []any{}, r.data()["items"])
	def := r.data()["defaults"].(map[string]any)
	assert.Equal(t, map[string]any{"low_mgdl": float64(80), "high_mgdl": float64(130), "high_inclusive": true}, def["fasting"])
	assert.Equal(t, map[string]any{"low_mgdl": nil, "high_mgdl": float64(180), "high_inclusive": false}, def["post_meal"])
	assert.Nil(t, def["bedtime"])

	glucose := func(tok string, v float64, ctx string) response {
		return e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", map[string]any{
			"type": "glucose", "value": v, "unit": "mg_dl", "context": ctx, "measured_at": "2026-10-06 07:30"})
	}
	assert.Equal(t, "below_target", classOf(t, glucose(tokA, 79, "fasting")))
	assert.Equal(t, "in_target", classOf(t, glucose(tokA, 130, "before_meal")))
	assert.Equal(t, "above_target", classOf(t, glucose(tokA, 180, "after_meal")))
	assert.Equal(t, "low", classOf(t, glucose(tokA, 65, "fasting")), "hypo wins")

	// Validation (422, never 401).
	for _, body := range []map[string]any{
		{"items": []any{map[string]any{"context": "fasting", "low_mgdl": 140, "high_mgdl": 90, "set_by": "doctor"}}},
		{"items": []any{map[string]any{"context": "fasting", "set_by": "doctor"}}},
		{"items": []any{map[string]any{"context": "fasting", "high_mgdl": 400, "set_by": "doctor"}}},
		{"items": []any{map[string]any{"context": "random", "high_mgdl": 140, "set_by": "doctor"}}},
		{"items": []any{map[string]any{"context": "fasting", "high_mgdl": 140, "set_by": "nurse"}}},
		{"items": []any{map[string]any{"context": "fasting", "high_mgdl": 140, "set_by": "user"},
			map[string]any{"context": "fasting", "high_mgdl": 150, "set_by": "user"}}},
	} {
		r = e.do(t, http.MethodPut, "/api/v1/vitals/targets", tokA, "en", body)
		assert.Equal(t, http.StatusUnprocessableEntity, r.status, body)
	}

	// The doctor's 90–140 override.
	r = e.do(t, http.MethodPut, "/api/v1/vitals/targets", tokA, "fa", map[string]any{"items": []any{
		map[string]any{"context": "fasting", "low_mgdl": 90, "high_mgdl": 140, "set_by": "doctor", "note": "دکتر احمدی"},
	}})
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, vitals.T("messages.targets_saved", "fa"), r.body["message"])
	items := r.data()["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "doctor", items[0].(map[string]any)["set_by"])
	assert.Equal(t, "in_target", classOf(t, glucose(tokA, 140, "fasting")))
	assert.Equal(t, "above_target", classOf(t, glucose(tokA, 141, "fasting")))
	assert.Equal(t, "below_target", classOf(t, glucose(tokA, 89, "fasting")))

	// Another user sees only her own (empty) targets; a non-diabetic user with a row keeps the general classes.
	r = e.do(t, http.MethodGet, "/api/v1/vitals/targets", tokB, "en", nil)
	assert.Equal(t, false, r.data()["diabetic"])
	assert.Equal(t, []any{}, r.data()["items"])
	r = e.do(t, http.MethodPut, "/api/v1/vitals/targets", tokB, "en", map[string]any{"items": []any{
		map[string]any{"context": "fasting", "low_mgdl": 90, "high_mgdl": 140, "set_by": "user"}}})
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "high", classOf(t, glucose(tokB, 110, "fasting")))
	r = e.do(t, http.MethodGet, "/api/v1/vitals/targets", tokA, "en", nil)
	require.Len(t, r.data()["items"], 1, "B's write did not touch A")

	// DELETE → registry defaults again.
	r = e.do(t, http.MethodDelete, "/api/v1/vitals/targets", tokA, "en", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, []any{}, r.data()["items"])
	assert.Equal(t, "above_target", classOf(t, glucose(tokA, 140, "fasting")))
}

// The urgent copy of message_contents takes the registry numbers through placeholders.
func TestVitals_AlertCopyPlaceholders(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000013")
	_, err := e.db.Exec(`UPDATE message_contents SET payload = JSON_SET(payload, '$.what_we_saw',
		'You logged {value}; above {systolic}/{diastolic} is an emergency.')
		WHERE ` + "`group`" + ` = 'vitals_alert' AND item_key = 'bp_crisis' AND locale = 'en'`)
	require.NoError(t, err)
	r := e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", map[string]any{
		"type": "bp", "systolic": 185, "diastolic": 112})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	saw := r.data()["alert"].(map[string]any)["what_we_saw"].(string)
	assert.Equal(t, "You logged 185/112 mmHg; above 180/120 is an emergency.", saw)
	assert.False(t, strings.Contains(saw, "{"))
}
