package model_test

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := jsonx.Marshal(v, jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	require.NoError(t, err)
	return string(b)
}

func TestColumnsCoverTheTable(t *testing.T) {
	// every column of the row = id, user_id, log_date, the data columns, 2 timestamps.
	total := reflect.TypeOf(store.DailyHealthLog{}).NumField()
	assert.Len(t, model.Columns, total-5)
	seen := map[string]bool{}
	for _, c := range model.Columns {
		assert.False(t, seen[c.Name], c.Name)
		seen[c.Name] = true
		assert.NotNil(t, c.Field(&store.DailyHealthLog{}), c.Name)
	}
	assert.Len(t, model.FromRow(store.DailyHealthLog{}).ToArray().Keys(), total)
}

func TestSetCastsLikeEloquent(t *testing.T) {
	l := model.FromRow(store.DailyHealthLog{ID: 7, UserID: 1004, LogDate: civildate.MustParse("2026-09-22")})
	medications := phpval.NewMap()
	medications.Set("painkillers", "ibuprofen")
	for k, v := range map[string]any{
		"spotting":           "1",
		"fatigue":            int64(0),
		"weight":             61.555,
		"blood_sugar":        "98.55",
		"heart_rate":         "72",
		"exercise_duration":  45.0,
		"moods":              []any{"happy", "calm"},
		"medications":        medications,
		"notes":              "a/b <c>",
		"bleeding_intensity": "low",
	} {
		require.NoError(t, l.Set(k, v), k)
	}
	require.NoError(t, l.Set("sexual_activities", phpval.NewMap())) // empty PHP array
	got := l.Attributes([]string{"id", "log_date", "spotting", "fatigue", "weight", "blood_sugar", "heart_rate",
		"exercise_duration", "moods", "medications", "sexual_activities", "notes", "bleeding_intensity", "nope"})
	assert.JSONEq(t, `{"id":7,"log_date":"2026-09-22","spotting":true,"fatigue":false,"weight":"61.56",
		"blood_sugar":"98.6","heart_rate":72,"exercise_duration":45,"moods":["happy","calm"],
		"medications":{"painkillers":"ibuprofen"},"sexual_activities":[],"notes":"a/b <c>","bleeding_intensity":"low"}`,
		marshal(t, got))
	assert.Equal(t, `["happy","calm"]`, string(l.Row.Moods.V))

	require.NoError(t, l.Set("weight", nil))
	assert.Equal(t, sql.NullString{}, l.Row.Weight)
	assert.Error(t, l.Set("unknown", 1))
}

func TestAccessorsAndTriggers(t *testing.T) {
	l := model.FromRow(store.DailyHealthLog{})
	assert.Nil(t, l.HeadacheIntensity())
	assert.Nil(t, l.Moods())
	assert.Empty(t, enums.RecommendationTriggerActiveFor(l))

	require.NoError(t, l.Set("headache_intensity", "high"))
	require.NoError(t, l.Set("moods", []any{"sad", int64(3)}))
	require.NoError(t, l.Set("fatigue", true))
	assert.Equal(t, "high", *l.HeadacheIntensity())
	assert.Equal(t, []string{"sad"}, l.Moods())
	assert.True(t, *l.Fatigue())
	assert.Contains(t, enums.RecommendationTriggerActiveFor(l), string(enums.RecommendationTriggerHeadache))
	assert.Contains(t, enums.RecommendationTriggerActiveFor(l), string(enums.RecommendationTriggerLowMood))
}

func TestDecodeArrayQuirks(t *testing.T) {
	assert.Equal(t, `[]`, marshal(t, model.DecodeArray([]byte(`{}`))))
	assert.Equal(t, `["a","b"]`, marshal(t, model.DecodeArray([]byte(`{"0":"a","1":"b"}`))))
	assert.Nil(t, model.DecodeArray([]byte(`{bad`)))
}

func TestEnumValuesKeyOrder(t *testing.T) {
	keys := model.EnumValues().Keys()
	assert.Equal(t, "bleeding_intensity", keys[0])
	assert.Equal(t, "discharge_smell", keys[len(keys)-1])
	assert.Len(t, keys, 21)
}
