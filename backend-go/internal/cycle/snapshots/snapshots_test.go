package snapshots

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func day(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

func period(start, end string) engine.Period {
	return engine.Period{Start: day(start), End: day(end), Source: engine.PeriodUserLogged}
}

// A first start: no previous real cycle (no `last`), `first` = start + the default 28 days.
func TestPredictAround_FirstStart(t *testing.T) {
	out := PredictAround(engine.Input{Periods: []engine.Period{period("2026-07-01", "2026-07-05")}}, civildate.MustParse("2026-07-01"))
	assert.Nil(t, out.Last)
	require.NotNil(t, out.First)
	assert.Equal(t, "2026-07-01", out.First.CycleStart.String())
	assert.Equal(t, "2026-07-29", out.First.PredictedNext.String())
	assert.Equal(t, uint8(0), out.First.BasisCycles)
}

// The next real start: `last` of 07-01 is the 07-29 prediction (actual 08-02), `first` of 08-02 from the new state.
func TestPredictAround_NextStart(t *testing.T) {
	in := engine.Input{Periods: []engine.Period{period("2026-07-01", "2026-07-05"), {Start: day("2026-08-02"), End: engine.NoDay}}}
	out := PredictAround(in, civildate.MustParse("2026-08-02"))
	require.NotNil(t, out.Last)
	assert.Equal(t, "2026-07-01", out.Last.CycleStart.String())
	assert.Equal(t, "2026-07-29", out.Last.PredictedNext.String())
	assert.Equal(t, "2026-08-02", out.Actual.String())
	require.NotNil(t, out.First)
	assert.Equal(t, "2026-08-02", out.First.CycleStart.String())
	assert.Equal(t, uint8(1), out.First.BasisCycles)
}

// An old period (not the newest start) and a start that is not in the input write nothing.
func TestPredictAround_NotNewest(t *testing.T) {
	in := engine.Input{Periods: []engine.Period{period("2026-07-01", "2026-07-05"), period("2026-08-02", "2026-08-06")}}
	assert.Equal(t, Outcome{}, PredictAround(in, civildate.MustParse("2026-07-01")))
	assert.Equal(t, Outcome{}, PredictAround(in, civildate.MustParse("2026-07-15")))
	assert.Equal(t, Outcome{}, PredictAround(engine.Input{}, civildate.MustParse("2026-07-15")))
}

// The declared onboarding anchor is never a cycle_start: no `last` for it.
func TestPredictAround_DeclaredAnchorIsNotACycle(t *testing.T) {
	in := engine.Input{
		Declared: engine.Declared{LastPeriodStart: engine.Some(day("2026-07-01"))},
		Periods:  []engine.Period{{Start: day("2026-08-02"), End: engine.NoDay}},
	}
	out := PredictAround(in, civildate.MustParse("2026-08-02"))
	assert.Nil(t, out.Last)
	assert.NotNil(t, out.First)
}

func TestSnapshotRow_Codes(t *testing.T) {
	in := engine.Input{Periods: []engine.Period{period("2026-07-01", "2026-07-05")}}
	st := engine.Prepare(in, engineparams.For(civildate.MustParse("2026-08-02")))
	d := day("2026-08-02")
	s := st.Resolve(d, d)
	row := SnapshotRow(st, s)
	assert.Equal(t, s.MainPhase.String(), row.MainPhase)
	assert.Equal(t, s.Subphase.String(), row.Subphase)
	assert.Equal(t, s.Confidence.String(), row.Confidence)
	assert.Equal(t, s.DataQuality.String(), row.DataQuality)
	assert.Equal(t, uint32(s.Warnings), row.WarningsMask)
	assert.Equal(t, "cycle", row.Mode)
	late, ok := s.DaysLate.Get()
	assert.Equal(t, ok, row.DaysLate.Valid)
	assert.Equal(t, int16(late), row.DaysLate.Int16)
	assert.Equal(t, int16(4), row.DaysLate.Int16, "07-29 predicted, 08-02 is 4 days late")
}

// Tehran 01:00 on 10-28 is still 10-27 in Berlin; a missing or invalid zone is Tehran.
func TestGroupByLocalDay(t *testing.T) {
	now := time.Date(2026, 10, 28, 1, 0, 0, 0, civildate.Tehran)
	got := groupByLocalDay([]store.ListActiveSnapshotUsersRow{
		{UserID: 1},
		{UserID: 2, Timezone: sql.NullString{String: "Europe/Berlin", Valid: true}},
		{UserID: 3, Timezone: sql.NullString{String: "Not/AZone", Valid: true}},
	}, now)
	assert.Equal(t, map[civildate.Date][]uint64{
		civildate.MustParse("2026-10-28"): {1, 3},
		civildate.MustParse("2026-10-27"): {2},
	}, got)
}
