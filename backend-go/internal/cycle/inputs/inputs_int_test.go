package inputs_test

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/platform/racedet"
)

func TestMain(m *testing.M) { testdb.Main(m) }

var today = civildate.New(2026, 10, 21)

func day(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

func exec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	_, err := db.Exec(q, args...)
	require.NoError(t, err, q)
}

func newUser(t *testing.T, db *sql.DB, mobile string) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('T', ?, '2025-09-01 09:00:00', '2025-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // auto-increment id
}

func addPeriod(t *testing.T, db *sql.DB, uid uint64, start, end, source string, confirmed, estimated bool) {
	t.Helper()
	var e any
	if end != "" {
		e = end
	}
	exec(t, db, `INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, is_estimated, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`, uid, start, e, confirmed, estimated, source)
}

func addEntry(t *testing.T, db *sql.DB, uid uint64, date, param string, code, num any) {
	t.Helper()
	exec(t, db, `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, source, created_at, updated_at)
		VALUES (?, ?, 'measurements', ?, '', ?, ?, 'manual', NOW(), NOW())`, uid, date, param, code, num)
}

// seedFull is a user with every input: declared data with the new anchor column, manual lengths, stored postpartum
// mode with a breastfeeding delivery, a saved hormonal IUD (onboarding said iud), periods of every source, skips
// and every marker kind (with the rows the rules drop).
func seedFull(t *testing.T, db *sql.DB) uint64 {
	uid := newUser(t, db, "09120000001")
	exec(t, db, `INSERT INTO user_profiles (user_id, birthday, period_duration, cycle_duration, last_period_start, onboarding_last_period_start,
		user_goal, calculation_status, created_at, updated_at) VALUES (?, '1995-03-10', 5, 29, '2026-01-01', '2026-01-01', 'ttc', 'completed', NOW(), NOW())`, uid)
	exec(t, db, `INSERT INTO cycle_preferences (user_id, lengths_auto, created_at, updated_at) VALUES (?, 0, NOW(), NOW())`, uid)
	exec(t, db, `INSERT INTO user_life_profiles (user_id, gender, life_mode, medications, created_at, updated_at)
		VALUES (?, 'female', 'postpartum', '["iud"]', NOW(), NOW())`, uid)
	exec(t, db, `INSERT INTO postpartum_profiles (user_id, birth_date, baby_count, source, breastfeeding, created_at, updated_at)
		VALUES (?, '2026-03-01', 1, 'direct', 1, NOW(), NOW())`, uid)
	exec(t, db, `INSERT INTO contraception_methods (user_id, method, inserted_on, created_at, updated_at) VALUES (?, 'hormonal_iud', '2026-05-01', NOW(), NOW())`, uid)

	addPeriod(t, db, uid, "2026-01-01", "2026-01-05", "user_profile_confirmed", true, false) // the column wins: ignored
	addPeriod(t, db, uid, "2026-07-01", "2026-07-05", "user_logged", true, false)
	addPeriod(t, db, uid, "2026-07-30", "2026-08-03", "bleeding_log", true, false)
	addPeriod(t, db, uid, "2026-08-28", "2026-09-01", "user_logged", false, false)
	addPeriod(t, db, uid, "2026-09-25", "", "onboarding_estimate", false, true) // estimated: ignored
	addPeriod(t, db, uid, "2026-10-12", "", "user_logged", true, false)         // open

	exec(t, db, `INSERT INTO cycle_skips (user_id, expected_start, created_at, updated_at) VALUES (?, '2026-06-02', NOW(), NOW()), (?, '2026-05-05', NOW(), NOW())`, uid, uid)

	addEntry(t, db, uid, "2026-10-01", "lh_test", "positive", nil)
	exec(t, db, `INSERT INTO fertility_logs (user_id, log_date, lh_test, created_at, updated_at) VALUES
		(?, '2026-10-01', 'positive', NOW(), NOW()), (?, '2026-10-02', 'faint', NOW(), NOW()), (?, '2026-09-30', 'positive', NOW(), NOW()),
		(?, '2026-10-25', 'positive', NOW(), NOW())`, uid, uid, uid, uid) // the last one is after today + 1
	addEntry(t, db, uid, "2026-10-03", "lh_test", "negative", nil)
	addEntry(t, db, uid, "2026-10-04", "bbt", nil, "36.45")
	exec(t, db, `INSERT INTO daily_health_logs (user_id, log_date, basal_body_temperature, created_at, updated_at) VALUES
		(?, '2026-10-04', 36.40, NOW(), NOW()), (?, '2026-10-05', 36.80, NOW(), NOW()), (?, '2026-10-06', 33.10, NOW(), NOW()),
		(?, '2024-01-01', 36.50, NOW(), NOW())`, uid, uid, uid, uid) // 33.10 out of range; 2024 before the window
	addEntry(t, db, uid, "2026-10-19", "pregnancy_test", "negative", nil)
	addEntry(t, db, uid, "2026-10-20", "pregnancy_test", "positive", nil)
	return uid
}

