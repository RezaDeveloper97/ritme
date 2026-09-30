package insights

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

var d0 = civildate.MustParse("2026-01-01")

// periods builds confirmed, closed periods of bleed days from consecutive cycle lengths; the last
// start is followed by no length (the current cycle).
func periods(bleed int, lengths ...int) []model.History {
	out := []model.History{}
	start := d0
	for i := 0; i <= len(lengths); i++ {
		out = append(out, model.History{
			ID: int64(i + 1), PeriodStart: start, PeriodEnd: start.AddDays(bleed - 1),
			IsConfirmed: true, Source: "user_logged",
		})
		if i < len(lengths) {
			start = start.AddDays(lengths[i])
		}
	}
	return out
}

func lastStart(h []model.History) civildate.Date { return h[len(h)-1].PeriodStart }

func TestBuildHistory_EmptyWithoutPeriods(t *testing.T) {
	h := BuildHistory(nil, nil, d0)
	assert.Equal(t, RegularityNotEnoughData, h.Regularity)
	assert.Empty(t, h.Cycles)
	assert.Nil(t, h.MedianCycle)
	assert.Equal(t, 28, h.Predicted)
}

func TestBuildHistory_DesignExample(t *testing.T) {
	// The artboard: 38, 29, 29, 28, 30 (oldest → newest) then the current cycle on day 25.
	hs := periods(5, 38, 29, 29, 28, 30)
	today := lastStart(hs).AddDays(24)
	h := BuildHistory(hs, nil, today)

	require.NotNil(t, h.MedianCycle)
	assert.Equal(t, 29, *h.MedianCycle)
	assert.Equal(t, 2, *h.Spread)
	assert.Equal(t, 5, *h.MedianPeriod)
	assert.Equal(t, 5, h.Total)
	assert.Equal(t, 4, h.InRange)
	assert.Equal(t, RegularityRegular, h.Regularity)

	require.Len(t, h.Cycles, 6)
	cur := h.Cycles[0]
	assert.True(t, cur.IsCurrent)
	assert.Equal(t, 25, cur.Length)
	assert.Nil(t, cur.InRange)
	assert.True(t, cur.End.IsZero())
	// Newest completed first; the oldest 38-day cycle is out of range.
	assert.Equal(t, 30, h.Cycles[1].Length)
	assert.True(t, *h.Cycles[1].InRange)
	assert.Equal(t, 38, h.Cycles[5].Length)
	assert.False(t, *h.Cycles[5].InRange)
	assert.Equal(t, h.Cycles[1].Start.AddDays(29), h.Cycles[1].End)
}

func TestBuildHistory_IrregularAndWindow(t *testing.T) {
	hs := periods(5, 30, 30, 24, 35, 29, 41, 26, 33)
	h := BuildHistory(hs, nil, lastStart(hs).AddDays(3))
	assert.Equal(t, HistoryWindow, h.Total, "only the newest six completed cycles count")
	assert.Len(t, h.Cycles, HistoryWindow+1)
	assert.Equal(t, RegularityIrregular, h.Regularity)
}

func TestBuildHistory_OpenCurrentPeriodIsCapped(t *testing.T) {
	hs := periods(5, 28, 28)
	hs[len(hs)-1].PeriodEnd = civildate.Date{}
	h := BuildHistory(hs, nil, lastStart(hs).AddDays(1))
	assert.True(t, h.Cycles[0].PeriodOngoing)
	assert.Equal(t, 2, h.Cycles[0].PeriodDays)

	h = BuildHistory(hs, nil, lastStart(hs).AddDays(12))
	assert.Equal(t, 5, h.Cycles[0].PeriodDays, "an open period never paints past the usual length")
}

func TestBuildHistory_IgnoresEstimatesUnconfirmedAndFuture(t *testing.T) {
	hs := periods(5, 28)
	hs = append(hs,
		model.History{ID: 90, PeriodStart: d0.AddDays(-40), IsConfirmed: false},
		model.History{ID: 91, PeriodStart: d0.AddDays(-80), IsConfirmed: true, IsEstimated: true},
		model.History{ID: 92, PeriodStart: d0.AddDays(200), IsConfirmed: true},
	)
	h := BuildHistory(hs, nil, d0.AddDays(30))
	assert.Equal(t, 1, h.Total)
	assert.Equal(t, RegularityNotEnoughData, h.Regularity)
}

