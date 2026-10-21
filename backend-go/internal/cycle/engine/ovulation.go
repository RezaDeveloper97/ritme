package engine

import (
	"cmp"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/engine/bbt"
)

// CycleOvulation is the marker-based ovulation of the real cycle that starts at Periods()[i] (Prepare part 2,
// §2.17, E-16). It is today-independent: a cycle uses every reading and test dated from its start to the day before
// the next real start (the current cycle: every later one). A reading or test on a menstrual day counts.
type CycleOvulation struct {
	Date     Day             // the ovulation used: BBT, else LH; NoDay = neither (the calendar estimate applies)
	Source   OvulationSource // bbt_confirmed | lh_positive | unknown (→ calendar at resolve)
	BBT      Day             // the BBT (3-over-6) ovulation, NoDay when the shift is not confirmed
	LH       Day             // the first positive LH test + 1, NoDay without a positive test
	Conflict bool            // both exist and |LH − BBT| > lh_bbt_conflict.days: BBT wins, warnings += lh_bbt_conflict
}

// prepareMarkers sorts Prepare's copies of the BBT readings (one per day: the later input record of a day wins, as
// for periods, E-02) and of the LH tests by date, dropping undated ones.
func (s *State) prepareMarkers() {
	r := slices.DeleteFunc(s.in.BBT, func(r BBTReading) bool { return r.Date == NoDay })
	slices.SortStableFunc(r, func(a, b BBTReading) int { return cmp.Compare(a.Date, b.Date) })
	out := r[:0]
	for _, x := range r {
		if n := len(out); n > 0 && out[n-1].Date == x.Date {
			out[n-1] = x
			continue
		}
		out = append(out, x)
	}
	s.bbt = out

	s.lh = slices.DeleteFunc(s.in.LH, func(t LHTest) bool { return t.Date == NoDay })
	slices.SortStableFunc(s.lh, func(a, b LHTest) int { return cmp.Compare(a.Date, b.Date) })
	s.in.BBT, s.in.LH = s.bbt, s.lh // the copies were sorted in place
}

// prepareOvulations is Prepare part 2: per real cycle the BBT and LH ovulation, priority BBT > LH > calendar and
// the LH/BBT conflict (§2.17, E-16); then effective_luteal_length (§2.16, E-16).
func (s *State) prepareOvulations() {
	s.prepareMarkers()
	if len(s.periods) > 0 {
		s.ovulations = make([]CycleOvulation, len(s.periods))
	}
	for i := range s.periods {
		from, to := s.periods[i].Start, NoDay
		if i+1 < len(s.periods) {
			to = s.periods[i+1].Start
		}
		s.ovulations[i] = s.cycleOvulation(from, to)
	}
	s.prepareLuteal()
}

// cycleOvulation detects the ovulation of the cycle [from, to) (to = NoDay: open-ended).
func (s *State) cycleOvulation(from, to Day) CycleOvulation {
	o := CycleOvulation{Date: NoDay, BBT: NoDay, LH: NoDay}

	if r := bbt.Detect(between(s.bbt, from, to, func(r BBTReading) Day { return r.Date }), s.p.BBT); r.OK {
		o.BBT = r.Ovulation
	}
	for _, t := range between(s.lh, from, to, func(t LHTest) Day { return t.Date }) {
		if t.Positive { // E-16: only positive; faint/negative arrive as not positive
			o.LH = t.Date + 1
			break
		}
	}

	switch {
	case o.BBT != NoDay:
		o.Date, o.Source = o.BBT, OvulationBBTConfirmed
		if o.LH != NoDay {
			gap := int(o.LH - o.BBT)
			o.Conflict = max(gap, -gap) > s.p.LHBBTConflictDays
		}
	case o.LH != NoDay:
		o.Date, o.Source = o.LH, OvulationLHPositive
	}
	return o
}

// between is the sub-slice of a date-sorted slice with from ≤ date < to (to = NoDay: no upper bound).
func between[T any](v []T, from, to Day, date func(T) Day) []T {
	cmpDate := func(x T, d Day) int { return cmp.Compare(date(x), d) }
	lo, _ := slices.BinarySearchFunc(v, from, cmpDate)
	hi := len(v)
	if to != NoDay {
		hi, _ = slices.BinarySearchFunc(v, to, cmpDate)
	}
	return v[lo:max(lo, hi)]
}

// prepareLuteal is effective_luteal_length (§2.16, E-16): the §2.9 median of the last ≤ HistoryWindow BBT-confirmed
// luteal lengths (next real start − BBT ovulation) within Luteal.Min..Luteal.Max, else Luteal.Default. LH-only cycles
// and skipped cycles (E-15) contribute nothing.
func (s *State) prepareLuteal() {
	s.luteal, s.lutealSrc = s.p.Luteal.Default, LengthDefault
	w := max(s.p.HistoryWindow, 1)
	var buf [8]int
	lens := buf[:0]
	for i := len(s.cycles) - 1; i >= 0 && len(lens) < w; i-- {
		o := s.ovulations[i]
		if s.cycles[i].Skipped || o.BBT == NoDay {
			continue
		}
		if n := int(s.cycles[i].NextStart - o.BBT); n >= s.p.Luteal.Min && n <= s.p.Luteal.Max {
			lens = append(lens, n)
		}
	}
	if len(lens) > 0 {
		s.luteal, s.lutealSrc = median(lens), LengthHistory
	}
}

// Ovulations are the per-cycle marker ovulations, parallel to Periods() (do not modify).
func (s *State) Ovulations() []CycleOvulation { return s.ovulations }

// LutealLength is effective_luteal_length and its source (history = BBT-confirmed cycles, default = Luteal.Default).
func (s *State) LutealLength() (int, LengthSource) { return s.luteal, s.lutealSrc }

// BBT are the BBT readings sorted by date, one per day (do not modify).
func (s *State) BBT() []BBTReading { return s.bbt }

// LH are the LH tests sorted by date (do not modify).
func (s *State) LH() []LHTest { return s.lh }
