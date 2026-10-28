package engine

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The invariant suite (README §9, T-M9-22): properties every Status, every future_cycles list and every State must
// keep on any history. violations / futureViolations / stateViolations are shared by the random-history test below
// and by FuzzPrepare (fuzz_test.go).

// userInputWarnings are the six §2.33 warnings, written out independently of Warning.RequiresUserInput.
func userInputWarnings() WarningSet {
	var s WarningSet
	for _, w := range []Warning{WarnPeriodEndMissingWarningCapExceeded, WarnPeriodEndMissingHardCapExceeded,
		WarnCycleUnresolvedAfterExpectedPeriod, WarnInsufficientAnchorData, WarnContradictoryAnchorData,
		WarnStaleOnboardingAnchor} {
		s.Add(w)
	}
	return s
}

// loggedBleeding reports whether d lies inside a closed real period, or is the start day of an open one.
func loggedBleeding(s *State, d Day) bool {
	i := s.lastRealAtOrBefore(d)
	if i < 0 {
		return false
	}
	p := s.Periods()[i]
	return d == p.Start || (!p.Open() && d <= p.End)
}

// noCycleSuppression is a suppression under which no cycle runs (E-25 pregnancy, E-14 postpartum not returned).
func noCycleSuppression(sup SuppressionReason) bool {
	return sup == SuppressionPregnancy || sup == SuppressionPostpartumNotReturned
}

