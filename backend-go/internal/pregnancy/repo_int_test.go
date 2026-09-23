package pregnancy

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/alerts"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/content"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func setup(t *testing.T) (*sql.DB, *store.Queries, uint64) {
	t.Helper()
	if !testdb.Enabled() {
		t.Skip("TEST_DB_DSN not set")
	}
	conn := testdb.New(t)
	res, err := conn.Exec("INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('p', '09900000099', NOW(), NOW())")
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return conn, store.New(conn), uint64(id) //nolint:gosec // G115
}

func toJSON(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := jsonx.Marshal(v, 0)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

func input(t *testing.T, js string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(js))
	require.NoError(t, err)
	return v.(phpval.Map)
}

var t0 = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

func TestSymptomUpsertShapes(t *testing.T) {
	_, q, uid := setup(t)
	ctx := context.Background()
	date := civildate.MustParse("2026-09-23")

	created, err := upsertSymptomLog(ctx, q, uid, date, "2026-09-23",
		input(t, `{"log_date":"2026-09-23","has_nausea":true,"nausea_severity":"mild","notes":"x"}`), t0)
	require.NoError(t, err)
	got := toJSON(t, created)
	// A fresh model carries only what was set (+ timestamps + id).
	assert.ElementsMatch(t, []string{"user_id", "log_date", "has_nausea", "nausea_severity", "notes", "updated_at", "created_at", "id"}, keys(got))
	assert.Equal(t, true, got["has_nausea"])
	assert.Equal(t, "2026-09-23", got["log_date"])
	assert.Equal(t, "2026-09-23T06:30:00.000000Z", got["created_at"])

	// Same values an hour later: not dirty → updated_at unchanged; full row returned.
	same, err := upsertSymptomLog(ctx, q, uid, date, "2026-09-23",
		input(t, `{"log_date":"2026-09-23","has_nausea":1,"nausea_severity":"mild"}`), t0.Add(time.Hour))
	require.NoError(t, err)
	got = toJSON(t, same)
	assert.Len(t, got, 34) // every column
	assert.Equal(t, "2026-09-23T06:30:00.000000Z", got["updated_at"])
	assert.Nil(t, got["has_vomiting"])

	changed, err := upsertSymptomLog(ctx, q, uid, date, "2026-09-23",
		input(t, `{"log_date":"2026-09-23","has_nausea":false}`), t0.Add(time.Hour))
	require.NoError(t, err)
	got = toJSON(t, changed)
	assert.Equal(t, false, got["has_nausea"])
	assert.Equal(t, "2026-09-23T07:30:00.000000Z", got["updated_at"])
	row, err := q.GetSymptomLog(ctx, store.GetSymptomLogParams{UserID: uid, LogDate: date})
	require.NoError(t, err)
	assert.Equal(t, sql.NullBool{Bool: false, Valid: true}, row.HasNausea)
	assert.Equal(t, "x", row.Notes.String)
}

func TestWeeklyUpsertDecimalsAndAlerts(t *testing.T) {
	_, q, uid := setup(t)
	ctx := context.Background()
	log, err := upsertWeeklyLog(ctx, q, uid, 27, int64(27), input(t,
		`{"log_date":"2026-09-23","pregnancy_week":27,"weight":72.85,"swelling_locations":["face","feet"],"systolic_pressure":165,"diastolic_pressure":112,"fasting_blood_sugar":130.5}`), t0)
	require.NoError(t, err)
	got := toJSON(t, log)
	assert.Equal(t, "72.85", got["weight"])
	assert.Equal(t, "130.50", got["fasting_blood_sugar"])
	assert.Equal(t, []any{"face", "feet"}, got["swelling_locations"])
	assert.NotContains(t, got, "has_blood_pressure_device")

	drafts := alerts.ForWeeklyLog(alerts.Context{Locale: "en", Week: 1}, log)
	require.Len(t, drafts, 2)
	models, err := createAlerts(ctx, q, uid, drafts, t0)
	require.NoError(t, err)
	a := toJSON(t, models[1])
	assert.Equal(t, map[string]any{"fasting": "130.50", "post_meal": nil}, a["trigger_symptoms"])
	assert.NotContains(t, a, "medical_history_flags")
	assert.NotContains(t, a, "is_read")

	cnt, err := q.CountActiveAlerts(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, store.CountActiveAlertsRow{Total: 2, Unread: 2, Warning: 2}, cnt)
	row, err := q.GetWeeklyLog(ctx, store.GetWeeklyLogParams{UserID: uid, PregnancyWeek: 27})
	require.NoError(t, err)
	assert.False(t, row.HasBloodPressureDevice)
	assert.Equal(t, "72.85", row.Weight.String)
}

func TestProfileSaveAndCalc(t *testing.T) {
	_, q, uid := setup(t)
	ctx := context.Background()
	require.NoError(t, saveProfile(ctx, q, uid, nil, jsonx.Obj("pregnancy_mode", true, "cycle_mode", false), t0))
	p, err := LoadProfile(ctx, q, uid)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.False(t, p.OnboardingCompleted)
	assert.EqualValues(t, 3, p.UncertaintyDays)

	require.NoError(t, saveProfile(ctx, q, uid, p, jsonx.Obj("age_source", "lmp", "lmp_date", "2026-07-01",
		"pre_existing_conditions", []any{"diabetes"}), t0.Add(time.Hour)))
	p, err = LoadProfile(ctx, q, uid)
	require.NoError(t, err)
	got := toJSON(t, ProfileJSON(p))
	assert.Equal(t, "2026-06-30T20:30:00.000000Z", got["lmp_date"], "plain date cast → previous day UTC")
	assert.Equal(t, []any{"diabetes"}, got["pre_existing_conditions"])
	assert.Equal(t, "2026-09-23T07:30:00.000000Z", got["updated_at"])
	assert.True(t, calc.IsHighRisk(p))
	assert.Equal(t, 13, calc.New(p, "en", civildate.MustParse("2026-09-23")).CurrentWeek())

	// Unchanged update keeps updated_at.
	require.NoError(t, saveProfile(ctx, q, uid, p, jsonx.Obj("age_source", "lmp"), t0.Add(2*time.Hour)))
	p2, err := LoadProfile(ctx, q, uid)
	require.NoError(t, err)
	assert.Equal(t, p.UpdatedAt, p2.UpdatedAt)

	none, err := LoadProfile(ctx, q, uid+1)
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestWeeklyContentLocalize(t *testing.T) {
	conn, q, _ := setup(t)
	_, err := conn.Exec(`INSERT INTO pregnancy_weekly_content (week_number, fetal_development, faq) VALUES (8, '{"en":"E","fa":"F"}', NULL)`)
	require.NoError(t, err)
	row, err := q.GetWeeklyContent(context.Background(), 8)
	require.NoError(t, err)
	assert.Equal(t, "F", toJSON(t, content.Localize(row, "fa"))["fetal_development"])
	whole := toJSON(t, content.Localize(row, "ar"))
	assert.Equal(t, map[string]any{"en": "E", "fa": "F"}, whole["fetal_development"], "missing locale → whole object")
	assert.Nil(t, whole["faq"])
	assert.Equal(t, "en", content.Locale("ar"))
	assert.Equal(t, "fa", content.Locale("fa"))
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
