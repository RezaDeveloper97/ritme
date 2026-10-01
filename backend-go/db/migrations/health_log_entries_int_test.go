package migrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const healthLogEntriesVersion int64 = 20 // 00020_health_log_entries.sql

// legacyFixtures are daily_health_logs rows covering every column kind plus the odd values old data can
// hold (unknown enum strings, empty strings, false booleans, contradictory pairs, PHP arrays with gaps,
// bare JSON scalars, nested JSON, nulls inside arrays, non-object medications, long notes).
var legacyFixtures = []string{
	`(user_id, log_date, bleeding_intensity, blood_color, has_clots, clots_amount, spotting, bleeding_smell,
	  headache_intensity, stomach_ache_intensity, pelvic_pain_intensity, breast_pain_intensity, back_pain_intensity,
	  ovarian_pain_intensity, nausea_intensity, bloating_intensity, diarrhea, constipation, appetite_change, food_craving,
	  breast_sensitivity_intensity, vaginal_dryness, vaginal_burning, vaginal_burning_intensity, vaginal_itching,
	  vaginal_itching_intensity, vaginal_smell_change, urination_change, urination_burning_intensity, acne, oily_skin,
	  hair_loss, swelling, fatigue, dizziness, hot_flashes, chills, moods, sleep_duration, sleep_quality, exercise_type,
	  exercise_duration, exercise_intensity, sexual_activities, sexual_desire, intercourse_type, weight,
	  basal_body_temperature, heart_rate, systolic_pressure, diastolic_pressure, blood_sugar, energy_level,
	  discharge_color, discharge_texture, discharge_amount, discharge_smell, discharge_itching, discharge_burning,
	  frequent_urination, medications, notes, created_at, updated_at)
	 VALUES (%[1]d, '2026-09-20', 'very_high', 'dark_red', 1, 'medium', 1, 'strong_unpleasant',
	  'low', 'medium', 'high', 'low', 'medium', 'high', 'low', 'high', 1, 0, 'loss', 1,
	  'medium', 1, 1, 'high', 1, NULL, 0, 'increase', 'medium', 1, 0,
	  1, 0, 1, 0, 1, 0, '["calm","sad","happy"]', '6_9', 'medium', '["yoga","walking"]',
	  45, 'low', '["dryness","protected_intercourse"]', 'higher', 'protected', 58.40,
	  36.42, 72, 118, 76, 97.5, 'very_low',
	  'egg_white', 'thick', 'low', 'normal', 0, 1,
	  1, '{"painkillers":"ibuprofen 400","supplements":"iron"}', 'یادداشت', '2026-09-20 21:00:00', '2026-09-20 22:00:00')`,
	`(user_id, log_date, bleeding_intensity, blood_color, spotting, vaginal_burning, vaginal_burning_intensity,
	  moods, exercise_type, sexual_activities, medications, notes, created_at, updated_at)
	 VALUES (%[1]d, '2026-09-21', 'extreme', '', 0, 0, 'high',
	  '{"1":"calm","3":"sad"}', '"yoga"', '["x", {"a": 1}, null, 5, "x"]', '["a","b"]', REPEAT('ب', 3000),
	  '2026-09-21 21:00:00', '2026-09-21 21:00:00')`,
	`(user_id, log_date, moods, exercise_type, sexual_activities, medications, created_at, updated_at)
	 VALUES (%[1]d, '2026-09-22', '[]', 'null', '{}', '{"vitamin d": 5, "x": {"a": [1, 2]}, "n": null, "q\\"k": "v"}',
	  '2026-09-22 21:00:00', NULL)`,
	`(user_id, log_date, created_at, updated_at) VALUES (%[1]d, '2026-09-23', NULL, NULL)`,
}

func insertUser(t *testing.T, db *sql.DB, mobile string) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('T', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id) //nolint:gosec // AUTO_INCREMENT ids are positive
}

func entryLine(userID uint64, date, slot string, code, num, text sql.NullString) string {
	s := func(v sql.NullString) string {
		if !v.Valid {
			return "∅"
		}
		return v.String
	}
	return fmt.Sprintf("%d|%s|%s|%s|%s|%s", userID, date, slot, s(code), s(num), s(text))
}

// stored is every health_log_entries row as a sorted line list.
func stored(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query("SELECT user_id, log_date, category, param, item, value_code, value_num, value_text, source FROM health_log_entries")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var uid uint64
		var date, cat, param, item, source string
		var code, num, text sql.NullString
		require.NoError(t, rows.Scan(&uid, &date, &cat, &param, &item, &code, &num, &text, &source))
		require.Equal(t, "legacy", source)
		out = append(out, entryLine(uid, date[:10], cat+"."+param+"."+item, code, num, text))
	}
	require.NoError(t, rows.Err())
	sort.Strings(out)
	return out
}

