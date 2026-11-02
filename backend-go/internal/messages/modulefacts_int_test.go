package messages_test

// T-M10-09b: the feed and the jobs' facts (BuildFacts, the inbox sweep and push slot) read the meno.* and cond.*
// facts through StoreSource — the stage tip reaches the feed and the inbox rows open without the menopause or nudges
// GET being called.

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/configcache"
)

// installModuleReads installs the module reads on e's database (what the route mount does) until the test ends.
func installModuleReads(e *feedEnv) {
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cat := catalog.NewReader(catalogstore.New(e.db), nil, 0, quiet)
	messages.InstallModuleReads(messages.NewModuleReads(e.db, menopause.NewService(e.db, cat)))
	e.t.Cleanup(func() { messages.InstallModuleReads(nil) })
}

func (e *feedEnv) ruleID(key string) uint64 {
	e.t.Helper()
	var id uint64
	require.NoError(e.t, e.db.QueryRow(`SELECT id FROM message_rules WHERE rule_key = ?`, key).Scan(&id))
	return id
}

func (e *feedEnv) inboxStates(key string) []string {
	e.t.Helper()
	rows, err := e.db.Query(`SELECT state FROM message_inbox WHERE user_id = ? AND rule_key = ? ORDER BY id`, e.uid, key)
	require.NoError(e.t, err)
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var s string
		require.NoError(e.t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(e.t, rows.Err())
	return out
}

func (e *feedEnv) jobFacts(day civildate.Date) map[string]any {
	e.t.Helper()
	f, err := messages.BuildFacts(context.Background(), messages.NewHandlers(e.db, clock.Real{}).WithEngine(engineSource(e.t, e.db)), e.uid, day)
	require.NoError(e.t, err)
	return f
}

func TestModuleFactsInt_MenopauseTipAndAlertInTheFeed(t *testing.T) {
	configcache.SetEnabled(true)
	defer configcache.SetEnabled(false)
	e := newFeedEnv(t, "09120006191", civildate.MustParse("2026-10-05"))
	e.exec(`INSERT INTO user_life_profiles (user_id, life_mode, gender, menopause_stage, menopause_last_period, created_at, updated_at)
		VALUES (?, 'menopause', 'female', 'meno', '2025-01-01', NOW(), NOW())`, e.uid)
	e.exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, source, created_at, updated_at)
		VALUES (?, '2026-10-15', 'bleeding', 'presence', '', 'spotting', 'manual', NOW(), NOW())`, e.uid)
	day := civildate.MustParse("2026-10-21")

	// Without the module reads (before T-M10-09b) the meno.* facts are absent. The routes install them
	// globally on Mount, so uninstall first.
	messages.InstallModuleReads(nil)
	assert.NotContains(t, e.jobFacts(day), "meno.stage")

	installModuleReads(e)
	f := e.jobFacts(day)
	assert.Equal(t, "meno", f["meno.stage"], "the sweep / push facts")
	assert.Equal(t, true, f["meno.postmenopausal_bleeding"])
	assert.Equal(t, "2026-10-15", f["meno_bleeding_on"])
	assert.InDelta(t, 0.0, f["cond.heavy_bleeding_days"], 0, "menopause is not a nudged mode")

	// The feed alone (GET /messages/menopause never called): the stage tip in daily_advice, the bleeding alert as an
	// open inbox row counted in the badge. (The seeded general advice outranks a stage tip (priority 150): it is
	// switched off so the slot shows what the meno.* facts select.)
	e.exec(`UPDATE message_rules SET is_active = 0 WHERE placement = 'daily_advice' AND (rule_key IS NULL OR rule_key NOT LIKE 'meno.%')`)
	configcache.Changed(context.Background(), rules.Table)
	st, body, _ := e.call(fiber.MethodGet, "/api/v1/messages/feed", feedDay1, nil)
	require.Equal(t, 200, st, body)
	tip := e.ruleID("meno.tip_stage_meno")
	var ids []uint64
	for _, it := range slotItems(body, "daily_advice") {
		ids = append(ids, uint64(it["id"].(float64)))
	}
	assert.Contains(t, ids, tip, "the meno stage tip is served by the feed")
	for _, other := range []string{"meno.tip_stage_peri", "meno.tip_stage_post", "meno.tip_stage_unsure"} {
		assert.NotContains(t, ids, e.ruleID(other), other)
	}
	assert.Equal(t, []string{"active"}, e.inboxStates("meno.postmenopausal_bleeding"), "opened by the feed")
	assert.GreaterOrEqual(t, body["data"].(map[string]any)["alerts"].(map[string]any)["open"], float64(1))

	// Leaving menopause mode: the «nothing» input (no menopause read) resolves the row on the next evaluation.
	e.exec(`UPDATE user_life_profiles SET life_mode = 'cycle' WHERE user_id = ?`, e.uid)
	assert.NotContains(t, e.jobFacts(day), "meno.stage")
	st, body, _ = e.call(fiber.MethodGet, "/api/v1/messages/feed?slots=urgent_modal", feedDay1, nil)
	require.Equal(t, 200, st, body)
	assert.Equal(t, []string{"resolved"}, e.inboxStates("meno.postmenopausal_bleeding"))
}

func TestModuleFactsInt_ConditionNudgeFromTheFeed(t *testing.T) {
	configcache.SetEnabled(true)
	defer configcache.SetEnabled(false)
	e := newFeedEnv(t, "09120006192", civildate.MustParse("2026-10-05"))
	for _, d := range []string{"2026-10-06", "2026-10-07"} { // inside the current cycle (10-05 .. 10-21)
		e.exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, source, created_at, updated_at)
			VALUES (?, ?, 'bleeding', 'flow', '', 'heavy', 'legacy', NOW(), NOW())`, e.uid, d)
	}
	installModuleReads(e)
	day := civildate.MustParse("2026-10-21")
	f := e.jobFacts(day)
	assert.InDelta(t, 2.0, f["cond.heavy_bleeding_days"], 0)
	assert.InDelta(t, 0.0, f["cond.heavy_pain_days"], 0)
	assert.Equal(t, []string{}, f["cond.programs"])
	assert.NotContains(t, f, "meno.stage", "not in menopause mode: no stage")
	assert.Equal(t, false, f["meno.postmenopausal_bleeding"], "the «nothing» input")

	// The feed alone (GET /messages/nudges never called) opens the heavy flow row.
	st, body, _ := e.call(fiber.MethodGet, "/api/v1/messages/feed?slots=urgent_modal", feedDay1, nil)
	require.Equal(t, 200, st, body)
	assert.Equal(t, []string{"active"}, e.inboxStates("cond.heavy_bleeding"))

	// Enrolled in the program: the fact carries it and the row resolves.
	e.exec(`INSERT INTO condition_enrolments (user_id, program, enrolled_on, created_at, updated_at)
		VALUES (?, 'heavy_bleeding', '2026-10-10', NOW(), NOW())`, e.uid)
	assert.Equal(t, []string{"heavy_bleeding"}, e.jobFacts(day)["cond.programs"])
	st, body, _ = e.call(fiber.MethodGet, "/api/v1/messages/feed?slots=urgent_modal", feedDay1, nil)
	require.Equal(t, 200, st, body)
	assert.Equal(t, []string{"resolved"}, e.inboxStates("cond.heavy_bleeding"))
}
