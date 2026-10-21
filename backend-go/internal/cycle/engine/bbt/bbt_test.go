package bbt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/cycle/engine/bbt"
)

type day int32

var strict = bbt.Params{Strict: true, MinReadings: 10, RiseCenti: 20}

var standard = bbt.Params{Strict: false, MinReadings: 10, RiseCenti: 20}

// series places readings on consecutive days from cycle day 1 (day value 1); 0 = no reading that day.
func series(values ...int32) []bbt.Reading[day] {
	var out []bbt.Reading[day]
	for i, v := range values {
		if v != 0 {
			out = append(out, bbt.Reading[day]{Date: day(i + 1), Centi: v})
		}
	}
	return out
}

// onDays places readings from cycle day `from` on.
func onDays(from int, values ...int32) []bbt.Reading[day] {
	lead := make([]int32, from-1)
	return series(append(lead, values...)...)
}

func TestDetect_TaskExample(t *testing.T) {
	// Baseline days 8–13 ≤ 36.50, highs 36.62/36.65/36.72 on days 14–16, a 10th reading on day 17.
	rs := onDays(8, 3640, 3645, 3650, 3630, 3648, 3650, 3662, 3665, 3672, 3670)
	for _, p := range []bbt.Params{strict, standard} {
		r := bbt.Detect(rs, p)
		assert.True(t, r.OK)
		assert.Equal(t, day(13), r.Ovulation, "the day before the first high reading")
		assert.Equal(t, 6, r.First)
		assert.Equal(t, 8, r.Confirm)
		assert.Equal(t, int32(3650), r.Baseline)
	}

	// Only 9 readings in the cycle: the shift exists but the cycle has too few readings.
	assert.False(t, bbt.Detect(rs[:9], strict).OK)
	assert.False(t, bbt.Detect(rs[:9], standard).OK)
}

func TestDetect_StrictVsStandard(t *testing.T) {
	// 36.51 / 36.52 / 36.60 over a 36.50 maximum, then lower readings.
	weak := onDays(1, 3640, 3645, 3650, 3630, 3648, 3650, 3651, 3652, 3660, 3600, 3600)
	assert.True(t, bbt.Detect(weak, standard).OK)
	assert.Equal(t, day(6), bbt.Detect(weak, standard).Ovulation)
	assert.False(t, bbt.Detect(weak, strict).OK, "36.60 < 36.50 + 0.20")

	// 36.55 / 36.60 / 36.72 over 36.50 → both (36.72 ≥ 36.70).
	strong := onDays(1, 3640, 3645, 3650, 3630, 3648, 3650, 3655, 3660, 3672, 3600)
	assert.True(t, bbt.Detect(strong, standard).OK)
	assert.True(t, bbt.Detect(strong, strict).OK)

	// Exactly +0.20 on the third reading confirms (≥).
	edge := onDays(1, 3640, 3645, 3650, 3630, 3648, 3650, 3655, 3660, 3670, 3600)
	assert.True(t, bbt.Detect(edge, strict).OK)
	// A reading equal to the baseline is not "above" it.
	equal := onDays(1, 3640, 3645, 3650, 3630, 3648, 3650, 3650, 3680, 3690, 3695, 3600)
	r := bbt.Detect(equal, standard)
	assert.True(t, r.OK)
	assert.Equal(t, day(7), r.Ovulation, "the run starts at the reading after the equal one")
}

func TestDetect_RunIsReadingsNotDays(t *testing.T) {
	// A missing day (day 8) inside the high run still confirms.
	gap := series(3640, 3645, 3650, 3630, 3648, 3650, 3672, 0, 3675, 3680, 3678)
	r := bbt.Detect(gap, strict)
	assert.True(t, r.OK)
	assert.Equal(t, day(6), r.Ovulation)
	assert.Equal(t, 8, r.Confirm)

	// A gap before the first high reading: ovulation is the calendar day before it, not the previous reading.
	before := series(3640, 3645, 3650, 3630, 3648, 3650, 0, 0, 3672, 3675, 3680, 3678)
	assert.Equal(t, day(8), bbt.Detect(before, strict).Ovulation)

	// A high run of only 2 readings does not confirm.
	two := series(3640, 3645, 3650, 3630, 3648, 3650, 3672, 3675, 3640, 3645, 3630, 3640)
	assert.False(t, bbt.Detect(two, strict).OK)
	assert.False(t, bbt.Detect(two, standard).OK)
}

