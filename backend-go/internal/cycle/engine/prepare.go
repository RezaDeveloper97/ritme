package engine

import (
	"cmp"
	"slices"

	"github.com/ritme/backend-go/internal/platform/phpround"
)

// Cycle is one real cycle: from a real start to the next real start.
type Cycle struct {
	Start     Day
	NextStart Day
	Length    int  // NextStart − Start
	Skipped   bool // contains a recorded skip (E-15): neither valid nor an outlier
	Short     bool // Length < CycleMin (short_cycle_outlier)
	Long      bool // Length > CycleMax (long_cycle_outlier)
	Valid     bool // CycleMin ≤ Length ≤ CycleMax and not skipped (§2.8)
}

// State is the today-independent preparation of one user's input (README §2.2). It never depends on "today", so
// it can be cached per (input revision, params fingerprint). A State is read-only after Prepare.
type State struct {
	p  Params
	in Input // the caller's input with every slice copied; in.Periods is the raw (unsorted) copy

	periods       []Period // consistent real periods, sorted by start
	excluded      []Period // records ignored because they conflict with an older one (E-02)
	contradictory bool
	skips         []Day // sorted, unique
	cycles        []Cycle

	validCycles  int // number of valid cycle lengths in the whole history
	validPeriods int // number of valid (closed, PeriodMin..PeriodMax) period lengths

	cycleLen, periodLen int
	cycleSrc, periodSrc LengthSource
	variability         Opt[int]

	// Declared dates, NoDay when absent (Opt none or Some(NoDay)).
	lmp, birth, delivery Day

	bbt        []BBTReading     // sorted by date, one per day
	lh         []LHTest         // sorted by date
	ovulations []CycleOvulation // parallel to periods (Prepare part 2)
	luteal     int              // effective_luteal_length
	lutealSrc  LengthSource

	// Quality bases (Prepare part 3, quality.go).
	freqBasis                 Opt[int]      // clinical_flags.frequency basis (E-06)
	duration                  Opt[Duration] // clinical_flags.duration (E-07)
	shortOutlier, longOutlier bool          // a non-skipped real cycle below / above the supported range (§2.7)
	onlyOutliers              bool          // real cycles exist and none is valid (§2.28 poor)

	// Skip anchors (Prepare part 5, anchors.go, E-15).
	skipAnchors  []Day // the skips that match a predicted start, sorted
	ignoredSkips []Day // the skips that match none (Diagnostics)
}

// Prepare builds the State. Part 1 (T-M9-15): sort and validate periods (E-02), real cycles and skips (§2.8,
// E-15), outliers, effective lengths (§2.9, E-20) and cycle variability (§2.27). Part 2 (T-M9-17): BBT and LH
// ovulation per real cycle with the LH/BBT conflict (§2.17) and effective_luteal_length (§2.16, E-16). Part 3
// (T-M9-18): the clinical_flags bases and outlier signals of the quality layer. Part 4 (T-M9-19): the positive
// pregnancy tests (E-26). Part 5 (T-M9-20): the skip anchors (E-15). It computes over all stored records and does not depend on any target date. It never panics on malformed input.
func Prepare(in Input, p Params) *State {
	s := &State{p: p, in: cloneInput(in)}
	s.lmp = dayOr(s.in.Declared.LastPeriodStart)
	s.birth = dayOr(s.in.BirthDate)
	s.delivery = dayOr(s.in.DeliveryDate)
	s.preparePeriods()
	s.prepareSkips()
	s.prepareCycles()
	s.prepareLengths()
	s.prepareOvulations()
	s.prepareQuality()
	s.preparePregnancyTests()
	s.prepareSkipAnchors()
	return s
}

// dayOr is the date of an optional day, NoDay when absent.
func dayOr(o Opt[Day]) Day { return o.Or(NoDay) }

func cloneInput(in Input) Input {
	in.Periods = slices.Clone(in.Periods)
	in.Skips = slices.Clone(in.Skips)
	in.LH = slices.Clone(in.LH)
	in.BBT = slices.Clone(in.BBT)
	in.PregnancyTests = slices.Clone(in.PregnancyTests)
	return in
}

// preparePeriods sorts the real periods by start (input order breaks ties, so the later record of two equal
// starts is the newer one) and drops conflicting records (E-02): an end before its start, a second open period,
// and a start on or before the end of the previous kept (closed) period — overlap and "a start inside another
// period" are the same test once starts are sorted. The newer record of a conflicting pair is the one ignored.
// A start after an older *open* period is not a conflict: the open period simply has no logged end.
func (s *State) preparePeriods() {
	recs := make([]Period, 0, len(s.in.Periods))
	for _, r := range s.in.Periods {
		if r.End == 0 && r.Start > 0 {
			// A zero End is the Day zero value (1970-01-01), never a real end after a later start: treat it as
			// "no end logged" instead of an end-before-start contradiction (loader trap, T-M9-17).
			r.End = NoDay
		}
		if r.Start != NoDay {
			recs = append(recs, r)
		}
	}
	slices.SortStableFunc(recs, func(a, b Period) int { return cmp.Compare(a.Start, b.Start) })

	kept := recs[:0]
	open := false
	for _, r := range recs {
		conflict := false
		switch {
		case !r.Open() && r.End < r.Start:
			conflict = true
		case r.Open() && open:
			conflict = true
		case len(kept) > 0:
			prev := kept[len(kept)-1]
			conflict = r.Start == prev.Start || (!prev.Open() && r.Start <= prev.End)
		}
		if conflict {
			s.contradictory = true
			s.excluded = append(s.excluded, r)
			continue
		}
		kept = append(kept, r)
		open = open || r.Open()
	}
	s.periods = kept
}

