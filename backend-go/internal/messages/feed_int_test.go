package messages_test

// T-M10-06 (D-163): GET /messages/feed, /messages/selected as its one-slot wrapper, POST /messages/events v6 and the
// status banner seed, over the real routes (internal/http Mount), MariaDB and a miniredis.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
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
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

const feedKeysDir = "../../contract/fixtures/keys"

type feedEnv struct {
	t     *testing.T
	db    *sql.DB
	app   *fiber.App
	token string
	uid   uint64
}

// newFeedEnv mounts every route (CYCLE_ENGINE=v6) over a fresh database and a cycle user whose last period started
// on last.
func newFeedEnv(t *testing.T, mobile string, last civildate.Date) *feedEnv {
	t.Helper()
	db := testdb.New(t)
	keysPath, err := filepath.Abs(feedKeysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe06', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`)
	require.NoError(t, err)
	uid := seedCycleUser(t, db, mobile, "non_ttc", "", last)
	tok, err := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365).Issue(context.Background(), uid, time.Now())
	require.NoError(t, err)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(metrics.HTTP())
	app.Use(func(c fiber.Ctx) error {
		err := c.Next()
		c.Set("X-Test-Queries", strconv.FormatInt(metrics.RequestQueries(c), 10))
		return err
	})
	app.Use(clock.Middleware(clock.Real{}, true))
	apihttp.Mount(app, &apihttp.Deps{
		Config: &config.Config{
			App: config.App{Env: "testing", URL: "http://localhost"}, SMS: config.SMS{Provider: "log"},
			Passport: config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30}, StoragePath: keysPath,
			CycleEngine: config.CycleEngineV6,
			EngineCache: config.EngineCache{Enabled: true, L1Entries: 1000},
		},
		DB: db, Cache: cache.NewFromClient(rdb, "ritme-go-feed:"), Logger: quiet,
	})
	t.Cleanup(func() { _ = app.ShutdownWithTimeout(5 * time.Second) })
	return &feedEnv{t: t, db: db, app: app, token: tok.AccessToken, uid: uid}
}

func (e *feedEnv) call(method, path, at string, body any) (int, map[string]any, nethttp.Header) {
	e.t.Helper()
	var rd io.Reader = nethttp.NoBody
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set(clock.Header, at)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	require.NoError(e.t, err)
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out, resp.Header
}

func (e *feedEnv) exec(q string, args ...any) {
	e.t.Helper()
	_, err := e.db.Exec(q, args...)
	require.NoError(e.t, err)
}

