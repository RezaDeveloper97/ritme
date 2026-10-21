package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bbtFrom places readings on consecutive cycle days from cycle day `from` of the cycle starting on start; 0 = none.
func bbtFrom(start Day, from int, values ...int32) []BBTReading {
	var out []BBTReading
	for i, v := range values {
		if v != 0 {
			out = append(out, BBTReading{Date: start + Day(from-1+i), Centi: v})
		}
	}
	return out
}

// shiftAt is a clean strict 3-over-6 shift with ovulation on day ov: 6 baseline readings ov−5..ov (max 36.50), 3 high
// readings ov+1..ov+3 and a 10th reading.
func shiftAt(ov Day) []BBTReading {
	vals := []int32{3640, 3645, 3650, 3630, 3648, 3650, 3672, 3675, 3680, 3678}
	out := make([]BBTReading, len(vals))
	for i, v := range vals {
		out[i] = BBTReading{Date: ov - 5 + Day(i), Centi: v}
	}
	return out
}

func lhOn(d Day, positive bool) LHTest { return LHTest{Date: d, Positive: positive} }

const (
	julyStart = "2026-07-01"
)

// current is a regular 28-day history whose current cycle starts on 2026-07-01, plus markers.
func current(bbt []BBTReading, lh ...LHTest) Input {
	in := regular(day(julyStart), 28, 5)
	in.BBT, in.LH = bbt, lh
	return in
}

// taskBBT is the T-M9-17 example: 6 baseline readings ≤ 36.50 on days 8–13, 36.62/36.65/36.72 on days 14–16 and a
// 10th reading on day 17.
func taskBBT(cur Day) []BBTReading {
	return bbtFrom(cur, 8, 3640, 3645, 3650, 3630, 3648, 3650, 3662, 3665, 3672, 3670)
}

func TestOvulation_BBTConfirmed(t *testing.T) {
	cur := day(julyStart)
	s := prep(current(taskBBT(cur)))
	st := s.Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationBBTConfirmed, st.Anchors.OvulationSource)
	assert.Equal(t, day("2026-07-13"), st.Anchors.EstimatedOvulationDate, "day 13")
	assert.Equal(t, day("2026-07-27"), st.Anchors.PredictedNextPeriodStart, "O + default luteal 14")
	assert.Equal(t, Some(-8), st.DaysToOvulation)
	assert.False(t, st.Warnings.Has(WarnLHBBTConflict))

	o := s.Ovulations()[len(s.Ovulations())-1]
	assert.Equal(t, CycleOvulation{Date: day("2026-07-13"), Source: OvulationBBTConfirmed, BBT: day("2026-07-13"),
		LH: NoDay}, o)

	// Only 9 readings in the cycle → not confirmed → calendar (next − 14).
	nine := taskBBT(cur)[:9]
	st = prep(current(nine)).Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationCalendar, st.Anchors.OvulationSource)
	assert.Equal(t, cur+28, st.Anchors.PredictedNextPeriodStart)
	assert.Equal(t, cur+14, st.Anchors.EstimatedOvulationDate)

	// … → LH when a positive test exists.
	st = prep(current(nine, lhOn(cur+11, true))).Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationLHPositive, st.Anchors.OvulationSource)
	assert.Equal(t, cur+12, st.Anchors.EstimatedOvulationDate)
}

func TestOvulation_StrictVsStandard(t *testing.T) {
	cur := day(julyStart)
	standard := DefaultParams()
	standard.BBT.Strict = false

	weak := bbtFrom(cur, 8, 3640, 3645, 3650, 3630, 3648, 3650, 3651, 3652, 3660, 3600)
	assert.Equal(t, OvulationBBTConfirmed, Prepare(current(weak), standard).Resolve(cur+20, cur+20).Anchors.OvulationSource)
	assert.Equal(t, OvulationCalendar, prep(current(weak)).Resolve(cur+20, cur+20).Anchors.OvulationSource)

	strong := bbtFrom(cur, 8, 3640, 3645, 3650, 3630, 3648, 3650, 3655, 3660, 3672, 3600)
	assert.Equal(t, OvulationBBTConfirmed, Prepare(current(strong), standard).Resolve(cur+20, cur+20).Anchors.OvulationSource)
	assert.Equal(t, OvulationBBTConfirmed, prep(current(strong)).Resolve(cur+20, cur+20).Anchors.OvulationSource)
}

