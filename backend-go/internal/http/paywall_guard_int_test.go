package http

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
)

// The no-paywall guarantee (T-M10-29, SPEC §4.3, DECISIONS A-03/A-04/A-11, D-178): «never a paywall on emergency
// paths, medication reminders or EPDS». No safety or clinical alert, emergency copy or core logging may ever sit
// behind Ritme Plus. Three layers prove it:
//
//  1. TestPaywallGuard_FreeUserSafetyRoutes: a free user (no plus_subscriptions / plus_trials row) calls every
//     emergency, medication-reminder, EPDS / postpartum-recovery, pregnancy alert / kick / contraction, vitals-alert
//     and alerts-inbox route through the production Mount with valid payloads (the alert branches run: crisis BP,
//     heavy lochia, EPDS item 10, spotting, 5-1-1): 2xx, never 402/429, never plus_required / plus_quota_exceeded.
//  2. The same test spies on the SQL the app sends: every plus entitlement path (Service.Consume / Entitlement /
//     Status, Gate.Require → tierOf → plus_subscriptions + plus_trials, usage on plus_usage, settings on
//     plus_settings) touches a plus_ table, so no plus_ statement during a guarded request means plus was never
//     consulted. A positive control proves the spy sees a Plus route.
//  3. TestPaywallGuard_SafetyPackagesNeverImportPlus (no database): the safety packages never import internal/plus.

// pgNow is the fixture instant: Saturday 2026-10-03 20:00 Tehran.
var pgNow = time.Date(2026, 10, 3, 20, 0, 0, 0, time.FixedZone("IRST", 12600))

// plusSQL matches a statement that names a plus_ table (plus_subscriptions, plus_trials, plus_usage, plus_settings,
// plus_plans, plus_invoices …).
var plusSQL = regexp.MustCompile("(?i)(^|[^a-z0-9_])plus_[a-z_]+")

// sqlRecorder keeps every statement the app's pool sends.
type sqlRecorder struct {
	mu   sync.Mutex
	stmt []string
}

func (r *sqlRecorder) add(q string) {
	r.mu.Lock()
	r.stmt = append(r.stmt, q)
	r.mu.Unlock()
}

func (r *sqlRecorder) reset() {
	r.mu.Lock()
	r.stmt = nil
	r.mu.Unlock()
}

// plusStatements returns the recorded statements that touch a plus_ table, and the total recorded.
func (r *sqlRecorder) plusStatements() ([]string, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, q := range r.stmt {
		if plusSQL.MatchString(q) {
			out = append(out, strings.Join(strings.Fields(q), " "))
		}
	}
	return out, len(r.stmt)
}

// spyConnector wraps the MariaDB connector; its conns record every query text and forward everything else (the
// optional interfaces mirror platform/db/instrument.go, which wraps this one so request deadlines and counting keep
// working).
type spyConnector struct {
	base driver.Connector
	rec  *sqlRecorder
}

func (c spyConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &spyConn{Conn: conn, rec: c.rec}, nil
}

func (c spyConnector) Driver() driver.Driver { return c.base.Driver() }

type spyConn struct {
	driver.Conn
	rec *sqlRecorder
}

var (
	_ driver.ConnPrepareContext = (*spyConn)(nil)
	_ driver.ConnBeginTx        = (*spyConn)(nil)
	_ driver.QueryerContext     = (*spyConn)(nil)
	_ driver.ExecerContext      = (*spyConn)(nil)
	_ driver.Pinger             = (*spyConn)(nil)
	_ driver.SessionResetter    = (*spyConn)(nil)
	_ driver.Validator          = (*spyConn)(nil)
	_ driver.NamedValueChecker  = (*spyConn)(nil)
)

func (c *spyConn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *spyConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	c.rec.add(query)
	var (
		st  driver.Stmt
		err error
	)
	if p, ok := c.Conn.(driver.ConnPrepareContext); ok {
		st, err = p.PrepareContext(ctx, query)
	} else {
		st, err = c.Conn.Prepare(query)
	}
	if err != nil {
		return nil, err
	}
	return &spyStmt{Stmt: st, conn: c.Conn, query: query, rec: c.rec}, nil
}

