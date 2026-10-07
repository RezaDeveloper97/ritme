package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
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
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

const companionNow = "2026-09-23T10:00:00+03:30"

var codeRe = regexp.MustCompile(`^[A-HJ-NP-Z2-9]{6}$`)

type companionEnv struct {
	t   *testing.T
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
	ids map[string]uint64
	mr  *miniredis.Miniredis
}

// newCompanionEnv mounts the real registry (routes_companion.go, routes_care.go, …) on a fresh test database and an
// in-memory Redis, so the throttles and the care delegation are the production wiring.
func newCompanionEnv(t *testing.T) *companionEnv {
	t.Helper()
	return newCompanionEnvWithLogger(t, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

// newCompanionEnvWithLogger is newCompanionEnv with the app's logger (CB-PRIV-01 reads the fake SMS provider's log).
func newCompanionEnvWithLogger(t *testing.T, logger *slog.Logger) *companionEnv {
	t.Helper()
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe48', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	Mount(app, &Deps{
		Config: &config.Config{
			App:         config.App{Env: "testing", URL: "http://localhost"},
			SMS:         config.SMS{Provider: "log"},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keysPath,
		},
		DB:     db,
		Cache:  cache.NewFromClient(rdb, "ritme-go-b4n2:"),
		Logger: logger,
	})
	return &companionEnv{t: t, db: db, app: app, iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365), ids: map[string]uint64{}, mr: mr}
}

// user creates an account and returns its bearer token.
func (e *companionEnv) user(name, mobile string) string {
	e.t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES (?, ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, name, mobile)
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	e.ids[name] = uint64(id)                                              //nolint:gosec // test ids
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
	require.NoError(e.t, err)
	return tok.AccessToken
}

type reply struct {
	Status int
	Body   map[string]any
	Raw    string
}

func (r reply) data() map[string]any {
	d, _ := r.Body["data"].(map[string]any)
	return d
}

func (r reply) list() []any {
	d, _ := r.Body["data"].([]any)
	return d
}