// violations lists every broken invariant of one status resolved with today.
func violations(s *State, st *Status, today Day) []string {
	var v []string
	bad := func(format string, a ...any) { v = append(v, fmt.Sprintf(format, a...)) }
	p := s.Params()
	a := &st.Anchors
	sup := st.Limits.SuppressionReason
	ov := a.EstimatedOvulationDate

	if st.MainPhase != st.Subphase.Main() {
		bad("main_phase %s is not the main phase of subphase %s", st.MainPhase, st.Subphase)
	}

	// cycle_day ≥ 1 whenever a start exists (1-based, §2.5); only a pregnancy (E-25) or postpartum before cycles
	// return (E-14, T-M9-53) hides it outside logged bleeding.
	if cd, ok := st.CycleDay.Get(); ok {
		switch {
		case noCycleSuppression(sup) && st.MainPhase != MainPhaseMenstrual:
			bad("cycle_day %d outside logged bleeding under %s", cd, sup)
		case cd < 1:
			bad("cycle_day %d < 1", cd)
		case !a.CurrentPeriodStart.IsSet():
			bad("cycle_day %d without current_period_start", cd)
		case cd != int(st.Date-a.CurrentPeriodStart)+1:
			bad("cycle_day %d ≠ date − start + 1 (start %s)", cd, a.CurrentPeriodStart)
		}
	} else if a.CurrentPeriodStart.IsSet() && (!noCycleSuppression(sup) || st.MainPhase == MainPhaseMenstrual) {
		bad("current_period_start %s without cycle_day", a.CurrentPeriodStart)
	}

	// Every start/end has a source (§2.40); is_confirmed ⇔ a logged end; ovulation date ⇔ its source.
	if a.CurrentPeriodStart.IsSet() != (a.CurrentPeriodStartSource != StartUnknown) {
		bad("current_period_start %q with source %s", a.CurrentPeriodStart, a.CurrentPeriodStartSource)
	}
	if a.CurrentPeriodEnd.IsSet() != (a.CurrentPeriodEndSource != EndUnknown) {
		bad("current_period_end %q with source %s", a.CurrentPeriodEnd, a.CurrentPeriodEndSource)
	}
	if a.CurrentPeriodEndIsConfirmed != (a.CurrentPeriodEndSource == EndUserLogged) {
		bad("is_confirmed %v with end source %s", a.CurrentPeriodEndIsConfirmed, a.CurrentPeriodEndSource)
	}
	if a.CurrentPeriodEnd.IsSet() && (!a.CurrentPeriodStart.IsSet() || a.CurrentPeriodEnd < a.CurrentPeriodStart) {
		bad("current_period_end %s before start %q", a.CurrentPeriodEnd, a.CurrentPeriodStart)
	}
	if ov.IsSet() != (a.OvulationSource != OvulationUnknown) {
		bad("estimated_ovulation_date %q with source %s", ov, a.OvulationSource)
	}
	if ov.IsSet() && a.PredictedNextPeriodStart.IsSet() && ov >= a.PredictedNextPeriodStart {
		bad("ovulation %s not before the next period %s", ov, a.PredictedNextPeriodStart)
	}

	// Never fertile on a menstrual day; O+1 (and every later day) is never fertile (§2.19, §2.40).
	if st.MainPhase == MainPhaseFertile {
		switch {
		case a.CurrentPeriodEnd.IsSet() && st.Date <= a.CurrentPeriodEnd:
			bad("fertile on or before current_period_end %s", a.CurrentPeriodEnd)
		case loggedBleeding(s, st.Date):
			bad("fertile on a logged period day")
		case !ov.IsSet():
			bad("fertile without an ovulation date")
		case st.Date > ov:
			bad("fertile after the ovulation %s", ov)
		case st.Date < ov-days(p.FertileBefore):
			bad("fertile before O − %d (O %s)", p.FertileBefore, ov)
		}
	}
	if st.MainPhase == MainPhaseMenstrual && st.FertilityLevel != FertilityLow && sup == SuppressionNone {
		bad("menstrual with fertility_level %s", st.FertilityLevel)
	}

	// fertility_level = unknown ⇔ period_expected / unknown / suppressed (§2.26, §2.42); otherwise the §2.26 table.
	wantUnknown := st.MainPhase == MainPhasePeriodExpected || st.MainPhase == MainPhaseUnknown ||
		st.Limits.FertilitySuppressed
	if (st.FertilityLevel == FertilityUnknown) != wantUnknown {
		bad("fertility_level %s with main_phase %s, suppressed %v", st.FertilityLevel, st.MainPhase,
			st.Limits.FertilitySuppressed)
	}
	if !st.Limits.FertilitySuppressed && st.FertilityLevel != specFertility(st.Subphase) {
		bad("fertility_level %s, §2.26 says %s for %s", st.FertilityLevel, specFertility(st.Subphase), st.Subphase)
	}

	// days_to_ovulation (§2.34): O − target; null without an ovulation, beyond max_late, or under suppression.
	wantDTO := ov.IsSet() && !st.Limits.FertilitySuppressed && st.DaysLate.Or(0) <= p.MaxLateDays
	if dto, ok := st.DaysToOvulation.Get(); ok != wantDTO {
		bad("days_to_ovulation present %v, want %v (O %q, late %v)", ok, wantDTO, ov, st.DaysLate)
	} else if ok && dto != int(ov-st.Date) {
		bad("days_to_ovulation %d ≠ O − date", dto)
	}

	// days_to_period / days_late (§2.22): both present exactly when a next start is predicted.
	next := a.PredictedNextPeriodStart
	dtp, okP := st.DaysToPeriod.Get()
	late, okL := st.DaysLate.Get()
	switch {
	case okP != next.IsSet() || okL != next.IsSet():
		bad("days_to_period %v / days_late %v with predicted_next %q", okP, okL, next)
	case okP && (dtp != max(0, int(next-st.Date)) || late != max(0, int(st.Date-next))):
		bad("days_to_period %d / days_late %d for next %s", dtp, late, next)
	}

	// requires_user_input ⇔ one of the six warnings (§2.33).
	if st.RequiresUserInput != (st.Warnings&userInputWarnings() != 0) {
		bad("requires_user_input %v with warnings %b", st.RequiresUserInput, st.Warnings)
	}

	// The E-24 limits block, and what suppression switches off (§2.42).
	l := st.Limits
	if l.FertilitySuppressed != (sup != SuppressionNone) || l.EngineSuspended != (sup == SuppressionPregnancy) {
		bad("limits %+v", l)
	}
	if l.FertilitySuppressed && (ov.IsSet() || st.SuggestPregnancyTest) {
		bad("suppressed (%s) with ovulation %q / suggest_pregnancy_test %v", sup, ov, st.SuggestPregnancyTest)
	}
	if sup == SuppressionPregnancy && (st.Warnings != 0 || st.ConfidenceReasons != 0 || next.IsSet()) {
		bad("pregnancy status with warnings, reasons or a predicted next")
	}

	// E-03 / E-28: a projected status is in the future and comes from a predicted_reference start.
	if st.FutureK < 0 || st.FutureK > max(1, p.FutureCycles) {
		bad("future_k %d", st.FutureK)
	}
	if st.FutureK > 0 && (st.Date <= today || a.CurrentPeriodStartSource != StartPredictedReference) {
		bad("future_k %d on %s (today %s, start source %s)", st.FutureK, st.Date, today, a.CurrentPeriodStartSource)
	}
	return v
}