// projected is the Go projection (taxonomy.ProjectRow) of every daily_health_logs row.
func projected(t *testing.T, db *sql.DB) []string {
	t.Helper()
	q := store.New(db)
	ids, err := db.Query("SELECT id FROM daily_health_logs")
	require.NoError(t, err)
	defer func() { _ = ids.Close() }()
	var out []string
	for ids.Next() {
		var id uint64
		require.NoError(t, ids.Scan(&id))
		row, err := q.GetDailyHealthLog(context.Background(), id)
		require.NoError(t, err)
		for _, e := range healthlog.ProjectLegacyRow(row) {
			out = append(out, entryLine(row.UserID, row.LogDate.String(), e.Slot(), e.Code, e.Num, e.Text))
		}
	}
	require.NoError(t, ids.Err())
	sort.Strings(out)
	return out
}

func runBackfill(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range taxonomy.BackfillStatements() {
		_, err := db.Exec(stmt)
		require.NoError(t, err, stmt)
	}
}

// B-N3-01: 00020 backfills every legacy row without loss (SQL ≡ the Go projection the live sync uses), is
// idempotent, and its Down drops only the new table.
func TestHealthLogEntries_BackfillDownUp(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	_, err = p.DownTo(ctx, healthLogEntriesVersion-1)
	require.NoError(t, err)
	var n int
	require.Error(t, db.QueryRow("SELECT COUNT(*) FROM health_log_entries").Scan(&n), "Down drops the table")

	a := insertUser(t, db, "09120000001")
	b := insertUser(t, db, "09120000002")
	for _, uid := range []uint64{a, b} {
		for _, f := range legacyFixtures {
			_, err := db.Exec(fmt.Sprintf("INSERT INTO daily_health_logs "+f, uid))
			require.NoError(t, err, f)
		}
	}
	var legacyBefore string
	require.NoError(t, db.QueryRow("CHECKSUM TABLE daily_health_logs").Scan(new(string), &legacyBefore))

	_, err = p.UpTo(ctx, healthLogEntriesVersion)
	require.NoError(t, err)
	got := stored(t, db)
	want := projected(t, db)
	require.NotEmpty(t, want)
	assert.Equal(t, want, got, "the SQL backfill equals the Go projection")

	// spot checks of the odd values
	joined := strings.Join(got, "\n")
	for _, s := range []string{
		fmt.Sprintf("%d|2026-09-20|bleeding.flow.|very_heavy|∅|∅", a),
		fmt.Sprintf("%d|2026-09-20|urogenital.symptoms.vaginal_burning|severe|∅|∅", a),
		fmt.Sprintf("%d|2026-09-20|urogenital.symptoms.vaginal_itching|yes|∅|∅", a),
		fmt.Sprintf("%d|2026-09-20|measurements.blood_sugar.|∅|97.50|∅", a),
		fmt.Sprintf("%d|2026-09-20|discharge.consistency.|sticky|∅|∅", a),
		fmt.Sprintf("%d|2026-09-21|bleeding.flow.|extreme|∅|∅", a),
		fmt.Sprintf("%d|2026-09-21|bleeding.color.||∅|∅", a),
		fmt.Sprintf("%d|2026-09-21|mood.moods.sad|yes|∅|∅", a),
		fmt.Sprintf("%d|2026-09-21|activity.types.yoga|yes|∅|∅", a),
		fmt.Sprintf(`%d|2026-09-21|sex.symptoms.{"a": 1}|yes|∅|∅`, a),
		fmt.Sprintf(`%d|2026-09-21|meds.other._raw|∅|∅|["a","b"]`, a),
		fmt.Sprintf("%d|2026-09-22|meds.other.vitamin d|∅|∅|5", a),
		fmt.Sprintf(`%d|2026-09-22|meds.other.x|∅|∅|{"a": [1, 2]}`, a),
		fmt.Sprintf(`%d|2026-09-22|meds.other.q"k|∅|∅|v`, a),
	} {
		assert.Contains(t, joined, s)
	}
	assert.NotContains(t, joined, "|2026-09-23|", "an empty legacy row projects nothing")
	assert.NotContains(t, joined, "meds.other.n|", "a null medication value is nothing")

	// idempotent: running the backfill again changes nothing
	runBackfill(t, db)
	assert.Equal(t, got, stored(t, db))

	// Down + Up again: the legacy table was never touched, the result is the same
	_, err = p.DownTo(ctx, healthLogEntriesVersion-1)
	require.NoError(t, err)
	var legacyAfter string
	require.NoError(t, db.QueryRow("CHECKSUM TABLE daily_health_logs").Scan(new(string), &legacyAfter))
	assert.Equal(t, legacyBefore, legacyAfter, "Up/Down never modify daily_health_logs")
	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, got, stored(t, db))
}

// Acceptance: the backfill on a copy of the seeded data (the contract fixtures).
func TestHealthLogEntries_BackfillOnSeededData(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))
	var logs int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM daily_health_logs").Scan(&logs))
	require.Positive(t, logs)

	want := projected(t, db)
	assert.Equal(t, want, stored(t, db), "the committed fixture already carries the backfill")

	_, err := db.Exec("DELETE FROM health_log_entries")
	require.NoError(t, err)
	runBackfill(t, db)
	assert.Equal(t, want, stored(t, db))
}
