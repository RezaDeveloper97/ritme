package inbox_test

// T-M10-07 (D-164): the alerts inbox lifecycle over MariaDB — merge, resolve, archive, missing evidence, the push slot
// — and the /api/v1/alerts routes plus the feed fan-out over the real Mount.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/inbox"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/usertz"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

func seedUser(t *testing.T, db *sql.DB, mobile, zone string) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Test', ?, NOW(), NOW())`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	var tz any
	if zone != "" {
		tz = zone
	}
	last := civildate.InTehran(time.Now()).AddDays(-10)
	_, err = db.Exec(`INSERT INTO user_profiles (user_id, birthday, last_period_start, cycle_duration, period_duration, user_goal, timezone, created_at, updated_at)
		VALUES (?, '1998-01-01', ?, 28, 5, 'non_ttc', ?, NOW(), NOW())`, id, last.AddDays(-140).String(), tz)
	require.NoError(t, err)
	for i := 5; i >= 0; i-- {
		s := last.AddDays(-28 * i)
		_, err = db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed,
			is_estimated, source, created_at, updated_at) VALUES (?, ?, ?, 5, 1, 0, 'user_logged', NOW(), NOW())`,
			id, s.String(), s.AddDays(4).String())
		require.NoError(t, err)
	}
	return uint64(id) //nolint:gosec // test id
}

func rule(id uint64, level, kind string, conds ...rules.Condition) rules.Rule {
	r := rules.Rule{ID: id, Level: level, Kind: kind, Placement: inbox.Slot, IsActive: true, Status: rules.StatusApproved,
		AudiencePercent: 100, HoursFrom: -1, HoursTo: -1, Version: 1, ApprovedVersion: 1, Priority: 500,
		Content: rules.Content{A: rules.Variant{Title: "BP {bp_sys}", Body: "Recheck"}, Seen: "{bp_sys} systolic",
			Confidence: "From your readings", Contact: "115"}}
	if kind == rules.KindSafety {
		r.Priority = 1000
	}
	if len(conds) > 0 {
		r.Groups = []rules.Group{{Op: "and", Items: conds}}
	}
	return r
}

var bpHigh = rules.Condition{Fact: "bp_sys", Op: "gte", Value: 140.0}

type svcEnv struct {
	t   *testing.T
	db  *sql.DB
	s   *inbox.Service
	uid uint64
}

func newSvc(t *testing.T, rs ...rules.Rule) *svcEnv {
	t.Helper()
	db := testdb.New(t)
	comp := compiled.CompileRules(rs)
	s := inbox.New(store.New(db), db, nil).WithLoader(func(context.Context) (*compiled.Compiled, error) { return comp, nil })
	return &svcEnv{t: t, db: db, s: s, uid: seedUser(t, db, "09120007001", "")}
}

type rowState struct {
	State   string
	Hits    int
	LastAt  time.Time
	AckedAt sql.NullTime
}

func (e *svcEnv) rows(ruleID uint64) []rowState {
	e.t.Helper()
	rs, err := e.db.Query(`SELECT state, hit_count, last_at, acked_at FROM message_inbox WHERE user_id = ? AND rule_id = ? ORDER BY id`, e.uid, ruleID)
	require.NoError(e.t, err)
	defer func() { _ = rs.Close() }()
	var out []rowState
	for rs.Next() {
		var r rowState
		require.NoError(e.t, rs.Scan(&r.State, &r.Hits, &r.LastAt, &r.AckedAt))
		out = append(out, r)
	}
	require.NoError(e.t, rs.Err())
	return out
}

func (e *svcEnv) eval(f rules.Facts, at time.Time) {
	e.t.Helper()
	_, err := e.s.Evaluate(context.Background(), e.uid, f, at)
	require.NoError(e.t, err)
}

var t0 = time.Date(2026, 10, 30, 10, 0, 0, 0, civildate.Tehran)

// Merge: same evidence key → unchanged (no write); new key → count 2 and new last_at; acknowledged urgent + new key →
// active again (M-16).
func TestInboxInt_Merge(t *testing.T) {
	e := newSvc(t, rule(1, rules.LevelWorthFollowUp, rules.KindClinical, bpHigh), rule(2, rules.LevelFollowSoon, rules.KindSafety, bpHigh))
	e.eval(rules.Facts{"bp_sys": 150.0}, t0)
	r := e.rows(1)
	require.Len(t, r, 1)
	assert.Equal(t, inbox.StateActive, r[0].State)
	assert.Equal(t, 1, r[0].Hits)

	e.eval(rules.Facts{"bp_sys": 150.0}, t0.Add(time.Hour))
	r = e.rows(1)
	assert.Equal(t, 1, r[0].Hits)
	assert.True(t, r[0].LastAt.Equal(t0), "no write without a change")

	e.eval(rules.Facts{"bp_sys": 155.0}, t0.Add(2*time.Hour))
	r = e.rows(1)
	require.Len(t, r, 1)
	assert.Equal(t, 2, r[0].Hits)
	assert.True(t, r[0].LastAt.Equal(t0.Add(2*time.Hour)))

	var urgentID uint64
	require.NoError(t, e.db.QueryRow(`SELECT id FROM message_inbox WHERE user_id = ? AND rule_id = 2`, e.uid).Scan(&urgentID))
	_, err := e.s.Ack(context.Background(), e.uid, urgentID, t0.Add(3*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, inbox.StateAcknowledged, e.rows(2)[0].State)
	e.eval(rules.Facts{"bp_sys": 170.0}, t0.Add(4*time.Hour))
	u := e.rows(2)
	require.Len(t, u, 1)
	assert.Equal(t, inbox.StateActive, u[0].State)
	assert.False(t, u[0].AckedAt.Valid)
	assert.Equal(t, 3, u[0].Hits) // 150 → open, 155 → 2, 170 → 3
}

// Resolve when the condition drops (facts known); a missing fact keeps the row open; the next hit opens a new row.
func TestInboxInt_ResolveAndReopen(t *testing.T) {
	e := newSvc(t, rule(1, rules.LevelWorthFollowUp, rules.KindClinical, bpHigh))
	e.eval(rules.Facts{"bp_sys": 150.0}, t0)
	e.eval(rules.Facts{}, t0.Add(time.Hour))
	assert.Equal(t, inbox.StateActive, e.rows(1)[0].State)
	e.eval(rules.Facts{"bp_sys": 120.0}, t0.Add(2*time.Hour))
	assert.Equal(t, inbox.StateResolved, e.rows(1)[0].State)
	e.eval(rules.Facts{"bp_sys": 150.0}, t0.Add(3*time.Hour))
	r := e.rows(1)
	require.Len(t, r, 2)
	assert.Equal(t, inbox.StateResolved, r[0].State)
	assert.Equal(t, inbox.StateActive, r[1].State)
}

// follow_up without evidence → no row and the metric + 1.
func TestInboxInt_MissingEvidence(t *testing.T) {
	r := rule(1, rules.LevelWorthFollowUp, rules.KindClinical, bpHigh)
	r.Content.Seen = ""
	e := newSvc(t, r)
	before := inbox.MissingEvidenceValue()
	e.eval(rules.Facts{"bp_sys": 150.0}, t0)
	assert.Empty(t, e.rows(1))
	assert.InDelta(t, before+1, inbox.MissingEvidenceValue(), 0)
}

// Archive at day 30 (fake clock), not at day 29; active rows are never archived by age.
func TestInboxInt_ArchiveDay30(t *testing.T) {
	e := newSvc(t, rule(1, rules.LevelWorthFollowUp, rules.KindClinical, bpHigh), rule(2, rules.LevelInfo, rules.KindContent, bpHigh))
	e.eval(rules.Facts{"bp_sys": 150.0}, t0)
	var id uint64
	require.NoError(t, e.db.QueryRow(`SELECT id FROM message_inbox WHERE user_id = ? AND rule_id = 1`, e.uid).Scan(&id))
	_, err := e.s.Ack(context.Background(), e.uid, id, t0.Add(time.Minute))
	require.NoError(t, err)
	ctx := context.Background()
	day29 := t0.AddDate(0, 0, 29)
	_, err = e.s.Archive(ctx, day29)
	require.NoError(t, err)
	assert.Equal(t, inbox.StateAcknowledged, e.rows(1)[0].State)
	n, err := e.s.Archive(ctx, t0.AddDate(0, 0, 30))
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	assert.Equal(t, inbox.StateArchived, e.rows(1)[0].State)
	assert.Equal(t, inbox.StateActive, e.rows(2)[0].State)
}

// The sweep re-evaluates users with open rows (resolve) through the fact builder.
func TestInboxInt_SweepResolves(t *testing.T) {
	e := newSvc(t, rule(1, rules.LevelWorthFollowUp, rules.KindClinical, bpHigh))
	e.eval(rules.Facts{"bp_sys": 150.0}, t0)
	e.s.WithFacts(func(_ context.Context, _ uint64, at time.Time) (rules.Facts, time.Time, error) {
		return rules.Facts{"bp_sys": 118.0}, at, nil
	})
	users, _, err := e.s.Sweep(context.Background(), t0.Add(18*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, users)
	assert.Equal(t, inbox.StateResolved, e.rows(1)[0].State)
}

// ---- push slot ----

func pushRule() rules.Rule {
	return rules.Rule{ID: 50, Level: rules.LevelInfo, Kind: rules.KindEngine, Placement: inbox.PushSlot, IsActive: true,
		Status: rules.StatusApproved, AudiencePercent: 100, HoursFrom: -1, HoursTo: -1, Version: 1, Priority: 900,
		Content: rules.Content{A: rules.Variant{Title: "Your period starts in 2 days", Body: "Log your symptoms",
			CTATarget: "/calendar"}}}
}

func newPush(t *testing.T) (*svcEnv, uint64) {
	t.Helper()
	e := newSvc(t, pushRule())
	auckland := seedUser(t, e.db, "09120007002", "Pacific/Auckland")
	zones := profilestore.New(e.db)
	e.s.WithLanguages(func(context.Context) []string { return []string{"fa", "en"} }).WithPreferences(zones).
		WithFacts(func(ctx context.Context, uid uint64, at time.Time) (rules.Facts, time.Time, error) {
			tz, err := zones.GetUserTimezone(ctx, uid)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, time.Time{}, err
			}
			return rules.Facts{}, at.In(usertz.ZoneOf(tz)), nil
		})
	for _, uid := range []uint64{e.uid, auckland} {
		_, err := e.db.Exec(`INSERT INTO user_activity_days (user_id, day, session, logged) VALUES (?, ?, 1, 0)`, uid,
			civildate.InTehran(time.Now()).String())
		require.NoError(t, err)
	}
	return e, auckland
}

func notifications(t *testing.T, db *sql.DB, uid uint64) []map[string]any {
	t.Helper()
	rs, err := db.Query(`SELECT type, title, body, COALESCE(action_url, ''), data FROM user_notifications WHERE user_id = ?`, uid)
	require.NoError(t, err)
	defer func() { _ = rs.Close() }()
	var out []map[string]any
	for rs.Next() {
		var typ, title, body, url, data string
		require.NoError(t, rs.Scan(&typ, &title, &body, &url, &data))
		out = append(out, map[string]any{"type": typ, "title": title, "body": body, "url": url, "data": data})
	}
	require.NoError(t, rs.Err())
	return out
}

// 09:15 Tehran: the Tehran user gets one neutral notification, the Auckland user (17:45) none; a rerun the same local
// day writes nothing more.
func TestPushInt_LocalNineOncePerDay(t *testing.T) {
	e, auckland := newPush(t)
	now := civildate.InTehran(time.Now()).TehranMidnight().Add(9*time.Hour + 15*time.Minute)
	res, err := e.s.Push(context.Background(), now)
	require.NoError(t, err)
	assert.Equal(t, inbox.PushResult{Candidates: 1, Sent: 1}, res)
	got := notifications(t, e.db, e.uid)
	require.Len(t, got, 1)
	assert.Equal(t, inbox.NotificationType, got[0]["type"])
	assert.Equal(t, "/calendar", got[0]["url"])
	var title, body map[string]string
	require.NoError(t, json.Unmarshal([]byte(got[0]["title"].(string)), &title))
	require.NoError(t, json.Unmarshal([]byte(got[0]["body"].(string)), &body))
	assert.Equal(t, "یادآور ریتمی", title["fa"])
	for _, s := range []string{got[0]["title"].(string), got[0]["body"].(string), got[0]["data"].(string)} {
		assert.NotContains(t, strings.ToLower(s), "period")
		assert.NotContains(t, strings.ToLower(s), "symptom")
	}
	assert.Empty(t, notifications(t, e.db, auckland))

	res, err = e.s.Push(context.Background(), now.Add(20*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 0, res.Sent)
	assert.Len(t, notifications(t, e.db, e.uid), 1)
}

// 23:30 Tehran: the Tehran user is skipped (quiet hours, not 09 local); Auckland is at 09:00 local and gets one.
func TestPushInt_LateNightSkipped(t *testing.T) {
	e, auckland := newPush(t)
	now := time.Date(2026, 10, 31, 23, 30, 0, 0, civildate.Tehran)
	res, err := e.s.Push(context.Background(), now)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Sent)
	assert.Empty(t, notifications(t, e.db, e.uid))
	assert.Len(t, notifications(t, e.db, auckland), 1)
}

// A tip (product) push is gated by the opt-in «tips and articles» switch: off by default → nothing.
func TestPushInt_PreferencesGate(t *testing.T) {
	e, _ := newPush(t)
	p := pushRule()
	p.Kind, p.Priority = rules.KindProduct, 50
	comp := compiled.CompileRules([]rules.Rule{p})
	e.s.WithLoader(func(context.Context) (*compiled.Compiled, error) { return comp, nil })
	now := civildate.InTehran(time.Now()).TehranMidnight().Add(9*time.Hour + 5*time.Minute)
	res, err := e.s.Push(context.Background(), now)
	require.NoError(t, err)
	assert.Equal(t, inbox.PushResult{Candidates: 1, Sent: 0}, res)
}

// 3 pushes in the last 7 local days → the weekly cap holds the fourth.
func TestPushInt_WeeklyCap(t *testing.T) {
	e, _ := newPush(t)
	now := civildate.InTehran(time.Now()).TehranMidnight().Add(9*time.Hour + 5*time.Minute)
	for d := 1; d <= 3; d++ {
		_, err := e.db.Exec(`INSERT INTO message_events (rule_id, user_id, version, variant, slot, event, created_at)
			VALUES (50, ?, 1, 'a', 'push', 'impression', ?)`, e.uid, now.AddDate(0, 0, -d))
		require.NoError(t, err)
	}
	res, err := e.s.Push(context.Background(), now)
	require.NoError(t, err)
	assert.Equal(t, inbox.PushResult{Candidates: 1, Sent: 0}, res)
	assert.Empty(t, notifications(t, e.db, e.uid))
}

// ---- API over the real routes ----

type apiEnv struct {
	t     *testing.T
	db    *sql.DB
	app   *fiber.App
	token string
	uid   uint64
}

func newAPI(t *testing.T) *apiEnv {
	t.Helper()
	db := testdb.New(t)
	keysPath, err := filepath.Abs("../../../contract/fixtures/keys")
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe07', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`)
	require.NoError(t, err)
	uid := seedUser(t, db, "09120007010", "")
	tok, err := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365).Issue(context.Background(), uid, time.Now())
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	apihttp.Mount(app, &apihttp.Deps{
		Config: &config.Config{
			App: config.App{Env: "testing", URL: "http://localhost"}, SMS: config.SMS{Provider: "log"},
			Passport: config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30}, StoragePath: keysPath,
			CycleEngine: config.CycleEngineV6, EngineCache: config.EngineCache{Enabled: true, L1Entries: 1000},
		},
		DB: db, Cache: cache.NewFromClient(rdb, "ritme-go-inbox:"), Logger: quiet,
	})
	t.Cleanup(func() { _ = app.ShutdownWithTimeout(5 * time.Second) })
	return &apiEnv{t: t, db: db, app: app, token: tok.AccessToken, uid: uid}
}