// futureViolations lists the broken invariants of today's future_cycles list (E-03…E-05, E-13, E-28): k = 1..N,
// strictly increasing, never overlapping, every period after today, the fertile window ending on the ovulation that
// precedes its period, nothing fertile under suppression and nothing at all while suspended or postpartum.
func futureViolations(s *State, fcs []FutureCycle, today Day) []string {
	var v []string
	bad := func(format string, a ...any) { v = append(v, fmt.Sprintf(format, a...)) }
	p := s.Params()
	n := max(1, p.FutureCycles)
	sup := s.Suppression(today).SuppressionReason
	if fcs == nil {
		bad("future_cycles is nil, want a list")
	}
	if len(fcs) != 0 && len(fcs) != n {
		bad("future_cycles has %d items, want 0 or %d", len(fcs), n)
	}
	if len(fcs) > 0 && (sup == SuppressionPregnancy || sup == SuppressionPostpartumNotReturned) {
		bad("future_cycles under %s", sup)
	}
	for i, c := range fcs {
		if c.K != i+1 {
			bad("item %d has k %d", i, c.K)
		}
		if c.PeriodStart <= today || c.PeriodEnd < c.PeriodStart {
			bad("k %d: period %s..%s (today %s)", c.K, c.PeriodStart, c.PeriodEnd, today)
		}
		if i > 0 {
			prev := fcs[i-1]
			if c.PeriodStart <= prev.PeriodEnd {
				bad("k %d starts %s on or before k %d ends %s", c.K, c.PeriodStart, prev.K, prev.PeriodEnd)
			}
			if int(c.PeriodStart-prev.PeriodStart) != s.ecl() {
				bad("k %d is %d days after k %d, want ECL %d", c.K, c.PeriodStart-prev.PeriodStart, prev.K, s.ecl())
			}
		}
		if c.K >= 2 && c.Confidence > ConfidenceMedium {
			bad("k %d confidence %s above medium", c.K, c.Confidence)
		}
		has := c.Ovulation.IsSet()
		if has != (c.OvulationSource != OvulationUnknown) || has != c.FertileStart.IsSet() || has != c.FertileEnd.IsSet() {
			bad("k %d: partial ovulation/fertile fields %+v", c.K, c)
		}
		if has && (c.Ovulation >= c.PeriodStart || c.FertileEnd != c.Ovulation ||
			c.FertileStart != c.Ovulation-days(p.FertileBefore)) {
			bad("k %d: fertile %s..%s, ovulation %s, period %s", c.K, c.FertileStart, c.FertileEnd, c.Ovulation,
				c.PeriodStart)
		}
		if has && sup != SuppressionNone {
			bad("k %d: ovulation under %s", c.K, sup)
		}
	}
	return v
}

