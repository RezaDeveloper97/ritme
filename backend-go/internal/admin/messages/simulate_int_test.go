package messages_test

// T-M10-11: POST /messages/simulate (v6 simulator) and GET /messages/rules/conflicts over MariaDB.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/rules"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// simUser seeds a cycle user whose last period started on last (six regular 28-day cycles).
func simUser(t *testing.T, e *admintest.Env, mobile, goal string, last civildate.Date) uint64 {
	t.Helper()
	res := e.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Test', ?, NOW(), NOW())`, mobile)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	e.Exec(`INSERT INTO user_profiles (user_id, birthday, last_period_start, cycle_duration, period_duration, user_goal, created_at, updated_at)
		VALUES (?, '1998-01-01', ?, 28, 5, ?, NOW(), NOW())`, id, last.AddDays(-140).String(), goal)
	for i := 5; i >= 0; i-- {
		s := last.AddDays(-28 * i)
		e.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed,
			is_estimated, source, created_at, updated_at) VALUES (?, ?, ?, 5, 1, 0, 'user_logged', NOW(), NOW())`,
			id, s.String(), s.AddDays(4).String())
	}
	return uint64(id) //nolint:gosec // test id
}

// simRule inserts an approved, active rule; extraCols / extraVals add v6 columns (caps, …).
func simRule(t *testing.T, e *admintest.Env, kind, placement string, priority int, body, conds string) uint64 {
	t.Helper()
	return simRuleCaps(t, e, kind, placement, priority, body, conds, "NULL")
}