func (e *companionEnv) do(method, path, token string, body any, now ...string) reply {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(e.t, err)
		rd = strings.NewReader(string(raw))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	at := companionNow
	if len(now) > 0 {
		at = now[0]
	}
	req.Header.Set(clock.Header, at)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	out := reply{Status: resp.StatusCode, Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func (e *companionEnv) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

func idOf(v any) uint64 {
	f, _ := v.(float64)
	return uint64(f)
}

// invite creates an invite as token and returns (link id, code).
func (e *companionEnv) invite(token string, body map[string]any) (uint64, string) {
	e.t.Helper()
	r := e.do(fiber.MethodPost, "/api/v1/companions", token, body)
	require.Equal(e.t, fiber.StatusCreated, r.Status, r.Raw)
	comp, _ := r.data()["companion"].(map[string]any)
	inv, _ := r.data()["invite"].(map[string]any)
	code, _ := inv["code"].(string)
	require.Regexp(e.t, codeRe, code)
	return idOf(comp["id"]), code
}

func medBody(extra map[string]any) map[string]any {
	b := map[string]any{"title": "Iron", "form": "tablet", "times": []string{"21:00"}}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func TestCompanionRoutes_LifecycleAndIDOR(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120002001") // owner A
	nina := e.user("Nina", "09120002002") // owner B
	ali := e.user("Ali", "09120002003")   // companion of A
	reza := e.user("Reza", "09120002004") // companion of B
	stranger := e.user("Stranger", "09120002005")
	saraID, ninaID := e.ids["Sara"], e.ids["Nina"]

	// 401 without a token, on an owner and a companion route.
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, "/api/v1/companions", "", nil).Status)
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, "/api/v1/companions/accept", "", map[string]any{"code": "ABCDEF"}).Status)

	// Validation.
	r := e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{"type": "friend"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, r.Raw, `"type"`)
	r = e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{"type": "partner", "grants": map[string]any{"bank": "view"}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, r.Raw, `"grants.bank"`)
	r = e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{"type": "partner", "grants": map[string]any{"meds": "admin"}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	r = e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{"type": "spouse", "child_ids": []int{7}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "children are not verifiable before B-N5-02")
	assert.Contains(t, r.Raw, `"child_ids"`)
	r = e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{"type": "partner", "phone": "09120002001"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "own number")

	// Sara invites Ali (phone-bound): meds edit, appointments + cycle view.
	r = e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{
		"type": "partner", "display_name": "Ali", "phone": "09120002003",
		"grants": map[string]any{"cycle": "view", "meds": "edit", "appointments": "view", "symptoms": "none"},
	})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	inv := r.data()["invite"].(map[string]any)
	codeA := inv["code"].(string)
	assert.Regexp(t, codeRe, codeA)
	assert.Equal(t, "0912****003", inv["phone"])
	assert.Equal(t, true, inv["sms_sent"], "fake provider outside production")
	linkA := idOf(r.data()["companion"].(map[string]any)["id"])
	assert.Equal(t, "invited", r.data()["companion"].(map[string]any)["status"])
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companion_invites WHERE code_hash = ?`, codeA), "plain code never stored")

	// Nothing is visible before acceptance.
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/companions/links/%d/sections/cycle", linkA), ali, nil).Status)

	// Refusals are one generic 422: unknown code, a code bound to another number, the owner's own code.
	unknown := e.do(fiber.MethodPost, "/api/v1/companions/accept", reza, map[string]any{"code": "ZZZZZZ"})
	wrongNumber := e.do(fiber.MethodPost, "/api/v1/companions/accept", reza, map[string]any{"code": codeA})
	own := e.do(fiber.MethodPost, "/api/v1/companions/accept", sara, map[string]any{"code": codeA})
	for _, x := range []reply{unknown, wrongNumber, own} {
		assert.Equal(t, fiber.StatusUnprocessableEntity, x.Status)
		assert.Equal(t, companion.ErrorCodeInviteInvalid, x.Body["error_code"])
		assert.Equal(t, unknown.Raw, x.Raw, "no oracle on whether a code exists")
	}

	// Ali accepts (Persian digits / lower case / spaces are normalized); Sara gets an inbox notice.
	r = e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": " " + strings.ToLower(codeA[:3]) + "-" + codeA[3:]})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "Sara", r.data()["owner"].(map[string]any)["name"])
	assert.Equal(t, []any{"meds"}, r.data()["can_record_for"])
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND type = 'companion'`, saraID))
	var title string
	require.NoError(t, e.db.QueryRow(`SELECT title FROM user_notifications WHERE user_id = ?`, saraID).Scan(&title))
	assert.Contains(t, title, "Ali accepted")
	// The code is one-time.
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/companions/accept", stranger, map[string]any{"code": codeA}).Status)

	// Nina invites Reza (code only, cycle view) and Reza accepts.
	linkB, codeB := e.invite(nina, map[string]any{"type": "spouse", "grants": map[string]any{"cycle": "view"}})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", reza, map[string]any{"code": codeB}).Status)

	// Owner lists only her own companions.
	r = e.do(fiber.MethodGet, "/api/v1/companions", sara, nil)
	require.Len(t, r.list(), 1)
	assert.Equal(t, "active", r.list()[0].(map[string]any)["status"])
	r = e.do(fiber.MethodGet, "/api/v1/companions/links", ali, nil)
	require.Len(t, r.list(), 1)
	assert.Equal(t, float64(saraID), r.list()[0].(map[string]any)["owner"].(map[string]any)["id"])

	sec := func(link uint64, s string) string {
		return fmt.Sprintf("/api/v1/companions/links/%d/sections/%s", link, s)
	}

	// Granted sections read; not granted → 403; another owner's link / a stranger / an unknown section → 404.
	r = e.do(fiber.MethodGet, sec(linkA, "meds"), ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "edit", r.data()["level"])
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, sec(linkA, "cycle"), ali, nil).Status)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, sec(linkA, "appointments"), ali, nil).Status)
	r = e.do(fiber.MethodGet, sec(linkA, "symptoms"), ali, nil)
	assert.Equal(t, fiber.StatusForbidden, r.Status)
	assert.Equal(t, companion.ErrorCodeSectionNotShared, r.Body["error_code"])
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, sec(linkA, "pregnancy"), ali, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, sec(linkB, "cycle"), ali, nil).Status, "companion of A on B's link")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, sec(linkA, "cycle"), reza, nil).Status, "companion of B on A's link")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, sec(linkA, "cycle"), stranger, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, sec(linkA, "cycle"), sara, nil).Status, "the owner is not her own companion")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, sec(linkA, "bank"), ali, nil).Status)
	assert.Equal(t, 3, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND action = 'read'`, saraID), "every read audited")

	// Owner routes are owner-only: another owner and the companion get 404, and nothing changes.
	for _, tok := range []string{nina, ali, stranger} {
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/companions/%d", linkA), tok, nil).Status)
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", linkA), tok,
			map[string]any{"grants": map[string]any{"symptoms": "edit", "pregnancy": "edit"}}).Status)
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/children", linkA), tok,
			map[string]any{"child_ids": []int{}}).Status)
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, fmt.Sprintf("/api/v1/companions/%d/renew", linkA), tok, map[string]any{}).Status)
		assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/%d", linkA), tok, nil).Status)
	}
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, sec(linkA, "symptoms"), ali, nil).Status, "companion cannot grant itself")
	// The companion-side leave route is companion-only.
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/links/%d", linkA), sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/links/%d", linkA), reza, nil).Status)
	// Renewing an accepted link is refused.
	r = e.do(fiber.MethodPost, fmt.Sprintf("/api/v1/companions/%d/renew", linkA), sara, map[string]any{})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Equal(t, companion.ErrorCodeNotPending, r.Body["error_code"])

	// «ثبت برای …»: Ali (meds edit) records a medication in Sara's list; Sara is notified, the write is audited.
	r = e.do(fiber.MethodPost, "/api/v1/care/medications", ali, medBody(map[string]any{"for_user_id": saraID}))
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	medID := idOf(r.data()["id"])
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM reminders WHERE id = ? AND user_id = ?`, medID, saraID))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, e.ids["Ali"]))
	assert.Equal(t, 2, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND type = 'companion'`, saraID))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND action = 'write' AND section = 'meds'`, saraID))
	var notice string
	require.NoError(t, e.db.QueryRow(`SELECT title FROM user_notifications WHERE user_id = ? ORDER BY id DESC LIMIT 1`, saraID).Scan(&notice))
	assert.NotContains(t, notice, "Iron", "no health payload in the notice")

	// Ali opens and edits that medication only through for_user_id; without it, it is not his.
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d?for_user_id=%d", medID, saraID), ali, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d", medID), ali, nil).Status)
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/care/medications/%d", medID), ali, map[string]any{"for_user_id": saraID, "title": "Iron 2"})
	assert.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/care/medications/%d", medID), ali, map[string]any{"title": "x"}).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/care/medications/%d", medID), ali, nil).Status, "deletes stay owner-only")

	// view cannot write; no link cannot write; a malformed id is a 422.
	r = e.do(fiber.MethodPost, "/api/v1/care/appointments", ali, map[string]any{
		"for_user_id": saraID, "kind": "in_person", "topic": "ultrasound", "scheduled_at": "2026-09-30 10:00",
	})
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Equal(t, "companion_forbidden", r.Body["error_code"])
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodPost, "/api/v1/care/medications", ali, medBody(map[string]any{"for_user_id": ninaID})).Status)
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodPost, "/api/v1/care/medications", reza, medBody(map[string]any{"for_user_id": ninaID})).Status, "cycle view only")
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodPost, "/api/v1/care/medications", stranger, medBody(map[string]any{"for_user_id": saraID})).Status)
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d?for_user_id=%d", medID, saraID), stranger, nil).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/care/medications", ali, medBody(map[string]any{"for_user_id": "abc"})).Status)
	// Naming oneself is the normal path.
	assert.Equal(t, fiber.StatusCreated, e.do(fiber.MethodPost, "/api/v1/care/medications", sara, medBody(map[string]any{"for_user_id": saraID})).Status)

	// Grants change takes effect on the next request: meds edit → view.
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", linkA), sara, map[string]any{"grants": map[string]any{"meds": "view", "cycle": "view"}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "none", r.data()["grants"].(map[string]any)["appointments"])
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodPost, "/api/v1/care/medications", ali, medBody(map[string]any{"for_user_id": saraID})).Status)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, sec(linkA, "meds"), ali, nil).Status)
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, sec(linkA, "appointments"), ali, nil).Status)

	// Audit trail: the owner's view, no payload.
	r = e.do(fiber.MethodGet, "/api/v1/companions/audit", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status)
	assert.NotEmpty(t, r.list())
	assert.NotContains(t, r.Raw, "Iron")
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/companions/audit", ali, nil).Status)
	assert.Empty(t, e.do(fiber.MethodGet, "/api/v1/companions/audit", ali, nil).list(), "a companion sees no one's trail")

	// Revoke: access ends immediately.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/%d", linkA), sara, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, sec(linkA, "meds"), ali, nil).Status)
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d?for_user_id=%d", medID, saraID), ali, nil).Status)
	assert.Empty(t, e.do(fiber.MethodGet, "/api/v1/companions/links", ali, nil).list())
	assert.Empty(t, e.do(fiber.MethodGet, "/api/v1/companions", sara, nil).list())

	// Leave: the companion ends B's link.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/links/%d", linkB), reza, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, sec(linkB, "cycle"), reza, nil).Status)
	assert.Empty(t, e.do(fiber.MethodGet, "/api/v1/companions", nina, nil).list())
}