// stateViolations lists the broken invariants of a prepared State against its input: kept + excluded = every dated
// record; kept periods sorted, non-overlapping, at most one open; cycles between consecutive kept starts; skips and
// markers sorted and unique per day; effective lengths inside the valid ranges (or the defaults).
func stateViolations(s *State, in Input) []string {
	var v []string
	bad := func(format string, a ...any) { v = append(v, fmt.Sprintf(format, a...)) }
	p := s.Params()
	dated := 0
	for _, r := range in.Periods {
		if r.Start != NoDay {
			dated++
		}
	}
	ps := s.Periods()
	if len(ps)+len(s.Excluded()) != dated {
		bad("%d kept + %d excluded periods ≠ %d dated records", len(ps), len(s.Excluded()), dated)
	}
	if (len(s.Excluded()) > 0) != s.Contradictory() {
		bad("contradictory %v with %d excluded records", s.Contradictory(), len(s.Excluded()))
	}
	open := 0
	for i, r := range ps {
		if r.Open() {
			open++
		} else if r.End < r.Start {
			bad("kept period %s..%s ends before it starts", r.Start, r.End)
		}
		if i > 0 && (r.Start <= ps[i-1].Start || (!ps[i-1].Open() && r.Start <= ps[i-1].End)) {
			bad("kept periods %d and %d overlap or are unsorted", i-1, i)
		}
	}
	if open > 1 {
		bad("%d open periods kept", open)
	}
	if want := max(0, len(ps)-1); len(s.Cycles()) != want {
		bad("%d cycles for %d periods", len(s.Cycles()), len(ps))
	}
	for i, c := range s.Cycles() {
		if c.Start != ps[i].Start || c.NextStart != ps[i+1].Start || c.Length != int(c.NextStart-c.Start) ||
			c.Valid != (!c.Skipped && !c.Short && !c.Long) {
			bad("cycle %d %+v", i, c)
		}
	}
	if len(s.Ovulations()) != len(ps) {
		bad("%d ovulations for %d periods", len(s.Ovulations()), len(ps))
	}
	if !slices.IsSorted(s.Skips()) || len(slices.Compact(slices.Clone(s.Skips()))) != len(s.Skips()) {
		bad("skips not sorted and unique")
	}
	for i := 1; i < len(s.BBT()); i++ {
		if s.BBT()[i].Date <= s.BBT()[i-1].Date {
			bad("BBT readings not sorted with one per day")
			break
		}
	}
	if !slices.IsSortedFunc(s.LH(), func(a, b LHTest) int { return int(a.Date) - int(b.Date) }) {
		bad("LH tests not sorted")
	}
	if l, src := s.CycleLength(); (src == LengthDefault && l != p.DefaultCycleLength) ||
		(src != LengthDefault && (l < p.CycleMin || l > p.CycleMax)) {
		bad("effective_cycle_length %d (%s)", l, src)
	}
	if l, src := s.PeriodLength(); (src == LengthDefault && l != p.DefaultPeriodLength) ||
		(src != LengthDefault && (l < p.PeriodMin || l > p.PeriodMax)) {
		bad("effective_period_length %d (%s)", l, src)
	}
	if l, _ := s.LutealLength(); l < p.Luteal.Min || l > p.Luteal.Max {
		bad("effective_luteal_length %d", l)
	}
	return v
}

