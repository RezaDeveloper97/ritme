// Package labor is the contraction maths of the contraction timer (bloom B-N5-03, nbl_Log_Contraction): per-
// contraction duration and interval, the averages of the last hour and the 5-1-1 rule — contractions about every 5
// minutes, each lasting about 1 minute, for 1 hour. Pure functions, no I/O: the pregnancy tools service and the
// pregnancy alert engine (rule contractions_511) both call it, with the thresholds read from the rule's
// admin-editable params.
package labor

import (
	"fmt"
	"sort"
	"time"
)

// Contraction is one timed contraction; a zero End means it is still in progress.
type Contraction struct {
	Start time.Time
	End   time.Time
}

// Done reports whether the contraction has ended.
func (c Contraction) Done() bool { return !c.End.IsZero() }

// Duration is End − Start (0 while in progress).
func (c Contraction) Duration() time.Duration {
	if !c.Done() || c.End.Before(c.Start) {
		return 0
	}
	return c.End.Sub(c.Start)
}

// Params are the 5-1-1 thresholds.
type Params struct {
	IntervalMax time.Duration // average start-to-start interval at most this
	DurationMin time.Duration // average duration at least this ("about one minute")
	Run         time.Duration // sustained for at least this long
}

// Defaults of the rule (the seeded contractions_511 row; [needs clinical review]).
const (
	DefaultIntervalMaxMinutes = 5
	DefaultDurationMinSeconds = 45
	DefaultRunMinutes         = 60
)

// DefaultParams are the seeded thresholds.
func DefaultParams() Params {
	return Params{
		IntervalMax: DefaultIntervalMaxMinutes * time.Minute,
		DurationMin: DefaultDurationMinSeconds * time.Second,
		Run:         DefaultRunMinutes * time.Minute,
	}
}

// completed are the ended contractions sorted by start.
func completed(cs []Contraction) []Contraction {
	out := make([]Contraction, 0, len(cs))
	for _, c := range cs {
		if c.Done() {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// Intervals are the start-to-start intervals of cs sorted by start: Intervals(cs)[i] is the gap before the i-th
// contraction (0 for the first, which has none).
func Intervals(cs []Contraction) []time.Duration {
	sorted := append([]Contraction(nil), cs...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Start.Before(sorted[j].Start) })
	out := make([]time.Duration, len(sorted))
	for i := 1; i < len(sorted); i++ {
		out[i] = sorted[i].Start.Sub(sorted[i-1].Start)
	}
	return out
}

// Stats are averages over a set of ended contractions; zero values mean "not enough data".
type Stats struct {
	Count       int
	AvgDuration time.Duration // 0 when Count == 0
	AvgInterval time.Duration // 0 when Count < 2
}

func stats(cs []Contraction) Stats {
	s := Stats{Count: len(cs)}
	if len(cs) == 0 {
		return s
	}
	var total time.Duration
	for _, c := range cs {
		total += c.Duration()
	}
	s.AvgDuration = total / time.Duration(len(cs))
	if len(cs) > 1 {
		s.AvgInterval = cs[len(cs)-1].Start.Sub(cs[0].Start) / time.Duration(len(cs)-1)
	}
	return s
}

// Recent are the averages of the ended contractions that started within window before the latest one (the
// «میانگین مدت / میانگین فاصله» of the timer).
func Recent(cs []Contraction, window time.Duration) Stats {
	done := completed(cs)
	if len(done) == 0 {
		return Stats{}
	}
	from := done[len(done)-1].Start.Add(-window)
	i := sort.Search(len(done), func(i int) bool { return !done[i].Start.Before(from) })
	return stats(done[i:])
}

// Result is the 5-1-1 verdict on the latest run of contractions.
type Result struct {
	Met bool
	// The trailing run: the latest contractions without a pause longer than twice IntervalMax, cut to the shortest
	// tail that spans Run (or the whole run when it is shorter).
	Count       int
	Span        time.Duration // first start → last end of the tail
	AvgInterval time.Duration
	AvgDuration time.Duration
	MetAt       time.Time // the last contraction's end when Met
}

// FiveOneOne checks the 5-1-1 rule on the ended contractions of cs: over a tail of the latest run spanning at least
// Run, the average interval is at most IntervalMax and the average duration at least DurationMin. A pause longer
// than twice IntervalMax starts a new run, so an earlier burst does not count.
func FiveOneOne(cs []Contraction, p Params) Result {
	done := completed(cs)
	if len(done) < 2 || p.Run <= 0 {
		r := Result{Count: len(done)}
		if len(done) == 1 {
			r.Span, r.AvgDuration = done[0].Duration(), done[0].Duration()
		}
		return r
	}
	last := len(done) - 1
	first := last
	pause := 2 * p.IntervalMax
	for first > 0 {
		if p.IntervalMax > 0 && done[first].Start.Sub(done[first-1].Start) > pause {
			break
		}
		if done[last].End.Sub(done[first].Start) >= p.Run {
			break // the shortest tail spanning Run
		}
		first--
	}
	tail := done[first:]
	s := stats(tail)
	r := Result{
		Count:       len(tail),
		Span:        done[last].End.Sub(done[first].Start),
		AvgInterval: s.AvgInterval,
		AvgDuration: s.AvgDuration,
	}
	r.Met = len(tail) >= 2 && r.Span >= p.Run && s.AvgInterval <= p.IntervalMax && s.AvgDuration >= p.DurationMin
	if r.Met {
		r.MetAt = done[last].End
	}
	return r
}

// Clock formats d as m:ss (h:mm:ss from an hour), the timer's notation.
func Clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int64(d / time.Second)
	if h := sec / 3600; h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, sec/60%60, sec%60)
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}
