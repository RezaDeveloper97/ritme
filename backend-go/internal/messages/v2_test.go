package messages

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
	pregv2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

func lmpProfile(lmp civildate.Date) *pstore.PregnancyProfile {
	return &pstore.PregnancyProfile{
		AgeSource: sql.NullString{String: "lmp", Valid: true},
		LmpDate:   civildate.NullDate{Date: lmp, Valid: true},
	}
}

// T-M9-03: pregnancy_week is the gestational week of the pregnancy screens (it used to be negative: DiffDays is
// other − d).
func TestPregnancyWeek_FromDating(t *testing.T) {
	today := civildate.MustParse("2026-10-21")

	week, ok := pregnancyWeek(lmpProfile(today.AddDays(-70)), today)
	require.True(t, ok)
	assert.InDelta(t, 11.0, week, 0)

	week, ok = pregnancyWeek(lmpProfile(today), today)
	require.True(t, ok)
	assert.InDelta(t, 1.0, week, 0)

	// The same number the pregnancy v2 screens show.
	p := lmpProfile(today.AddDays(-100))
	d, ok := pregv2.Resolve(p, today)
	require.True(t, ok)
	week, ok = pregnancyWeek(p, today)
	require.True(t, ok)
	assert.InDelta(t, float64(d.CurrentWeek()), week, 0)

	_, ok = pregnancyWeek(nil, today)
	assert.False(t, ok)
	_, ok = pregnancyWeek(&pstore.PregnancyProfile{AgeSource: sql.NullString{String: "lmp", Valid: true}}, today)
	assert.False(t, ok, "no LMP → no fact")
}

func TestPregnancyWeek_V2RuleBetweenMatches(t *testing.T) {
	today := civildate.MustParse("2026-10-21")
	week, ok := pregnancyWeek(lmpProfile(today.AddDays(-70)), today)
	require.True(t, ok)
	facts := rules.Facts{"mode": "pregnancy", "pregnancy_week": week}
	r := rules.Rule{Groups: []rules.Group{{Op: "and", Items: []rules.Condition{
		{Fact: "pregnancy_week", Op: "between", Value: []any{10.0, 12.0}},
	}}}}
	assert.True(t, rules.MatchRule(&r, facts))
}