// randomHistory is one seeded random user for the invariant suite: 0–30 real periods with regular, irregular and
// outlier cycles, open current and old periods, unsorted / duplicated / overlapping records, skips at (or near) the
// predicted start, biphasic BBT curves (strict passes and fails), LH tests (agreeing and conflicting), the pill,
// postpartum and breastfeeding, every mode, ages 14–55, declared and manual lengths and pregnancy tests.
func randomHistory(r *rand.Rand, today Day) Input {
	in := none()
	base := 21 + r.IntN(25)
	if r.IntN(10) == 0 {
		base = 15 + r.IntN(50) // outlier-only histories
	}
	jitter := r.IntN(3)
	if r.IntN(4) == 0 {
		jitter = 3 + r.IntN(10) // irregular
	}
	plen := 2 + r.IntN(9)
	n := r.IntN(31)
	if r.IntN(6) == 0 {
		n = 0
	}

	// Gaps between consecutive starts, oldest first; a skipped cycle doubles its gap (E-15).
	gaps := make([]int, max(0, n-1))
	skipAfter := make([]bool, len(gaps))
	sum := 0
	for i := range gaps {
		gaps[i] = max(1, base+r.IntN(2*jitter+1)-jitter)
		if r.IntN(12) == 0 {
			gaps[i] = 10 + r.IntN(80) // a lone outlier
		}
		if r.IntN(10) == 0 {
			skipAfter[i] = true
			gaps[i] += base
		}
		sum += gaps[i]
	}
	last := today - Day(r.IntN(150)) + Day(r.IntN(4)) // occasionally a start logged just after today
	d := last - Day(sum)
	for i := range n {
		ln := max(1, plen+r.IntN(3)-1)
		pr := Period{Start: d, End: d + Day(ln-1)}
		if r.IntN(4) == 0 {
			pr.Source = PeriodBleedingLog
		}
		switch {
		case i == n-1 && r.IntN(3) == 0: // the current period is open
			pr.End = NoDay
		case r.IntN(30) == 0: // an old open period
			pr.End = NoDay
		}
		in.Periods = append(in.Periods, pr)

		next := today + Day(base)
		if i < len(gaps) {
			next = d + Day(gaps[i])
			if skipAfter[i] {
				in.Skips = append(in.Skips, d+Day(base+r.IntN(3)-1))
			}
		}
		if r.IntN(3) == 0 {
			in.BBT = append(in.BBT, randomBBT(r, d, next, today)...)
		}
		if r.IntN(3) == 0 {
			ov := next - Day(12+r.IntN(5))
			if r.IntN(4) == 0 {
				ov += Day(r.IntN(9) - 4) // LH/BBT conflict
			}
			in.LH = append(in.LH, LHTest{Date: ov - 2, Positive: false}, LHTest{Date: ov - 1, Positive: r.IntN(5) != 0})
		}
		if i < len(gaps) {
			d += Day(gaps[i])
		}
	}

	// Malformed records: duplicates, overlaps, ends before starts, records without a start, unsorted input.
	if len(in.Periods) > 0 && r.IntN(12) == 0 {
		c := in.Periods[r.IntN(len(in.Periods))]
		c.Start += Day(r.IntN(5) - 2)
		in.Periods = append(in.Periods, c)
	}
	if r.IntN(25) == 0 {
		in.Periods = append(in.Periods, Period{Start: today - 40, End: today - 45}, Period{Start: NoDay, End: today})
	}
	if r.IntN(3) == 0 {
		r.Shuffle(len(in.Periods), func(i, j int) { in.Periods[i], in.Periods[j] = in.Periods[j], in.Periods[i] })
	}
	if r.IntN(8) == 0 {
		in.Skips = append(in.Skips, today-Day(r.IntN(200)), NoDay)
	}

	if r.IntN(3) == 0 {
		in.Declared.LastPeriodStart = Some(today - Day(r.IntN(150)))
	}
	if r.IntN(3) == 0 {
		in.Declared.CycleLength = Some(18 + r.IntN(32))
		in.Declared.PeriodLength = Some(1 + r.IntN(11))
		in.Declared.Manual = r.IntN(4) == 0
	}

	switch r.IntN(6) {
	case 0: // unknown age
		in.TeenBandOnly = r.IntN(3) == 0
	default: // ages 14–55
		in.BirthDate = Some(today - Day(365*(14+r.IntN(42))+r.IntN(365)))
	}
	switch x := r.IntN(100); {
	case x < 50:
		in.Mode = ModeCycle
	case x < 65:
		in.Mode = ModeTTC
	case x < 73:
		in.Mode = ModePregnancy
	case x < 82:
		in.Mode = ModePostpartum
	case x < 90:
		in.Mode = ModeMenopause
	default:
		in.Mode = ModeTeen
	}
	in.Hormonal = r.IntN(8) == 0 // the pill
	if in.Mode == ModePostpartum || r.IntN(12) == 0 {
		in.Breastfeeding = r.IntN(2) == 0
		if r.IntN(5) != 0 {
			in.DeliveryDate = Some(today - Day(r.IntN(500)))
		}
	}
	if r.IntN(8) == 0 {
		in.PregnancyTests = append(in.PregnancyTests,
			PregnancyTest{Date: today - Day(r.IntN(45)), Positive: r.IntN(3) != 0},
			PregnancyTest{Date: today - Day(r.IntN(10)), Positive: false})
	}
	return in
}