func wantFull() engine.Input {
	return engine.Input{
		Periods: []engine.Period{
			{Start: day("2026-07-01"), End: day("2026-07-05"), Source: engine.PeriodUserLogged},
			{Start: day("2026-07-30"), End: day("2026-08-03"), Source: engine.PeriodBleedingLog},
			{Start: day("2026-08-28"), End: day("2026-09-01"), Source: engine.PeriodUserLogged},
			{Start: day("2026-10-12"), End: engine.NoDay, Source: engine.PeriodUserLogged},
		},
		Skips: []engine.Day{day("2026-05-05"), day("2026-06-02")},
		Declared: engine.Declared{
			LastPeriodStart: engine.Some(day("2026-01-01")), CycleLength: engine.Some(29), PeriodLength: engine.Some(5),
			Manual: true,
		},
		BirthDate:     engine.Some(day("1995-03-10")),
		Mode:          engine.ModePostpartum,
		Hormonal:      true,
		HormonalFrom:  engine.Some(day("2026-05-01")), // inserted_on, before created_at = NOW() (T-M9-48d)
		DeliveryDate:  engine.Some(day("2026-03-01")),
		Breastfeeding: true,
		LH: []engine.LHTest{
			{Date: day("2026-09-30"), Positive: true}, {Date: day("2026-10-01"), Positive: true},
		},
		BBT: []engine.BBTReading{
			{Date: day("2026-10-04"), Centi: 3645}, {Date: day("2026-10-05"), Centi: 3680},
		},
		PregnancyTests: []engine.PregnancyTest{{Date: day("2026-10-20"), Positive: true}},
	}
}

// seedTeen: no birthday, a teen age band, onboarding pill, the declared anchor in its column and a stray
// user_profile_confirmed row (T-M9-27: never a period, never the anchor).
func seedTeen(t *testing.T, db *sql.DB) uint64 {
	uid := newUser(t, db, "09120000002")
	exec(t, db, `INSERT INTO user_profiles (user_id, period_duration, onboarding_last_period_start, user_goal, calculation_status, created_at, updated_at)
		VALUES (?, 6, '2026-09-20', 'non_ttc', 'completed', NOW(), NOW())`, uid)
	exec(t, db, `INSERT INTO user_life_profiles (user_id, gender, life_mode, medications, created_at, updated_at)
		VALUES (?, 'female', 'teen', '["contraceptive_pill"]', NOW(), NOW())`, uid)
	exec(t, db, `INSERT INTO teen_profiles (user_id, age_band, menarche, created_at, updated_at) VALUES (?, '13_15', 'under_1y', NOW(), NOW())`, uid)
	addPeriod(t, db, uid, "2026-08-01", "2026-08-05", "user_profile_confirmed", true, false)
	return uid
}

// seedPregnant: an active pregnancy beats the stored postpartum mode.
func seedPregnant(t *testing.T, db *sql.DB) uint64 {
	uid := newUser(t, db, "09120000003")
	exec(t, db, `INSERT INTO user_profiles (user_id, birthday, user_goal, calculation_status, created_at, updated_at)
		VALUES (?, '1990-01-01', 'non_ttc', 'completed', NOW(), NOW())`, uid)
	exec(t, db, `INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, 'female', 'postpartum', NOW(), NOW())`, uid)
	exec(t, db, `INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-09-01', NOW(), NOW())`, uid)
	addPeriod(t, db, uid, "2026-09-01", "2026-09-05", "user_logged", true, false)
	addEntry(t, db, uid, "2026-10-05", "pregnancy_test", "positive", nil)
	return uid
}

func counted(t *testing.T, fn func(ctx context.Context)) int64 {
	t.Helper()
	ctx, c := metrics.WithQueryCounter(context.Background())
	fn(ctx)
	return c.Load()
}

