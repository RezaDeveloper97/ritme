package engine

// days converts a small day count (a cycle or period length, a cap) to a Day offset.
func days(n int) Day { return Day(n) } //nolint:gosec // engine lengths and caps are small (|n| < 2^31)

// layout is the day map of one cycle (§2.11, §2.16–§2.19): where the period ends, when the next period is due,
// the ovulation estimate and the display fertile window. Every phase rule from §2.21 to §2.25 reads it.
type layout struct {
	start        Day
	end          Day             // current_period_end: the logged end, or the assumed end start + EPL − 1 (§2.11)
	endConfirmed bool            // the end is a logged one
	next         Day             // predicted_next_period_start, or the actual next real start in a completed cycle (E-22)
	ov           Day             // estimated_ovulation_date
	fertileStart Day             // display_fertile_start = max(O − before, end + 1) (§2.19); the display end is ov
	ovSrc        OvulationSource // ovulation_source of ov (§2.10)
	conflict     bool            // lh_bbt_conflict in this cycle (§2.17)
}

// newLayout builds a layout from its anchors with a calendar ovulation source; ov is the ovulation estimate, so
// tests can inject any ovulation date (layoutFor sets the BBT/LH source).
func newLayout(start, end Day, endConfirmed bool, next, ov Day, fertileBefore int) layout {
	return layout{
		start:        start,
		end:          end,
		endConfirmed: endConfirmed,
		next:         next,
		ov:           ov,
		fertileStart: max(ov-days(fertileBefore), end+1),
		ovSrc:        OvulationCalendar,
	}
}

// fertileEmpty reports an empty display window (§2.19: display end < display start, e.g. a short cycle whose
// period covers the whole biological window).
func (l *layout) fertileEmpty() bool { return l.ov < l.fertileStart }

// cyclePhase resolves priorities 3–5 of §2.20 — the display fertile window, luteal, follicular — for a day that
// is neither menstrual nor period_expected. It answers SubphaseUnknown when no rule applies (priority 6).
func (l *layout) cyclePhase(t Day, p *Params) Subphase {
	if !l.fertileEmpty() && t >= l.fertileStart && t <= l.ov {
		return fertileSubphase(int(l.ov - t))
	}
	if t > l.ov && t < l.next {
		return lutealSubphase(t, l.ov, l.next, p)
	}
	if fs, fe := l.end+1, l.fertileStart-1; t >= fs && t <= fe {
		return follicularSubphase(int(t-fs)+1, int(fe-fs)+1)
	}
	return SubphaseUnknown
}

// fertileSubphase is §2.24 by days_to_ovulation: 5, 4, 3 → fertile_rising; 2, 1 → high_fertility; 0 →
// ovulation_likely.
func fertileSubphase(dto int) Subphase {
	switch {
	case dto >= 3:
		return SubphaseFertileRising
	case dto >= 1:
		return SubphaseHighFertility
	default:
		return SubphaseOvulationLikely
	}
}

// follicularSubphase is the §2.23 split of a follicular range of length n at 1-based position pos: 1 day → mid;
// 2 days → early, late; ≥ 3 → early = max(1, ⌊0.4n⌋), mid = max(1, ⌊0.4n⌋), late = the rest.
func follicularSubphase(pos, n int) Subphase {
	switch n {
	case 1:
		return SubphaseMidFollicular
	case 2:
		if pos == 1 {
			return SubphaseEarlyFollicular
		}
		return SubphaseLateFollicularTransition
	}
	early := max(1, n*2/5)
	mid := max(1, n*2/5)
	switch {
	case pos <= early:
		return SubphaseEarlyFollicular
	case pos <= early+mid:
		return SubphaseMidFollicular
	default:
		return SubphaseLateFollicularTransition
	}
}

// lutealSubphase is §2.25 for ov < t < next, in the spec priority pms_possible > late_luteal > post_ovulation >
// early/mid.
func lutealSubphase(t, ov, next Day, p *Params) Subphase {
	untilPeriod := int(next - t)
	switch {
	case untilPeriod >= 1 && untilPeriod <= p.PMSDays:
		return SubphasePMSPossible
	case untilPeriod > p.PMSDays && untilPeriod <= p.LateLutealDays:
		return SubphaseLateLuteal
	case t-ov == 1:
		return SubphasePostOvulation
	default:
		return assignableLuteal(t, ov, next, p)
	}
}

// assignableLuteal splits the assignable range O + 2 .. next − (LateLutealDays + 1) (= next − 7): early_luteal
// gets the first ⌈n/2⌉ days, mid_luteal the remaining ⌊n/2⌋. A day outside the range, or any day when the range is
// empty, is mid_luteal.
func assignableLuteal(t, ov, next Day, p *Params) Subphase {
	as, ae := ov+2, next-days(p.LateLutealDays+1)
	if ae < as || t < as || t > ae {
		return SubphaseMidLuteal
	}
	n := int(ae-as) + 1
	if int(t-as) < (n+1)/2 {
		return SubphaseEarlyLuteal
	}
	return SubphaseMidLuteal
}

// fertilityOf is the §2.26 table. period_expected and unknown (and suppression, T-M9-19) are unknown.
func fertilityOf(s Subphase) FertilityLevel {
	switch s {
	case SubphaseMenstrual, SubphaseMenstrualPossible, SubphaseEarlyFollicular, SubphaseMidFollicular,
		SubphasePostOvulation, SubphaseEarlyLuteal, SubphaseMidLuteal, SubphaseLateLuteal, SubphasePMSPossible:
		return FertilityLow
	case SubphaseLateFollicularTransition, SubphaseFertileRising:
		return FertilityMedium
	case SubphaseHighFertility:
		return FertilityHigh
	case SubphaseOvulationLikely:
		return FertilityPeak
	default:
		return FertilityUnknown
	}
}