func simRuleCaps(t *testing.T, e *admintest.Env, kind, placement string, priority int, body, conds, caps string) uint64 {
	t.Helper()
	content := fmt.Sprintf(`{"a":{"title":"%s %d","body":%q}}`, placement, priority, body)
	res := e.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, variants_enabled, ab_split,
		conditions, exceptions, cap_count, cap_period, cooldown_days, audience_percent, status, is_active, version,
		approved_version, caps, created_at, updated_at)
		VALUES (?, 'info', ?, ?, ?, ?, 0, 50, ?, '[]', NULL, NULL, 0, 100, 'approved', 1, 1, 1, `+caps+`, NOW(), NOW())`,
		fmt.Sprintf("%s %d", placement, priority), kind, priority, placement, content, conds)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // test id
}

func between(fact string, lo, hi int) string {
	return fmt.Sprintf(`[{"op":"and","items":[{"fact":%q,"op":"between","value":[%d,%d]}]}]`, fact, lo, hi)
}

// grant is an approved 30-minute health-access request (timestamps are Tehran wall clock, like the app writes them).
func grant(e *admintest.Env, adminID, userID uint64) {
	expires := time.Now().In(civildate.Tehran).Add(30 * time.Minute).Format(time.DateTime)
	e.Exec(`INSERT INTO admin_health_access_requests (admin_id, user_id, reason, ticket, status, decided_by, decided_at,
		expires_at, created_at, updated_at) VALUES (?, ?, 'message complaint', '#7', 'approved', 1, NOW(), ?, NOW(), NOW())`,
		adminID, userID, expires)
}

type pick struct {
	id      uint64
	variant string
}

func simWinners(t *testing.T, r admintest.Resp) map[string][]pick {
	t.Helper()
	out := map[string][]pick{}
	for _, s := range r.Data()["slots"].([]any) {
		sl := s.(map[string]any)
		ps := []pick{}
		for _, w := range sl["winners"].([]any) {
			wm := w.(map[string]any)
			ps = append(ps, pick{uint64(wm["rule_id"].(float64)), wm["variant"].(string)})
		}
		out[sl["slot"].(string)] = ps
	}
	return out
}

func simDropped(r admintest.Resp, slot string) map[uint64]string {
	out := map[uint64]string{}
	for _, s := range r.Data()["slots"].([]any) {
		sl := s.(map[string]any)
		if sl["slot"] != slot {
			continue
		}
		for _, d := range sl["dropped"].([]any) {
			dm := d.(map[string]any)
			out[uint64(dm["rule_id"].(float64))] = dm["reason"].(string)
		}
	}
	return out
}

// The golden: the simulator's winners per slot are the feed's for the same user and clock, over 5 personas.
func TestSimulateV6_MatchesFeedOverPersonas(t *testing.T) {
	e := rulesEnv(t)
	today := civildate.MustParse("2026-10-21")
	now := time.Date(2026, 10, 21, 10, 0, 0, 0, civildate.Tehran)

	simRule(t, e, rules.KindEngine, "status_banner", 900, "early", between("cycle_day", 1, 5))
	simRule(t, e, rules.KindEngine, "status_banner", 850, "late", between("cycle_day", 20, 35))
	simRule(t, e, rules.KindClinical, "smart_tip", 400, "week {pregnancy_week}", "[]") // never renders: missing_variable
	simRule(t, e, rules.KindClinical, "smart_tip", 300, "ttc tip",
		`[{"op":"and","items":[{"fact":"user_goal","op":"eq","value":"ttc"}]}]`)
	simRule(t, e, rules.KindClinical, "smart_tip", 200, "general", "[]")
	simRule(t, e, rules.KindContent, "daily_advice", 500, "first half", between("cycle_day", 1, 13))
	simRule(t, e, rules.KindContent, "daily_advice", 400, "second half", between("cycle_day", 14, 40))
	simRule(t, e, rules.KindContent, "daily_advice", 300, "{days_to_period} days left", "[]")
	simRule(t, e, rules.KindContent, "daily_advice", 200, "any", "[]")
	simRule(t, e, rules.KindContent, "daily_advice", 100, "any 2", "[]")
	simRuleCaps(t, e, rules.KindProduct, "daily_challenge", 50, "challenge", "[]", `'[{"count":1,"period":"day"}]'`)
	simRule(t, e, rules.KindMarketing, "modal", 40, "plus", `[{"op":"and","items":[{"fact":"is_premium","op":"eq","value":false}]}]`)

	personas := []struct {
		mobile, goal string
		day          int // cycle day on today
	}{{"09120000001", "non_ttc", 3}, {"09120000002", "non_ttc", 10}, {"09120000003", "ttc", 14},
		{"09120000004", "non_ttc", 25}, {"09120000005", "ttc", 8}}
	q := mstore.New(e.DB)
	svc := feed.New(q, nil, nil, nil).WithLoader(func(ctx context.Context) (*compiled.Compiled, error) {
		rows, err := rules.ActiveRows(ctx, q, "")
		if err != nil {
			return nil, err
		}
		return compiled.Compile(rows)
	})
	user := messages.NewHandlers(e.DB, clock.Real{})
	root := e.As(admintest.SuperID)
	distinct := map[string]bool{}
	for i, p := range personas {
		uid := simUser(t, e, p.mobile, p.goal, today.AddDays(-(p.day - 1)))
		if i == 1 { // one persona has a challenge shown today: the cap holds it back in both paths
			e.Exec(`INSERT INTO message_events (rule_id, user_id, variant, event, slot, created_at)
				SELECT id, ?, 'a', 'impression', 'daily_challenge', '2026-10-21 09:00:00' FROM message_rules WHERE placement = 'daily_challenge'`, uid)
		}
		grant(e, admintest.SuperID, uid)
		r := root.JSON(fiber.MethodPost, "/messages/simulate", map[string]any{"user_id": uid, "date": "2026-10-21", "time": "10:00"})
		require.Equal(t, 200, r.Status, r.Body)
		got := simWinners(t, r)

		res, err := svc.Build(context.Background(), feed.Request{UserID: uid, Slots: feed.DefaultSlots, Now: now,
			Facts: func(ctx context.Context, _ []string) (rules.Facts, error) {
				return messages.BuildFacts(ctx, user, uid, today)
			}})
		require.NoError(t, err)
		want := map[string][]pick{}
		for _, s := range res.Slots {
			ps := []pick{}
			for _, ch := range s.Choices {
				ps = append(ps, pick{ch.Rule.ID, ch.Variant})
			}
			want[s.Name] = ps
		}
		assert.Equal(t, want, got, "persona %d (%s, day %d)", i, p.goal, p.day)
		distinct[fmt.Sprint(got)] = true
	}
	assert.GreaterOrEqual(t, len(distinct), 3, "the personas exercise different winners")
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM message_events WHERE event <> 'impression'"), "the simulator writes no events")
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM message_events"))
}

func TestSimulateV6_GrantAuditCapAndSynthetic(t *testing.T) {
	e := rulesEnv(t)
	uid := simUser(t, e, "09120000009", "non_ttc", civildate.FromTime(time.Now().In(civildate.Tehran)).AddDays(-4))
	ch := simRuleCaps(t, e, rules.KindProduct, "daily_challenge", 50, "challenge", "[]", `'[{"count":1,"period":"day"}]'`)
	tip := simRule(t, e, rules.KindClinical, "smart_tip", 200, "hello", "[]")
	root, mk := e.As(admintest.SuperID), e.As(marketingID)
	body := map[string]any{"user_id": uid, "slots": []string{"daily_challenge", "smart_tip"}}

	// No grant → 403, nothing audited, no facts built.
	r := root.JSON(fiber.MethodPost, "/messages/simulate", body)
	assert.Equal(t, 403, r.Status, r.Body)
	assert.Equal(t, "health_access_required", r.Code())
	// A grant row is not enough without the health capability.
	grant(e, marketingID, uid)
	r = mk.JSON(fiber.MethodPost, "/messages/simulate", body)
	assert.Equal(t, 403, r.Status)
	assert.Equal(t, "health_access_required", r.Code())
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE target_type = 'user' AND target_id = ?", uid))
	// Unknown user with a grant row is still a 403 without that user's grant; with one → 404.
	assert.Equal(t, 403, root.JSON(fiber.MethodPost, "/messages/simulate", map[string]any{"user_id": 999999}).Status)

	// With a grant → 200 and the audit rows (chained health read first).
	grant(e, admintest.SuperID, uid)
	r = root.JSON(fiber.MethodPost, "/messages/simulate", body)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'health_access.read' AND target_id = ? AND admin_id = 1", uid))
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'messages.simulate' AND target_id = ?", uid))
	d := r.Data()
	assert.Equal(t, "user", d["source"])
	assert.Equal(t, "real", d["history"])
	assert.Equal(t, "current_inputs", d["resolution"])
	assert.Equal(t, "Asia/Tehran", d["zone"])
	assert.Contains(t, d["engine"].(map[string]any), "cycle_day")
	assert.Equal(t, []pick{{ch, "a"}}, simWinners(t, r)["daily_challenge"])
	assert.Equal(t, []pick{{tip, "a"}}, simWinners(t, r)["smart_tip"])

	// The seeded history fills the daily cap → cap_day.
	e.Exec(`INSERT INTO message_events (rule_id, user_id, variant, event, slot, created_at) VALUES (?, ?, 'a', 'impression', 'daily_challenge', NOW())`, ch, uid)
	r = root.JSON(fiber.MethodPost, "/messages/simulate", body)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Empty(t, simWinners(t, r)["daily_challenge"])
	assert.Equal(t, map[uint64]string{ch: rules.ReasonCapDay}, simDropped(r, "daily_challenge"))
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM message_events"), "no event written by the simulator")

	// Synthetic: no user, empty history (the cap never triggers), override facts only.
	r = mk.JSON(fiber.MethodPost, "/messages/simulate", map[string]any{"slots": []string{"daily_challenge"},
		"facts_override": map[string]any{"cycle_day": 3, "confidence": "high"}})
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, "synthetic", r.Data()["source"])
	assert.Equal(t, "empty", r.Data()["history"])
	assert.Nil(t, r.Data()["user_id"])
	assert.Equal(t, []any{"confidence", "cycle_day"}, r.Data()["facts_overridden"])
	assert.Equal(t, 3.0, r.Data()["engine"].(map[string]any)["cycle_day"])
	assert.Equal(t, []pick{{ch, "a"}}, simWinners(t, r)["daily_challenge"])

	// Validation.
	assert.Equal(t, 422, mk.JSON(fiber.MethodPost, "/messages/simulate", map[string]any{"slots": []string{"nope"}}).Status)
	assert.Equal(t, 422, mk.JSON(fiber.MethodPost, "/messages/simulate", map[string]any{"date": "21-10-2026"}).Status)
	assert.Equal(t, 422, mk.JSON(fiber.MethodPost, "/messages/simulate", map[string]any{"time": "25:00"}).Status)
	assert.Equal(t, 401, e.Anonymous().JSON(fiber.MethodPost, "/messages/simulate", map[string]any{}).Status)
}

func TestSimulateV6_UnknownUserWithGrant(t *testing.T) {
	e := rulesEnv(t)
	// goose 00129 made admin_health_access_requests.user_id a cascading FK, so a grant for a user that does not exist
	// can no longer be stored normally; a dedicated connection with FK checks off still reproduces that orphan row.
	conn, err := e.DB.Conn(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = conn.ExecContext(t.Context(), "SET FOREIGN_KEY_CHECKS=0")
	require.NoError(t, err)
	expires := time.Now().In(civildate.Tehran).Add(30 * time.Minute).Format(time.DateTime)
	_, err = conn.ExecContext(t.Context(), `INSERT INTO admin_health_access_requests (admin_id, user_id, reason, ticket, status, decided_by, decided_at,
		expires_at, created_at, updated_at) VALUES (?, 999999, 'message complaint', '#7', 'approved', 1, NOW(), ?, NOW(), NOW())`,
		admintest.SuperID, expires)
	require.NoError(t, err)
	_, err = conn.ExecContext(t.Context(), "SET FOREIGN_KEY_CHECKS=1")
	require.NoError(t, err)
	r := e.As(admintest.SuperID).JSON(fiber.MethodPost, "/messages/simulate", map[string]any{"user_id": 999999})
	assert.Equal(t, 404, r.Status, r.Body)
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE target_id = 999999"))
}

func TestConflictPairsV2_API(t *testing.T) {
	e := rulesEnv(t)
	a := simRule(t, e, rules.KindClinical, "smart_tip", 300, "a", between("cycle_day", 1, 7))
	b := simRule(t, e, rules.KindClinical, "smart_tip", 200, "b", between("cycle_day", 5, 10))
	simRule(t, e, rules.KindClinical, "smart_tip", 100, "c", between("cycle_day", 20, 25))
	safety := simRule(t, e, rules.KindSafety, "push", 1000, "s", between("days_late", 3, 30))
	product := simRuleCaps(t, e, rules.KindProduct, "push", 50, "p", between("days_late", 0, 10), `'[{"count":1,"period":"day"}]'`)

	r := e.As(marketingID).Get("/messages/rules/conflicts")
	require.Equal(t, 200, r.Status, r.Body)
	cs := r.Data()["conflicts"].([]any)
	require.Len(t, cs, 2, "1–7 × 5–10 and safety × product; 20–25 overlaps nothing")
	bySlot := map[string]map[string]any{}
	for _, c := range cs {
		bySlot[c.(map[string]any)["slot"].(string)] = c.(map[string]any)
	}
	tipC := bySlot["smart_tip"]
	assert.EqualValues(t, a, tipC["winner"])
	assert.EqualValues(t, b, tipC["b"].(map[string]any)["rule_id"])
	assert.Equal(t, "priority", tipC["decided_by"])
	f := tipC["overlap"].(map[string]any)["facts"].([]any)[0].(map[string]any)
	assert.Equal(t, "cycle_day", f["fact"])
	assert.EqualValues(t, 5, f["min"])
	assert.EqualValues(t, 7, f["max"])
	pushC := bySlot["push"]
	assert.EqualValues(t, safety, pushC["winner"])
	assert.Equal(t, "safety", pushC["decided_by"])
	assert.EqualValues(t, product, pushC["b"].(map[string]any)["rule_id"])
	assert.Equal(t, []any{[]any{8.0, 23.0}}, pushC["overlap"].(map[string]any)["hours"], "push quiet hours removed")

	r = e.As(marketingID).Get("/messages/rules/conflicts?slot=push")
	require.Equal(t, 200, r.Status)
	assert.EqualValues(t, 1, r.Data()["count"])
	assert.Equal(t, 422, e.As(marketingID).Get("/messages/rules/conflicts?slot=nope").Status)
}