func TestDetect_FirstShiftAndSlidingBaseline(t *testing.T) {
	// A spike on day 3 lifts the baseline of the first windows; the shift is found once it slides out.
	rs := series(3640, 3630, 3700, 3630, 3640, 3635, 3630, 3640, 3635, 3638, 3660, 3665, 3670, 3600)
	r := bbt.Detect(rs, standard)
	assert.True(t, r.OK)
	assert.Equal(t, day(10), r.Ovulation)
	assert.Equal(t, int32(3640), r.Baseline)

	// Two shifts: the first one wins.
	two := series(3600, 3600, 3600, 3600, 3600, 3600, 3630, 3630, 3630, 3700, 3700, 3700, 3700)
	assert.Equal(t, day(6), bbt.Detect(two, strict).Ovulation)
}

func TestDetect_Degenerate(t *testing.T) {
	assert.False(t, bbt.Detect[day](nil, strict).OK)
	// MinReadings below 9 still needs 6 + 3 readings.
	p := bbt.Params{MinReadings: 0}
	assert.False(t, bbt.Detect(series(1, 1, 1, 1, 1, 1, 2, 2), p).OK)
	assert.True(t, bbt.Detect(series(1, 1, 1, 1, 1, 1, 2, 2, 2), p).OK)
	// Flat readings never confirm.
	assert.False(t, bbt.Detect(series(3650, 3650, 3650, 3650, 3650, 3650, 3650, 3650, 3650, 3650, 3650), standard).OK)
	// The zero result is all zero.
	assert.Equal(t, bbt.Result[day]{}, bbt.Detect(series(3650), standard))
}

func TestDetect_NoAllocs(t *testing.T) {
	rs := onDays(8, 3640, 3645, 3650, 3630, 3648, 3650, 3662, 3665, 3672, 3670)
	assert.Zero(t, testing.AllocsPerRun(100, func() { _ = bbt.Detect(rs, strict) }))
}

// FuzzBBTDetect: random readings never panic, never confirm with fewer than MinReadings readings, and a confirmed
// shift always satisfies the rule it reports.
func FuzzBBTDetect(f *testing.F) {
	f.Add([]byte{40, 45, 50, 30, 48, 50, 62, 65, 72, 70}, true, uint8(10), uint8(20))
	f.Add([]byte{40, 45, 50, 30, 48, 50, 51, 52, 60, 0, 0}, false, uint8(10), uint8(20))
	f.Add([]byte{}, true, uint8(0), uint8(0))
	f.Fuzz(func(t *testing.T, raw []byte, isStrict bool, minReadings, rise uint8) {
		p := bbt.Params{Strict: isStrict, MinReadings: int(minReadings % 32), RiseCenti: int32(rise % 64)}
		rs := make([]bbt.Reading[day], 0, len(raw))
		d := day(0)
		for _, b := range raw {
			d += day(1 + b%2) // occasional missing days
			rs = append(rs, bbt.Reading[day]{Date: d, Centi: 3600 + int32(b)})
		}
		r := bbt.Detect(rs, p)
		if !r.OK {
			return
		}
		if len(rs) < p.MinReadings || len(rs) < bbt.Baseline+bbt.Run {
			t.Fatalf("confirmed with %d readings (min %d)", len(rs), p.MinReadings)
		}
		if r.First < bbt.Baseline || r.Confirm != r.First+bbt.Run-1 || r.Confirm >= len(rs) {
			t.Fatalf("bad indexes %+v", r)
		}
		base := int32(0)
		for _, x := range rs[r.First-bbt.Baseline : r.First] {
			base = max(base, x.Centi)
		}
		if base != r.Baseline || r.Ovulation != rs[r.First].Date-1 {
			t.Fatalf("bad result %+v (baseline %d)", r, base)
		}
		for _, x := range rs[r.First : r.Confirm+1] {
			if x.Centi <= base {
				t.Fatalf("reading %d not above %d", x.Centi, base)
			}
		}
		if p.Strict && rs[r.Confirm].Centi < base+p.RiseCenti {
			t.Fatalf("strict rise not met: %+v", r)
		}
	})
}