func TestCompanionRoutes_ExpiryAndRenew(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120003001")
	ali := e.user("Ali", "09120003002")

	link, code := e.invite(sara, map[string]any{"type": "partner"})
	// 24 h later the code is refused with the same generic 422.
	r := e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code}, "2026-09-24T10:00:01+03:30")
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Equal(t, companion.ErrorCodeInviteInvalid, r.Body["error_code"])

	// Renew issues a new code; the old one stops working, the new one is accepted.
	r = e.do(fiber.MethodPost, fmt.Sprintf("/api/v1/companions/%d/renew", link), sara, map[string]any{}, "2026-09-24T11:00:00+03:30")
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	fresh := r.data()["invite"].(map[string]any)["code"].(string)
	assert.NotEqual(t, code, fresh)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code}, "2026-09-24T11:01:00+03:30").Status)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": fresh}, "2026-09-24T11:01:00+03:30").Status)
}

// TestCompanionRoutes_AcceptThrottle: CompanionAcceptPerUser attempts per account and CompanionAcceptPerIP per client
// IP per hour; the 429 is the localized companion envelope.
func TestCompanionRoutes_AcceptThrottle(t *testing.T) {
	e := newCompanionEnv(t)
	guess := func(tok string) reply {
		return e.do(fiber.MethodPost, "/api/v1/companions/accept", tok, map[string]any{"code": "ZZZZZZ"})
	}
	a := e.user("A", "09120004001")
	for i := range CompanionAcceptPerUser {
		require.Equal(t, fiber.StatusUnprocessableEntity, guess(a).Status, "attempt %d", i+1)
	}
	r := guess(a)
	assert.Equal(t, fiber.StatusTooManyRequests, r.Status)
	assert.Equal(t, companion.ErrorCodeTooManyAttempts, r.Body["error_code"])

	// Fresh accounts behind the same IP share the per-IP budget (a's 11 hits were counted there).
	used := CompanionAcceptPerUser + 1
	n := 2
	for used < CompanionAcceptPerIP {
		tok := e.user(fmt.Sprintf("U%d", n), fmt.Sprintf("091200040%02d", n))
		n++
		for i := 0; i < CompanionAcceptPerUser && used < CompanionAcceptPerIP; i++ {
			require.Equal(t, fiber.StatusUnprocessableEntity, guess(tok).Status)
			used++
		}
	}
	fresh := e.user("Fresh", "09120004099")
	assert.Equal(t, fiber.StatusTooManyRequests, guess(fresh).Status, "per-IP cap")
}