// randomBBT is a biphasic curve over [from, to) (and never after today + 3): a low phase around 36.40 °C, a rise of
// 0.08–0.45 °C after an ovulation 12–16 days before to, one day in seven unmeasured.
func randomBBT(r *rand.Rand, from, to, today Day) []BBTReading {
	ov := to - Day(12+r.IntN(5))
	rise := int32(8 + r.IntN(38))
	var out []BBTReading
	for d := from; d < to && d <= today+3; d++ {
		if r.IntN(7) == 0 {
			continue
		}
		c := int32(3640 + r.IntN(9) - 4)
		if d > ov {
			c += rise
		}
		out = append(out, BBTReading{Date: d, Centi: c})
	}
	return out
}

// TestInvariants_RandomHistories is the invariant suite over 2 000 seeded random histories × a 400-day window
// crossing today: every status keeps violations() empty, Range == map(Resolve) byte for byte, future_cycles keep
// their invariants, Prepare does not modify its input, and Resolve / Today are deterministic (a second Prepare of the
// same input gives the same bytes). The generator must reach every main phase, ovulation source, suppression reason
// and the warnings named below, so the properties are not vacuous.
func TestInvariants_RandomHistories(t *testing.T) {
	r := rand.New(rand.NewPCG(22, 2026))
	histories := 2000
	if testing.Short() {
		histories = 200
	}
	seen := map[string]int{}
	var a, b []byte
	for i := range histories {
		today := day("2026-10-21") + Day(r.IntN(1461)-730)
		in := randomHistory(r, today)
		orig := cloneInput(in)
		s := prep(in)
		require.Equal(t, orig, in, "history %d: Prepare modified its input", i)
		if v := stateViolations(s, in); len(v) > 0 {
			t.Fatalf("history %d: state: %v", i, v)
		}

		from := today - Day(r.IntN(MaxRangeDays))
		got := s.Range(from, from+MaxRangeDays-1, today)
		require.Len(t, got, MaxRangeDays)
		for j := range got {
			st := &got[j]
			if v := violations(s, st, today); len(v) > 0 {
				t.Fatalf("history %d, day %s (today %s): %v\n%s", i, st.Date, today, v, AppendJSON(nil, st, nil))
			}
			want := s.Resolve(st.Date, today)
			a, b = AppendJSON(a[:0], st, nil), AppendJSON(b[:0], &want, nil)
			if !bytes.Equal(a, b) {
				t.Fatalf("history %d, day %s: Range ≠ Resolve\nrange   %s\nresolve %s", i, st.Date, a, b)
			}
			seen["main:"+st.MainPhase.String()]++
			seen["ov:"+st.Anchors.OvulationSource.String()]++
			seen["sup:"+st.Limits.SuppressionReason.String()]++
			for w := range st.Warnings.All() {
				seen["w:"+w.String()]++
			}
			if st.FutureK > 0 {
				seen["future"]++
			}
		}

		fcs := s.FutureCycles(today)
		if v := futureViolations(s, fcs, today); len(v) > 0 {
			t.Fatalf("history %d (today %s): future_cycles: %v", i, today, v)
		}

		// Determinism: the same input prepared again resolves to the same bytes.
		st1, ui1 := s.Today(today)
		st2, ui2 := prep(in).Today(today)
		a, b = AppendJSON(a[:0], &st1, &ui1), AppendJSON(b[:0], &st2, &ui2)
		require.Equal(t, string(a), string(b), "history %d: Today is not deterministic", i)
		again := s.Resolve(from, today)
		a, b = AppendJSON(a[:0], &got[0], nil), AppendJSON(b[:0], &again, nil)
		require.Equal(t, string(a), string(b), "history %d: Resolve is not deterministic", i)
	}

	if testing.Short() {
		return
	}
	for _, k := range []string{"main:menstrual", "main:follicular", "main:fertile", "main:luteal",
		"main:period_expected", "main:unknown", "ov:calendar", "ov:bbt_confirmed", "ov:lh_positive", "ov:unknown",
		"sup:", "sup:hormonal_contraception", "sup:pregnancy", "sup:postpartum_not_returned", "future",
		"w:period_end_missing_hard_cap_exceeded", "w:predicted_period_overdue", "w:cycle_unresolved_after_expected_period",
		"w:insufficient_anchor_data", "w:stale_onboarding_anchor", "w:skipped_cycle_recorded", "w:lh_bbt_conflict",
		"w:contradictory_anchor_data", "w:perimenopause_age_range", "w:long_absence_no_period",
		"w:short_cycle_outlier_detected", "w:long_cycle_outlier_detected", "w:high_cycle_variability"} {
		assert.Positive(t, seen[k], "the generator never produced %s", k)
	}
}