func TestOvulation_RunOfReadings(t *testing.T) {
	cur := day(julyStart)
	// A missing day (day 15) inside the high run still confirms.
	gap := bbtFrom(cur, 8, 3640, 3645, 3650, 3630, 3648, 3650, 3662, 0, 3665, 3672, 3670)
	st := prep(current(gap)).Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationBBTConfirmed, st.Anchors.OvulationSource)
	assert.Equal(t, cur+12, st.Anchors.EstimatedOvulationDate)

	// A high run of only 2 readings does not.
	two := bbtFrom(cur, 8, 3640, 3645, 3650, 3630, 3648, 3650, 3672, 3675, 3640, 3645, 3630)
	assert.Equal(t, OvulationCalendar, prep(current(two)).Resolve(cur+20, cur+20).Anchors.OvulationSource)
}

func TestOvulation_LH(t *testing.T) {
	cur := day(julyStart)
	s := prep(current(nil, lhOn(cur+13, true), lhOn(cur+11, true), lhOn(cur+9, false)))
	st := s.Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationLHPositive, st.Anchors.OvulationSource)
	assert.Equal(t, day("2026-07-13"), st.Anchors.EstimatedOvulationDate, "first positive on day 12 → day 13")
	assert.Equal(t, day("2026-07-27"), st.Anchors.PredictedNextPeriodStart)

	// A faint result (loaded as not positive) alone → calendar.
	st = prep(current(nil, lhOn(cur+11, false))).Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationCalendar, st.Anchors.OvulationSource)

	// A positive test on a menstrual day still counts (scope 7): O = day 3, the display window is empty.
	s = prep(current(nil, lhOn(cur+1, true)))
	st = s.Resolve(cur+2, cur+2)
	assert.Equal(t, OvulationLHPositive, st.Anchors.OvulationSource)
	assert.Equal(t, cur+2, st.Anchors.EstimatedOvulationDate)
	assert.Equal(t, SubphaseMenstrual, st.Subphase, "menstrual overrides fertile")
	assert.Equal(t, cur+16, st.Anchors.PredictedNextPeriodStart)
	for d := cur; d < cur+16; d++ {
		assert.NotEqual(t, MainPhaseFertile, s.Resolve(d, d).MainPhase, "%s", d)
	}

	// Tests before the cycle start belong to the previous cycle.
	st = prep(current(nil, lhOn(cur-3, true))).Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationCalendar, st.Anchors.OvulationSource)
}

func TestOvulation_Conflict(t *testing.T) {
	cur := day(julyStart)
	bbtO := cur + 13 // day 14
	cases := []struct {
		name     string
		lhDay    Day // the positive test
		conflict bool
	}{
		{"LH O = 11, BBT O = 14 → BBT + lh_bbt_conflict", cur + 9, true},
		{"LH O = 12, BBT O = 14 → BBT, no warning", cur + 10, false},
		{"LH O = 16, BBT O = 14 → BBT, no warning", cur + 14, false},
		{"LH O = 17, BBT O = 14 → BBT + lh_bbt_conflict", cur + 15, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := prep(current(shiftAt(bbtO), lhOn(c.lhDay, true)))
			for _, d := range []Day{cur, cur + 13, cur + 25} { // every target of that cycle
				st := s.Resolve(d, d)
				assert.Equal(t, OvulationBBTConfirmed, st.Anchors.OvulationSource)
				assert.Equal(t, bbtO, st.Anchors.EstimatedOvulationDate)
				assert.Equal(t, c.conflict, st.Warnings.Has(WarnLHBBTConflict))
			}
			// The previous cycle carries no warning.
			assert.False(t, s.Resolve(cur-10, cur).Warnings.Has(WarnLHBBTConflict))
		})
	}

	// The threshold is clinical config.
	p := DefaultParams()
	p.LHBBTConflictDays = 3
	st := Prepare(current(shiftAt(bbtO), lhOn(cur+9, true)), p).Resolve(cur, cur)
	assert.False(t, st.Warnings.Has(WarnLHBBTConflict))
}