// TestCompanionRoutes_DisabledWithoutPepper: production without COMPANION_CODE_PEPPER answers 503 on the code routes.
func TestCompanionRoutes_DisabledWithoutPepper(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120005001")
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	guard := auth.NewGuardWith(keys.Public, authstore.New(e.db), clock.Real{}, nil).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(e.db), nil, nil))
	h := companion.NewHandlers(e.db, clock.Real{}, companion.HandlerOptions{Disabled: true})
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Post("/api/v1/companions", locale, guard, h.Store)
	app.Post("/api/v1/companions/accept", locale, guard, h.Accept)
	app.Get("/api/v1/companions", locale, guard, h.Index)
	e.app = app

	r := e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{"type": "partner"})
	assert.Equal(t, fiber.StatusServiceUnavailable, r.Status)
	assert.Equal(t, companion.ErrorCodeUnavailable, r.Body["error_code"])
	assert.Equal(t, fiber.StatusServiceUnavailable, e.do(fiber.MethodPost, "/api/v1/companions/accept", sara, map[string]any{"code": "ABCDEF"}).Status)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/companions", sara, nil).Status, "reads keep working")
}

func smsSent(t *testing.T, r reply) bool {
	t.Helper()
	inv, ok := r.data()["invite"].(map[string]any)
	require.True(t, ok, r.Raw)
	sent, _ := inv["sms_sent"].(bool)
	return sent
}