// TestInvariants_DetectBreakage guards the checkers themselves: each corruption of a valid status, list or state is
// reported, so an empty violations() result means something.
func TestInvariants_DetectBreakage(t *testing.T) {
	cur := day(julyStart)
	in := regular(cur, 28, 5)
	s := prep(in)
	today := cur + 12
	good := s.Resolve(today, today)
	require.Empty(t, violations(s, &good, today))
	require.Equal(t, MainPhaseFertile, good.MainPhase)

	corrupt := map[string]func(st *Status){
		"main phase":         func(st *Status) { st.MainPhase = MainPhaseLuteal },
		"cycle day 0":        func(st *Status) { st.CycleDay = Some(0) },
		"cycle day missing":  func(st *Status) { st.CycleDay = None[int]() },
		"start source":       func(st *Status) { st.Anchors.CurrentPeriodStartSource = StartUnknown },
		"end source":         func(st *Status) { st.Anchors.CurrentPeriodEndSource = EndUnknown },
		"confirmed":          func(st *Status) { st.Anchors.CurrentPeriodEndIsConfirmed = false },
		"ovulation source":   func(st *Status) { st.Anchors.OvulationSource = OvulationUnknown },
		"fertile on period":  func(st *Status) { st.Date, st.CycleDay = cur+2, Some(3) },
		"fertile after O":    func(st *Status) { st.Anchors.EstimatedOvulationDate = st.Date - 1 },
		"fertility unknown":  func(st *Status) { st.FertilityLevel = FertilityUnknown },
		"dto missing":        func(st *Status) { st.DaysToOvulation = None[int]() },
		"dto value":          func(st *Status) { st.DaysToOvulation = Some(-9) },
		"days late":          func(st *Status) { st.DaysLate = Some(3) },
		"requires input":     func(st *Status) { st.RequiresUserInput = true },
		"requires input off": func(st *Status) { st.Warnings.Add(WarnStaleOnboardingAnchor) },
		"limits":             func(st *Status) { st.Limits.EngineSuspended = true },
		"future k":           func(st *Status) { st.FutureK = 2 },
	}
	for name, f := range corrupt {
		st := good
		f(&st)
		assert.NotEmpty(t, violations(s, &st, today), name)
	}

	fcs := s.FutureCycles(today)
	require.Empty(t, futureViolations(s, fcs, today))
	for name, f := range map[string]func(c []FutureCycle){
		"k":       func(c []FutureCycle) { c[1].K = 7 },
		"overlap": func(c []FutureCycle) { c[2].PeriodStart = c[1].PeriodEnd },
		"past":    func(c []FutureCycle) { c[0].PeriodStart = today },
		"medium":  func(c []FutureCycle) { c[3].Confidence = ConfidenceHigh },
		"fertile": func(c []FutureCycle) { c[4].FertileStart = NoDay },
		"ov":      func(c []FutureCycle) { c[5].Ovulation = c[5].PeriodStart },
	} {
		c := slices.Clone(fcs)
		f(c)
		assert.NotEmpty(t, futureViolations(s, c, today), name)
	}
	assert.NotEmpty(t, futureViolations(s, nil, today), "nil list")
	assert.NotEmpty(t, futureViolations(s, fcs[:3], today), "short list")

	require.Empty(t, stateViolations(s, in))
	more := in
	more.Periods = append(slices.Clone(in.Periods), Period{Start: cur + 1, End: cur + 3})
	assert.NotEmpty(t, stateViolations(s, more), "a record the state does not account for")
}