func TestHistoryJSONShape(t *testing.T) {
	hs := periods(5, 28, 29)
	b, err := jsonx.Marshal(BuildHistory(hs, nil, lastStart(hs).AddDays(4)).JSON(), 0)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"cycle_length":{"median":29,"spread_days":2,"range":{"min":27,"max":31},"predicted":29}`)
	assert.Contains(t, s, `"regularity":{"status":"regular","in_range":2,"total":2}`)
	assert.Contains(t, s, `"is_current":true,"in_range":null`)
}

func TestSourceDay(t *testing.T) {
	// Luteal tail lines up with the end, whatever the cycle length.
	assert.Equal(t, 29, sourceDay(27, 28, 30))
	assert.Equal(t, 16, sourceDay(14, 28, 30))
	// Day 0 is always day 0; the head stretches proportionally.
	assert.Equal(t, 0, sourceDay(0, 28, 35))
	assert.Equal(t, 20, sourceDay(13, 28, 35))
	for n := validCycleMin; n <= validCycleMax; n++ {
		for typical := validCycleMin; typical <= validCycleMax; typical++ {
			for tt := range typical {
				s := sourceDay(tt, typical, n)
				require.GreaterOrEqual(t, s, 0)
				require.Less(t, s, n)
			}
		}
	}
}

func TestBuildPattern_NeedsThreeCycles(t *testing.T) {
	hs := periods(5, 28, 28)
	p := BuildPattern(hs, nil, nil, lastStart(hs).AddDays(3))
	assert.False(t, p.Ready)
	assert.Equal(t, 2, p.CyclesCounted)
	assert.Empty(t, p.Groups["symptoms"])
	b, err := jsonx.Marshal(p.JSON(), 0)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"groups":[{"key":"symptoms","items":[]},{"key":"mood","items":[]},{"key":"pain","items":[]}]`)
}

func TestBuildPattern_BreastTendernessBeforePeriod(t *testing.T) {
	hs := periods(5, 28, 28, 28, 28)
	logs := map[civildate.Date]DayLog{}
	// Breast tenderness on the last 4 days of every completed cycle (days 25–28), cramps on day 1.
	for i := 0; i+1 < len(hs); i++ {
		next := hs[i+1].PeriodStart
		for k := 1; k <= 4; k++ {
			logs[next.AddDays(-k)] = DayLog{"symptoms.breast_tenderness": 1}
		}
		logs[hs[i].PeriodStart] = DayLog{"symptoms.cramps": 2.0 / 3, "mood.sensitive": 1}
	}
	p := BuildPattern(hs, nil, logs, lastStart(hs).AddDays(2))
	require.True(t, p.Ready)
	assert.Equal(t, 28, p.CycleLength)
	assert.Equal(t, 14, p.OvulationDay)

	items := p.Groups["symptoms"]
	require.Len(t, items, 2)
	var bt SymptomPattern
	for _, it := range items {
		if it.Key == "breast_tenderness" {
			bt = it
		}
	}
	assert.Equal(t, 4, bt.Cycles)
	require.Len(t, bt.Strip, 28)
	assert.InDelta(t, 1.0, bt.Strip[27], 0.001)
	assert.InDelta(t, 0.0, bt.Strip[10], 0.001)
	require.NotNil(t, bt.Window)
	assert.Equal(t, RelationBeforePeriod, bt.Window.Relation)
	assert.Equal(t, 4, bt.Window.Days)

	cramps := items[0]
	if cramps.Key != "cramps" {
		cramps = items[1]
	}
	assert.InDelta(t, 0.67, cramps.Strip[0], 0.001)
	assert.Equal(t, RelationEarly, cramps.Window.Relation)
	assert.Len(t, p.Groups["mood"], 1)
	assert.Empty(t, p.Groups["pain"])
}
