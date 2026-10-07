package services_test

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
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/checkups"
	checkupsstore "github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/enums"
	healthrecordstore "github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/services"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-10-07T10:00:00+03:30"
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
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-10-07 09:00:00', '2026-10-07 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	cat := catalog.NewReader(catalogstore.New(db), nil, 0, quiet) // no cache: admin edits show at once
	cq := checkupsstore.New(db)
	mode := func(ctx context.Context, uid uint64) (enums.LifeMode, error) {
		return checkups.UserLifeMode(ctx, cq, uid)
	}
	h := services.NewHandlers(services.NewService(cat, healthrecordstore.New(db), mode, nil), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/services", locale, guard, h.Show)
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

func (e *env) get(t *testing.T, token, lang string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/services", nil)
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
	data, _ := m["data"].(map[string]any)
	return resp.StatusCode, data, string(raw)
}

func codes(list any) []string {
	var out []string
	for _, x := range list.([]any) {
		out = append(out, x.(map[string]any)["code"].(string))
	}
	return out
}

func byCode(list any, code string) map[string]any {
	for _, x := range list.([]any) {
		if m := x.(map[string]any); m["code"] == code {
			return m
		}
	}
	return nil
}

func TestUnauthenticated(t *testing.T) {
	e := setup(t)
	status, _, raw := e.get(t, "", "")
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, raw)
}

func TestHub_SeededCatalog(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09900047001")

	status, d, raw := e.get(t, tok, "fa")
	require.Equal(t, http.StatusOK, status, raw)
	assert.Equal(t, services.SectionCodes, codes(d["sections"]))
	assert.Nil(t, d["upcoming_booking"])
	assert.Equal(t, []string{"assistant", "doctors", "record", "labs", "vitals", "insurance"}, codes(d["care"]))
	// no profile → cycle mode: every program incl. contraception
	assert.Equal(t, []string{"pain_endometriosis", "pmdd", "heavy_bleeding", "pelvic_floor", "contraception"}, codes(d["programs"]))

	care := byCode(d["care"], "record")
	assert.Equal(t, "پرونده سلامت", care["title"])
	assert.Equal(t, "/record", care["href"])
	assert.Equal(t, "live", care["status"])
	assert.Equal(t, float64(0), care["count"])
	for _, c := range []string{"assistant", "doctors", "insurance"} {
		assert.Equal(t, "soon", byCode(d["care"], c)["status"], c)
	}
	sos := byCode(d["sections"], "emergency")
	assert.Equal(t, "115", sos["phone"])

	_, en, _ := e.get(t, tok, "en")
	assert.Equal(t, "Health record", byCode(en["care"], "record")["title"])
	assert.Equal(t, "Layette & baby", byCode(en["sections"], "shop")["categories"].([]any)[0].(map[string]any)["title"])
}

func TestHub_AdminToggleAndRecordCountIsolation(t *testing.T) {
	e := setup(t)
	a, tokA := e.user(t, "09900047002")
	_, tokB := e.user(t, "09900047003")
	for i := 0; i < 2; i++ {
		_, err := e.db.Exec(`INSERT INTO record_documents (user_id, kind, created_at, updated_at) VALUES (?, 'visit', NOW(), NOW())`, a)
		require.NoError(t, err)
	}
	_, err := e.db.Exec("UPDATE catalog_items SET is_active = 0 WHERE `group` = 'services_sections' AND code IN ('shop', 'emergency')")
	require.NoError(t, err)

	_, da, _ := e.get(t, tokA, "fa")
	assert.Equal(t, float64(2), byCode(da["care"], "record")["count"])
	assert.NotContains(t, codes(da["sections"]), "shop")
	assert.Equal(t, "emergency", codes(da["sections"])[len(codes(da["sections"]))-1], "the emergency card cannot be switched off")

	_, db, _ := e.get(t, tokB, "fa")
	assert.Equal(t, float64(0), byCode(db["care"], "record")["count"], "user B never sees A's documents")
}

func TestHub_TeenHidesShopAndContraception(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09900047004")
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, 'teen', NOW(), NOW())`, uid)
	require.NoError(t, err)
	_, d, raw := e.get(t, tok, "fa")
	assert.NotContains(t, codes(d["programs"]), "contraception", raw)
	assert.NotContains(t, codes(d["sections"]), "shop", "teen mode has no shop")
}