// TestCompanionRoutes_InviteSMSLimits (review M-1): renew re-sends only after SMSResendAfter or to a new number; one
// number gets at most SMSPerRecipientPerDay invites a day from all owners; one owner triggers at most
// SMSPerOwnerPerDay. A refused send still returns the code (sms_sent=false).
func TestCompanionRoutes_InviteSMSLimits(t *testing.T) {
	e := newCompanionEnv(t)
	const target = "09120007100"
	o1 := e.user("O1", "09120007001")
	r := e.do(fiber.MethodPost, "/api/v1/companions", o1, map[string]any{"type": "partner", "phone": target})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.True(t, smsSent(t, r))
	link := idOf(r.data()["companion"].(map[string]any)["id"])
	renew := func(at string, body map[string]any) reply {
		r := e.do(fiber.MethodPost, fmt.Sprintf("/api/v1/companions/%d/renew", link), o1, body, at)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		assert.Regexp(t, codeRe, r.data()["invite"].(map[string]any)["code"], "the code is returned either way")
		return r
	}
	assert.False(t, smsSent(t, renew("2026-09-23T10:01:00+03:30", map[string]any{})), "within the resend gap")
	e.mr.FastForward(companion.SMSResendAfter + time.Second)                                          // Redis TTLs run on Redis time, not X-Test-Now
	assert.True(t, smsSent(t, renew("2026-09-23T10:03:00+03:30", map[string]any{})), "after the gap") // target: 2
	o2 := e.user("O2", "09120007002")
	r = e.do(fiber.MethodPost, "/api/v1/companions", o2, map[string]any{"type": "partner", "phone": target})
	assert.True(t, smsSent(t, r)) // target: 3
	o3 := e.user("O3", "09120007003")
	r = e.do(fiber.MethodPost, "/api/v1/companions", o3, map[string]any{"type": "partner", "phone": target})
	require.Equal(t, fiber.StatusCreated, r.Status)
	assert.False(t, smsSent(t, r), "per-recipient daily cap")

	// A changed number is sent at once (new resend key) — counts toward O1's daily cap (3 so far after this).
	assert.True(t, smsSent(t, renew("2026-09-23T10:04:00+03:30", map[string]any{"phone": "09120007101"})))

	// Per-owner daily cap: O4 reaches SMSPerOwnerPerDay distinct numbers, the next is not sent.
	o4 := e.user("O4", "09120007004")
	for i := range companion.SMSPerOwnerPerDay {
		r := e.do(fiber.MethodPost, "/api/v1/companions", o4, map[string]any{"type": "partner", "phone": fmt.Sprintf("091200072%02d", i)})
		require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
		assert.True(t, smsSent(t, r), "invite %d", i+1)
	}
	r = e.do(fiber.MethodPost, "/api/v1/companions", o4, map[string]any{"type": "partner", "phone": "09120007299"})
	require.Equal(t, fiber.StatusCreated, r.Status)
	assert.False(t, smsSent(t, r), "per-owner daily cap")
}

