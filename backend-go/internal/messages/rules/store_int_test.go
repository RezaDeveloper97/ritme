package rules_test

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// countingDB counts the statements sent through it.
type countingDB struct {
	*sql.DB
	n atomic.Int64
}

func (d *countingDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	d.n.Add(1)
	return d.DB.ExecContext(ctx, q, args...)
}

func (d *countingDB) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	d.n.Add(1)
	return d.DB.QueryContext(ctx, q, args...)
}

func (d *countingDB) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	d.n.Add(1)
	return d.DB.QueryRowContext(ctx, q, args...)
}

func (d *countingDB) PrepareContext(ctx context.Context, q string) (*sql.Stmt, error) {
	return d.DB.PrepareContext(ctx, q)
}

// legacyHistory is the pre-T-M9-13 DBHistory: two probes per rule (no Prefetch).
type legacyHistory struct{ h rules.DBHistory }

func (l legacyHistory) Count(r uint64, e string, s time.Time) (int, error) { return l.h.Count(r, e, s) }
func (l legacyHistory) Last(r uint64, e string) (time.Time, bool, error)   { return l.h.Last(r, e) }

func insertRule(t *testing.T, db *sql.DB, prio, capCount int, period string, cooldown int) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, conditions, exceptions,
		cap_count, cap_period, cooldown_days, audience_percent, status, is_active, created_at, updated_at)
		VALUES ('r', 'info', 'product', ?, 'smart_tip', '{"a":{"title":"t","body":"b"}}', '[]', '[]', NULLIF(?, 0), NULLIF(?, ''), ?, 100,
		'approved', 1, '2026-10-01 09:00:00', '2026-10-01 09:00:00')`, prio, capCount, period, cooldown)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // test id
}

// ensureUsers inserts users rows with fixed ids: message_events.user_id is a foreign key since goose 00129 (T-M12-13).
func ensureUsers(t *testing.T, db *sql.DB, ids ...uint64) {
	t.Helper()
	for _, id := range ids {
		_, err := db.Exec(`INSERT INTO users (id, mobile, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`,
			id, fmt.Sprintf("0912%07d", id))
		require.NoError(t, err)
	}
}

// /messages/selected's history read is constant: 50 active rules, every one capped or cooling down, cost one query.
func TestSelect_FiftyRulesOneHistoryQuery(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, noon.Location())
	const uid = 42
	ensureUsers(t, db, uid)
	periods := []string{"day", "week", "cycle"}
	mine := map[uint64]bool{}
	var first uint64
	for i := range 50 {
		id := insertRule(t, db, 90-i, 1, periods[i%3], i%3)
		mine[id] = true
		if i == 0 {
			first = id
		}
		if i < 49 { // the last rule has no history → it is the pick
			_, err := db.Exec(`INSERT INTO message_events (rule_id, user_id, variant, event, created_at) VALUES (?, ?, 'a', 'impression', ?)`,
				id, uid, now.Add(-time.Duration(i%5+1)*time.Hour))
			require.NoError(t, err)
		}
	}
	_, err := db.Exec(`INSERT INTO message_events (rule_id, user_id, variant, event, created_at) VALUES (?, ?, 'a', 'dismiss', ?)`, first, uid, now.Add(-time.Hour))
	require.NoError(t, err)

	rows, err := store.New(db).ListActiveMessageRules(ctx, store.ListActiveMessageRulesParams{})
	require.NoError(t, err)
	// only this test's rules: the migrations seed the status banner rules (00112, T-M10-06)
	rows = slices.DeleteFunc(rows, func(r store.MessageRule) bool { return !mine[r.ID] })
	require.Len(t, rows, 50)
	all := make([]rules.Rule, 0, len(rows))
	for i := range rows {
		all = append(all, rules.FromRow(&rows[i]))
	}
	facts := rules.Facts{"cycle_length": 28.0}

	counter := &countingDB{DB: db}
	h := rules.DBHistory{Ctx: ctx, Q: store.New(counter), UserID: uid}
	got, err := rules.Select(all, facts, uid, now, h)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(1), counter.n.Load(), "one grouped history query for 50 rules")

	counter.n.Store(0)
	want, err := rules.Select(all, facts, uid, now, legacyHistory{rules.DBHistory{Ctx: ctx, Q: store.New(counter), UserID: uid}})
	require.NoError(t, err)
	assert.Equal(t, want.Rule.ID, got.Rule.ID, "same pick as the per-rule probes")
	assert.Greater(t, counter.n.Load(), int64(50), "the old path probed every rule")

	// The grouped row also carries the last dismissal.
	st, err := store.New(db).ListUserMessageEventStats(ctx, store.ListUserMessageEventStatsParams{
		DaySince: sql.NullTime{Time: now.Add(-24 * time.Hour), Valid: true}, WeekSince: sql.NullTime{Time: now, Valid: true},
		CycleSince: sql.NullTime{Time: now, Valid: true}, UserID: uid, RuleIds: []uint64{rows[len(rows)-1].ID, first},
	})
	require.NoError(t, err)
	byEvent := map[string]store.ListUserMessageEventStatsRow{}
	for _, r := range st {
		if r.RuleID == first {
			byEvent[r.Event] = r
		}
	}
	assert.Equal(t, int64(1), byEvent["impression"].DayCount)
	assert.Equal(t, int64(0), byEvent["impression"].WeekCount)
	assert.True(t, byEvent["dismiss"].LastAt.Equal(now.Add(-time.Hour)), byEvent["dismiss"].LastAt.String())
}

// ActiveRows: one query per process; an admin write that publishes message_rules is visible on the next read.
func TestActiveRows_SnapshotAndLocalInvalidation(t *testing.T) {
	configcache.SetEnabled(true)
	t.Cleanup(func() { configcache.SetEnabled(false) })
	db := testdb.New(t)
	configcache.Default.Invalidate(rules.Table) // drop what an earlier test may have loaded
	ctx := context.Background()
	counter := &countingDB{DB: db}
	q := store.New(counter)
	insertRule(t, db, 50, 0, "", 0)

	got, err := rules.ActiveRows(ctx, q, "smart_tip")
	require.NoError(t, err)
	require.Len(t, got, 1)
	_, _ = rules.ActiveRows(ctx, q, "")
	got, _ = rules.ActiveRows(ctx, q, "push")
	assert.Empty(t, got)
	assert.Equal(t, int64(1), counter.n.Load(), "the snapshot answers every placement")

	insertRule(t, db, 60, 0, "", 0) // admin write…
	got, _ = rules.ActiveRows(ctx, q, "smart_tip")
	assert.Len(t, got, 1, "not yet published")
	configcache.Changed(ctx, rules.Table) // …then the publish after its commit
	got, _ = rules.ActiveRows(ctx, q, "smart_tip")
	require.Len(t, got, 2)
	assert.Equal(t, int16(60), int16(got[0].Priority), "priority DESC kept") //nolint:gosec // small
}

// Selection v6 reads the whole history in one grouped query (T-M10-04): per rule shown / dismissed, per slot (a legacy
// event without a slot counts for its rule's placement), the last safety shown; the windows are user-local.
func TestLoadHistory_OneQuery(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	const uid = 43
	ensureUsers(t, db, uid, 999)                             // 999 = the other user whose event must not count
	berlinNow := time.Date(2026, 10, 21, 9, 0, 0, 0, berlin) // 11:30 Tehran
	tip := insertRule(t, db, 300, 0, "", 0)                  // smart_tip, product kind
	res, err := db.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, conditions, exceptions,
		cooldown_days, audience_percent, status, is_active, created_at, updated_at)
		VALUES ('s', 'follow_soon', 'safety', 1000, 'urgent_modal', '{"a":{"title":"t","body":"b"}}', '[]', '[]', 0, 100,
		'approved', 1, '2026-10-01 09:00:00', '2026-10-01 09:00:00')`)
	require.NoError(t, err)
	safety, _ := res.LastInsertId()
	ev := func(rule uint64, event, slot string, at time.Time) {
		var s any
		if slot != "" {
			s = slot
		}
		_, err := db.Exec(`INSERT INTO message_events (rule_id, user_id, variant, event, slot, created_at) VALUES (?, ?, 'a', ?, ?, ?)`,
			rule, uid, event, s, at)
		require.NoError(t, err)
	}
	ev(tip, "impression", "", berlinNow.Add(-time.Hour))                          // legacy: no slot → smart_tip; today
	ev(tip, "impression", "push", berlinNow.Add(-8*time.Hour))                    // 01:00 Berlin today, but yesterday in Tehran
	ev(tip, "dismiss", "push", berlinNow.Add(-30*time.Minute))                    //
	ev(tip, "impression", "smart_tip", berlinNow.Add(-10*24*time.Hour))           // outside the week
	ev(uint64(safety), "impression", "urgent_modal", berlinNow.Add(-3*time.Hour)) //nolint:gosec // test id
	ev(tip, "impression", "push", berlinNow.Add(-time.Minute))                    // another user's event must not count
	_, err = db.Exec(`UPDATE message_events SET user_id = 999 ORDER BY id DESC LIMIT 1`)
	require.NoError(t, err)

	counter := &countingDB{DB: db}
	w := rules.CapWindows(berlinNow, rules.Facts{"current_period_start": "2026-10-15"})
	h, err := rules.LoadHistory(ctx, store.New(counter), uid, w)
	require.NoError(t, err)
	assert.Equal(t, int64(1), counter.n.Load(), "one history query per request")

	r := h.Rules[tip]
	assert.Equal(t, rules.WindowCounts{Day: 2, Week: 2, Cycle: 2}, r.Shown)
	assert.Equal(t, rules.WindowCounts{Day: 1, Week: 1, Cycle: 1}, r.Dismissed)
	assert.True(t, r.LastShown.Equal(berlinNow.Add(-time.Hour)), r.LastShown.String())
	assert.True(t, r.LastDismissed.Equal(berlinNow.Add(-30*time.Minute)), r.LastDismissed.String())
	assert.Equal(t, rules.WindowCounts{Day: 1, Week: 1}, h.Slots["push"])
	assert.Equal(t, rules.WindowCounts{Day: 1, Week: 1}, h.Slots["smart_tip"])
	assert.Equal(t, rules.WindowCounts{Day: 1, Week: 1}, h.Slots["urgent_modal"])
	assert.True(t, h.LastSafetyShown.Equal(berlinNow.Add(-3*time.Hour)), h.LastSafetyShown.String())

	// The legacy prefetch still filters on its candidate ids and folds the slots of a rule together.
	st, err := rules.DBHistory{Ctx: ctx, Q: store.New(db), UserID: uid}.Prefetch([]uint64{tip}, rules.Windows(berlinNow, nil))
	require.NoError(t, err)
	last, ok, err := st.Last(tip, "impression")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, last.Equal(berlinNow.Add(-time.Hour)))
	n, err := st.Count(tip, "impression", rules.Windows(berlinNow, nil)[1])
	require.NoError(t, err)
	assert.Equal(t, 2, n, "the week count adds the rule's smart_tip and push rows")
}
