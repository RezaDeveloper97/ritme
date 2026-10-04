package shared_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/companion/shared"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// modeOwner is an owner whose last logged period started 2026-07-20 (28-day cycle): on 2026-09-23 the engine sees her
// ~37 days late — what a pregnancy she keeps private looks like through a cycle view.
func modeOwner(t *testing.T, db *sql.DB, mobile string) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-07-01 09:00:00', '2026-07-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_profiles (user_id, last_period_start, period_duration, cycle_duration, created_at, updated_at)
		VALUES (?, '2026-07-20', 5, 28, '2026-07-01 09:00:00', '2026-07-01 09:00:00')`, id)
	require.NoError(t, err)
	entry := `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at)
		VALUES (?, '2026-09-22', ?, ?, ?, ?, NULL, NULL, 'manual', '2026-09-22 09:00:00', '2026-09-22 09:00:00')`
	for _, row := range [][]any{
		{"mood", "moods", "calm", "1"},
		{"symptoms", "general", "fatigue", "mild"},
		{"symptoms", "general", "leg_cramps", "mild"},  // pregnancy only
		{"pain", "location", "leg", "mild"},            // pregnancy only
		{"pain", "location", "stitches", "mild"},       // postpartum only
		{"symptoms", "general", "night_sweats", "yes"}, // menopause only
		{"pain", "location", "joints", "mild"},         // menopause only
	} {
		_, err := db.Exec(entry, append([]any{id}, row...)...)
		require.NoError(t, err)
	}
	return uint64(id) //nolint:gosec // test id
}

func jsonOf(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

func rawOf(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return string(raw)
}

func assertNeutral(t *testing.T, view map[string]any, msg string) {
	t.Helper()
	assert.Equal(t, false, view["has_data"], msg)
	for _, k := range []string{"cycle_day", "cycle_length", "main_phase", "days_to_period", "days_late", "predicted_next_period_start", "confidence"} {
		v, ok := view[k]
		assert.True(t, ok, "%s: key %s kept", msg, k)
		assert.Nil(t, v, "%s: %s", msg, k)
	}
}

// CMP-H1: a pregnant owner who shares cycle + symptoms but not pregnancy leaks neither lateness nor pregnancy-only
// symptoms; with the pregnancy grant the pregnancy items show, lateness past MaxLateDays still never does.
func TestReadFor_PregnancyWithoutGrantIsNeutral(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	owner := modeOwner(t, db, "09120007001")
	r := shared.NewReader(db)
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	noPreg := companion.Link{OwnerID: owner, Type: companion.TypePartner,
		Grants: companion.Grants{companion.SectionCycle: companion.LevelView, companion.SectionSymptoms: companion.LevelView}}
	withPreg := companion.Link{OwnerID: owner, Type: companion.TypePartner, Grants: companion.Grants{
		companion.SectionCycle: companion.LevelView, companion.SectionSymptoms: companion.LevelView, companion.SectionPregnancy: companion.LevelView}}

	// Cycle mode first: the lateness itself is past the cap, so it is not shown to anyone.
	v, err := r.ReadFor(ctx, noPreg, companion.SectionCycle, "en", now)
	require.NoError(t, err)
	cyc := jsonOf(t, v)
	assert.Equal(t, true, cyc["has_data"])
	assert.Nil(t, cyc["days_late"], "days_late past MaxLateDays is null")
	assert.Nil(t, cyc["cycle_day"], "and so is the cycle day that would reveal it")
	v, err = r.ReadFor(ctx, noPreg, companion.SectionSymptoms, "en", now)
	require.NoError(t, err)
	s := rawOf(t, v)
	assert.Contains(t, s, `"calm"`)
	assert.Contains(t, s, `"fatigue"`)
	for _, hidden := range []string{"leg_cramps", `"leg"`, "stitches", "night_sweats", "joints"} {
		assert.NotContains(t, s, hidden, "cycle mode")
	}

	// Pregnant (an active pregnancy profile wins over every stored mode).
	_, err = db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-07-20', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, owner)
	require.NoError(t, err)

	v, err = r.ReadFor(ctx, noPreg, companion.SectionCycle, "en", now)
	require.NoError(t, err)
	assertNeutral(t, jsonOf(t, v), "pregnant, no pregnancy grant")
	v, err = r.Read(ctx, owner, companion.SectionCycle, "en", now) // Read = no grants: the strictest view
	require.NoError(t, err)
	assertNeutral(t, jsonOf(t, v), "Read without a link")

	v, err = r.ReadFor(ctx, noPreg, companion.SectionSymptoms, "en", now)
	require.NoError(t, err)
	s = rawOf(t, v)
	assert.Contains(t, s, `"fatigue"`)
	for _, hidden := range []string{"leg_cramps", `"leg"`, "stitches", "night_sweats", "joints"} {
		assert.NotContains(t, s, hidden, "pregnant, no pregnancy grant")
	}

	v, err = r.ReadFor(ctx, withPreg, companion.SectionCycle, "en", now)
	require.NoError(t, err)
	cyc = jsonOf(t, v)
	assert.Equal(t, true, cyc["has_data"], "the pregnancy grant shows the cycle view")
	assert.Nil(t, cyc["days_late"])
	v, err = r.ReadFor(ctx, withPreg, companion.SectionSymptoms, "en", now)
	require.NoError(t, err)
	s = rawOf(t, v)
	assert.Contains(t, s, "leg_cramps", "pregnancy items with the pregnancy grant")
	assert.Contains(t, s, `"leg"`)
	for _, hidden := range []string{"stitches", "night_sweats", "joints"} {
		assert.NotContains(t, s, hidden, "other modes' items never")
	}
}

// Postpartum without the pregnancy grant is neutral too; menopause is neutral whatever the grants.
func TestReadFor_PostpartumAndMenopause(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	r := shared.NewReader(db)
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	all := companion.Grants{companion.SectionCycle: companion.LevelView, companion.SectionSymptoms: companion.LevelView,
		companion.SectionPregnancy: companion.LevelView}

	for _, tc := range []struct{ mode, mobile, keep, hide string }{
		{"postpartum", "09120007002", "stitches", "leg_cramps"},
		{"menopause", "09120007003", "", "night_sweats"},
	} {
		owner := modeOwner(t, db, tc.mobile)
		_, err := db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, gender, created_at, updated_at)
			VALUES (?, ?, 'female', NOW(), NOW())`, owner, tc.mode)
		require.NoError(t, err)

		v, err := r.ReadFor(ctx, companion.Link{OwnerID: owner, Grants: companion.Grants{companion.SectionCycle: companion.LevelView}},
			companion.SectionCycle, "en", now)
		require.NoError(t, err)
		assertNeutral(t, jsonOf(t, v), tc.mode+" without the pregnancy grant")

		v, err = r.ReadFor(ctx, companion.Link{OwnerID: owner, Grants: all}, companion.SectionCycle, "en", now)
		require.NoError(t, err)
		if tc.mode == "menopause" {
			assertNeutral(t, jsonOf(t, v), "menopause, whatever the grants")
		} else {
			assert.Equal(t, true, jsonOf(t, v)["has_data"], "postpartum with the pregnancy grant")
		}

		v, err = r.ReadFor(ctx, companion.Link{OwnerID: owner, Grants: all}, companion.SectionSymptoms, "en", now)
		require.NoError(t, err)
		s := rawOf(t, v)
		assert.Contains(t, s, `"fatigue"`, tc.mode)
		assert.NotContains(t, s, tc.hide, tc.mode)
		assert.NotContains(t, s, "joints", tc.mode)
		if tc.keep != "" {
			assert.Contains(t, s, tc.keep, tc.mode)
		}
		v, err = r.ReadFor(ctx, companion.Link{OwnerID: owner}, companion.SectionSymptoms, "en", now)
		require.NoError(t, err)
		assert.NotContains(t, rawOf(t, v), "stitches", tc.mode+" without the pregnancy grant")
	}
}