func (c *spyConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if b, ok := c.Conn.(driver.ConnBeginTx); ok {
		return b.BeginTx(ctx, opts)
	}
	return c.Conn.Begin() //nolint:staticcheck // fallback for drivers without BeginTx
}

func (c *spyConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	q, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	c.rec.add(query)
	return q.QueryContext(ctx, query, args)
}

func (c *spyConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	e, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	c.rec.add(query)
	return e.ExecContext(ctx, query, args)
}

func (c *spyConn) Ping(ctx context.Context) error {
	if p, ok := c.Conn.(driver.Pinger); ok {
		return p.Ping(ctx)
	}
	return nil
}

func (c *spyConn) ResetSession(ctx context.Context) error {
	if r, ok := c.Conn.(driver.SessionResetter); ok {
		return r.ResetSession(ctx)
	}
	return nil
}

func (c *spyConn) IsValid() bool {
	if v, ok := c.Conn.(driver.Validator); ok {
		return v.IsValid()
	}
	return true
}

func (c *spyConn) CheckNamedValue(nv *driver.NamedValue) error {
	if n, ok := c.Conn.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

// spyStmt records each execution of a prepared statement too (a statement prepared once and run again).
type spyStmt struct {
	driver.Stmt
	conn  driver.Conn
	query string
	rec   *sqlRecorder
}

func (s *spyStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	s.rec.add(s.query)
	if q, ok := s.Stmt.(driver.StmtQueryContext); ok {
		return q.QueryContext(ctx, args)
	}
	return nil, fmt.Errorf("spy: %T has no QueryContext", s.Stmt)
}

func (s *spyStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	s.rec.add(s.query)
	if e, ok := s.Stmt.(driver.StmtExecContext); ok {
		return e.ExecContext(ctx, args)
	}
	return nil, fmt.Errorf("spy: %T has no ExecContext", s.Stmt)
}

func (s *spyStmt) CheckNamedValue(nv *driver.NamedValue) error {
	if n, ok := s.Stmt.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	if n, ok := s.conn.(driver.NamedValueChecker); ok {
		return n.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

func (s *spyStmt) ColumnConverter(idx int) driver.ValueConverter { //nolint:staticcheck // forwarded for drivers that use it
	if cc, ok := s.Stmt.(driver.ColumnConverter); ok { //nolint:staticcheck // see above
		return cc.ColumnConverter(idx)
	}
	return driver.DefaultParameterConverter
}

// paywallEnv is the production Mount over the spied pool; seed is the plain test pool (not recorded).
type paywallEnv struct {
	t    *testing.T
	seed *sql.DB
	app  *fiber.App
	rec  *sqlRecorder
	iss  *passport.Issuer
	mr   *miniredis.Miniredis
}

func newPaywallEnv(t *testing.T) *paywallEnv {
	t.Helper()
	seed := testdb.New(t)
	cfg, err := mysql.ParseDSN(testdb.DSN(t))
	require.NoError(t, err)
	cfg.InterpolateParams = true // as platformdb.OpenDSN: the production wire protocol
	connector, err := mysql.NewConnector(cfg)
	require.NoError(t, err)
	rec := &sqlRecorder{}
	db := sql.OpenDB(platformdb.Instrument(spyConnector{base: connector, rec: rec}))
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.Ping())

	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = seed.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-0000000a0129', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	Mount(app, &Deps{
		Config: &config.Config{
			App:         config.App{Env: "testing", URL: "http://localhost"},
			SMS:         config.SMS{Provider: "log"},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keysPath,
		},
		DB:     db,
		Cache:  cache.NewFromClient(rdb, "ritme-go-pg29:"),
		Logger: quiet,
	})
	return &paywallEnv{t: t, seed: seed, app: app, rec: rec, iss: passport.NewIssuer(keys.Private, authstore.New(seed), clock.Real{}, 365), mr: mr}
}