func TestLoad_EveryInput(t *testing.T) {
	db := testdb.New(t)
	q := store.New(db)
	full, teen, pregnant := seedFull(t, db), seedTeen(t, db), seedPregnant(t, db)
	bare := newUser(t, db, "09120000004") // a users row and nothing else

	var got engine.Input
	n := counted(t, func(ctx context.Context) {
		var err error
		got, err = inputs.Load(ctx, q, full, today)
		require.NoError(t, err)
	})
	assert.Equal(t, wantFull(), got)
	assert.LessOrEqual(t, n, int64(4), "Load query budget")
	t.Logf("Load: %d queries", n)

	teenIn, err := inputs.Load(context.Background(), q, teen, today)
	require.NoError(t, err)
	assert.Equal(t, engine.Input{
		// No cycle_preferences row = manual: a history median is only a suggestion until she confirms it.
		Declared:     engine.Declared{LastPeriodStart: engine.Some(day("2026-09-20")), PeriodLength: engine.Some(6), Manual: true},
		TeenBandOnly: true, Mode: engine.ModeTeen, Hormonal: true,
	}, teenIn)

	pregIn, err := inputs.Load(context.Background(), q, pregnant, today)
	require.NoError(t, err)
	assert.Equal(t, engine.ModePregnancy, pregIn.Mode)
	assert.Equal(t, []engine.PregnancyTest{{Date: day("2026-10-05"), Positive: true}}, pregIn.PregnancyTests)

	bareIn, err := inputs.Load(context.Background(), q, bare, today) // no profile row: nothing declared, manual default
	require.NoError(t, err)
	assert.Equal(t, engine.Input{Declared: engine.Declared{Manual: true}}, bareIn)
	noneIn, err := inputs.Load(context.Background(), q, 999_999, today) // no user at all
	require.NoError(t, err)
	assert.Equal(t, engine.Input{}, noneIn)

	// LoadMany of three users = three Loads, in the same four queries.
	ids := []uint64{full, teen, pregnant}
	var many map[uint64]engine.Input
	n = counted(t, func(ctx context.Context) {
		many, err = inputs.LoadMany(ctx, q, ids, today)
		require.NoError(t, err)
	})
	assert.LessOrEqual(t, n, int64(4), "LoadMany query budget")
	require.Len(t, many, 3)
	for _, id := range ids {
		one, err := inputs.Load(context.Background(), q, id, today)
		require.NoError(t, err)
		assert.Equal(t, one, many[id], id)
	}

	// The snapshot round-trips what the loader produced.
	b, err := inputs.Encode(got)
	require.NoError(t, err)
	back, err := inputs.Decode(b)
	require.NoError(t, err)
	assert.Equal(t, got, back)
}

// TestLoad_Budget: 120 periods and 400 markers in ≤ 4 queries; p95 ≤ 5 ms on the local test DB.
// T-M9-27 (I-01, D-123): an onboarding-only user (declared LMP in its column, no cycle_histories row) resolves from
// the onboarding_declared anchor; 90 days on, the anchor is stale (§2.41: > effective cycle + 14) and rolled forward.
func TestLoad_OnboardingOnlyUserIsDeclaredThenStale(t *testing.T) {
	db := testdb.New(t)
	q := store.New(db)
	uid := newUser(t, db, "09120000009")
	exec(t, db, `INSERT INTO user_profiles (user_id, period_duration, cycle_duration, last_period_start, onboarding_last_period_start,
		user_goal, calculation_status, created_at, updated_at) VALUES (?, 5, 28, '2026-10-10', '2026-10-10', 'non_ttc', 'completed', NOW(), NOW())`, uid)

	in, err := inputs.Load(context.Background(), q, uid, today)
	require.NoError(t, err)
	assert.Empty(t, in.Periods, "the declared LMP is not a period")
	assert.Equal(t, engine.Some(day("2026-10-10")), in.Declared.LastPeriodStart)

	st := engine.Prepare(in, engine.DefaultParams())
	now := st.Resolve(day(today.String()), day(today.String()))
	assert.Equal(t, engine.StartOnboardingDeclared, now.Anchors.CurrentPeriodStartSource)
	assert.Equal(t, day("2026-10-10"), now.Anchors.CurrentPeriodStart)
	assert.NotEqual(t, engine.ResolutionUserLogged, now.ResolutionSource, "an onboarding-only user is never user_logged")
	assert.False(t, now.Warnings.Has(engine.WarnStaleOnboardingAnchor))

	later := day(today.AddDays(90).String())
	in, err = inputs.Load(context.Background(), q, uid, today.AddDays(90))
	require.NoError(t, err)
	stale := engine.Prepare(in, engine.DefaultParams()).Resolve(later, later)
	assert.True(t, stale.Warnings.Has(engine.WarnStaleOnboardingAnchor), "§2.41 stale anchor after 90 days")
	assert.Equal(t, engine.StartPredictedReference, stale.Anchors.CurrentPeriodStartSource)

	// T-M9-52: on a rolled-forward "period" day (85 days after the 2026-10-10 anchor: n = 3, cycle day 2) the
	// engine asks for the last period and paints no period day.
	rolled := day(today.AddDays(74).String())
	in, err = inputs.Load(context.Background(), q, uid, today.AddDays(74))
	require.NoError(t, err)
	st2, ui := engine.Prepare(in, engine.DefaultParams()).Today(rolled)
	assert.Equal(t, engine.Some(2), st2.CycleDay)
	assert.Equal(t, engine.BannerAskLastPeriod, ui.Banner)
	assert.Equal(t, engine.MainPhaseUnknown, st2.MainPhase, "ask_last_period never sits next to a period main_phase")
}

