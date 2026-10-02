package postpartum

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func d(s string) civildate.Date {
	v, err := civildate.Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

func full(scores ...int) map[string]int {
	out := map[string]int{}
	for i, it := range Items {
		out[it.Code] = scores[i]
	}
	return out
}

func TestItems_ReverseScoredAsEPDS(t *testing.T) {
	require.Len(t, Items, 10)
	for _, it := range Items {
		switch it.Code {
		case "q1", "q2", "q4":
			assert.Equal(t, ascending, it.Scores, it.Code)
		default:
			assert.Equal(t, descending, it.Scores, it.Code)
		}
	}
	codes := []string{}
	for _, it := range ItemsOf(KindShort) {
		codes = append(codes, it.Code)
	}
	assert.Equal(t, []string{"q3", "q4", "q5"}, codes, "EPDS-3 anxiety subscale")
}

func TestScore_Full(t *testing.T) {
	cases := []struct {
		name     string
		scores   []int
		total    int
		band     string
		urgent   bool
		reasons  []string
		selfHarm int
	}{
		{"all zero", []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 0, BandLow, false, nil, 0},
		{"9 low", []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 0}, 9, BandLow, false, nil, 0},
		{"10 possible", []int{1, 1, 1, 1, 1, 1, 1, 1, 2, 0}, 10, BandPossible, false, nil, 0},
		{"12 possible", []int{2, 2, 1, 1, 1, 1, 1, 1, 2, 0}, 12, BandPossible, false, nil, 0},
		{"13 likely urgent", []int{2, 2, 2, 1, 1, 1, 1, 1, 2, 0}, 13, BandLikely, true, []string{ReasonScore}, 0},
		{"q10 any positive is urgent even at a low total", []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, 1, BandLow, true, []string{ReasonSelfHarm}, 1},
		{"both", []int{3, 3, 3, 3, 3, 3, 3, 3, 3, 3}, 30, BandLikely, true, []string{ReasonSelfHarm, ReasonScore}, 3},
	}
	for _, c := range cases {
		r := Score(KindFull, full(c.scores...))
		assert.Equal(t, c.total, r.Total, c.name)
		assert.Equal(t, 30, r.Max, c.name)
		assert.Equal(t, c.band, r.Band, c.name)
		assert.Equal(t, c.urgent, r.Urgent, c.name)
		assert.Equal(t, c.reasons, r.Reasons, c.name)
		require.NotNil(t, r.SelfHarm, c.name)
		assert.Equal(t, c.selfHarm, *r.SelfHarm, c.name)
		assert.Empty(t, r.FollowUp, c.name)
	}
}

func TestScore_ShortNeverUrgentButSuggestsTheFullCheck(t *testing.T) {
	r := Score(KindShort, map[string]int{"q3": 2, "q4": 2, "q5": 1, "q10": 3})
	assert.Equal(t, 5, r.Total, "only the EPDS-3 items count")
	assert.Equal(t, 9, r.Max)
	assert.Equal(t, BandLow, r.Band)
	assert.False(t, r.Urgent)
	assert.Nil(t, r.SelfHarm)
	assert.Empty(t, r.FollowUp)
	assert.NotContains(t, r.Answers, "q10")

	r = Score(KindShort, map[string]int{"q3": 2, "q4": 2, "q5": 2})
	assert.Equal(t, 6, r.Total)
	assert.Equal(t, BandElevated, r.Band)
	assert.Equal(t, KindFull, r.FollowUp)
	assert.False(t, r.Urgent)
}

func TestScore_ClampsOutOfRange(t *testing.T) {
	r := Score(KindShort, map[string]int{"q3": 9, "q4": -2, "q5": 1})
	assert.Equal(t, 4, r.Total)
}

func TestCheckBand(t *testing.T) {
	assert.Equal(t, BandElevated, Check{Kind: KindShort, Total: 6}.Band())
	assert.Equal(t, BandLow, Check{Kind: KindShort, Total: 5}.Band())
	assert.Equal(t, BandPossible, Check{Kind: KindFull, Total: 11}.Band())
	assert.Equal(t, BandLikely, Check{Kind: KindFull, Total: 13}.Band())
	assert.Equal(t, 9, Check{Kind: KindShort}.Max())
	assert.Equal(t, 30, Check{Kind: KindFull}.Max())
}

func TestScheduleOn(t *testing.T) {
	birth := d("2026-09-01")
	chk := func(kind, on string, total int) *Check { return &Check{Kind: kind, TakenOn: d(on), Total: total} }

	// First two weeks: the short check only.
	s := ScheduleOn(d("2026-09-05"), birth, nil, nil)
	assert.Equal(t, Schedule{Due: KindShort, NextDueOn: d("2026-09-05")}, s)
	short := chk(KindShort, "2026-09-05", 2)
	s = ScheduleOn(d("2026-09-08"), birth, short, nil)
	assert.Equal(t, Schedule{NextDueOn: d("2026-09-12")}, s)
	s = ScheduleOn(d("2026-09-12"), birth, short, nil)
	assert.Equal(t, KindShort, s.Due)

	// From day 14 the full check (never taken) is due.
	s = ScheduleOn(d("2026-09-15"), birth, chk(KindShort, "2026-09-12", 1), nil)
	assert.Equal(t, KindFull, s.Due)

	// After a full check: weekly short in between, the next full 14 days later.
	f := chk(KindFull, "2026-09-15", 4)
	s = ScheduleOn(d("2026-09-16"), birth, f, f)
	assert.Equal(t, Schedule{NextDueOn: d("2026-09-22")}, s)
	s = ScheduleOn(d("2026-09-22"), birth, f, f)
	assert.Equal(t, KindShort, s.Due)
	s = ScheduleOn(d("2026-09-29"), birth, chk(KindShort, "2026-09-22", 1), f)
	assert.Equal(t, KindFull, s.Due)

	// An elevated short check makes the full one due right away.
	s = ScheduleOn(d("2026-09-23"), birth, chk(KindShort, "2026-09-22", ShortElevatedMin), f)
	assert.Equal(t, Schedule{Due: KindFull, NextDueOn: d("2026-09-23")}, s)
}
