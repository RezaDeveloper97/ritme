package postpartum

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

func toMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func TestSafetyJSON(t *testing.T) {
	ctx := context.Background()
	cp := guide.NewCopy(nil, "en", "")

	v, err := SafetyJSON(ctx, cp, Score(KindFull, full(1, 1, 1, 1, 1, 1, 1, 1, 2, 0)))
	require.NoError(t, err)
	m := toMap(t, v)
	assert.Equal(t, LevelAdvice, m["level"])
	assert.Equal(t, []any{}, m["actions"], "advice has no call action")

	v, err = SafetyJSON(ctx, cp, Score(KindFull, full(0, 0, 0, 0, 0, 0, 0, 0, 0, 2)))
	require.NoError(t, err)
	m = toMap(t, v)
	assert.Equal(t, LevelUrgent, m["level"])
	assert.Len(t, m["actions"], 3)

	v, err = SafetyJSON(ctx, cp, Score(KindShort, map[string]int{"q3": 0, "q4": 1, "q5": 1}))
	require.NoError(t, err)
	assert.Nil(t, v)
}

// Every EPDS item has its text and four option labels in fa and en.
func TestEPDSCopyComplete(t *testing.T) {
	for _, locale := range []string{"fa", "en"} {
		for _, it := range Items {
			_, ok := translator().Get("postpartum.epds.items."+it.Code+".text", locale)
			assert.True(t, ok, "%s %s", locale, it.Code)
			assert.Len(t, optionLabels(it.Code, locale), 4, "%s %s", locale, it.Code)
		}
		for _, b := range []string{BandLow, BandElevated, BandPossible, BandLikely} {
			assert.NotEmpty(t, line("epds.bands."+b, locale), "%s %s", locale, b)
		}
		for _, p := range []string{guide.PhasePuerperium, guide.PhaseRecovery, guide.PhaseLate} {
			assert.NotEmpty(t, line("phases."+p, locale), "%s %s", locale, p)
		}
	}
}

func entry(cat, param, item, code, num string) taxonomy.Entry {
	e := taxonomy.Entry{Category: cat, Param: param, Item: item}
	if code != "" {
		e.Code = sql.NullString{String: code, Valid: true}
	}
	if num != "" {
		e.Num = sql.NullString{String: num, Valid: true}
	}
	return e
}

func TestRecoveryOf(t *testing.T) {
	r := RecoveryOf([]taxonomy.Entry{
		entry("bleeding", "lochia_amount", "", "medium", ""),
		entry("bleeding", "clot_size", "", "large", ""),
		entry("pain", "location", "back", "mild", ""),
		entry("pain", "location", "stitches", "severe", ""),
		entry("pain", "location", "pelvis", "severe", ""), // not a recovery location (pregnancy/cycle item)
		entry("breasts", "symptoms", "redness", "yes", ""),
		entry("breasts", "symptoms", "engorgement", "no", ""),
		entry("baby", "feeds_count", "", "", "7.00"),
		entry("sleep", "hours", "", "", "5.50"),
	})
	assert.Equal(t, "medium", r.LochiaAmount)
	assert.Equal(t, "severe", r.PainLevel, "the highest location level")
	assert.Equal(t, []string{"stitches", "back"}, r.PainLocations)
	assert.Equal(t, []string{"redness"}, r.Breasts)
	require.NotNil(t, r.FeedsCount)
	assert.Equal(t, 7, *r.FeedsCount)
	assert.InDelta(t, 5.5, *r.SleepHours, 0.001)
	assert.Equal(t, []string{guide.AlertLargeClots}, r.Alerts())

	none := RecoveryOf([]taxonomy.Entry{entry("pain", "none", "", "yes", "")})
	assert.Equal(t, "none", none.PainLevel)
	assert.Nil(t, none.Breasts, "not logged")
}

func TestRecoveryChanges(t *testing.T) {
	h, n := 9.0, 0
	in := RecoveryInput{
		Set:       map[string]bool{"pain_level": true, "breasts": true, "sleep_hours": true, "feeds_count": true, "lochia_color": true},
		PainLevel: "mild", PainLocations: []string{"abdomen"}, Breasts: []string{}, SleepHours: &h, FeedsCount: &n,
	}
	keys := map[string]int{}
	for _, ch := range in.Changes() {
		keys[ch.Key()] = len(ch.Entries)
	}
	assert.Equal(t, map[string]int{
		"pain.none": 0, "pain.location": 1, "breasts.symptoms": 3, "baby.feeds_count": 1,
		"sleep.hours": 1, "sleep.duration": 1, "bleeding.lochia_color": 0,
	}, keys)
	assert.Equal(t, "9_plus", sleepBucket(9))
	assert.Equal(t, "0_3", sleepBucket(2.5))
}
