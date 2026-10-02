package guide

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func day(s string) civildate.Date {
	d, err := civildate.Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestStatusOn(t *testing.T) {
	birth := day("2026-09-01")
	cases := []struct {
		today            string
		days, w, d, week int
		phase            string
		left             int
	}{
		{"2026-09-01", 0, 0, 0, 1, PhasePuerperium, 42},
		{"2026-09-18", 17, 2, 3, 3, PhasePuerperium, 25}, // nbl_v15_Main «۲ هفته و ۳ روز»
		{"2026-10-12", 41, 5, 6, 6, PhasePuerperium, 1},
		{"2026-10-13", 42, 6, 0, 7, PhaseRecovery, 0},
		{"2027-03-02", 182, 26, 0, 27, PhaseLate, 0},
		{"2026-08-30", 0, 0, 0, 1, PhasePuerperium, 42}, // a birth date after today counts as today
	}
	for _, c := range cases {
		s := StatusOn(birth, day(c.today))
		assert.Equal(t, c.days, s.DaysSinceBirth, c.today)
		assert.Equal(t, c.w, s.Weeks, c.today)
		assert.Equal(t, c.d, s.Days, c.today)
		assert.Equal(t, c.week, s.Week, c.today)
		assert.Equal(t, c.phase, s.Phase, c.today)
		assert.Equal(t, c.left, s.PuerperiumDaysLeft, c.today)
		assert.LessOrEqual(t, s.Progress, 1.0)
	}
}

func TestTipKey(t *testing.T) {
	assert.Equal(t, "1", TipKey(0))
	assert.Equal(t, "1", TipKey(1))
	assert.Equal(t, "12", TipKey(12))
	assert.Equal(t, LateTipKey, TipKey(13))
	assert.Len(t, TipKeys(), MaxTipWeek+1)
}

// Every slot has an embedded fa and en copy with every field filled.
func TestEmbeddedCopyComplete(t *testing.T) {
	for _, locale := range []string{"fa", "en"} {
		c := NewCopy(nil, locale, "")
		for _, s := range Slots() {
			txt, err := c.Text(context.Background(), s.Group, s.Key)
			require.NoError(t, err)
			for _, f := range s.Fields {
				line, ok := translator().Get("postpartum_copy."+s.Group+"."+s.Key+"."+f, locale)
				require.True(t, ok, "%s %s.%s.%s", locale, s.Group, s.Key, f)
				assert.Equal(t, line, txt[f])
				assert.NotEmpty(t, txt[f], "%s %s.%s.%s", locale, s.Group, s.Key, f)
			}
		}
	}
}

type fakeRows map[string]any // "group|key|locale" → payload

func (f fakeRows) Payload(_ context.Context, group, key, locale string) (any, bool, error) {
	v, ok := f[group+"|"+key+"|"+locale]
	return v, ok, nil
}

func TestCopy_AdminRowFirstThenDefaultLanguageThenEmbedded(t *testing.T) {
	row := phpval.NewMap()
	row.Set("title", " Admin title ")
	row.Set("body", "")
	rows := fakeRows{TipGroup + "|2|fa": row}
	ctx := context.Background()

	got, err := NewCopy(rows, "fa", "fa").Text(ctx, TipGroup, "2")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"title": "Admin title", "body": ""}, got, "a written row is used as is")

	got, err = NewCopy(rows, "ar", "fa").Text(ctx, TipGroup, "2")
	require.NoError(t, err)
	assert.Equal(t, "Admin title", got["title"], "default language row before the embedded copy")

	got, err = NewCopy(rows, "en", "").Text(ctx, TipGroup, "3")
	require.NoError(t, err)
	assert.Equal(t, "Bleeding gets lighter", got["title"])

	got, err = NewCopy(nil, "ar", "").Text(ctx, SafetyGroup, SafetyUrgent)
	require.NoError(t, err)
	assert.Equal(t, "Emergency 115", got["emergency_label"], "a language without a file falls back to English")
}
