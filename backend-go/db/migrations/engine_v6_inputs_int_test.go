package migrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	ppstore "github.com/ritme/backend-go/internal/postpartum/store"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

const engineV6InputsVersion int64 = 98 // 00098_engine_v6_inputs.sql

func newV6User(t *testing.T, conn *sql.DB, mobile string) uint64 {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('V6', ?, NOW(), NOW())`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id)
}

func columnExists(t *testing.T, conn *sql.DB, table, column string) bool {
	t.Helper()
	var n int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, column).Scan(&n))
	return n == 1
}

func v6Now() sql.NullTime { return sql.NullTime{Time: time.Now().In(civildate.Tehran), Valid: true} }

// T-M9-24: every new column and table is writable and readable through its queries, cascades with the user, and Down
// restores the 00097 schema without touching the rows that were there before.
func TestEngineV6InputsMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	p, err := platformdb.NewMigrator(conn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	cq, pq, ppq := cyclestore.New(conn), profilestore.New(conn), ppstore.New(conn)
	uid := newV6User(t, conn, "09129990098")
	other := newV6User(t, conn, "09129990099")

	// user_profiles: an existing profile row (written before the migration's columns are used) keeps its values.
	_, err = conn.Exec(`INSERT INTO user_profiles (user_id, cycle_duration, last_period_start, created_at, updated_at)
		VALUES (?, 29, '2026-09-30', NOW(), NOW())`, uid)
	require.NoError(t, err)
	in, err := pq.GetProfileEngineInputs(ctx, uid)
	require.NoError(t, err)
	assert.False(t, in.OnboardingLastPeriodStart.Valid, "existing rows read NULL")
	assert.False(t, in.Timezone.Valid)

	lmp := civildate.MustParse("2026-09-28")
	require.NoError(t, pq.SetOnboardingLastPeriodStart(ctx, profilestore.SetOnboardingLastPeriodStartParams{
		UserID: uid, OnboardingLastPeriodStart: civildate.NullDate{Date: lmp, Valid: true}, Now: v6Now()}))
	require.NoError(t, pq.SetProfileTimezone(ctx, profilestore.SetProfileTimezoneParams{
		UserID: uid, Timezone: sql.NullString{String: "Europe/Berlin", Valid: true}, Now: v6Now()}))
	in, err = pq.GetProfileEngineInputs(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, civildate.NullDate{Date: lmp, Valid: true}, in.OnboardingLastPeriodStart)
	assert.Equal(t, "Europe/Berlin", in.Timezone.String)
	var rows, cycle int
	var lps string
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*), MAX(cycle_duration), DATE_FORMAT(MAX(last_period_start), '%Y-%m-%d')
		FROM user_profiles WHERE user_id = ?`, uid).Scan(&rows, &cycle, &lps))
	assert.Equal(t, []any{1, 29, "2026-09-30"}, []any{rows, cycle, lps}, "upsert: one row, other columns kept")

	// A user without a profile row gets one from the timezone upsert (user_id is UNIQUE since 00096).
	require.NoError(t, pq.SetProfileTimezone(ctx, profilestore.SetProfileTimezoneParams{
		UserID: other, Timezone: sql.NullString{String: "Asia/Tehran", Valid: true}, Now: v6Now()}))
	in, err = pq.GetProfileEngineInputs(ctx, other)
	require.NoError(t, err)
	assert.Equal(t, "Asia/Tehran", in.Timezone.String)

	// postpartum_profiles.breastfeeding: NULL until answered, then true / false / NULL again.
	_, err = conn.Exec(`INSERT INTO postpartum_profiles (user_id, birth_date, created_at, updated_at) VALUES (?, '2026-08-01', NOW(), NOW())`, uid)
	require.NoError(t, err)
	bf, err := ppq.GetPostpartumBreastfeeding(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, civildate.MustParse("2026-08-01"), bf.BirthDate)
	assert.False(t, bf.Breastfeeding.Valid)
	for _, want := range []sql.NullBool{{Bool: true, Valid: true}, {Bool: false, Valid: true}, {}} {
		n, err := ppq.SetPostpartumBreastfeeding(ctx, ppstore.SetPostpartumBreastfeedingParams{Breastfeeding: want, Now: v6Now(), UserID: uid})
		require.NoError(t, err)
		assert.EqualValues(t, 1, n)
		bf, err = ppq.GetPostpartumBreastfeeding(ctx, uid)
		require.NoError(t, err)
		assert.Equal(t, want, bf.Breastfeeding)
	}
	n, err := ppq.SetPostpartumBreastfeeding(ctx, ppstore.SetPostpartumBreastfeedingParams{
		Breastfeeding: sql.NullBool{Bool: true, Valid: true}, Now: v6Now(), UserID: other})
	require.NoError(t, err)
	assert.Zero(t, n, "no postpartum profile, nothing written")
	require.NoError(t, ppq.UpsertPostpartumProfile(ctx, ppstore.UpsertPostpartumProfileParams{
		UserID: uid, BirthDate: civildate.MustParse("2026-08-02"), BabyCount: 1, Source: "direct", Now: v6Now()}))
	_, _ = ppq.SetPostpartumBreastfeeding(ctx, ppstore.SetPostpartumBreastfeedingParams{
		Breastfeeding: sql.NullBool{Bool: true, Valid: true}, Now: v6Now(), UserID: uid})
	require.NoError(t, ppq.UpsertPostpartumProfile(ctx, ppstore.UpsertPostpartumProfileParams{
		UserID: uid, BirthDate: civildate.MustParse("2026-08-03"), BabyCount: 1, Source: "direct", Now: v6Now()}))
	bf, err = ppq.GetPostpartumBreastfeeding(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, sql.NullBool{Bool: true, Valid: true}, bf.Breastfeeding, "the profile upsert keeps the answer")

	// cycle_skips: idempotent insert, ordered list, user-scoped delete.
	d1, d2 := civildate.MustParse("2026-10-28"), civildate.MustParse("2026-09-30")
	for _, d := range []civildate.Date{d1, d2} {
		n, err := cq.InsertCycleSkip(ctx, cyclestore.InsertCycleSkipParams{UserID: uid, ExpectedStart: d, Now: v6Now()})
		require.NoError(t, err)
		assert.EqualValues(t, 1, n)
	}
	n, err = cq.InsertCycleSkip(ctx, cyclestore.InsertCycleSkipParams{UserID: uid, ExpectedStart: d1, Now: v6Now()})
	require.NoError(t, err)
	assert.Zero(t, n, "same skip again changes nothing")
	skips, err := cq.ListCycleSkips(ctx, uid)
	require.NoError(t, err)
	require.Len(t, skips, 2)
	assert.Equal(t, []civildate.Date{d2, d1}, []civildate.Date{skips[0].ExpectedStart, skips[1].ExpectedStart})
	n, err = cq.DeleteCycleSkip(ctx, cyclestore.DeleteCycleSkipParams{UserID: other, ExpectedStart: d1})
	require.NoError(t, err)
	assert.Zero(t, n, "another user's delete never touches her skip")
	n, err = cq.DeleteCycleSkip(ctx, cyclestore.DeleteCycleSkipParams{UserID: uid, ExpectedStart: d1})
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	// user_revisions: absent = no row; engine bumps move both, analysis bumps only analysis_rev.
	_, err = cq.GetUserRevisions(ctx, uid)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, cq.BumpUserRevisions(ctx, cyclestore.BumpUserRevisionsParams{UserID: uid, Now: v6Now()}))
	require.NoError(t, cq.BumpUserRevisions(ctx, cyclestore.BumpUserRevisionsParams{UserID: uid, Now: v6Now()}))
	rev, err := cq.GetUserRevisions(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, cyclestore.GetUserRevisionsRow{EngineRev: 2, AnalysisRev: 2}, rev)
	require.NoError(t, cq.BumpAnalysisRevision(ctx, cyclestore.BumpAnalysisRevisionParams{UserID: uid, Now: v6Now()}))
	rev, err = cq.GetUserRevisions(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, cyclestore.GetUserRevisionsRow{EngineRev: 2, AnalysisRev: 3}, rev)
	require.NoError(t, cq.BumpAnalysisRevision(ctx, cyclestore.BumpAnalysisRevisionParams{UserID: other, Now: v6Now()}))
	rev, err = cq.GetUserRevisions(ctx, other)
	require.NoError(t, err)
	assert.Equal(t, cyclestore.GetUserRevisionsRow{EngineRev: 0, AnalysisRev: 1}, rev)

	// FK cascade: deleting the user deletes her skips, revisions and postpartum profile.
	_, err = conn.Exec(`DELETE FROM users WHERE id = ?`, uid)
	require.NoError(t, err)
	for _, table := range []string{"cycle_skips", "user_revisions", "postpartum_profiles", "user_profiles"} {
		var c int
		require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, uid).Scan(&c))
		assert.Zero(t, c, "%s cascades", table)
	}

	// Down restores 00097: new tables and columns gone, the other user's profile row stays.
	_, err = p.DownTo(ctx, engineV6InputsVersion-1)
	require.NoError(t, err)
	assert.False(t, tableExists(t, conn, "cycle_skips"))
	assert.False(t, tableExists(t, conn, "user_revisions"))
	assert.False(t, columnExists(t, conn, "user_profiles", "onboarding_last_period_start"))
	assert.False(t, columnExists(t, conn, "user_profiles", "timezone"))
	assert.False(t, columnExists(t, conn, "postpartum_profiles", "breastfeeding"))
	var kept int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM user_profiles WHERE user_id = ?`, other).Scan(&kept))
	assert.Equal(t, 1, kept)

	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.True(t, tableExists(t, conn, "cycle_skips"))
	assert.True(t, tableExists(t, conn, "user_revisions"))
	assert.True(t, columnExists(t, conn, "user_profiles", "timezone"))
	assert.True(t, columnExists(t, conn, "postpartum_profiles", "breastfeeding"))
}

// T-M9-24: concurrent engine-input writes never lose a revision bump (the upsert increments in place).
func TestBumpUserRevisions_Concurrent(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	q := cyclestore.New(conn)
	uid := newV6User(t, conn, "09129990100")

	const workers = 10
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := q.BumpUserRevisions(ctx, cyclestore.BumpUserRevisionsParams{UserID: uid, Now: v6Now()}); err != nil {
				errs <- fmt.Errorf("worker %d: %w", i, err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	rev, err := q.GetUserRevisions(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, cyclestore.GetUserRevisionsRow{EngineRev: workers, AnalysisRev: workers}, rev)
}