// lutealHistory is a 28-day history whose completed cycles have BBT ovulations with the given luteal lengths
// (oldest first; 0 = no BBT in that cycle), the current cycle starting on 2026-07-01.
func lutealHistory(luteal ...int) Input {
	cur := day(julyStart)
	lengths := make([]int, len(luteal))
	for i := range lengths {
		lengths[i] = 28
	}
	in := withPeriods(periodsEnding(cur, 5, lengths...))
	for i, l := range luteal {
		if l > 0 {
			next := in.Periods[i+1].Start
			in.BBT = append(in.BBT, shiftAt(next-Day(l))...)
		}
	}
	return in
}

func TestOvulation_EffectiveLuteal(t *testing.T) {
	cases := []struct {
		name    string
		luteal  []int
		want    int
		fromHis bool
	}{
		{"[12, 13] → 13 (half up)", []int{12, 13}, 13, true},
		{"[9, 17] → both invalid → 14", []int{9, 17}, 14, false},
		{"[11, 12, 15] → 12", []int{11, 12, 15}, 12, true},
		{"only the last 3 valid count", []int{16, 16, 11, 12, 15}, 12, true},
		{"invalid ones are skipped, not counted", []int{16, 9, 17}, 16, true},
		{"bounds 10 and 16 are valid", []int{10, 16}, 13, true},
		{"cycles without BBT contribute nothing", []int{0, 13, 0}, 13, true},
		{"none", nil, 14, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, src := prep(lutealHistory(c.luteal...)).LutealLength()
			assert.Equal(t, c.want, n)
			assert.Equal(t, c.fromHis, src == LengthHistory)
		})
	}

	// LH-only cycles do not contribute (E-16).
	in := lutealHistory(0, 0)
	in.LH = []LHTest{lhOn(in.Periods[0].Start+14, true), lhOn(in.Periods[1].Start+14, true)}
	n, src := prep(in).LutealLength()
	assert.Equal(t, 14, n)
	assert.Equal(t, LengthDefault, src)

	// Skipped cycles (E-15) do not contribute.
	in = lutealHistory(12, 13)
	in.Skips = []Day{in.Periods[1].Start + 5}
	n, _ = prep(in).LutealLength()
	assert.Equal(t, 12, n)

	// Luteal bounds and default are clinical config.
	p := DefaultParams()
	p.Luteal = LutealParams{Default: 13, Min: 12, Max: 14}
	n, src = Prepare(lutealHistory(11, 15), p).LutealLength()
	assert.Equal(t, 13, n)
	assert.Equal(t, LengthDefault, src)
}