func (e *feedEnv) count(q string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

// rule inserts an approved, active rule of kind in placement and publishes the change.
func (e *feedEnv) rule(kind, level, placement string, priority int, extra string) uint64 {
	e.t.Helper()
	return e.ruleWhen(kind, level, placement, priority, extra, "[]")
}

// ruleWhen is rule with the conditions JSON.
func (e *feedEnv) ruleWhen(kind, level, placement string, priority int, extra, conds string) uint64 {
	e.t.Helper()
	content := fmt.Sprintf(`{"a":{"title":"%s %d","body":"body","cta_label":"Go","cta_target":"ritme://x"},"seen":"s","confidence":"c","contact":"tel:115","translations":{"en":{"a":{"title":"EN %s %d","body":"en body","cta_label":"Go en"}}}}`,
		placement, priority, placement, priority)
	res, err := e.db.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, variants_enabled, ab_split,
		conditions, exceptions, cap_count, cap_period, cooldown_days, audience_percent, status, is_active, version,
		approved_version, goal_event, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, 50, ?, '[]', NULL, NULL, 0, 100, 'approved', 1, 2, 2, `+extra+`, NOW(), NOW())`,
		placement, level, kind, priority, placement, content, conds)
	require.NoError(e.t, err)
	id, _ := res.LastInsertId()
	configcache.Changed(context.Background(), rules.Table)
	return uint64(id) //nolint:gosec // test id
}

func slotItems(body map[string]any, slot string) []map[string]any {
	data := body["data"].(map[string]any)
	raw := data["slots"].(map[string]any)[slot].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(map[string]any))
	}
	return out
}

func (e *feedEnv) eventually(q string, want int, args ...any) {
	e.t.Helper()
	require.Eventually(e.t, func() bool { return e.count(q, args...) == want }, 5*time.Second, 50*time.Millisecond,
		"%s: want %d, have %d", q, want, e.count(q, args...))
}

const (
	feedDay1 = "2026-10-21T10:00:00+03:30"
	feedDay2 = "2026-10-22T10:00:00+03:30"
)

func TestFeedInt_SlotsShownEligibleAndSelected(t *testing.T) {
	configcache.SetEnabled(true)
	defer configcache.SetEnabled(false)
	e := newFeedEnv(t, "09120006101", civildate.MustParse("2026-10-05"))
	urgent := e.rule(rules.KindSafety, rules.LevelFollowSoon, "urgent_modal", 1000, "NULL")
	tip := e.rule(rules.KindClinical, rules.LevelInfo, "smart_tip", 500, "'period_logged'")
	var advice []uint64
	for i := range 4 {
		advice = append(advice, e.rule(rules.KindContent, rules.LevelInfo, "daily_advice", 400-i, "NULL"))
	}
	challenge := e.rule(rules.KindProduct, rules.LevelSuggestion, "daily_challenge", 50, "NULL")
	shown := `SELECT COUNT(*) FROM message_events WHERE user_id = ? AND event = 'impression'`
	shownOf := `SELECT COUNT(*) FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'impression'`

	// /messages/selected: the pre-feed shape, for one slot of the feed (and it logs that slot's shown).
	st, body, _ := e.call(fiber.MethodGet, "/api/v1/messages/selected?placement=daily_challenge", feedDay1, nil)
	require.Equal(t, 200, st, body)
	msg := body["data"].(map[string]any)["message"].(map[string]any)
	keys := make([]string, 0, len(msg))
	for k := range msg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	assert.Equal(t, []string{"body", "cta", "dismissible", "kind", "level", "placement", "rule_id", "title", "variant"}, keys)
	assert.EqualValues(t, challenge, msg["rule_id"])
	assert.Equal(t, "EN daily_challenge 50", msg["title"], "the en translation of the content")
	assert.Equal(t, "ritme://x", msg["cta"].(map[string]any)["target"], "the CTA target falls back to the base text")
	e.eventually(shownOf, 1, e.uid, challenge)

	// The feed: every default slot, in order, per-slot winners.
	st, body, hdr := e.call(fiber.MethodGet, "/api/v1/messages/feed", feedDay1, nil)
	require.Equal(t, 200, st, body)
	assert.Equal(t, "private, no-store", hdr.Get("Cache-Control"))
	data := body["data"].(map[string]any)
	assert.Positive(t, data["rules_rev"])
	assert.Equal(t, map[string]any{"open": float64(0), "urgent": float64(0)}, data["alerts"])
	assert.Len(t, data["slots"].(map[string]any), len(feed.DefaultSlots))
	u := slotItems(body, "urgent_modal")
	require.Len(t, u, 1)
	assert.EqualValues(t, urgent, u[0]["id"])
	assert.Equal(t, "urgent", u[0]["level"], "the spec level name")
	assert.Equal(t, false, u[0]["dismissible"])
	assert.EqualValues(t, 2, u[0]["version"], "the approved version")
	assert.Equal(t, map[string]any{"seen": "s", "confidence": "c", "contact": "tel:115"}, u[0]["evidence"])
	require.Len(t, slotItems(body, "smart_tip"), 1)
	assert.EqualValues(t, tip, slotItems(body, "smart_tip")[0]["id"])
	assert.NotContains(t, slotItems(body, "smart_tip")[0], "evidence", "an info message has no evidence")
	da := slotItems(body, "daily_advice")
	require.Len(t, da, 3, "daily_advice holds 3")
	for i, it := range da {
		assert.EqualValues(t, advice[i], it["id"], "priority order")
	}
	assert.EqualValues(t, challenge, slotItems(body, "daily_challenge")[0]["id"])
	served := 0
	for _, s := range feed.DefaultSlots {
		served += len(slotItems(body, s))
	}

	// shown once per (message, slot, local day): selected's challenge shown is not logged again.
	e.eventually(shown, served, e.uid)
	assert.Equal(t, 1, e.count(shownOf, e.uid, challenge))
	// eligible: the top 5 of daily_advice (4 candidates, content kind = always logged); the product challenge only
	// for a sampled user.
	e.eventually(`SELECT COUNT(*) FROM message_events WHERE user_id = ? AND event = 'eligible' AND slot = 'daily_advice'`, 4, e.uid)
	wantChallenge := 0
	if feed.Sampled(e.uid) {
		wantChallenge = 1
	}
	assert.Equal(t, wantChallenge, e.count(`SELECT COUNT(*) FROM message_events WHERE user_id = ? AND event = 'eligible' AND rule_id = ?`, e.uid, challenge))
	assert.Equal(t, feed.Sampled(e.uid), feed.Sampled(e.uid), "stable per user")
	var ver sql.NullInt64
	var slot sql.NullString
	require.NoError(t, e.db.QueryRow(`SELECT version, slot FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'impression'`, e.uid, tip).Scan(&ver, &slot))
	assert.Equal(t, []any{int64(2), "smart_tip"}, []any{ver.Int64, slot.String})

	// A second fetch the same local day adds no shown for a message served again. (daily_advice is left out: its
	// 1-per-week message cap counts the shown just logged, so a refetch serves the next advice, a first shown.)
	st, body, _ = e.call(fiber.MethodGet, "/api/v1/messages/feed?slots=urgent_modal,daily_challenge", feedDay1, nil)
	require.Equal(t, 200, st, body)
	require.Len(t, slotItems(body, "urgent_modal"), 1)
	assert.Empty(t, slotItems(body, "daily_challenge"), "no product message right after a safety one (SPEC §5.4.1)")
	time.Sleep(1500 * time.Millisecond) // past one flush interval
	assert.Equal(t, served, e.count(shown, e.uid))

	// A switched-off module empties its slot: daily_advice still has a candidate today (advice[3]).
	feed.SlotModules["daily_advice"] = "todo"
	e.exec(`INSERT INTO app_modules (` + "`key`" + `, enabled, created_at, updated_at) VALUES ('todo', 0, NOW(), NOW())
		ON DUPLICATE KEY UPDATE enabled = 0`)
	configcache.Changed(context.Background(), "app_modules")
	_, body, _ = e.call(fiber.MethodGet, "/api/v1/messages/feed?slots=daily_advice,urgent_modal", feedDay1, nil)
	assert.Empty(t, slotItems(body, "daily_advice"))
	assert.Len(t, body["data"].(map[string]any)["slots"].(map[string]any), 2, "only the asked slots")
	delete(feed.SlotModules, "daily_advice")
	_, body, _ = e.call(fiber.MethodGet, "/api/v1/messages/feed?slots=daily_advice", feedDay1, nil)
	require.Len(t, slotItems(body, "daily_advice"), 1, "the module gate was the only stop")
	assert.EqualValues(t, advice[3], slotItems(body, "daily_advice")[0]["id"])

	// The next local day adds one for the safety message (no cap).
	_, body, _ = e.call(fiber.MethodGet, "/api/v1/messages/feed", feedDay2, nil)
	require.Len(t, slotItems(body, "urgent_modal"), 1)
	e.eventually(shownOf, 2, e.uid, urgent)
	assert.Empty(t, slotItems(body, "smart_tip"), "smart_tip: once per cycle")

	st, body, _ = e.call(fiber.MethodGet, "/api/v1/messages/feed?slots=push", feedDay2, nil)
	assert.Equal(t, 422, st, body)
}

func TestFeedInt_EventsAPI(t *testing.T) {
	configcache.SetEnabled(true)
	defer configcache.SetEnabled(false)
	e := newFeedEnv(t, "09120006102", civildate.MustParse("2026-10-05"))
	tip := e.rule(rules.KindClinical, rules.LevelInfo, "smart_tip", 500, "NULL")
	const ev = "/api/v1/messages/events"

	// A client impression after the server logged shown today is ignored; on another day it counts. (First: the
	// dismiss below takes the rule out of the feed.)
	_, _, _ = e.call(fiber.MethodGet, "/api/v1/messages/feed?slots=smart_tip", feedDay1, nil)
	shownOf := `SELECT COUNT(*) FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'impression'`
	e.eventually(shownOf, 1, e.uid, tip)
	st, body, _ := e.call(fiber.MethodPost, ev, feedDay1, map[string]any{"rule_id": tip, "event": "impression", "slot": "smart_tip"})
	require.Equal(t, 201, st)
	assert.Equal(t, false, body["data"].(map[string]any)["recorded"])
	st, body, _ = e.call(fiber.MethodPost, ev, feedDay2, map[string]any{"rule_id": tip, "event": "shown"})
	require.Equal(t, 201, st)
	assert.Equal(t, true, body["data"].(map[string]any)["recorded"])
	e.eventually(shownOf, 2, e.uid, tip)

	st, body, _ = e.call(fiber.MethodPost, ev, feedDay1, map[string]any{"rule_id": tip, "event": "clicked", "version": 2, "slot": "smart_tip"})
	require.Equal(t, 201, st, body)
	assert.Equal(t, true, body["data"].(map[string]any)["recorded"])
	st, _, _ = e.call(fiber.MethodPost, ev, feedDay1, map[string]any{"rule_id": tip, "event": "dismiss"})
	require.Equal(t, 201, st)
	e.eventually(`SELECT COUNT(*) FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'click' AND version = 2 AND slot = 'smart_tip'`, 1, e.uid, tip)
	e.eventually(`SELECT COUNT(*) FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'dismiss' AND version = 2 AND slot = 'smart_tip'`, 1, e.uid, tip)

	for _, bad := range []map[string]any{
		{"rule_id": tip, "event": "clicked", "version": 3},
		{"rule_id": tip, "event": "clicked", "version": "x"},
		{"rule_id": tip, "event": "clicked", "version": 0},
	} {
		st, body, _ = e.call(fiber.MethodPost, ev, feedDay1, bad)
		assert.Equal(t, 422, st, bad)
		assert.Contains(t, body["errors"], "version")
	}
	st, body, _ = e.call(fiber.MethodPost, ev, feedDay1, map[string]any{"rule_id": tip, "event": "clicked", "slot": "modal"})
	assert.Equal(t, 422, st)
	assert.Contains(t, body["errors"], "slot", "a slot the rule does not run in")
	st, body, _ = e.call(fiber.MethodPost, ev, feedDay1, map[string]any{"rule_id": tip, "event": "clicked", "variant": "z"})
	assert.Equal(t, 422, st)
	assert.Contains(t, body["errors"], "variant")

	// Rate limit: 120 / minute / user.
	limited := false
	for range 125 {
		if st, _, _ = e.call(fiber.MethodPost, ev, feedDay1, map[string]any{"rule_id": tip, "event": "click"}); st == 429 {
			limited = true
			break
		}
	}
	assert.True(t, limited)
}

// M-14: each today_ui.banner value of the engine has exactly one approved status_banner rule, with T-M9-42's copy.
func TestFeedInt_BannerSeed(t *testing.T) {
	db := testdb.New(t)
	def, ok := facts.Lookup("today_ui.banner")
	require.True(t, ok)
	read := func(lang string) map[string]any {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "frontend", "messages", lang, "home.json"))
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m))
		return m["nb"].(map[string]any)
	}
	fa, en := read("fa"), read("en")
	copyOf := func(nb map[string]any, banner string) rules.Variant {
		get := func(path ...string) string {
			var cur any = nb
			for _, p := range path {
				cur = cur.(map[string]any)[p]
			}
			return cur.(string)
		}
		switch banner {
		case "end_missing":
			return rules.Variant{Title: get("endMissing", "title"), Body: get("endMissing", "text"), CTALabel: get("endMissing", "cta")}
		case "overdue":
			return rules.Variant{Title: strings.ReplaceAll(get("banner", "overdue", "title"), "{n}", "{days_late}"),
				Body: get("banner", "overdue", "text"), CTALabel: get("banner", "overdue", "test")}
		}
		return rules.Variant{Title: get("banner", "askLast", "title"), Body: get("banner", "askLast", "text"), CTALabel: get("banner", "askLast", "cta")}
	}
	values := slices.DeleteFunc(slices.Clone(def.Enum), func(v string) bool { return v == facts.BannerNone })
	require.Len(t, values, 3)
	for _, v := range values {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM message_rules WHERE placement = 'status_banner' AND status = 'approved'
			AND is_active = 1 AND kind = 'engine' AND JSON_CONTAINS(conditions, JSON_OBJECT('fact', 'today_ui.banner', 'op', 'eq', 'value', ?), '$[0].items')`, v).Scan(&n))
		assert.Equal(t, 1, n, v)
		var content string
		require.NoError(t, db.QueryRow(`SELECT content FROM message_rules WHERE rule_key = ?`, "banner."+v).Scan(&content))
		var c struct {
			A            rules.Variant `json:"a"`
			Translations map[string]struct {
				A rules.Variant `json:"a"`
			} `json:"translations"`
		}
		require.NoError(t, json.Unmarshal([]byte(content), &c))
		want := copyOf(fa, v)
		assert.Equal(t, []string{want.Title, want.Body, want.CTALabel}, []string{c.A.Title, c.A.Body, c.A.CTALabel}, v)
		want = copyOf(en, v)
		got := c.Translations["en"].A
		assert.Equal(t, []string{want.Title, want.Body, want.CTALabel}, []string{got.Title, got.Body, got.CTALabel}, v)
		assert.True(t, strings.HasPrefix(c.A.CTATarget, "ritme://"), v)
		assert.Equal(t, c.A.CTATarget, got.CTATarget, v)
	}
}