// TestCompanionRoutes_DelegatedWriteAuditFailClosed (review M-2, L-4, L-5): a companion write is audited before it
// happens — without an audit row there is no write; a failed owner notice is logged, never a 500; a delegated
// appointment update cannot touch the prep checklist.
func TestCompanionRoutes_DelegatedWriteAuditFailClosed(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120008001")
	ali := e.user("Ali", "09120008002")
	saraID := e.ids["Sara"]
	_, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"meds": "edit", "appointments": "edit"}})

	// L-5: the inbox insert fails → accept still 200 (link active).
	_, err := e.db.Exec("RENAME TABLE user_notifications TO user_notifications_off")
	require.NoError(t, err)
	r := e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code})
	assert.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	// M-2: a failed owner notice after a committed write is not a 500.
	r = e.do(fiber.MethodPost, "/api/v1/care/medications", ali, medBody(map[string]any{"for_user_id": saraID}))
	assert.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	_, err = e.db.Exec("RENAME TABLE user_notifications_off TO user_notifications")
	require.NoError(t, err)
	before := e.count(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, saraID)

	// M-2: no audit trail → no write.
	_, err = e.db.Exec("RENAME TABLE companion_audit_logs TO companion_audit_logs_off")
	require.NoError(t, err)
	r = e.do(fiber.MethodPost, "/api/v1/care/medications", ali, medBody(map[string]any{"for_user_id": saraID, "title": "Unaudited"}))
	assert.Equal(t, fiber.StatusInternalServerError, r.Status)
	_, err = e.db.Exec("RENAME TABLE companion_audit_logs_off TO companion_audit_logs")
	require.NoError(t, err)
	assert.Equal(t, before, e.count(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, saraID), "write refused without audit")

	// L-4: the owner's prep checklist is owner-only; a companion's update keeps it as stored.
	r = e.do(fiber.MethodPost, "/api/v1/care/appointments", sara, map[string]any{
		"kind": "in_person", "topic": "ultrasound", "scheduled_at": "2026-09-30 10:00", "prep": []map[string]any{{"text": "Insurance card"}},
	})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	apptID := idOf(r.data()["id"])
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/care/appointments/%d", apptID), ali, map[string]any{
		"for_user_id": saraID, "location": "Clinic", "prep": []map[string]any{{"text": "Injected", "done": true}},
	})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "Clinic", r.data()["location"])
	prep, _ := r.data()["prep"].([]any)
	require.Len(t, prep, 1)
	assert.Equal(t, "Insurance card", prep[0].(map[string]any)["text"])
	assert.Equal(t, false, prep[0].(map[string]any)["done"])
}