func TestOvulation_NextFromLuteal(t *testing.T) {
	cur := day(julyStart)

	// BBT O 07-14 + luteal 13 → next 07-27; period_expected from 07-27.
	in := lutealHistory(13)
	in.BBT = append(in.BBT, shiftAt(day("2026-07-14"))...)
	s := prep(in)
	st := s.Resolve(cur+20, cur+20)
	assert.Equal(t, day("2026-07-27"), st.Anchors.PredictedNextPeriodStart)
	assert.Equal(t, Some(6), st.DaysToPeriod)
	assert.Equal(t, SubphasePMSPossible, s.Resolve(day("2026-07-26"), day("2026-07-26")).Subphase)
	pe := s.Resolve(day("2026-07-27"), day("2026-07-27"))
	assert.Equal(t, SubphasePeriodExpected, pe.Subphase)
	assert.Equal(t, Some(0), pe.DaysLate)
	assert.Equal(t, Some(1), s.Resolve(day("2026-07-28"), day("2026-07-28")).DaysLate)
	// Fertile window O−5..O with the BBT O.
	assert.Equal(t, SubphaseFertileRising, s.Resolve(day("2026-07-09"), day("2026-07-09")).Subphase)
	assert.Equal(t, SubphaseOvulationLikely, s.Resolve(day("2026-07-14"), day("2026-07-14")).Subphase)
	assert.Equal(t, SubphaseLateFollicularTransition, s.Resolve(day("2026-07-08"), day("2026-07-08")).Subphase)

	// LH O 07-14 (no BBT) + default 14 → 07-28; the luteal history comes from BBT cycles only.
	in = regular(cur, 28, 5)
	in.LH = []LHTest{lhOn(day("2026-07-13"), true)}
	st = prep(in).Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationLHPositive, st.Anchors.OvulationSource)
	assert.Equal(t, day("2026-07-28"), st.Anchors.PredictedNextPeriodStart)

	// LH in the current cycle uses the BBT luteal history.
	in = lutealHistory(12)
	in.LH = []LHTest{lhOn(day("2026-07-13"), true)}
	assert.Equal(t, day("2026-07-26"), prep(in).Resolve(cur+20, cur+20).Anchors.PredictedNextPeriodStart)
}

// Luteal 10 with BBT: assignable range O+2..O+3 → 1 early + 1 mid; late/pms follow the moved next.
func TestOvulation_ShortLutealLayout(t *testing.T) {
	cur := day(julyStart)
	o := cur + 13
	in := lutealHistory(10)
	in.BBT = append(in.BBT, shiftAt(o)...)
	s := prep(in)
	require.Equal(t, o+10, s.Resolve(o, o).Anchors.PredictedNextPeriodStart)
	want := []Subphase{
		SubphaseOvulationLikely, SubphasePostOvulation, SubphaseEarlyLuteal, SubphaseMidLuteal,
		SubphaseLateLuteal, SubphaseLateLuteal, SubphaseLateLuteal,
		SubphasePMSPossible, SubphasePMSPossible, SubphasePMSPossible, SubphasePeriodExpected,
	}
	for i, w := range want {
		d := o + Day(i)
		assert.Equal(t, w.String(), s.Resolve(d, d).Subphase.String(), "O+%d", i)
	}
}

// E-22: a completed past cycle with a BBT ovulation is laid out around it, up to its actual next start.
func TestOvulation_CompletedCycle(t *testing.T) {
	cur := day(julyStart)
	in := regular(cur, 28, 5)
	prev := in.Periods[1].Start // the completed cycle before the current one
	in.BBT = shiftAt(prev + 12) // O on day 13, luteal 16
	in.LH = []LHTest{lhOn(prev+15, true)}
	s := prep(in)

	st := s.Resolve(prev+12, cur+5)
	assert.Equal(t, SubphaseOvulationLikely, st.Subphase)
	assert.Equal(t, OvulationBBTConfirmed, st.Anchors.OvulationSource)
	assert.Equal(t, cur, st.Anchors.PredictedNextPeriodStart, "the actual next start, not O + luteal")
	assert.True(t, st.Warnings.Has(WarnLHBBTConflict), "LH O = 17, BBT O = 13")
	assert.Equal(t, SubphasePostOvulation, s.Resolve(prev+13, cur).Subphase)
	assert.Equal(t, SubphasePMSPossible, s.Resolve(cur-1, cur).Subphase)
	for d := prev; d < cur; d++ {
		assert.NotEqual(t, MainPhasePeriodExpected, s.Resolve(d, cur).MainPhase)
	}

	// Its luteal length (16) is effective_luteal_length, but a calendar ovulation stays next − 14 (§2.17).
	st = s.Resolve(cur+3, cur+3)
	assert.Equal(t, OvulationCalendar, st.Anchors.OvulationSource)
	assert.Equal(t, cur+14, st.Anchors.EstimatedOvulationDate)
	assert.False(t, st.Warnings.Has(WarnLHBBTConflict))
	n, _ := s.LutealLength()
	assert.Equal(t, 16, n)
}