// user creates a free account (no plus rows); pregnantLMP non-empty also writes a pregnancy profile dated by it.
func (e *paywallEnv) user(mobile, pregnantLMP string) (uint64, string) {
	e.t.Helper()
	res, err := e.seed.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	if pregnantLMP != "" {
		_, err = e.seed.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
			VALUES (?, 1, 'lmp', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id, pregnantLMP)
		require.NoError(e.t, err)
	}
	var n int
	require.NoError(e.t, e.seed.QueryRow(`SELECT (SELECT COUNT(*) FROM plus_subscriptions WHERE user_id = ?) + (SELECT COUNT(*) FROM plus_trials WHERE user_id = ?)`, id, id).Scan(&n))
	require.Zero(e.t, n, "a free user")
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test id
	require.NoError(e.t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test id
}

// send runs one request at the instant at and decodes the envelope.
func (e *paywallEnv) send(method, path, token string, body any, at time.Time) reply {
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
	req.Header.Set(clock.Header, at.Format(time.RFC3339))
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	out := reply{Status: resp.StatusCode, Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

// guarded runs one guarded request as its own subtest and asserts the guarantee: the expected 2xx (or want), never
// 402/429 or a plus error code, and no plus_ statement while it ran. The write throttle (a non-Plus 429) is reset
// first so the long scenarios never trip it.
func (e *paywallEnv) guarded(name, method, path, token string, body any, at time.Time, want int) reply {
	e.t.Helper()
	var out reply
	e.t.Run(name, func(t *testing.T) {
		e.mr.FlushAll()
		e.rec.reset()
		out = e.send(method, path, token, body, at)
		stmts, total := e.rec.plusStatements()
		assert.NotEqual(t, fiber.StatusPaymentRequired, out.Status, out.Raw)
		assert.NotEqual(t, fiber.StatusTooManyRequests, out.Status, out.Raw)
		assert.NotContains(t, out.Raw, `"error_code":"`+plus.CodePlusRequired+`"`)
		assert.NotContains(t, out.Raw, `"error_code":"`+plus.CodeQuotaExceeded+`"`)
		assert.Equal(t, want, out.Status, "%s %s: %s", method, path, out.Raw)
		assert.Positive(t, total, "the spy saw the request's SQL")
		assert.Empty(t, stmts, "%s %s consulted Ritme Plus", method, path)
	})
	return out
}

func pgID(t *testing.T, v any) string {
	t.Helper()
	f, ok := v.(float64)
	require.True(t, ok, "id %v", v)
	return strconv.FormatUint(uint64(f), 10)
}

func pgObj(v any) map[string]any { m, _ := v.(map[string]any); return m }

func pgList(v any) []any { l, _ := v.([]any); return l }

func TestPaywallGuard_FreeUserSafetyRoutes(t *testing.T) {
	e := newPaywallEnv(t)
	at := pgNow
	_, tok := e.user("09120029001", "")

	t.Run("positive control: the spy sees a Plus route", func(t *testing.T) {
		e.rec.reset()
		r := e.send(fiber.MethodGet, "/api/v1/plus/status", tok, nil, at)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		stmts, _ := e.rec.plusStatements()
		assert.NotEmpty(t, stmts, "GET /plus/status resolves the tier on plus_subscriptions / plus_trials")
	})

	// Emergency card: owner read, lock-screen card, edit, public link on/off and the public (no-auth) card.
	const card = "/api/v1/health-record/emergency-card"
	e.guarded("emergency show", fiber.MethodGet, card, tok, nil, at, fiber.StatusOK)
	e.guarded("emergency update", fiber.MethodPut, card, tok, map[string]any{
		"show_on_lock_screen": true, "show_pregnancy": true,
		"emergency_contact": map[string]any{"name": "Ali", "relation": "partner", "phone": "09121234567"},
	}, at, fiber.StatusOK)
	e.guarded("emergency lock", fiber.MethodGet, card+"/lock", tok, nil, at, fiber.StatusOK)
	r := e.guarded("emergency public-link on", fiber.MethodPost, card+"/public-link", tok, nil, at, fiber.StatusCreated)
	if pub, _ := r.data()["token"].(string); assert.NotEmpty(t, pub, r.Raw) {
		e.guarded("emergency public card", fiber.MethodGet, "/api/v1/emergency-cards/"+pub, "", nil, at, fiber.StatusOK)
	}
	e.guarded("emergency public-link off", fiber.MethodDelete, card+"/public-link", tok, nil, at, fiber.StatusOK)

	// Medication reminders (care) and their intakes.
	e.guarded("care enums", fiber.MethodGet, "/api/v1/care/enums", tok, nil, at, fiber.StatusOK)
	r = e.guarded("medication store", fiber.MethodPost, "/api/v1/care/medications", tok, map[string]any{
		"title": "Folic acid", "dose": "400", "unit": "mcg", "form": "tablet", "times": []string{"08:00", "20:00"},
	}, at, fiber.StatusCreated)
	if id, ok := r.data()["id"]; assert.True(t, ok, r.Raw) {
		med := "/api/v1/care/medications/" + pgID(t, id)
		e.guarded("medication list", fiber.MethodGet, "/api/v1/care/medications", tok, nil, at, fiber.StatusOK)
		e.guarded("medication show", fiber.MethodGet, med, tok, nil, at, fiber.StatusOK)
		e.guarded("medication update", fiber.MethodPut, med, tok, map[string]any{"notes": "after breakfast"}, at, fiber.StatusOK)
		e.guarded("care today", fiber.MethodGet, "/api/v1/care/today", tok, nil, at, fiber.StatusOK)
		intake := map[string]any{"date": "2026-10-03", "slot": "08:00"}
		e.guarded("medication intake take", fiber.MethodPost, med+"/intakes", tok, intake, at, fiber.StatusOK)
		e.guarded("medication intake untake", fiber.MethodDelete, med+"/intakes", tok, intake, at, fiber.StatusOK)
		e.guarded("medication destroy", fiber.MethodDelete, med, tok, nil, at, fiber.StatusOK)
	}

	// Legacy reminders (medication type).
	e.guarded("reminder enums", fiber.MethodGet, "/api/v1/reminders/enums", tok, nil, at, fiber.StatusOK)
	r = e.guarded("reminder store", fiber.MethodPost, "/api/v1/reminders", tok, map[string]any{
		"type": "medication", "title": "Iron", "recurrence": "daily", "recurrence_time": "09:00",
	}, at, fiber.StatusCreated)
	if id, ok := r.data()["id"]; assert.True(t, ok, r.Raw) {
		rem := "/api/v1/reminders/" + pgID(t, id)
		e.guarded("reminder list", fiber.MethodGet, "/api/v1/reminders", tok, nil, at, fiber.StatusOK)
		e.guarded("reminder update", fiber.MethodPut, rem, tok, map[string]any{"title": "Iron 30mg"}, at, fiber.StatusOK)
		e.guarded("reminder destroy", fiber.MethodDelete, rem, tok, nil, at, fiber.StatusOK)
	}

	// Vitals: a crisis BP runs the urgent alert path.
	e.guarded("vitals hub", fiber.MethodGet, "/api/v1/vitals", tok, nil, at, fiber.StatusOK)
	e.guarded("vitals thresholds", fiber.MethodGet, "/api/v1/vitals/thresholds", tok, nil, at, fiber.StatusOK)
	r = e.guarded("vitals crisis BP", fiber.MethodPost, "/api/v1/vitals/readings", tok, map[string]any{
		"type": "bp", "systolic": 185, "diastolic": 125,
	}, at, fiber.StatusCreated)
	assert.NotNil(t, r.data()["alert"], "the crisis alert ran: %s", r.Raw)
	if reading := pgObj(r.data()["reading"]); assert.NotNil(t, reading, r.Raw) {
		e.guarded("vitals reading show", fiber.MethodGet, "/api/v1/vitals/readings/"+pgID(t, reading["id"]), tok, nil, at, fiber.StatusOK)
	}
	e.guarded("vitals readings", fiber.MethodGet, "/api/v1/vitals/readings", tok, nil, at, fiber.StatusOK)

	// Alerts inbox: one urgent safety row (ack, then dismiss) and one clinical row (visit note).
	uid := e.seedInboxUser()
	safetyID, clinicalID := e.inboxRow(uid, rules.KindSafety, rules.LevelFollowSoon, 1000), e.inboxRow(uid, rules.KindClinical, rules.LevelWorthFollowUp, 500)
	r = e.guarded("alerts inbox list", fiber.MethodGet, "/api/v1/alerts", tok, nil, at, fiber.StatusOK)
	assert.Len(t, pgList(r.data()["items"]), 2, r.Raw)
	e.guarded("alerts inbox ack", fiber.MethodPost, fmt.Sprintf("/api/v1/alerts/%d/ack", safetyID), tok, nil, at, fiber.StatusOK)
	e.guarded("alerts inbox dismiss", fiber.MethodPost, fmt.Sprintf("/api/v1/alerts/%d/dismiss", safetyID), tok, nil, at, fiber.StatusOK)
	e.guarded("alerts inbox visit-note", fiber.MethodPost, fmt.Sprintf("/api/v1/alerts/%d/visit-note", clinicalID), tok, nil, at, fiber.StatusOK)
	e.guarded("alerts inbox history", fiber.MethodGet, "/api/v1/alerts?state=history", tok, nil, at, fiber.StatusOK)
	e.guarded("alerts inbox missing id", fiber.MethodPost, "/api/v1/alerts/999999/ack", tok, nil, at, fiber.StatusNotFound)

	// Postpartum recovery (heavy lochia → the red-flag alert) and EPDS (item 10 positive → the urgent safety path).
	_, mom := e.user("09120029002", "")
	e.guarded("postpartum activate", fiber.MethodPost, "/api/v1/postpartum/activate", mom, map[string]any{"birth_date": "2026-09-14"}, at, fiber.StatusOK)
	e.guarded("postpartum show", fiber.MethodGet, "/api/v1/postpartum", mom, nil, at, fiber.StatusOK)
	e.guarded("postpartum recovery show", fiber.MethodGet, "/api/v1/postpartum/recovery", mom, nil, at, fiber.StatusOK)
	r = e.guarded("postpartum recovery save (red flag)", fiber.MethodPut, "/api/v1/postpartum/recovery", mom, map[string]any{
		"lochia_amount": "heavy", "pain_level": "moderate", "pain_locations": []string{"stitches"},
	}, at, fiber.StatusOK)
	assert.NotEmpty(t, pgList(r.data()["alerts"]), "the heavy bleeding alert ran: %s", r.Raw)
	e.guarded("epds questions", fiber.MethodGet, "/api/v1/postpartum/epds/questions", mom, nil, at, fiber.StatusOK)
	e.guarded("epds questions full", fiber.MethodGet, "/api/v1/postpartum/epds/questions?kind=full", mom, nil, at, fiber.StatusOK)
	answers := map[string]int{"q10": 1}
	for i := 1; i <= 9; i++ {
		answers[fmt.Sprintf("q%d", i)] = 0
	}
	r = e.guarded("epds save (item 10 positive)", fiber.MethodPost, "/api/v1/postpartum/epds", mom, map[string]any{"kind": "full", "answers": answers}, at, fiber.StatusOK)
	assert.Equal(t, "urgent", pgObj(r.data()["safety"])["level"], "the self-harm safety branch ran: %s", r.Raw)
	e.guarded("epds history", fiber.MethodGet, "/api/v1/postpartum/epds", mom, nil, at, fiber.StatusOK)

	e.pregnancyRoutes(at)
}

// pregnancyRoutes covers the pregnancy alert (v1 + v2), day-log, fetal-movement, kick and contraction routes for a
// pregnant free user (LMP 2026-02-20: week 33 on the fixture day).
func (e *paywallEnv) pregnancyRoutes(at time.Time) {
	t := e.t
	_, tok := e.user("09120029003", "2026-02-20")
	const v2 = "/api/v1/pregnancy/v2"
	e.guarded("pregnancy status", fiber.MethodGet, "/api/v1/pregnancy/status", tok, nil, at, fiber.StatusOK)
	r := e.guarded("pregnancy day log (spotting → urgent alert)", fiber.MethodPut, v2+"/days/2026-10-03", tok,
		map[string]any{"symptoms": map[string]any{"spotting": "severe"}}, at, fiber.StatusOK)
	assert.NotEmpty(t, pgList(r.data()["alerts"]), "the critical-symptom alert ran: %s", r.Raw)
	e.guarded("pregnancy fetal movement (reduced)", fiber.MethodPost, "/api/v1/pregnancy/fetal-movement", tok, map[string]any{
		"log_date": "2026-10-03", "pregnancy_week": 33, "movement_status": "reduced", "movement_count": 3,
	}, at, fiber.StatusCreated)

	r = e.guarded("pregnancy v2 alerts", fiber.MethodGet, v2+"/alerts", tok, nil, at, fiber.StatusOK)
	if list := pgList(r.data()["alerts"]); assert.NotEmpty(t, list, r.Raw) {
		id := pgID(t, pgObj(list[0])["id"])
		e.guarded("pregnancy v2 alert ack", fiber.MethodPost, v2+"/alerts/"+id+"/actions/ack", tok, nil, at, fiber.StatusOK)
		e.guarded("pregnancy v2 alert visit note", fiber.MethodPost, v2+"/alerts/"+id+"/actions/add_to_visit_note", tok, nil, at, fiber.StatusOK)
	}

	const v1 = "/api/v1/pregnancy/alerts"
	e.guarded("pregnancy v1 alert summary", fiber.MethodGet, v1+"/summary", tok, nil, at, fiber.StatusOK)
	r = e.guarded("pregnancy v1 alerts", fiber.MethodGet, v1, tok, nil, at, fiber.StatusOK)
	if list := pgList(r.data()["alerts"]); assert.NotEmpty(t, list, r.Raw) {
		id := pgID(t, pgObj(list[len(list)-1])["id"])
		e.guarded("pregnancy v1 alert show", fiber.MethodGet, v1+"/"+id, tok, nil, at, fiber.StatusOK)
		e.guarded("pregnancy v1 alert read", fiber.MethodPost, v1+"/"+id+"/read", tok, nil, at, fiber.StatusOK)
		e.guarded("pregnancy v1 alert dismiss", fiber.MethodPost, v1+"/"+id+"/dismiss", tok, nil, at, fiber.StatusOK)
	}
	e.guarded("pregnancy v1 alerts read-all", fiber.MethodPost, v1+"/read-all", tok, nil, at, fiber.StatusOK)

	// Kick counter.
	const k = "/api/v1/pregnancy/kick-sessions"
	r = e.guarded("kicks start", fiber.MethodPost, k, tok, nil, at, fiber.StatusCreated)
	if id, ok := r.data()["id"]; assert.True(t, ok, r.Raw) {
		s := k + "/" + pgID(t, id)
		e.guarded("kicks kick", fiber.MethodPost, s+"/kicks", tok, nil, at.Add(time.Minute), fiber.StatusOK)
		e.guarded("kicks kick again", fiber.MethodPost, s+"/kicks", tok, nil, at.Add(2*time.Minute), fiber.StatusOK)
		e.guarded("kicks undo", fiber.MethodDelete, s+"/kicks", tok, nil, at.Add(3*time.Minute), fiber.StatusOK)
		e.guarded("kicks overview", fiber.MethodGet, k, tok, nil, at.Add(4*time.Minute), fiber.StatusOK)
		e.guarded("kicks stop", fiber.MethodPost, s+"/stop", tok, nil, at.Add(5*time.Minute), fiber.StatusOK)
		e.guarded("kicks destroy", fiber.MethodDelete, s, tok, nil, at.Add(6*time.Minute), fiber.StatusOK)
	}

	// Contraction timer: 13 contractions every 5 minutes, 60 s each — the 13th stop meets 5-1-1 and raises the
	// urgent alert.
	const c = "/api/v1/pregnancy/contractions"
	t0 := at.Add(time.Hour)
	var session string
	for i := range 13 {
		start := t0.Add(time.Duration(i) * 5 * time.Minute)
		r = e.guarded(fmt.Sprintf("contraction %d start", i+1), fiber.MethodPost, c+"/start", tok, nil, start, fiber.StatusOK)
		if id, ok := r.data()["id"]; ok {
			session = pgID(t, id)
		}
		r = e.guarded(fmt.Sprintf("contraction %d stop", i+1), fiber.MethodPost, c+"/stop", tok, nil, start.Add(time.Minute), fiber.StatusOK)
	}
	assert.NotEmpty(t, pgList(r.data()["alerts"]), "the 5-1-1 alert ran: %s", r.Raw)
	end := t0.Add(62 * time.Minute)
	e.guarded("contractions overview", fiber.MethodGet, c, tok, nil, end, fiber.StatusOK)
	if assert.NotEmpty(t, session) {
		e.guarded("contraction session show", fiber.MethodGet, c+"/sessions/"+session, tok, nil, end, fiber.StatusOK)
		e.guarded("contraction session finish", fiber.MethodPost, c+"/sessions/"+session+"/finish", tok, nil, end, fiber.StatusOK)
		e.guarded("contraction session destroy", fiber.MethodDelete, c+"/sessions/"+session, tok, nil, end, fiber.StatusOK)
	}
}

// seedInboxUser returns the id of the first account of the test (09120029001).
func (e *paywallEnv) seedInboxUser() uint64 {
	e.t.Helper()
	var id uint64
	require.NoError(e.t, e.seed.QueryRow(`SELECT id FROM users WHERE mobile = '09120029001'`).Scan(&id))
	return id
}

// inboxRow writes a served alerts_inbox rule (its copy) and one open inbox row of it for uid; no route of this test
// runs the feed fan-out, so the unconditional rule reaches nobody else.
func (e *paywallEnv) inboxRow(uid uint64, kind, level string, priority int) uint64 {
	e.t.Helper()
	content := `{"a":{"title":"Alert ` + kind + `","body":"body"},"seen":"what we saw","confidence":"how sure","contact":"115"}`
	res, err := e.seed.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, variants_enabled, ab_split,
		conditions, exceptions, cap_count, cap_period, cooldown_days, audience_percent, status, is_active, version,
		approved_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'alerts_inbox', ?, 0, 50, '[]', '[]', NULL, NULL, 0, 100, 'approved', 1, 1, 1, NOW(), NOW())`,
		"paywall guard "+kind, level, kind, priority, content)
	require.NoError(e.t, err)
	ruleID, _ := res.LastInsertId()
	configcache.Changed(context.Background(), rules.Table)
	last := pgNow.Add(-time.Hour)
	res, err = e.seed.Exec(`INSERT INTO message_inbox (user_id, rule_id, version, slot, kind, level, state, hit_count, evidence,
		first_at, last_at, created_at, updated_at) VALUES (?, ?, 1, 'alerts_inbox', ?, ?, 'active', 1, '{"vars":{}}', ?, ?, NOW(), NOW())`,
		uid, ruleID, kind, level, last, last)
	require.NoError(e.t, err)
	id, _ := res.LastInsertId()
	return uint64(id) //nolint:gosec // test id
}

// paywallFreeDirs are the safety packages that must never import internal/plus (relative to this file). Direct
// imports only: internal/emergency reaches plus transitively today (emergency → healthrecord → analysis → plus, for
// the analysis range gate), which is fine as long as the emergency code itself never calls it — the runtime test
// above proves the emergency routes never consult it.
var paywallFreeDirs = []string{
	"../emergency",
	"../reminder",
	"../reminders",
	"../care",
	"../postpartum",
	"../safety",
	"../messages/inbox",
}

// TestPaywallGuard_SafetyPackagesNeverImportPlus needs no database (it runs in every `go test`).
func TestPaywallGuard_SafetyPackagesNeverImportPlus(t *testing.T) {
	const plusPkg = "github.com/ritme/backend-go/internal/plus"
	for _, dir := range paywallFreeDirs {
		t.Run(dir, func(t *testing.T) {
			info, err := os.Stat(dir)
			require.NoError(t, err, "a guarded package was moved or renamed: update paywallFreeDirs")
			require.True(t, info.IsDir())
			files := 0
			err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return nil
				}
				f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
				if err != nil {
					return err
				}
				files++
				for _, imp := range f.Imports {
					p, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						return err
					}
					assert.False(t, p == plusPkg || strings.HasPrefix(p, plusPkg+"/"), "%s imports %s: no Plus on safety paths (D-178)", path, p)
				}
				return nil
			})
			require.NoError(t, err)
			assert.Positive(t, files, "no Go files under %s", dir)
		})
	}
}
