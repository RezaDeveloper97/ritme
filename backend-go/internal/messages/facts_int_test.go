package messages_test

// T-M10-03 (D-161): the fact registry's StoreSource over a real database — contraception (I-03), plan, pregnancy
// dating, postpartum week, vitals, log activity, the v6 facts of BuildFacts and the cold-build query budget.

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/facts"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// countingDB counts the statements the message stores send (the engine state loads through its own store).
type countingDB struct {
	db *sql.DB
	n  atomic.Int64
}

func (c *countingDB) ExecContext(ctx context.Context, q string, a ...any) (sql.Result, error) {
	c.n.Add(1)
	return c.db.ExecContext(ctx, q, a...)
}

func (c *countingDB) PrepareContext(ctx context.Context, q string) (*sql.Stmt, error) {
	c.n.Add(1)
	return c.db.PrepareContext(ctx, q)
}

func (c *countingDB) QueryContext(ctx context.Context, q string, a ...any) (*sql.Rows, error) {
	c.n.Add(1)
	return c.db.QueryContext(ctx, q, a...)
}

func (c *countingDB) QueryRowContext(ctx context.Context, q string, a ...any) *sql.Row {
	c.n.Add(1)
	return c.db.QueryRowContext(ctx, q, a...)
}

var factsToday = civildate.MustParse("2026-10-21")

func registryFacts(t *testing.T, db *sql.DB, uid uint64, names ...string) (facts.Values, int64) {
	t.Helper()
	cdb := &countingDB{db: db}
	src := messages.NewStoreSource(mstore.New(cdb), pstore.New(cdb), uid, factsToday).
		WithEngine(engineSource(t, db)).WithNow(factsToday.TehranMidnight().Add(10 * time.Hour))
	fs := facts.New(src.FactSources(), factsToday, "fa")
	ctx := context.Background()
	if names == nil {
		require.NoError(t, fs.All(ctx))
	} else {
		require.NoError(t, fs.Need(ctx, names...))
	}
	return fs.Values(), cdb.n.Load()
}