func TestOvulation_MarkerHygiene(t *testing.T) {
	cur := day(julyStart)
	rs := taskBBT(cur)
	// Shuffle, add an undated reading and a duplicate day whose later record wins: a high duplicate of day 13 would
	// lift the baseline; the later (low) one keeps the shift.
	in := current(append([]BBTReading{rs[9], {Date: NoDay, Centi: 3700}, {Date: cur + 12, Centi: 3700}},
		append(rs[:9:9], rs[5])...))
	in.LH = []LHTest{{Date: NoDay, Positive: true}}
	s := prep(in)
	assert.Len(t, s.BBT(), 10)
	assert.Len(t, s.LH(), 0)
	st := s.Resolve(cur+20, cur+20)
	assert.Equal(t, OvulationBBTConfirmed, st.Anchors.OvulationSource)
	assert.Equal(t, cur+12, st.Anchors.EstimatedOvulationDate)

	// Prepare copies the markers.
	in.BBT[0].Centi = 0
	assert.Equal(t, OvulationBBTConfirmed, s.Resolve(cur+20, cur+20).Anchors.OvulationSource)

	// Readings before the first real period are ignored; with no periods there are no ovulations.
	s = prep(Input{BBT: taskBBT(cur), LH: []LHTest{lhOn(cur, true)}})
	assert.Empty(t, s.Ovulations())
	assert.Equal(t, OvulationUnknown, s.Resolve(cur+20, cur+20).Anchors.OvulationSource)
}

// A declared onboarding start is not a real cycle: markers do not move it (T-M9-17 scope 3: per real cycle).
func TestOvulation_DeclaredStartUsesCalendar(t *testing.T) {
	decl := day(julyStart)
	in := declared(Declared{LastPeriodStart: Some(decl)})
	in.LH = []LHTest{lhOn(decl+11, true)}
	st := prep(in).Resolve(decl+20, decl+20)
	assert.Equal(t, OvulationCalendar, st.Anchors.OvulationSource)
	assert.Equal(t, decl+28, st.Anchors.PredictedNextPeriodStart)
}

// The zero Input means "no data" (T-M9-16 open item): Day 0 = 1970-01-01 is never a declared date.
func TestInput_ZeroMeansNoData(t *testing.T) {
	for _, in := range []Input{{}, {Declared: Declared{LastPeriodStart: Some(NoDay)}, BirthDate: Some(NoDay),
		DeliveryDate: Some(NoDay)}} {
		s := prep(in)
		for _, d := range []Day{0, 1, day(julyStart)} {
			st := s.Resolve(d, d)
			assert.Equal(t, ResolutionUnknown, st.ResolutionSource, "%s", d)
			assert.True(t, st.Warnings.Has(WarnInsufficientAnchorData))
		}
		assert.Equal(t, NoDay, s.lmp)
		assert.Equal(t, NoDay, s.birth)
		assert.Equal(t, NoDay, s.delivery)
	}
	// An explicit 1970-01-01 is still a date.
	s := prep(Input{Declared: Declared{LastPeriodStart: Some(Day(0))}, BirthDate: Some(Day(1)),
		DeliveryDate: Some(Day(2))})
	assert.Equal(t, StartOnboardingDeclared, s.Resolve(3, 3).Anchors.CurrentPeriodStartSource)
	assert.Equal(t, Day(1), s.birth)
	assert.Equal(t, Day(2), s.delivery)
}

func TestResolve_NoAllocsWithMarkers(t *testing.T) {
	cur := day(julyStart)
	in := lutealHistory(12, 13, 0, 14)
	in.BBT = append(in.BBT, taskBBT(cur)...)
	in.LH = []LHTest{lhOn(cur+9, true)}
	s := prep(in)
	var st Status
	targets := []Day{cur + 3, cur + 12, cur + 20, cur + 40, cur - 30, cur - 100}
	allocs := testing.AllocsPerRun(200, func() {
		for _, t := range targets {
			st = s.Resolve(t, cur)
		}
	})
	assert.Zero(t, allocs)
	_ = st
}