func (e *apiEnv) call(method, path string) (int, map[string]any, nethttp.Header) {
	e.t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(nil))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Authorization", "Bearer "+e.token)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	require.NoError(e.t, err)
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out, resp.Header
}

// dbRule inserts a served rule of the alerts_inbox slot (no conditions: it matches every user).
func (e *apiEnv) dbRule(kind, level string, priority int) uint64 {
	e.t.Helper()
	content := `{"a":{"title":"Alert ` + kind + `","body":"body"},"seen":"what we saw","confidence":"how sure","contact":"115","translations":{"en":{"a":{"title":"EN alert ` + kind + `","body":"en body"}}}}`
	res, err := e.db.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, variants_enabled, ab_split,
		conditions, exceptions, cap_count, cap_period, cooldown_days, audience_percent, status, is_active, version,
		approved_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'alerts_inbox', ?, 0, 50, '[]', '[]', NULL, NULL, 0, 100, 'approved', 1, 1, 1, NOW(), NOW())`,
		kind, level, kind, priority, content)
	require.NoError(e.t, err)
	id, _ := res.LastInsertId()
	configcache.Changed(context.Background(), rules.Table)
	return uint64(id) //nolint:gosec // test id
}

// insertRow writes an inbox row directly (ordering / paging fixtures).
func (e *apiEnv) insertRow(uid, ruleID uint64, level, state string, last time.Time) uint64 {
	e.t.Helper()
	res, err := e.db.Exec(`INSERT INTO message_inbox (user_id, rule_id, version, slot, kind, level, state, hit_count, evidence,
		first_at, last_at, created_at, updated_at) VALUES (?, ?, 1, 'alerts_inbox', 'clinical', ?, ?, 1, '{"vars":{}}', ?, ?, NOW(), NOW())`,
		uid, ruleID, level, state, last, last)
	require.NoError(e.t, err)
	id, _ := res.LastInsertId()
	return uint64(id) //nolint:gosec // test id
}

func items(body map[string]any) []map[string]any {
	data := body["data"].(map[string]any)
	var out []map[string]any
	for _, it := range data["items"].([]any) {
		out = append(out, it.(map[string]any))
	}
	return out
}

func slotIDs(body map[string]any, slot string) []float64 {
	data := body["data"].(map[string]any)
	var out []float64
	for _, it := range data["slots"].(map[string]any)[slot].([]any) {
		out = append(out, it.(map[string]any)["id"].(float64))
	}
	return out
}

// The feed opens the safety row and shows it in urgent_modal; dismiss is 409 ack_required until acknowledged; after
// the ack the modal is empty and dismiss is 200 (archived).
func TestAlertsAPI_FanOutAckDismiss(t *testing.T) {
	e := newAPI(t)
	sid := e.dbRule(rules.KindSafety, rules.LevelFollowSoon, 1000)

	code, body, _ := e.call("GET", "/api/v1/messages/feed")
	require.Equal(t, 200, code, body)
	assert.Equal(t, []float64{float64(sid)}, slotIDs(body, "urgent_modal"))
	alerts := body["data"].(map[string]any)["alerts"].(map[string]any)
	assert.EqualValues(t, 1, alerts["open"])
	assert.EqualValues(t, 1, alerts["urgent"])

	code, body, hdr := e.call("GET", "/api/v1/alerts")
	require.Equal(t, 200, code, body)
	assert.Equal(t, "private, no-store", hdr.Get("Cache-Control"))
	list := items(body)
	require.Len(t, list, 1)
	it := list[0]
	assert.Equal(t, "urgent", it["level"])
	assert.Equal(t, "active", it["state"])
	assert.Equal(t, "EN alert safety", it["title"])
	assert.Equal(t, false, it["dismissible"])
	assert.Equal(t, "115", it["evidence"].(map[string]any)["contact"])
	id := fmt.Sprint(it["id"])

	code, body, _ = e.call("POST", "/api/v1/alerts/"+id+"/dismiss")
	assert.Equal(t, 409, code)
	assert.Equal(t, inbox.CodeAckRequired, body["error_code"])

	code, body, _ = e.call("POST", "/api/v1/alerts/"+id+"/ack")
	require.Equal(t, 200, code, body)
	assert.Equal(t, "acknowledged", body["data"].(map[string]any)["alert"].(map[string]any)["state"])

	code, body, _ = e.call("GET", "/api/v1/messages/feed")
	require.Equal(t, 200, code)
	assert.Empty(t, slotIDs(body, "urgent_modal"))

	code, body, _ = e.call("POST", "/api/v1/alerts/"+id+"/dismiss")
	require.Equal(t, 200, code, body)
	assert.Equal(t, "archived", body["data"].(map[string]any)["alert"].(map[string]any)["state"])
	_, body, _ = e.call("GET", "/api/v1/alerts?state=history")
	assert.Len(t, items(body), 1)
}

// Order urgent → follow_up → suggestion → info, then last_at DESC; cursor paging covers every row once; another
// user's id is 404 for every action; a bad state / cursor is 422.
func TestAlertsAPI_OrderPagingForeign(t *testing.T) {
	e := newAPI(t)
	levels := []string{rules.LevelInfo, rules.LevelSuggestion, rules.LevelWorthFollowUp, rules.LevelFollowSoon}
	base := time.Now().In(civildate.Tehran).Truncate(time.Second).Add(-48 * time.Hour)
	for i := range 24 {
		e.insertRow(e.uid, uint64(1000+i), levels[i%4], inbox.StateActive, base.Add(time.Duration(i)*time.Minute)) //nolint:gosec // small
	}
	var got []map[string]any
	cursor := ""
	for page := 0; page < 5; page++ {
		path := "/api/v1/alerts?state=open"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		code, body, _ := e.call("GET", path)
		require.Equal(t, 200, code, body)
		got = append(got, items(body)...)
		next := body["data"].(map[string]any)["next_cursor"]
		if next == nil {
			break
		}
		cursor = next.(string)
	}
	require.Len(t, got, 24)
	rank := map[string]int{"urgent": 0, "follow_up": 1, "suggestion": 2, "info": 3}
	for i := 1; i < len(got); i++ {
		a, b := got[i-1], got[i]
		ra, rb := rank[a["level"].(string)], rank[b["level"].(string)]
		require.LessOrEqual(t, ra, rb, i)
		if ra == rb {
			require.GreaterOrEqual(t, a["last_at"].(string), b["last_at"].(string), i)
		}
	}
	seen := map[float64]bool{}
	for _, it := range got {
		require.False(t, seen[it["id"].(float64)])
		seen[it["id"].(float64)] = true
	}

	other := seedUser(t, e.db, "09120007011", "")
	foreign := e.insertRow(other, 9, rules.LevelInfo, inbox.StateActive, base)
	for _, act := range []string{"ack", "dismiss"} {
		code, _, _ := e.call("POST", fmt.Sprintf("/api/v1/alerts/%d/%s", foreign, act))
		assert.Equal(t, 404, code, act)
	}
	var state string
	require.NoError(t, e.db.QueryRow(`SELECT state FROM message_inbox WHERE id = ?`, foreign).Scan(&state))
	assert.Equal(t, inbox.StateActive, state)

	code, _, _ := e.call("GET", "/api/v1/alerts?state=bogus")
	assert.Equal(t, 422, code)
	code, _, _ = e.call("GET", "/api/v1/alerts?cursor=%25%25")
	assert.Equal(t, 422, code)
}

func TestAlertsAPI_Unauthenticated(t *testing.T) {
	e := newAPI(t)
	e.token = "nope"
	code, body, _ := e.call("GET", "/api/v1/alerts")
	assert.Equal(t, 401, code)
	assert.NotEmpty(t, body["error_code"])
}
