package fertility

import (
	"database/sql"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/fertility/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func TestChanceOf(t *testing.T) {
	cases := map[enums.FertilityLevel][2]any{
		enums.FertilityLevelLow:    {"low", 1},
		enums.FertilityLevelMedium: {"medium", 3},
		enums.FertilityLevelHigh:   {"high", 4},
		enums.FertilityLevelPeak:   {"peak", 5},
		enums.FertilityLevelNone:   {"none", 0},
	}
	for in, want := range cases {
		c := ChanceOf(in)
		require.NotNil(t, c.Level, in)
		assert.Equal(t, want[0], *c.Level, in)
		assert.Equal(t, want[1], c.Bars, in)
	}
	unknown := ChanceOf(enums.FertilityLevelUnknown)
	assert.Nil(t, unknown.Level)
	b, err := jsonx.Marshal(unknown.JSON("en"), jsonx.UnescapedUnicode)
	require.NoError(t, err)
	assert.JSONEq(t, `{"level":null,"label":"Unknown","bars":0}`, string(b))
}

func TestDayFromRow(t *testing.T) {
	d := DayFromRow(civildate.MustParse("2026-09-22"), store.GetMergedDayRow{
		BasalBodyTemperature: sql.NullString{String: "36.50", Valid: true},
		BloatingIntensity:    sql.NullString{String: "high", Valid: true},
		OvarianPainIntensity: sql.NullString{String: "low", Valid: true},
		Spotting:             sql.NullBool{Bool: false, Valid: true},
		BbtTime:              sql.NullString{String: "06:05:00", Valid: true},
	})
	assert.Equal(t, []string{"ovarian_pain", "bloating"}, d.Symptoms)
	day := 3
	b, err := jsonx.Marshal(d.JSON(Context{CycleDay: &day, Chance: ChanceOf(enums.FertilityLevelLow)}, "en"), jsonx.UnescapedUnicode)
	require.NoError(t, err)
	assert.Equal(t, `{"date":"2026-09-22","cycle_day":3,"lh":null,"mucus":null,"bbt":"36.50","bbt_time":"06:05",`+
		`"intercourse":null,"symptoms":["ovarian_pain","bloating"],"note":null,"chance":{"level":"low","label":"Low","bars":1}}`, string(b))

	empty := DayFromRow(civildate.MustParse("2026-09-22"), store.GetMergedDayRow{Spotting: sql.NullBool{Bool: true, Valid: true}})
	assert.Equal(t, []string{"spotting"}, empty.Symptoms)
}

// Every language file carries the same keys as English, and every enum value has a label.
func TestLangFiles(t *testing.T) {
	codes, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	require.NotEmpty(t, codes)
	groups := map[string][]string{
		"lh": LHResults, "mucus": CervicalMucus, "intercourse": IntercourseTypes,
		"chance": {"none", "low", "medium", "high", "peak", "unknown"},
	}
	for _, c := range codes {
		for g, values := range groups {
			for _, v := range values {
				assert.NotEmpty(t, Label(g, v, c.Name()), "%s %s.%s", c.Name(), g, v)
			}
		}
		for _, k := range []string{"messages.validation_failed", "messages.day_saved", "validation.bbt_range", "validation.date_future"} {
			assert.NotEqual(t, "fertility."+k, T(k, c.Name()), "%s %s", c.Name(), k)
		}
		assert.NotEmpty(t, attributes(c.Name()))
	}
}