// seedBudget is the budget user: 120 periods and 400 markers (200 days of BBT + LH).
func seedBudget(t *testing.T, db *sql.DB) uint64 {
	uid := newUser(t, db, "09120000010")
	exec(t, db, `INSERT INTO user_profiles (user_id, birthday, cycle_duration, user_goal, calculation_status, created_at, updated_at)
		VALUES (?, '1995-03-10', 28, 'ttc', 'completed', NOW(), NOW())`, uid)

	start := today.AddDays(-28 * 120)
	var rows []string
	for i := range 120 {
		s := start.AddDays(28 * i)
		rows = append(rows, fmt.Sprintf("(%d, '%s', '%s', 1, 0, 'user_logged', NOW(), NOW())", uid, s, s.AddDays(4)))
	}
	exec(t, db, `INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, is_estimated, source, created_at, updated_at) VALUES `+strings.Join(rows, ","))
	rows = rows[:0]
	for i := range 200 { // 200 days × (BBT + LH) = 400 markers
		d := today.AddDays(-i)
		lh := "negative"
		if i%28 == 14 {
			lh = "positive"
		}
		rows = append(rows,
			fmt.Sprintf("(%d, '%s', 'measurements', 'bbt', '', NULL, %d.%02d, 'manual', NOW(), NOW())", uid, d, 36, 30+i%50),
			fmt.Sprintf("(%d, '%s', 'measurements', 'lh_test', '', '%s', NULL, 'manual', NOW(), NOW())", uid, d, lh))
	}
	exec(t, db, `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, source, created_at, updated_at) VALUES `+strings.Join(rows, ","))
	return uid
}

func TestLoad_Budget(t *testing.T) {
	if racedet.Enabled {
		t.Skip("latency budget is meaningless under -race")
	}
	db := testdb.New(t)
	q := store.New(db)
	uid := seedBudget(t, db)

	n := counted(t, func(ctx context.Context) {
		in, err := inputs.Load(ctx, q, uid, today)
		require.NoError(t, err)
		assert.Len(t, in.Periods, 120)
		assert.Len(t, in.BBT, 200)
	})
	assert.LessOrEqual(t, n, int64(4))

	loadP50, loadP95 := percentiles(t, func() {
		_, err := inputs.Load(context.Background(), q, uid, today)
		require.NoError(t, err)
	})
	// The floor: one trivial prepared round trip (the driver prepares, executes and closes like every sqlc query).
	_, floorP95 := percentiles(t, func() {
		var one int
		require.NoError(t, db.QueryRowContext(context.Background(), "SELECT ?", 1).Scan(&one))
	})
	t.Logf("Load 120 periods + 400 markers: p50 %v, p95 %v (trivial round trip p95 %v)", loadP50, loadP95, floorP95)
	if floorP95 > time.Millisecond {
		// A trivial query alone is slow: the host is loaded (parallel test runs), so wall-clock says nothing about
		// the loader. The query budget above still holds; the time budget is checked on an idle database.
		t.Logf("time budget not checked: the database answers SELECT 1 in %v at p95 (> 1ms)", floorP95)
		return
	}
	assert.LessOrEqual(t, loadP95, 5*time.Millisecond, "p95 budget")
}

// percentiles runs fn 60 times (after 3 warm-ups) and returns its p50 and p95 wall-clock times.
func percentiles(t *testing.T, fn func()) (p50, p95 time.Duration) {
	t.Helper()
	const runs = 60
	for range 3 {
		fn()
	}
	times := make([]time.Duration, 0, runs)
	for range runs {
		t0 := time.Now()
		fn()
		times = append(times, time.Since(t0))
	}
	slices.Sort(times)
	return times[runs/2], times[runs*95/100]
}