// Budget (T-M10-06): the feed with a warm engine state and 200 active rules, ≤ 20 ms p95 and ≤ 3 queries plus auth.
func TestFeedInt_Budget(t *testing.T) {
	configcache.SetEnabled(true)
	defer configcache.SetEnabled(false)
	e := newFeedEnv(t, "09120006103", civildate.MustParse("2026-10-05"))
	slotKind := [][3]string{{"urgent_modal", rules.KindSafety, rules.LevelFollowSoon}, {"smart_tip", rules.KindClinical, rules.LevelInfo},
		{"daily_advice", rules.KindContent, rules.LevelInfo}, {"daily_challenge", rules.KindProduct, rules.LevelSuggestion},
		{"modal", rules.KindMarketing, rules.LevelInfo}}
	for i := range 197 { // + the 3 seeded banner rules = 200
		sk := slotKind[i%len(slotKind)]
		lo, hi := rules.Band(sk[1])
		prio := lo + i%(hi-lo+1)
		// conditions on engine facts, so the prefetch has work to do
		e.ruleWhen(sk[1], sk[2], sk[0], prio, "NULL",
			fmt.Sprintf(`[{"op":"and","items":[{"fact":"cycle_day","op":"gte","value":%d}]}]`, i%30))
	}
	configcache.Changed(context.Background(), rules.Table)
	require.Equal(t, 200, e.count(`SELECT COUNT(*) FROM message_rules WHERE is_active = 1 AND (rule_key IS NULL OR rule_key NOT LIKE 'preg.%')`)) // the 00113 preg.* inbox rules are not feed rules
	for range 3 {                                                                                                                                 // warm: rules compiled, engine state cached, marks set
		st, body, _ := e.call(fiber.MethodGet, "/api/v1/messages/feed", feedDay1, nil)
		require.Equal(t, 200, st, body)
	}
	const n = 200
	durs := make([]time.Duration, 0, n)
	var queries []int
	for range n {
		start := time.Now()
		st, _, h := e.call(fiber.MethodGet, "/api/v1/messages/feed", feedDay1, nil)
		durs = append(durs, time.Since(start))
		require.Equal(t, 200, st)
		q, _ := strconv.Atoi(h.Get("X-Test-Queries"))
		queries = append(queries, q)
	}
	slices.Sort(durs)
	p50, p95 := durs[n/2], durs[n*95/100]
	t.Logf("feed warm, 200 rules: p50 %s, p95 %s, queries %d (auth included)", p50, p95, slices.Max(queries))
	assert.LessOrEqual(t, p95, 20*time.Millisecond, "p95 over budget")
	assert.LessOrEqual(t, slices.Max(queries), 3+1, "≤ 3 queries plus auth")
}