// TestInvariants_FuzzRegressions are the engine bugs FuzzPrepare found (T-M9-22), as readable cases; the raw inputs
// are in testdata/fuzz/FuzzPrepare.
func TestInvariants_FuzzRegressions(t *testing.T) {
	today := day("2026-11-30")

	t.Run("a logged period running past today is not projected as fertile", func(t *testing.T) {
		// 49 logged days from today (a period longer than the cycle): the projection's k = 1 starts on day 29.
		s := prep(withPeriods([]Period{{Start: today, End: today + 48}}))
		for d := today; d <= today+48; d++ {
			st := s.Resolve(d, today)
			require.Empty(t, violations(s, &st, today), "day %s", d)
			assert.Equal(t, MainPhaseMenstrual, st.MainPhase, "day %s", d)
			assert.Equal(t, ResolutionUserLogged, st.ResolutionSource, "day %s", d)
			assert.Zero(t, st.FutureK, "day %s", d)
		}
		after := s.Resolve(today+60, today)
		assert.Positive(t, after.FutureK, "after the logged end the projection applies again")
	})

	t.Run("a skip inside a logged period is ignored", func(t *testing.T) {
		start := today - 60
		in := withPeriods([]Period{{Start: start, End: start + 40}})
		in.Skips = []Day{start + 28} // the predicted next, but a logged bleeding day
		s := prep(in)
		assert.Equal(t, []Day{start + 28}, s.Diagnostics().IgnoredSkips)
		st := s.Resolve(start+30, today)
		require.Empty(t, violations(s, &st, today))
		assert.Equal(t, MainPhaseMenstrual, st.MainPhase)
		assert.False(t, st.Warnings.Has(WarnSkippedCycleRecorded))
	})

	t.Run("an LH positive on the last cycle day is no ovulation", func(t *testing.T) {
		start := today - 40
		in := withPeriods([]Period{{Start: start, End: start + 4}, {Start: start + 30, End: start + 34}})
		in.LH = []LHTest{{Date: start + 29, Positive: true}}
		s := prep(in)
		assert.Equal(t, OvulationUnknown, s.Ovulations()[0].Source)
		st := s.Resolve(start+20, today)
		require.Empty(t, violations(s, &st, today))
		assert.Equal(t, OvulationCalendar, st.Anchors.OvulationSource)
		assert.Equal(t, start+16, st.Anchors.EstimatedOvulationDate) // next real start − 14 (E-22)

		in.LH[0].Date = start + 28 // the day before: ovulation start + 29, the cycle's last day
		assert.Equal(t, start+29, prep(in).Ovulations()[0].LH)
	})
}
