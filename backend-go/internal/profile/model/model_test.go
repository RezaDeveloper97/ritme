package model_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/profile/model"
	"github.com/ritme/backend-go/internal/profile/store"
)

// laterColumns applies the `ALTER TABLE … ADD COLUMN` / `DROP COLUMN` statements of the Up sections after the baseline
// to a table's column list, in migration order (AFTER honoured, so the result is the real table order).
func laterColumns(t *testing.T, table string, cols []string) []string {
	t.Helper()
	files, err := filepath.Glob("../../../db/migrations/000*.sql")
	require.NoError(t, err)
	slices.Sort(files)
	stmt := regexp.MustCompile("(?i)ALTER TABLE `" + table + "` (ADD|DROP) COLUMN (?:IF (?:NOT )?EXISTS )?`([a-z0-9_]+)`([^;]*);")
	after := regexp.MustCompile("(?i) AFTER `([a-z0-9_]+)`")
	for _, f := range files {
		if strings.HasSuffix(f, "00001_baseline.sql") {
			continue
		}
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		up, _, _ := strings.Cut(string(raw), "-- +goose Down")
		for _, m := range stmt.FindAllStringSubmatch(up, -1) {
			op, col := strings.ToUpper(m[1]), m[2]
			i := slices.Index(cols, col)
			if op == "DROP" {
				if i >= 0 {
					cols = slices.Delete(cols, i, i+1)
				}
				continue
			}
			if i >= 0 {
				continue
			}
			if a := after.FindStringSubmatch(m[3]); a != nil {
				j := slices.Index(cols, a[1])
				require.GreaterOrEqual(t, j, 0, "%s: AFTER %s", f, a[1])
				cols = slices.Insert(cols, j+1, col)
				continue
			}
			cols = append(cols, col)
		}
	}
	return cols
}

// The generic serializer names columns from sqlc field names; every model it renders must
// map back to the exact column list (and order) of the migrated schema (baseline + later ALTERs).
func TestColumnNamesMatchSchema(t *testing.T) {
	raw, err := os.ReadFile("../../../db/migrations/00001_baseline.sql")
	require.NoError(t, err)
	col := regexp.MustCompile("(?m)^  `([a-z0-9_]+)` ")
	for table, row := range map[string]any{
		"users": store.User{}, "user_profiles": store.UserProfile{}, "cycle_histories": store.CycleHistory{},
		"daily_health_logs": store.DailyHealthLog{}, "pregnancy_profiles": store.PregnancyProfile{},
		"pregnancy_symptom_logs": store.PregnancySymptomLog{}, "pregnancy_weekly_logs": store.PregnancyWeeklyLog{},
		"pregnancy_fetal_movements": store.PregnancyFetalMovement{}, "reminders": store.Reminder{},
	} {
		block := regexp.MustCompile("(?s)CREATE TABLE `" + table + "` \\((.*?)\n\\)").FindSubmatch(raw)
		require.NotNil(t, block, table)
		var want []string
		for _, m := range col.FindAllSubmatch(block[1], -1) {
			want = append(want, string(m[1]))
		}
		want = laterColumns(t, table, want)
		var got []string
		rt := reflect.TypeOf(row)
		for i := range rt.NumField() {
			got = append(got, model.ColumnName(rt.Field(i).Name))
		}
		assert.Equal(t, want, got, table)
	}
}

func TestProfileJSON(t *testing.T) {
	ts := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	p := &model.UserProfile{
		ID: 1, UserID: 2,
		Birthday:          civildate.NullDate{Date: civildate.MustParse("1995-02-14"), Valid: true},
		Weight:            sql.NullString{String: "70.00", Valid: true},
		Height:            sql.NullInt16{Int16: 165, Valid: true},
		UserGoal:          "non_ttc",
		ChronicConditions: sql.Null[json.RawMessage]{V: json.RawMessage(`[]`), Valid: true},
		SubscriptionType:  "free", CalculationStatus: "pending",
		CalculationStartedAt: sql.NullTime{Time: ts, Valid: true},
		CreatedAt:            sql.NullTime{Time: ts, Valid: true},
	}
	b, err := jsonx.Marshal(model.ProfileJSON(p), 0)
	require.NoError(t, err)
	assert.Equal(t, `{"id":1,"user_id":2,"birthday":"1995-02-14","weight":70,"height":165,"period_duration":null,`+
		`"cycle_duration":null,"last_period_start":null,"user_goal":"non_ttc","pregnancy_intention":null,`+
		`"chronic_conditions":[],"subscription_type":"free","calculation_status":"pending",`+
		`"calculation_started_at":"2026-09-23T06:30:00.000000Z","calculation_completed_at":null,`+
		`"calculation_version":0,"created_at":"2026-09-23T06:30:00.000000Z","updated_at":null,`+
		`"onboarding_last_period_start":null,"timezone":null}`, string(b))
	assert.Nil(t, model.ProfileJSON(nil))
}

func TestAttributesCasts(t *testing.T) {
	r := store.Reminder{
		ID: 3, UserID: 2, Type: "medication", Title: "x", Recurrence: "daily",
		RecurrenceTime: sql.NullString{String: "08:00:00", Valid: true},
		StartsOn:       civildate.NullDate{Date: civildate.MustParse("2026-09-23"), Valid: true},
		IsActive:       true,
		Meta:           sql.Null[json.RawMessage]{V: json.RawMessage(`{}`), Valid: true},
	}
	b, err := jsonx.Marshal(model.Attributes(&r, model.ReminderCasts), 0)
	require.NoError(t, err)
	assert.Equal(t, `{"id":3,"user_id":2,"type":"medication","title":"x","subtitle":null,"notes":null,`+
		`"scheduled_at":null,"recurrence":"daily","recurrence_time":"08:00:00",`+
		`"starts_on":"2026-09-22T20:30:00.000000Z","ends_on":null,"is_active":true,"meta":[],`+
		`"created_at":null,"updated_at":null}`, string(b))

	w := store.PregnancyWeeklyLog{Weight: sql.NullString{String: "71.1", Valid: true}, HasSwelling: sql.NullBool{Valid: true}}
	m := model.Attributes(&w, model.PregnancyWeeklyLogCasts).(*jsonx.OrderedMap)
	weight, _ := m.Get("weight")
	swelling, _ := m.Get("has_swelling")
	assert.Equal(t, jsonx.MustDecimal("71.1", 2), weight)
	assert.Equal(t, false, swelling)
	assert.Equal(t, []any{}, model.List([]store.Reminder(nil), model.ReminderCasts))
}