func (s *State) prepareSkips() {
	s.skips = slices.DeleteFunc(slices.Clone(s.in.Skips), func(d Day) bool { return d == NoDay })
	slices.Sort(s.skips)
	s.skips = slices.Compact(s.skips)
}

// hasSkipBetween reports whether a skip lies strictly between a and b.
func (s *State) hasSkipBetween(a, b Day) bool {
	i, _ := slices.BinarySearch(s.skips, a+1)
	return i < len(s.skips) && s.skips[i] < b
}

func (s *State) prepareCycles() {
	if len(s.periods) < 2 {
		return
	}
	s.cycles = make([]Cycle, 0, len(s.periods)-1)
	for i := 1; i < len(s.periods); i++ {
		c := Cycle{Start: s.periods[i-1].Start, NextStart: s.periods[i].Start}
		c.Length = int(c.NextStart) - int(c.Start)
		c.Skipped = s.hasSkipBetween(c.Start, c.NextStart)
		if !c.Skipped {
			c.Short = c.Length < s.p.CycleMin
			c.Long = c.Length > s.p.CycleMax
			c.Valid = !c.Short && !c.Long
		}
		if c.Valid {
			s.validCycles++
		}
		s.cycles = append(s.cycles, c)
	}
}

// periodLength is the length of a closed period (0 for an open one).
func periodLength(p Period) int {
	if p.Open() {
		return 0
	}
	return int(p.End) - int(p.Start) + 1
}

func (s *State) validPeriodLength(n int) bool { return n >= s.p.PeriodMin && n <= s.p.PeriodMax }

func (s *State) prepareLengths() {
	w := max(s.p.HistoryWindow, 1)

	// Last ≤ w valid cycle lengths, oldest first.
	cyc := make([]int, 0, w)
	for i := len(s.cycles) - 1; i >= 0 && len(cyc) < w; i-- {
		if s.cycles[i].Valid {
			cyc = append(cyc, s.cycles[i].Length)
		}
	}
	slices.Reverse(cyc)

	per := make([]int, 0, w)
	for i := len(s.periods) - 1; i >= 0; i-- {
		if n := periodLength(s.periods[i]); s.validPeriodLength(n) {
			s.validPeriods++
			if len(per) < w {
				per = append(per, n)
			}
		}
	}

	if len(cyc) >= 2 {
		s.variability = Some(slices.Max(cyc) - slices.Min(cyc))
	}

	d := s.in.Declared
	validCycle := func(n int) bool { return n >= s.p.CycleMin && n <= s.p.CycleMax }
	s.cycleLen, s.cycleSrc = effective(cyc, d.CycleLength, d.Manual, validCycle, s.p.DefaultCycleLength)
	s.periodLen, s.periodSrc = effective(per, d.PeriodLength, d.Manual, s.validPeriodLength, s.p.DefaultPeriodLength)
}

// effective is §2.9: a manual declared value (E-20) wins; else the half-up median of the history window; else the
// declared value; else the default. A declared value outside the valid range is ignored.
func effective(history []int, declared Opt[int], manual bool, valid func(int) bool, def int) (int, LengthSource) {
	dv, ok := declared.Get()
	ok = ok && valid(dv)
	switch {
	case ok && manual:
		return dv, LengthDeclared
	case len(history) > 0:
		return median(history), LengthHistory
	case ok:
		return dv, LengthDeclared
	default:
		return def, LengthDefault
	}
}

// median of a small non-empty slice; an even count averages the two middle values and rounds half up
// (PHP round, §2.9: median(28, 29) = 28.5 → 29). The slice is sorted in place.
func median(v []int) int {
	slices.Sort(v)
	mid := len(v) / 2
	if len(v)%2 == 1 {
		return v[mid]
	}
	return int(phpround.Round(float64(v[mid-1]+v[mid])/2, 0))
}

// Params is the parameter set the state was prepared with.
func (s *State) Params() Params { return s.p }

// Mode is the life mode of the input (the legacy daily card drops its fertility copy in teen and menopause, T-M9-32).
func (s *State) Mode() Mode { return s.in.Mode }

// Declared is the onboarding/profile data of the input (the legacy profile_values block, T-M9-32).
func (s *State) Declared() Declared { return s.in.Declared }

// Periods are the consistent real periods, sorted by start (do not modify).
func (s *State) Periods() []Period { return s.periods }

// Excluded are the conflicting period records Prepare ignored (E-02).
func (s *State) Excluded() []Period { return s.excluded }

// Contradictory reports contradictory_anchor_data (E-02).
func (s *State) Contradictory() bool { return s.contradictory }

// Skips are the recorded skipped-cycle dates, sorted and unique.
func (s *State) Skips() []Day { return s.skips }

// Cycles are the real cycles between consecutive real starts, oldest first (do not modify).
func (s *State) Cycles() []Cycle { return s.cycles }

// CycleLength is effective_cycle_length and its source.
func (s *State) CycleLength() (int, LengthSource) { return s.cycleLen, s.cycleSrc }

// PeriodLength is effective_period_length and its source.
func (s *State) PeriodLength() (int, LengthSource) { return s.periodLen, s.periodSrc }

// Variability is cycle_variability (§2.27): max − min of the last ≤ HistoryWindow valid cycle lengths, none with
// fewer than 2.
func (s *State) Variability() Opt[int] { return s.variability }

// ValidCycleCount is the number of valid cycle lengths in the whole history.
func (s *State) ValidCycleCount() int { return s.validCycles }

// ValidPeriodCount is the number of valid period lengths in the whole history.
func (s *State) ValidPeriodCount() int { return s.validPeriods }