// I-03: a combined pill is hormonal, a copper IUD is not; without a method the onboarding pill answer counts.
func TestFactsInt_HormonalContraception(t *testing.T) {
	db := testdb.New(t)
	for _, tc := range []struct {
		mobile, method, meds string
		want                 bool
	}{
		{"09120007301", "combined_pill", "", true},
		{"09120007302", "copper_iud", `["contraceptive_pill"]`, false},
		{"09120007303", "", `["contraceptive_pill"]`, true},
		{"09120007304", "", `["iud"]`, false},
	} {
		uid := seedCycleUser(t, db, tc.mobile, "non_ttc", "", factsToday.AddDays(-12))
		if tc.method != "" {
			_, err := db.Exec(`INSERT INTO contraception_methods (user_id, method, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, uid, tc.method)
			require.NoError(t, err)
		}
		if tc.meds != "" {
			_, err := db.Exec(`INSERT INTO user_life_profiles (user_id, medications, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, uid, tc.meds)
			require.NoError(t, err)
		}
		v, _ := registryFacts(t, db, uid, "hormonal_contraception")
		assert.Equal(t, tc.want, v["hormonal_contraception"], tc.method+tc.meds)
	}
}

// Account, logs and vitals of one cycle user; the cold full build stays within 4 queries beyond the engine state.
func TestFactsInt_AccountLogsVitalsAndBudget(t *testing.T) {
	db := testdb.New(t)
	uid := seedCycleUser(t, db, "09120007311", "non_ttc", "", factsToday.AddDays(-1))
	exec := func(q string, a ...any) {
		_, err := db.Exec(q, a...)
		require.NoError(t, err)
	}
	exec(`UPDATE users SET created_at = ? WHERE id = ?`, factsToday.AddDays(-40).TehranMidnight().Add(9*time.Hour), uid)
	exec(`INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`,
		uid, factsToday.AddDays(-2).TehranMidnight(), factsToday.AddDays(5).TehranMidnight())
	exec(`INSERT INTO vital_readings (user_id, type, measured_at, systolic, diastolic, created_at, updated_at)
		VALUES (?, 'bp', ?, 118, 76, NOW(), NOW()), (?, 'bp', ?, 135, 85, NOW(), NOW())`,
		uid, factsToday.AddDays(-20).TehranMidnight().Add(9*time.Hour), uid, factsToday.AddDays(-2).TehranMidnight().Add(9*time.Hour))
	for _, d := range []civildate.Date{factsToday.AddDays(-3), factsToday.AddDays(-10), factsToday.AddDays(-45)} {
		exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, created_at, updated_at)
			VALUES (?, ?, 'pain', 'location', 'abdomen', 'severe', NOW(), NOW())`, uid, d.String())
	}

	v, n := registryFacts(t, db, uid)
	assert.Equal(t, "trial", v["plan"])
	assert.Equal(t, 40.0, v["days_since_signup"])
	assert.Equal(t, 28.0, v["age"])
	assert.Equal(t, "non_ttc", v["user_goal"])
	assert.Equal(t, "Sara", v["name"])
	assert.Equal(t, 3.0, v["days_since_last_log"])
	assert.Equal(t, 2.0, v["logs_30d"], "the day 45 days ago is outside the window")
	assert.Equal(t, "stage1", v["bp_last_category"], "the newest reading (135/85), not the older normal one")
	assert.NotContains(t, v, "glucose_last_category")
	assert.Equal(t, false, v["hormonal_contraception"])
	assert.Equal(t, "menstrual", v["main_phase"])
	assert.LessOrEqual(t, n, int64(4), "cold full build: ≤ 4 queries beyond the engine state")

	// Engine-only rules read no message-store query at all.
	_, n = registryFacts(t, db, uid, "main_phase", "days_late", "warnings")
	assert.Zero(t, n)
}

// Pregnant 30+2 (LMP dating, the pregnancy screens' function): week 31, third trimester.
func TestFactsInt_PregnancyDating(t *testing.T) {
	db := testdb.New(t)
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Mina', '09120007321', NOW(), NOW())`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	uid := uint64(id) //nolint:gosec // test id
	_, err = db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, age_source, lmp_date, onboarding_completed, created_at, updated_at)
		VALUES (?, 1, 0, 'lmp', ?, 1, NOW(), NOW())`, uid, factsToday.AddDays(-(30*7 + 2)).String())
	require.NoError(t, err)
	v, _ := registryFacts(t, db, uid, "pregnancy_week", "pregnancy_trimester", "mode")
	assert.Equal(t, 31.0, v["pregnancy_week"])
	assert.Equal(t, 3.0, v["pregnancy_trimester"])
	assert.Equal(t, "pregnancy", v["mode"])
}

// Postpartum week 5 (30 days after birth) only while the stored life mode is postpartum.
func TestFactsInt_PostpartumWeek(t *testing.T) {
	db := testdb.New(t)
	uid := seedCycleUser(t, db, "09120007331", "non_ttc", "postpartum", factsToday.AddDays(-300))
	_, err := db.Exec(`INSERT INTO postpartum_profiles (user_id, birth_date, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`,
		uid, factsToday.AddDays(-30).String())
	require.NoError(t, err)
	v, _ := registryFacts(t, db, uid, "postpartum_week", "life_mode")
	assert.Equal(t, 5.0, v["postpartum_week"])
	assert.Equal(t, "postpartum", v["life_mode"])

	_, err = db.Exec(`UPDATE user_life_profiles SET life_mode = 'cycle' WHERE user_id = ?`, uid)
	require.NoError(t, err)
	v, _ = registryFacts(t, db, uid, "postpartum_week")
	assert.NotContains(t, v, "postpartum_week")
}

// BuildFacts under v6 is the registry (all groups); v1 keeps the legacy builder.
func TestFactsInt_BuildFactsUsesRegistry(t *testing.T) {
	db := testdb.New(t)
	uid := seedCycleUser(t, db, "09120007341", "non_ttc", "", factsToday.AddDays(-36))
	f, err := messages.BuildFacts(context.Background(), messages.NewHandlers(db, clock.Real{}).WithEngine(engineSource(t, db)), uid, factsToday)
	require.NoError(t, err)
	assert.Equal(t, 8.0, f["days_late"])
	assert.Contains(t, f["warnings"], "predicted_period_overdue")
	assert.Equal(t, "overdue", f["today_ui.banner"])
	assert.Equal(t, "cycle", f["mode"])
	assert.Equal(t, "free", f["plan"])
	assert.Contains(t, f, "symptom_history.cramps")

	v1, err := messages.BuildFacts(context.Background(), messages.NewHandlers(db, clock.Real{}), uid, factsToday)
	require.NoError(t, err)
	assert.NotContains(t, v1, "warnings")
	assert.NotContains(t, v1, "plan")
}
